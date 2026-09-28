// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
)

// TestCheckRequest is the contract: for every way a target can be judged,
// a request about such a target and the answer the gateway must get.
func TestCheckRequest(t *testing.T) {
	dir := t.TempDir()

	good := writeStatement(t, dir, "good.json", valid())

	low := valid()
	low.Predicate.Score = &attestation.Score{Total: 40, Grade: "E", Assessed: 6, Of: 6}
	lowPath := writeStatement(t, dir, "low.json", low)

	unscored := valid()
	unscored.Predicate.Score = nil
	unscoredPath := writeStatement(t, dir, "unscored.json", unscored)

	authFail := valid()
	authFail.Predicate.Verdicts[0].Status = "fail"
	authFail.Predicate.Verdicts[0].Severity = "critical"
	authFail.Predicate.Counts = attestation.Counts{Fail: 2}
	authFailPath := writeStatement(t, dir, "authfail.json", authFail)

	tampered := valid()
	tampered.Predicate.Target.Endpoint = "https://other.example.com/mcp" // digest no longer matches
	tamperedPath := writeStatement(t, dir, "tampered.json", tampered)

	notJSON := writeFile(t, dir, "garbage.json", []byte("not a statement"))

	targets := map[string]Target{
		"good":     {Attestation: good, Endpoint: testEndpoint},
		"low":      {Attestation: lowPath, Endpoint: testEndpoint},
		"unscored": {Attestation: unscoredPath, Endpoint: testEndpoint},
		"authfail": {Attestation: authFailPath, Endpoint: testEndpoint},
		"tampered": {Attestation: tamperedPath, Endpoint: testEndpoint},
		"garbage":  {Attestation: notJSON, Endpoint: testEndpoint},
		"wrongurl": {Attestation: good, Endpoint: "https://elsewhere.example.com/mcp"},
		"stdio":    {Attestation: good, Endpoint: testEndpoint, Transport: "stdio"},
		"absent":   {Attestation: dir + "/absent.json", Endpoint: testEndpoint},
	}
	strict := Config{
		Targets:            targets,
		Policy:             Policy{MinScore: 70, DenyFailIn: []string{"auth"}},
		PassthroughMethods: []string{"initialize"},
	}
	lenient := Config{
		Targets: targets,
		Policy:  Policy{MinScore: 70, DenyFailIn: []string{"auth"}, RequireAttestation: boolPtr(false)},
	}
	protocolToo := Config{
		Targets: targets,
		Policy:  Policy{DenyFailIn: []string{"auth", "protocol"}},
	}
	clients := map[string]extmcp.ExtMcpClient{}
	for name, cfg := range map[string]Config{"strict": strict, "lenient": lenient, "protocolToo": protocolToo} {
		clients[name] = serve(t, NewServer(loadedStore(t, t.TempDir(), cfg), quiet))
	}

	for name, tc := range map[string]struct {
		policy  string
		method  string
		targets []string
		deny    string // substring of the denial reason; "" means Pass
	}{
		"pass":                              {"strict", "tools/call", []string{"good"}, ""},
		"pass with no targets":              {"strict", "ping", nil, ""},
		"pass-through skips the table":      {"strict", "initialize", []string{"unknown"}, ""},
		"deny unknown target":               {"strict", "tools/call", []string{"unknown"}, `target "unknown": no attestation configured`},
		"deny below score":                  {"strict", "tools/call", []string{"low"}, `target "low": score 40 (E) is below the minimum 70`},
		"deny no score":                     {"strict", "tools/call", []string{"unscored"}, `carries no score; policy requires at least 70`},
		"deny fail in category":             {"strict", "tools/call", []string{"authfail"}, `target "authfail": failing checks in category auth (auth.unauthenticated_tools)`},
		"deny invalid statement":            {"strict", "tools/call", []string{"tampered"}, `target "tampered": attestation unusable: invalid:`},
		"deny unparsable statement":         {"strict", "tools/call", []string{"garbage"}, `attestation unusable: invalid:`},
		"deny statement about another url":  {"strict", "tools/call", []string{"wrongurl"}, `does not cover http https://elsewhere.example.com/mcp`},
		"deny statement of other transport": {"strict", "tools/call", []string{"stdio"}, `does not cover stdio`},
		"deny unreadable statement":         {"strict", "tools/call", []string{"absent"}, `attestation unusable: unreadable:`},
		"fanout denies on the first bad":    {"strict", "tools/list", []string{"good", "low", "unknown"}, `target "low"`},
		"fanout passes when all good":       {"strict", "tools/list", []string{"good", "good"}, ""},
		"lenient passes unknown":            {"lenient", "tools/call", []string{"unknown"}, ""},
		"lenient passes unusable":           {"lenient", "tools/call", []string{"tampered"}, ""},
		"lenient still scores":              {"lenient", "tools/call", []string{"low"}, `below the minimum 70`},
		"protocol category":                 {"protocolToo", "tools/call", []string{"good"}, `category protocol (protocol.origin)`},
		"no score floor":                    {"protocolToo", "tools/call", []string{"unscored"}, `category protocol`},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := clients[tc.policy].CheckRequest(context.Background(), &extmcp.McpRequest{
				ServiceNames: tc.targets, Method: tc.method,
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.deny == "" {
				if res.GetPass() == nil {
					t.Fatalf("want Pass, got %v", res)
				}
				return
			}
			e := res.GetError()
			if e == nil {
				t.Fatalf("want a denial mentioning %q, got %v", tc.deny, res)
			}
			if e.GetCode() != extmcp.AuthorizationError_PERMISSION_DENIED {
				t.Errorf("code = %v, want PERMISSION_DENIED", e.GetCode())
			}
			if !strings.Contains(e.GetReason(), tc.deny) {
				t.Errorf("reason = %q, want it to mention %q", e.GetReason(), tc.deny)
			}
		})
	}
}

