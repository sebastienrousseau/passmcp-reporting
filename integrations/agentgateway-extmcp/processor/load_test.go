// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"bytes"
	"context"
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
