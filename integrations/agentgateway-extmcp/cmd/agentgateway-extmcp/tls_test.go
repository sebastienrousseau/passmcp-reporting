// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
)

// keyPair writes a self-signed certificate for 127.0.0.1 and its key as
// PEM files in dir, and returns their paths and the certificate.
func keyPair(t *testing.T, dir, name string) (certFile, keyFile string, cert *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "agentgateway-extmcp test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if cert, err = x509.ParseCertificate(der); err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, name+".crt")
	keyFile = filepath.Join(dir, name+".key")
	writePEM(t, certFile, "CERTIFICATE", der)
	writePEM(t, keyFile, "PRIVATE KEY", keyDER)
	return certFile, keyFile, cert
}

func writePEM(t *testing.T, path, kind string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

// serveTLS starts the command with a TLS listener and returns its address
// and the certificate it serves. The server stops when the test ends.
func serveTLS(t *testing.T) (net.Addr, *x509.Certificate, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	certFile, keyFile, cert := keyPair(t, dir, "server")
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	stderr := &bytes.Buffer{}
	args := []string{"-config", cfg, "-listen", "127.0.0.1:0", "-tls-cert", certFile, "-tls-key", keyFile}
	go func() { done <- run(ctx, args, stderr, ready) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run: %v", err)
		}
	})
	select {
	case addr := <-ready:
		return addr, cert, stderr
	case err := <-done:
		t.Fatalf("run returned early: %v", err)
		return nil, nil, nil
	}
}

// check makes one CheckRequest over conn options and returns its error.
func check(t *testing.T, addr net.Addr, opt grpc.DialOption) error {
	t.Helper()
	conn, err := grpc.NewClient(addr.String(), opt)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := extmcp.NewExtMcpClient(conn).CheckRequest(ctx, &extmcp.McpRequest{ServiceNames: []string{"mcp"}, Method: "tools/call"})
	if err == nil && res.GetPass() == nil {
		t.Fatalf("got %v, want a pass", res)
	}
	return err
}

func trusting(cert *x509.Certificate, maxVersion uint16) grpc.DialOption {
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS10, // offered on purpose, to see the server refuse it
		MaxVersion: maxVersion,
	}))
}