func TestPassCarriesScoresAsMetadata(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Targets: map[string]Target{"good": {Attestation: writeStatement(t, dir, "s.json", valid()), Endpoint: testEndpoint}}}
	c := serve(t, NewServer(loadedStore(t, dir, cfg), quiet))
	res, err := c.CheckRequest(context.Background(), &extmcp.McpRequest{ServiceNames: []string{"good"}, Method: "tools/call"})
	if err != nil {
		t.Fatal(err)
	}
	passmcp := res.GetMetadata().GetFields()["passmcp"].GetStructValue().GetFields()["good"].GetStructValue().GetFields()
	if passmcp["score"].GetNumberValue() != 88 || passmcp["grade"].GetStringValue() != "B" {
		t.Fatalf("metadata = %v", res.GetMetadata())
	}
	// A request that names no gated target carries no bag at all.
	res, err = c.CheckRequest(context.Background(), &extmcp.McpRequest{Method: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetMetadata() != nil {
		t.Fatalf("metadata = %v, want none", res.GetMetadata())
	}
}

func TestCheckResponseAlwaysPasses(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Targets: map[string]Target{"good": {Attestation: writeStatement(t, dir, "s.json", valid()), Endpoint: testEndpoint}}}
	c := serve(t, NewServer(loadedStore(t, dir, cfg), quiet))
	res, err := c.CheckResponse(context.Background(), &extmcp.McpResponse{ServiceNames: []string{"anything"}, Method: "tools/call"})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetPass() == nil {
		t.Fatalf("got %v", res)
	}
}

// TestUnloadedStoreIsAnErrorNotADecision: before the first successful
// reload the processor must not answer, so the gateway's failureMode
// applies instead of a made-up verdict.
func TestUnloadedStoreIsAnErrorNotADecision(t *testing.T) {
	c := serve(t, NewServer(NewStore(t.TempDir()+"/none.json", nil, nil), nil))
	_, err := c.CheckRequest(context.Background(), &extmcp.McpRequest{ServiceNames: []string{"x"}, Method: "tools/call"})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v, want Unavailable", err)
	}
}

// TestReload: files are read at load time and not again until a reload,
// and a reload that cannot read the configuration keeps the old table.
func TestReload(t *testing.T) {
	dir := t.TempDir()
	path := writeStatement(t, dir, "s.json", valid())
	cfgPath := writeConfig(t, dir, Config{
		Targets: map[string]Target{"good": {Attestation: path, Endpoint: testEndpoint}},
		Policy:  Policy{MinScore: 70},
	})
	store := NewStore(cfgPath, nil, quiet)
	if store.Snapshot() != nil {
		t.Fatal("snapshot before the first reload")
	}
	if err := store.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	c := serve(t, NewServer(store, quiet))
	check := func(want bool) {
		t.Helper()
		res, err := c.CheckRequest(context.Background(), &extmcp.McpRequest{ServiceNames: []string{"good"}, Method: "tools/call"})
		if err != nil {
			t.Fatal(err)
		}
		if got := res.GetPass() != nil; got != want {
			t.Fatalf("allowed = %v, want %v (%v)", got, want, res.GetError().GetReason())
		}
	}
	check(true)

	// The statement on disk gets worse; the running processor does not
	// see it until told to.
	low := valid()
	low.Predicate.Score.Total = 10
	writeStatement(t, dir, "s.json", low)
	check(true)
	if err := store.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	check(false)

	// A broken configuration is refused and the previous table stays.
	writeFile(t, dir, "config.json", []byte("{"))
	if err := store.Reload(context.Background()); err == nil {
		t.Fatal("reload of a broken configuration succeeded")
	}
	check(false)
	if len(store.Snapshot().Statements) != 1 || len(store.Snapshot().Unusable) != 0 {
		t.Fatalf("snapshot = %+v", store.Snapshot())
	}

	// A cancelled context stops a build before it reads anything.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Build(ctx, store.Snapshot().Config, nil, nil); err == nil {
		t.Fatal("build with a cancelled context succeeded")
	}
}

func TestSnapshotRecordsWhyAStatementIsUnusable(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Targets: map[string]Target{
		"ok":  {Attestation: writeStatement(t, dir, "s.json", valid()), Endpoint: testEndpoint},
		"bad": {Attestation: writeFile(t, dir, "bad.json", []byte("{}")), Endpoint: testEndpoint},
	}}
	snap := loadedStore(t, dir, cfg).Snapshot()
	if _, ok := snap.Statements["ok"]; !ok {
		t.Error("ok not verified")
	}
	if why := snap.Unusable["bad"]; !strings.HasPrefix(why, "invalid:") {
		t.Errorf("bad recorded as %q", why)
	}
}
