// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/processor"
)

const endpoint = "https://mcp.example.com/mcp"

func statement(t *testing.T, dir string, score float64) string {
	t.Helper()
	tg := attestation.Target{Transport: "http", Endpoint: endpoint}
	st := &attestation.Statement{
		Type:          attestation.StatementType,
		PredicateType: attestation.PredicateType,
		Subject:       []attestation.Subject{attestation.SubjectFor(tg)},
		Predicate: attestation.Evaluation{
			SubjectKind:   attestation.SubjectKindDescriptor,
			Target:        tg,
			JudgedAgainst: attestation.Basis{SpecRevision: "2026-07-28", Rubric: "1", CheckInventory: "1"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.3", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
			Took:          "1s",
			Verdicts:      []attestation.Verdict{{ID: "auth.unauthenticated_tools", Phase: "auth", Status: "pass"}},
			Counts:        attestation.Counts{Pass: 1},
			Score:         &attestation.Score{Total: score, Grade: "B", Assessed: 6, Of: 6},
		},
	}
	b, err := st.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "statement.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func config(t *testing.T, dir, statementPath string) string {
	t.Helper()
	b, err := json.Marshal(processor.Config{
		Targets: map[string]processor.Target{"mcp": {Attestation: statementPath, Endpoint: endpoint}},
		Policy:  processor.Policy{MinScore: 70},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunServesUntilCancelled(t *testing.T) {
	dir := t.TempDir()
	cfg := config(t, dir, statement(t, dir, 88))
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	var stderr bytes.Buffer
	go func() {
		done <- run(ctx, []string{"-config", cfg, "-listen", "127.0.0.1:0", "-reload-interval", "10ms", "-log-level", "debug"}, &stderr, ready)
	}()
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
	res, err := c.CheckRequest(ctx, &extmcp.McpRequest{ServiceNames: []string{"mcp"}, Method: "tools/call"})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetPass() == nil {
		t.Fatalf("got %v", res)
	}

	// The interval reload picks up a worse statement without a restart.
	statement(t, dir, 10)
	deadline := time.Now().Add(5 * time.Second)
	for {
		res, err = c.CheckRequest(ctx, &extmcp.McpRequest{ServiceNames: []string{"mcp"}, Method: "tools/call"})
		if err != nil {
			t.Fatal(err)
		}
		if res.GetError() != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("interval reload never took effect")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(res.GetError().GetReason(), "below the minimum 70") {
		t.Fatalf("reason = %q", res.GetError().GetReason())
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), `"msg":"listening"`) {
		t.Errorf("stderr lacks the listening line:\n%s", stderr.String())
	}
}

func TestRunRefusesBadInvocations(t *testing.T) {
	dir := t.TempDir()
	good := config(t, dir, statement(t, dir, 88))
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"no config flag":   {nil, "-config is required"},
		"missing config":   {[]string{"-config", dir + "/absent.json"}, "config:"},
		"bad log level":    {[]string{"-config", good, "-log-level", "loud"}, "-log-level"},
		"unknown flag":     {[]string{"-bogus"}, "flag provided but not defined"},
		"unusable address": {[]string{"-config", good, "-listen", "256.256.256.256:1"}, "listen"},
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			err := run(context.Background(), tc.args, &stderr, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
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
