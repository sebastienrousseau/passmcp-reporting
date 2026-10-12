// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// refused runs the command with args, expects it to stop before listening,
// and returns the error. A run that wrongly starts serving ends with the
// context and fails here instead of hanging the test binary.
func refused(t *testing.T, args ...string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var stderr bytes.Buffer
	err := run(ctx, args, &stderr, nil)
	if strings.Contains(stderr.String(), "listening") {
		t.Errorf("listened before refusing:\n%s", stderr.String())
	}
	if err == nil {
		t.Fatalf("%v: started; want a refusal", args)
	}
	return err
}

// wantMentions fails unless err's message contains every one of words.
func wantMentions(t *testing.T, err error, words ...string) {
	t.Helper()
	for _, w := range words {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("err = %q, want it to mention %q", err, w)
		}
	}
}

func TestRunRefusesToStartWithoutAKeyPair(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	for name, listen := range map[string]string{
		"default address": "",
		"loopback":        "127.0.0.1:0",
		"all interfaces":  "0.0.0.0:0",
	} {
		t.Run(name, func(t *testing.T) {
			args := []string{"-config", cfg}
			if listen != "" {
				args = append(args, "-listen", listen)
			}
			wantMentions(t, refused(t, args...), "-tls-cert", "-tls-key", "-plaintext")
		})
	}
}

// startPlaintext runs the command with -plaintext on listen and returns
// the bound address and the diagnostics. The server stops when the test
// ends.
func startPlaintext(t *testing.T, listen string) (net.Addr, *lockedBuffer) {
	t.Helper()
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	stderr := &lockedBuffer{}
	go func() { done <- run(ctx, []string{"-config", cfg, "-listen", listen, "-plaintext"}, stderr, ready) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run: %v", err)
		}
	})
	select {
	case addr := <-ready:
		return addr, stderr
	case err := <-done:
		done <- nil // the cleanup waits on done; run has already returned
		t.Fatalf("run returned early: %v", err)
		return nil, nil
	}
}

func TestPlaintextServesOnLoopback(t *testing.T) {
	for _, listen := range []string{"127.0.0.1:0", "[::1]:0", "localhost:0"} {
		t.Run(listen, func(t *testing.T) {
			if strings.HasPrefix(listen, "[::1]") {
				l, err := net.Listen("tcp", "[::1]:0")
				if err != nil {
					t.Skipf("no IPv6 loopback here: %v", err)
				}
				_ = l.Close()
			}
			addr, stderr := startPlaintext(t, listen)
			if err := check(t, addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stderr.String(), `"tls":false`) {
				t.Errorf("the listening line does not say the listener is plaintext:\n%s", stderr.String())
			}
		})
	}
}

// TestPlaintextRefusesANonLoopbackAddress. Only a literal loopback address
// or the name localhost is accepted: a wildcard, a routable address and
// any other host name are refused before anything is bound. A host name
// resolves to whatever DNS or /etc/hosts says on the day, so it is not
// evidence the listener stays on this host.
func TestPlaintextRefusesANonLoopbackAddress(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	for _, listen := range []string{
		"0.0.0.0:4400",
		":4400",
		"[::]:4400",
		"192.0.2.10:4400",
		"[2001:db8::1]:4400",
		"extmcp.internal:4400",
		"localhost.example.com:4400",
		"[::ffff:192.0.2.10]:4400",
	} {
		t.Run(listen, func(t *testing.T) {
			err := refused(t, "-config", cfg, "-listen", listen, "-plaintext")
			wantMentions(t, err, "-plaintext", "loopback", "-tls-cert")
		})
	}
	err := refused(t, "-config", cfg, "-listen", "no-port", "-plaintext")
	wantMentions(t, err, "-listen")
}

func TestPlaintextWithAKeyPairIsRefused(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	certFile, keyFile, _ := keyPair(t, dir, "server")
	for name, extra := range map[string][]string{
		"both":             {"-tls-cert", certFile, "-tls-key", keyFile},
		"certificate only": {"-tls-cert", certFile},
		"key only":         {"-tls-key", keyFile},
	} {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"-config", cfg, "-listen", "127.0.0.1:0", "-plaintext"}, extra...)
			wantMentions(t, refused(t, args...), "-plaintext", "-tls-cert")
		})
	}
}

// TestPlaintextChecksTheBoundAddress. localhost is accepted by name, so
// what it resolved to is checked once bound: a listener that did not land
// on loopback is closed rather than served in the clear.
func TestPlaintextChecksTheBoundAddress(t *testing.T) {
	for addr, ok := range map[net.Addr]bool{
		&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}:  true,
		&net.TCPAddr{IP: net.IPv6loopback, Port: 1}:        true,
		&net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 1}: false,
		&net.TCPAddr{IP: net.IPv4zero, Port: 1}:            false,
		&net.UnixAddr{Name: "/tmp/sock", Net: "unix"}:      false,
	} {
		err := boundToLoopback(addr)
		if (err == nil) != ok {
			t.Errorf("boundToLoopback(%v) = %v, want ok = %v", addr, err, ok)
		}
	}
}

// TestAMissingKeyPairNamesTheOptOut. The container image names the key
// pair in its default arguments, so an operator who mounts none meets
// this error, not errNoKeyPair; it must say what to do as well.
func TestAMissingKeyPairNamesTheOptOut(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	err := refused(t, "-config", cfg, "-listen", "0.0.0.0:0",
		"-tls-cert", dir+"/tls.crt", "-tls-key", dir+"/tls.key")
	wantMentions(t, err, "-tls-cert/-tls-key", "tls.crt", "-plaintext", "loopback")
}
