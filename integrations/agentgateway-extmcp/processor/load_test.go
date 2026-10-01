// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoaderReadsFilesAndHTTPS(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"hello":"world"}`)
	path := writeFile(t, dir, "s.json", body)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write(body)
		case "/big":
			_, _ = w.Write(bytes.Repeat([]byte("x"), 100))
		default:
			http.NotFound(w, r)
		}
	}))
	// Closing the server mid-handshake on a kept-alive connection logs a
	// line that means nothing here.
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	l := &Loader{Client: srv.Client(), MaxBytes: 64}

	for name, tc := range map[string]struct {
		source string
		want   string // substring of the error, or "" for success
	}{
		"file":            {path, ""},
		"https":           {srv.URL + "/ok", ""},
		"https 404":       {srv.URL + "/missing", "404"},
		"https too large": {srv.URL + "/big", "exceeds 64 bytes"},
		"http refused":    {strings.Replace(srv.URL, "https://", "http://", 1) + "/ok", "only https"},
		"other scheme":    {"ftp://example.com/s.json", "only https"},
		"missing file":    {dir + "/absent.json", "no such file"},
		"file too large":  {writeFile(t, dir, "big.json", bytes.Repeat([]byte("x"), 65)), "exceeds 64 bytes"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := l.Load(context.Background(), tc.source)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, body) {
					t.Fatalf("got %q", got)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestLoaderDefaultsAreApplied(t *testing.T) {
	l := &Loader{}
	if l.maxBytes() != DefaultMaxBytes {
		t.Errorf("maxBytes = %d", l.maxBytes())
	}
	// A cancelled context must stop a fetch before it starts, whichever
	// client is in use.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Load(ctx, "https://127.0.0.1:1/never"); err == nil {
		t.Error("fetch with a cancelled context succeeded")
	}
}

// A redirect is followed only while it stays on https. The configured URL
// being https is the whole of the "not swapped in transit" guarantee, and
// a redirect to http:// would quietly give it up.
func TestLoaderRefusesARedirectOffHTTPS(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(plain.Close)
	var secure *httptest.Server
	secure = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/to-http":
			http.Redirect(w, r, plain.URL+"/s.json", http.StatusFound)
		case "/to-https":
			http.Redirect(w, r, secure.URL+"/s.json", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, secure.URL+"/loop", http.StatusFound)
		default:
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(secure.Close)
	injected := secure.Client()
	l := &Loader{Client: injected}

	if _, err := l.Load(context.Background(), secure.URL+"/to-http"); err == nil || !strings.Contains(err.Error(), "only https") {
		t.Errorf("redirect to http: err = %v, want a refusal", err)
	}
	if got, err := l.Load(context.Background(), secure.URL+"/to-https"); err != nil || !bytes.Equal(got, body) {
		t.Errorf("redirect within https: got %q, err = %v", got, err)
	}
	if _, err := l.Load(context.Background(), secure.URL+"/loop"); err == nil || !strings.Contains(err.Error(), "redirects") {
		t.Errorf("redirect loop: err = %v, want it stopped", err)
	}
	if injected.CheckRedirect != nil {
		t.Error("the injected client was modified")
	}

	// An injected client's own redirect policy still applies after the
	// https rule.
	strict := secure.Client()
	strict.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("no redirects here") }
	if _, err := (&Loader{Client: strict}).Load(context.Background(), secure.URL+"/to-https"); err == nil || !strings.Contains(err.Error(), "no redirects here") {
		t.Errorf("injected policy: err = %v", err)
	}
}