func TestRunServesOverTLS(t *testing.T) {
	addr, cert, _ := serveTLS(t)
	for name, v := range map[string]uint16{"TLS 1.2": tls.VersionTLS12, "TLS 1.3": tls.VersionTLS13} {
		if err := check(t, addr, trusting(cert, v)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestTLSListenerRefusesOldVersionsAndPlaintext(t *testing.T) {
	addr, cert, _ := serveTLS(t)
	if err := check(t, addr, trusting(cert, tls.VersionTLS11)); err == nil {
		t.Error("a TLS 1.1 client was served")
	}
	if err := check(t, addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
		t.Error("a plaintext client was served by a TLS listener")
	}
}

func TestServerTLSRefusesAnIncompletePair(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := keyPair(t, dir, "one")
	_, otherKey, _ := keyPair(t, dir, "two")
	for name, tc := range map[string]struct {
		cert, key, want string
	}{
		"certificate alone":  {certFile, "", "must be given together"},
		"key alone":          {"", keyFile, "must be given together"},
		"missing file":       {filepath.Join(dir, "absent.crt"), keyFile, "-tls-cert/-tls-key"},
		"key of another one": {certFile, otherKey, "-tls-cert/-tls-key"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := serverTLS(tc.cert, tc.key); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
	// Neither file is not a request for plaintext: that takes -plaintext.
	if opts, kp, err := serverTLS("", ""); err == nil || opts != nil || kp != nil {
		t.Errorf("no TLS flags: opts = %v, pair = %v, err = %v; want a refusal naming -plaintext", opts, kp, err)
	} else if !strings.Contains(err.Error(), "-plaintext") {
		t.Errorf("err = %v, want it to name -plaintext", err)
	}
}

func TestTLSConfigFloorIsTLS12(t *testing.T) {
	certFile, keyFile, _ := keyPair(t, t.TempDir(), "server")
	cfg, _, err := tlsConfig(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %#x, want TLS 1.2 (%#x)", cfg.MinVersion, tls.VersionTLS12)
	}
	if cfg.GetCertificate == nil {
		t.Fatal("no GetCertificate: the key pair could not be swapped on reload")
	}
	got, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil || got == nil || len(got.Certificate) != 1 {
		t.Errorf("GetCertificate = %v, %v; want the one configured", got, err)
	}
}

// startTLS runs the command with the key pair at certFile and keyFile and
// extra flags, and returns its address and its diagnostics.
func startTLS(t *testing.T, certFile, keyFile string, extra ...string) (net.Addr, *lockedBuffer) {
	t.Helper()
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	stderr := &lockedBuffer{}
	args := append([]string{"-config", cfg, "-listen", "127.0.0.1:0", "-tls-cert", certFile, "-tls-key", keyFile}, extra...)
	go func() { done <- run(ctx, args, stderr, ready) }()
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
		t.Fatalf("run returned early: %v", err)
		return nil, nil
	}
}

// served opens a new TLS connection to addr, trusting roots, and returns
// the leaf certificate the listener presented.
func served(t *testing.T, addr net.Addr, roots ...*x509.Certificate) *x509.Certificate {
	t.Helper()
	pool := x509.NewCertPool()
	for _, c := range roots {
		pool.AddCert(c)
	}
	conn, err := tls.Dial("tcp", addr.String(), &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12, NextProtos: []string{"h2"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	return conn.ConnectionState().PeerCertificates[0]
}

// copyFile replaces dst's contents with src's, as a certificate manager
// rotating a key pair in place does.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// eventually polls cond until it holds or five seconds pass.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("%s never happened", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestTLSKeyPairReloads. A certificate rotated on disk is served to the
// next connection without a restart; a key pair that does not load is
// logged and the certificate already in service stays. The reload here is
// -reload-interval's, so the test runs on every OS; TestSIGHUPReloadsTheKeyPair
// sends the signal.
func TestTLSKeyPairReloads(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, first := keyPair(t, dir, "server")
	secondCert, secondKey, second := keyPair(t, dir, "second")
	addr, stderr := startTLS(t, certFile, keyFile, "-reload-interval", "20ms")
	if got := served(t, addr, first, second); !got.Equal(first) {
		t.Fatal("the listener does not serve the certificate it was started with")
	}

	// A key that does not match the certificate: refused, and the first
	// certificate stays in service.
	copyFile(t, secondKey, keyFile)
	eventually(t, "a logged TLS reload failure", func() bool {
		return strings.Contains(stderr.String(), "keeping the previous certificate")
	})
	if !strings.Contains(stderr.String(), "private key does not match public key") {
		t.Errorf("the log does not name the failure:\n%s", stderr.String())
	}
	if got := served(t, addr, first, second); !got.Equal(first) {
		t.Fatal("a key pair that does not load replaced the certificate in service")
	}

	// The matching certificate arrives: the next connection sees it.
	copyFile(t, secondCert, certFile)
	eventually(t, "the second certificate being served", func() bool {
		return served(t, addr, first, second).Equal(second)
	})
}

func TestRunRefusesAHalfTLSConfigurationBeforeListening(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	certFile, _, _ := keyPair(t, dir, "server")
	var stderr bytes.Buffer
	// A run that wrongly starts serving ends with this context, and fails
	// below, instead of hanging the test binary.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := run(ctx, []string{"-config", cfg, "-listen", "127.0.0.1:0", "-tls-cert", certFile}, &stderr, nil)
	if err == nil || !strings.Contains(err.Error(), "must be given together") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(stderr.String(), "listening") {
		t.Errorf("listened before refusing:\n%s", stderr.String())
	}
}
