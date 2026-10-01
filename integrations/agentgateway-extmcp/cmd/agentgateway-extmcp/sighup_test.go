// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"context"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
)

// TestSIGHUPReloads sends the process a real SIGHUP and expects the next
// decision to reflect the statement now on disk.
func TestSIGHUPReloads(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	var stderr lockedBuffer
	go func() { done <- run(ctx, []string{"-config", cfg, "-listen", "127.0.0.1:0"}, &stderr, ready) }()
	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("run returned early: %v", err)
	}
	conn, err := grpc.NewClient(addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := extmcp.NewExtMcpClient(conn)

	statement(t, dir, 10)
	// The handler is installed before ready is sent, so the first signal
	// cannot kill the process; the loop only waits for the reload to land.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
		res, err := c.CheckRequest(ctx, &extmcp.McpRequest{ServiceNames: []string{"mcp"}, Method: "tools/call"})
		if err != nil {
			t.Fatal(err)
		}
		if res.GetError() != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("SIGHUP never reloaded")
		}
	}
	if !strings.Contains(stderr.String(), `"trigger":"SIGHUP"`) {
		t.Errorf("stderr lacks the SIGHUP reload line:\n%s", stderr.String())
	}
}

// TestSIGHUPReloadsTheKeyPair sends a real SIGHUP after the key pair is
// replaced on disk and expects the next connection to see the new
// certificate.
func TestSIGHUPReloadsTheKeyPair(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, first := keyPair(t, dir, "server")
	secondCert, secondKey, second := keyPair(t, dir, "second")
	addr, stderr := startTLS(t, certFile, keyFile)
	copyFile(t, secondCert, certFile)
	copyFile(t, secondKey, keyFile)
	eventually(t, "the second certificate being served after SIGHUP", func() bool {
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
		return served(t, addr, first, second).Equal(second)
	})
	if !strings.Contains(stderr.String(), `"msg":"reloaded the TLS key pair","trigger":"SIGHUP"`) {
		t.Errorf("stderr lacks the TLS reload line:\n%s", stderr.String())
	}
}
