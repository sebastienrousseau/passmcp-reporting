// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
)

const testEndpoint = "https://mcp.example.com/mcp"

// valid mirrors the helper in the root module's attestation_test.go: a
// statement that passes Validate, about testEndpoint, scoring 88 with one
// failing protocol check.
func valid() *attestation.Statement {
	t := attestation.Target{Transport: "http", Endpoint: testEndpoint}
	return &attestation.Statement{
		Type:          attestation.StatementType,
		PredicateType: attestation.PredicateType,
		Subject:       []attestation.Subject{attestation.SubjectFor(t)},
		Predicate: attestation.Evaluation{
			SubjectKind:   attestation.SubjectKindDescriptor,
			Target:        t,
			JudgedAgainst: attestation.Basis{SpecRevision: "2026-07-28", Rubric: "1", CheckInventory: "1"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.3", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
			Took:          "3.2s",
			Verdicts: []attestation.Verdict{
				{ID: "auth.unauthenticated_tools", Phase: "auth", Status: "pass"},
				{ID: "protocol.origin", Phase: "protocol", Status: "fail", Severity: "major", Evidence: []string{"req#12"}},
			},
			Counts: attestation.Counts{Pass: 1, Fail: 1},
			Score:  &attestation.Score{Total: 88, Grade: "B", Assessed: 6, Of: 6},
		},
	}
}

// quiet is a logger that discards everything; tests assert on results,
// not log lines.
var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// writeStatement writes st to a file under dir and returns its path.
func writeStatement(t *testing.T, dir, name string, st *attestation.Statement) string {
	t.Helper()
	b, err := st.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return writeFile(t, dir, name, b)
}

func writeFile(t *testing.T, dir, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeConfig writes cfg as JSON under dir and returns its path.
func writeConfig(t *testing.T, dir string, cfg Config) string {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return writeFile(t, dir, "config.json", b)
}

// loadedStore builds a store over cfg with every attestation loaded.
func loadedStore(t *testing.T, dir string, cfg Config) *Store {
	t.Helper()
	s := NewStore(writeConfig(t, dir, cfg), &Loader{}, quiet)
	if err := s.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

// serve runs srv on an in-process listener and returns a client to it.
func serve(t *testing.T, srv extmcp.ExtMcpServer) extmcp.ExtMcpClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer()
	extmcp.RegisterExtMcpServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return extmcp.NewExtMcpClient(conn)
}

func boolPtr(b bool) *bool { return &b }
