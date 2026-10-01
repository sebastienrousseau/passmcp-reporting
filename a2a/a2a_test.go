// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-reporting/attestation"
)

// valid is a statement this package writes and accepts.
func valid() *Statement {
	target := Target{Transport: "https", Endpoint: "https://agent.example.com",
		Agent: &AgentIdentity{Name: "planner", Version: "1.2.0", ProtocolVersion: "1.0"}}
	return &Statement{
		Type:          attestation.StatementType,
		Subject:       []attestation.Subject{SubjectFor(target)},
		PredicateType: PredicateType,
		Predicate: Evaluation{
			SubjectKind:   SubjectKindDescriptor,
			Target:        target,
			JudgedAgainst: Basis{ProtocolVersion: "1.0", Rubric: "a2a-1", CheckInventory: "0.0.9"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.9", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC),
			Took:          "210ms",
			Card: &Card{URL: "https://agent.example.com/.well-known/agent-card.json",
				Digest: strings.Repeat("b", 64), Signed: true, KeyID: "key-1"},
			Verdicts: []attestation.Verdict{
				{ID: "a2a.card_schema", Phase: "card", Status: "pass", Evidence: []string{"req#1"}},
				{ID: "a2a.card_signature", Phase: "card", Status: "pass", Evidence: []string{"req#1", "req#2"}},
				{ID: "a2a.unauthenticated", Phase: "auth", Status: "fail", Severity: "critical", Evidence: []string{"req#3"}},
				{ID: "a2a.transport", Phase: "net", Status: "pass"},
			},
			Counts: attestation.Counts{Pass: 3, Fail: 1},
			Score:  &attestation.Score{Total: 62, Grade: "D", Assessed: 3, Of: 3},
		},
	}
}

func TestAValidStatementParses(t *testing.T) {
	b, err := valid().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(b)
	if err != nil {
		t.Fatalf("a valid statement did not parse: %v", err)
	}
	if !s.Covers("https://agent.example.com") || s.Covers("https://other.example.com") {
		t.Fatal("Covers does not identify the agent")
	}
	if v, ok := s.VerdictFor("a2a.unauthenticated"); !ok || v.Status != "fail" {
		t.Fatalf("VerdictFor: %+v %v", v, ok)
	}
	if _, ok := s.VerdictFor("absent"); ok {
		t.Fatal("found a verdict that is not there")
	}
}

// An MCP evaluation is refused rather than read as an A2A one, and an A2A
// evaluation is refused by the MCP verifier: the predicate types keep the
// two apart.
func TestThePredicateTypesDoNotCross(t *testing.T) {
	b, err := valid().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attestation.Parse(b); err == nil {
		t.Fatal("the MCP verifier accepted an A2A statement")
	}
	mcp, err := os.ReadFile("../attestation/testdata/statement.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(mcp); err == nil || !strings.Contains(err.Error(), "predicateType") {
		t.Fatalf("the A2A verifier accepted an MCP statement: %v", err)
	}
	mcpDigest := attestation.SubjectFor(attestation.Target{Transport: "http", Endpoint: "https://agent.example.com"}).Digest["sha256"]
	if SubjectFor(Target{Endpoint: "https://agent.example.com"}).Digest["sha256"] == mcpDigest {
		t.Fatal("an agent and an MCP server at one URL share a digest")
	}
}

// Every rule has a statement that breaks it.
func TestValidateRefusesEachBrokenRule(t *testing.T) {
	cases := map[string]func(*Statement){
		"type":            func(s *Statement) { s.Type = "x" },
		"predicate":       func(s *Statement) { s.PredicateType = "x" },
		"no subject":      func(s *Statement) { s.Subject = nil },
		"two subjects":    func(s *Statement) { s.Subject = append(s.Subject, s.Subject[0]) },
		"subject name":    func(s *Statement) { s.Subject[0].Name = "https://other" },
		"no digest":       func(s *Statement) { s.Subject[0].Digest = map[string]string{} },
		"edited":          func(s *Statement) { s.Subject[0].Digest["sha256"] = strings.Repeat("0", 64) },
		"kind":            func(s *Statement) { s.Predicate.SubjectKind = "mcp-target-descriptor" },
		"transport":       func(s *Statement) { s.Predicate.Target.Transport = "stdio" },
		"scheme mismatch": func(s *Statement) { s.Predicate.Target.Transport = "http" },
		"no endpoint":     func(s *Statement) { s.Predicate.Target.Endpoint = ""; s.Subject[0].Name = "" },
		"not a URL":       func(s *Statement) { s.Predicate.Target.Endpoint = "agent"; s.Subject[0].Name = "agent" },
		"instrument":      func(s *Statement) { s.Predicate.Instrument.Version = "" },
		"ran at":          func(s *Statement) { s.Predicate.RanAt = time.Time{} },
		"inventory":       func(s *Statement) { s.Predicate.JudgedAgainst.CheckInventory = "" },
		"rubric":          func(s *Statement) { s.Predicate.JudgedAgainst.Rubric = "" },
		"card url":        func(s *Statement) { s.Predicate.Card.URL = "" },
		"card digest":     func(s *Statement) { s.Predicate.Card.Digest = "zz" },
		"unsigned key":    func(s *Statement) { s.Predicate.Card.Signed = false },
		"no verdicts":     func(s *Statement) { s.Predicate.Verdicts = nil },
		"verdict id":      func(s *Statement) { s.Predicate.Verdicts[0].ID = "" },
		"status":          func(s *Statement) { s.Predicate.Verdicts[0].Status = "maybe" },
		"counts":          func(s *Statement) { s.Predicate.Counts.Pass = 9 },
	}
	for name, breakIt := range cases {
		s := valid()
		breakIt(s)
		if err := s.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	s := valid()
	s.Predicate.Card = nil
	s.Predicate.Verdicts = append(s.Predicate.Verdicts,
		attestation.Verdict{ID: "a2a.x", Status: "warn"}, attestation.Verdict{ID: "a2a.y", Status: "skip"}, attestation.Verdict{ID: "a2a.z", Status: "info"})
	s.Predicate.Counts = attestation.Counts{Pass: 3, Fail: 1, Warn: 1, Skip: 1, Info: 1}
	if err := s.Validate(); err != nil {
		t.Fatalf("a statement with no card and every status: %v", err)
	}
	if _, err := Parse([]byte("{")); err == nil {
		t.Fatal("parsed invalid JSON")
	}
}

// TestTheFixtureIsCurrent keeps testdata/statement.json equal to what valid()
// marshals, so a consumer testing against the fixture tests a statement this
// package accepts. Regenerate with PASSMCP_UPDATE_FIXTURES=1.
func TestTheFixtureIsCurrent(t *testing.T) {
	want, err := valid().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	const path = "testdata/statement.json"
	if os.Getenv("PASSMCP_UPDATE_FIXTURES") == "1" {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, want, 0o644); err != nil { // #nosec G306 -- a test fixture
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run PASSMCP_UPDATE_FIXTURES=1 go test ./a2a/", path)
	}
	if _, err := Parse(got); err != nil {
		t.Fatalf("the fixture does not verify: %v", err)
	}
}
