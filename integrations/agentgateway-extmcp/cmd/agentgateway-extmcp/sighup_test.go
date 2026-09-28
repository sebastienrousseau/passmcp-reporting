// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"sync"
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

// lockedBuffer is a bytes.Buffer safe to read while run is still writing
// to it from another goroutine.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
