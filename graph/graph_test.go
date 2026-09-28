// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"satellion.com/passmcp-reporting/attestation"
)

func bp(b bool) *bool       { return &b }
func fp(f float64) *float64 { return &f }
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// sample builds a small graph the way an ingest would: an agent using a
// server that exposes a tool, an identity on the server, the source it was
// found in and the attestation about it.
func sample() *Graph {
	g := New()
	srv := ServerID("http", "https://mcp.example.com/mcp")
	tool := ToolID(srv, "delete_repo")
	agent := AgentID("claude-desktop", "/home/u/claude.json", "work")
	ident := IdentityID("https://auth.example.com", "client-123")
	src := SourceID("config", "/home/u/claude.json")
	att := AttestationID(strings.Repeat("a", 64))
	g.Upsert(Node{ID: srv, Kind: KindServer, Label: "https://mcp.example.com/mcp",
		Server: &ServerProps{Transport: "http", Endpoint: "https://mcp.example.com/mcp", Auth: "none",
			Score: fp(71), Grade: "C", Attestation: strings.Repeat("a", 64),
			Failing: []Finding{{ID: "auth.required", Severity: "critical"}}}})
	g.Upsert(Node{ID: tool, Kind: KindTool, Label: "delete_repo", Tool: &ToolProps{Name: "delete_repo", ReadOnlyHint: bp(false), DestructiveHint: bp(true)}})
	g.Upsert(Node{ID: agent, Kind: KindAgent, Label: "work", Agent: &AgentProps{Client: "claude-desktop", ConfigFile: "/home/u/claude.json"}})
	g.Upsert(Node{ID: ident, Kind: KindIdentity, Label: "client-123", Identity: &IdentityProps{Issuer: "https://auth.example.com", ClientID: "client-123"}})
	g.Upsert(Node{ID: src, Kind: KindSource, Label: "/home/u/claude.json", Source: &SourceProps{Type: "config", Ref: "/home/u/claude.json"}})
	g.Upsert(Node{ID: att, Kind: KindAttestation, Label: "statement", Attestation: &AttestationProps{Digest: strings.Repeat("a", 64), PredicateType: attestation.PredicateType}})
	g.Link(Edge{From: agent, To: srv, Kind: Uses, Ref: &ConfigRef{File: "/home/u/claude.json", Key: "mcpServers.github"}})
	g.Link(Edge{From: srv, To: tool, Kind: Exposes})
	g.Link(Edge{From: ident, To: srv, Kind: Authorizes, Scopes: []string{"repo:write", "read"}})
	g.Link(Edge{From: srv, To: src, Kind: DiscoveredBy})
	g.Link(Edge{From: srv, To: att, Kind: AttestedBy})
	return g
}

// A server's ID is the attestation subject digest of the same target, so a
// graph and a statement name a server the same way.
func TestServerIDIsTheAttestationSubjectDigest(t *testing.T) {
	target := attestation.Target{Transport: "http", Endpoint: "https://mcp.example.com/mcp"}
	want := "server:" + attestation.SubjectFor(target).Digest["sha256"]
	if got := ServerID("http", "https://mcp.example.com/mcp"); got != want {
		t.Fatalf("ServerID = %s, want %s", got, want)
	}
	if ServerID("stdio", "https://mcp.example.com/mcp") == want {
		t.Fatal("the transport is not part of the identity")
	}
}

// The ID functions are deterministic and carry their kind.
func TestIDsAreStableAndTyped(t *testing.T) {
	cases := map[string]string{
		"tool:":        ToolID("server:x", "t"),
		"agent:":       AgentID("cursor", "f", "a"),
		"identity:":    IdentityID("iss", "c"),
		"source:":      SourceID("targets", "t.txt"),
		"attestation:": AttestationID(Digest([]byte("x"))),
	}
	for prefix, id := range cases {
		if !strings.HasPrefix(id, prefix) {
			t.Errorf("%s does not start with %s", id, prefix)
		}
	}
	a, b := ToolID("server:x", "t"), ToolID("server:x", "t")
	if a != b || a == ToolID("server:y", "t") {
		t.Error("tool IDs are not stable, or ignore the server")
	}
	// The SHA-256 of "x", so the digest is pinned, not just self-consistent.
	if d := Digest([]byte("x")); d != "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881" {
		t.Errorf("Digest(x) = %s", d)
	}
}

// Ingesting the same evidence twice changes nothing, down to the bytes.
func TestUpsertingTheSameEvidenceTwiceChangesNothing(t *testing.T) {
	once, err := sample().Marshal()
	must(t, err)
	g := sample()
	g.Merge(sample())
	twice, err := g.Marshal()
	must(t, err)
	if !bytes.Equal(once, twice) {
		t.Fatalf("a second ingest changed the graph:\n%s\n---\n%s", once, twice)
	}
}

// Evidence from two sources accumulates: a server first seen in a client
// configuration keeps its endpoint when an attestation adds its score, and a
// newer attestation replaces the failing checks rather than adding to them.
func TestEvidenceAccumulatesAndANewerAttestationReplacesFindings(t *testing.T) {
	g := New()
	id := ServerID("http", "https://s")
	g.Upsert(Node{ID: id, Kind: KindServer, Label: "s", Server: &ServerProps{Transport: "http", Endpoint: "https://s", Auth: "none"}})
	g.Upsert(Node{ID: id, Kind: KindServer, Server: &ServerProps{Name: "srv", Score: fp(50), Grade: "F", Attestation: "d1",
		Failing: []Finding{{ID: "auth.required", Severity: "critical"}}}})
	n, _ := g.Node(id)
	if n.Label != "s" || n.Server.Endpoint != "https://s" || n.Server.Name != "srv" || n.Server.Auth != "none" || *n.Server.Score != 50 {
		t.Fatalf("evidence did not accumulate: %+v", *n.Server)
	}
	g.Upsert(Node{ID: id, Kind: KindServer, Server: &ServerProps{Score: fp(90), Grade: "A", Attestation: "d2"}})
	n, _ = g.Node(id)
	if len(n.Server.Failing) != 0 || n.Server.Attestation != "d2" || *n.Server.Score != 90 {
		t.Fatalf("the newer attestation did not replace the findings: %+v", *n.Server)
	}
	g.Upsert(Node{ID: id, Kind: KindServer, Server: &ServerProps{Score: fp(95)}})
	if n, _ = g.Node(id); *n.Server.Score != 95 || n.Server.Attestation != "d2" {
		t.Fatalf("a score without an attestation: %+v", *n.Server)
	}
}

// The graph never shares a property struct with its caller: changing what
// was upserted afterwards does not change the graph.
func TestTheGraphOwnsItsProperties(t *testing.T) {
	g := New()
	p := &ServerProps{Transport: "http", Endpoint: "https://s", Failing: []Finding{{ID: "x"}}}
	id := ServerID("http", "https://s")
	g.Upsert(Node{ID: id, Kind: KindServer, Server: p})
	p.Endpoint = "changed"
	p.Failing[0].ID = "changed"
	ref := &ConfigRef{File: "f", Key: "k"}
	a := AgentID("c", "f", "a")
	g.Upsert(Node{ID: a, Kind: KindAgent, Agent: &AgentProps{Client: "c"}})
	g.Link(Edge{From: a, To: id, Kind: Uses, Ref: ref})
	ref.Key = "changed"
	n, _ := g.Node(id)
	if n.Server.Endpoint != "https://s" || n.Server.Failing[0].ID != "x" || g.Edges[0].Ref.Key != "k" {
		t.Fatal("the graph shares memory with its caller")
	}
}

// Tools and agents merge field by field too.
func TestToolAndAgentPropertiesMerge(t *testing.T) {
	g := New()
	tl := ToolID("server:x", "t")
	g.Upsert(Node{ID: tl, Kind: KindTool, Tool: &ToolProps{Name: "t"}})
	g.Upsert(Node{ID: tl, Kind: KindTool, Tool: &ToolProps{ReadOnlyHint: bp(true)}})
	g.Upsert(Node{ID: tl, Kind: KindTool, Tool: &ToolProps{DestructiveHint: bp(false)}})
	n, _ := g.Node(tl)
	if n.Tool.Name != "t" || !*n.Tool.ReadOnlyHint || *n.Tool.DestructiveHint {
		t.Fatalf("tool merge: %+v", *n.Tool)
	}
	ag := AgentID("c", "f", "a")
	g.Upsert(Node{ID: ag, Kind: KindAgent, Agent: &AgentProps{Client: "c"}})
	g.Upsert(Node{ID: ag, Kind: KindAgent, Agent: &AgentProps{ConfigFile: "f"}})
	g.Upsert(Node{ID: ag, Kind: KindAgent, Label: "a"})
	if n, _ = g.Node(ag); n.Agent.Client != "c" || n.Agent.ConfigFile != "f" || n.Label != "a" {
		t.Fatalf("agent merge: %+v", *n.Agent)
	}
	for _, k := range []Node{
		{ID: IdentityID("i", "c"), Kind: KindIdentity, Identity: &IdentityProps{Issuer: "i"}},
		{ID: SourceID("t", "r"), Kind: KindSource, Source: &SourceProps{Type: "targets"}},
		{ID: AttestationID("d"), Kind: KindAttestation, Attestation: &AttestationProps{Digest: "d"}},
	} {
		g.Upsert(k)
		g.Upsert(k)
	}
	if _, ok := g.Node("missing"); ok {
		t.Fatal("found a node that was never added")
	}
}

// Linking an edge twice unions its scopes; Unlink removes one relation.
func TestLinkUnionsScopesAndUnlinkRemoves(t *testing.T) {
	g := sample()
	srv := ServerID("http", "https://mcp.example.com/mcp")
	ident := IdentityID("https://auth.example.com", "client-123")
	g.Link(Edge{From: ident, To: srv, Kind: Authorizes, Scopes: []string{"admin", "read"}})
	es := g.EdgesFrom(ident, Authorizes)
	if len(es) != 1 || !reflect.DeepEqual(es[0].Scopes, []string{"admin", "read", "repo:write"}) {
		t.Fatalf("scopes: %+v", es)
	}
	agent := AgentID("claude-desktop", "/home/u/claude.json", "work")
	g.Link(Edge{From: agent, To: srv, Kind: Uses, Ref: &ConfigRef{File: "/home/u/claude.json", Key: "mcpServers.gh"}})
	if in := g.EdgesTo(srv, Uses); len(in) != 1 || in[0].Ref.Key != "mcpServers.gh" {
		t.Fatalf("uses edges: %+v", in)
	}
	g.Unlink(srv, Exposes, ToolID(srv, "delete_repo"))
	g.Unlink(srv, Exposes, "tool:absent")
	if len(g.EdgesFrom(srv, Exposes)) != 0 || len(g.EdgesFrom(srv, AttestedBy)) != 1 {
		t.Fatal("Unlink removed the wrong edge")
	}
	g.Link(Edge{From: srv, To: ToolID(srv, "delete_repo"), Kind: Exposes})
	if len(g.EdgesFrom(srv, Exposes)) != 1 {
		t.Fatal("the index is stale after Unlink")
	}
}

// Marshal and Parse round-trip, and the output is canonical whatever order
// the evidence arrived in.
func TestMarshalIsCanonicalAndRoundTrips(t *testing.T) {
	b, err := sample().Marshal()
	must(t, err)
	g, err := Parse(b)
	must(t, err)
	again, err := g.Marshal()
	must(t, err)
	if !bytes.Equal(b, again) {
		t.Fatal("a round trip changed the bytes")
	}
	rev := New()
	s := sample()
	for i := len(s.Nodes) - 1; i >= 0; i-- {
		rev.Upsert(s.Nodes[i])
	}
	for i := len(s.Edges) - 1; i >= 0; i-- {
		rev.Link(s.Edges[i])
	}
	rb, err := rev.Marshal()
	must(t, err)
	if !bytes.Equal(b, rb) {
		t.Fatal("insertion order leaks into the encoding")
	}
	empty, err := (&Graph{}).Marshal()
	must(t, err)
	if _, err := Parse(empty); err != nil {
		t.Fatalf("an empty graph does not round-trip: %v", err)
	}
	if _, err := Parse([]byte("{")); err == nil {
		t.Fatal("parsed invalid JSON")
	}
	if _, err := Parse([]byte(`{"version":"x","nodes":[],"edges":[]}`)); err == nil {
		t.Fatal("parsed an unknown version")
	}
}

// Validate refuses every way a graph can break the model.
func TestValidateRefusesABrokenGraph(t *testing.T) {
	srv := ServerID("http", "https://s")
	tool := ToolID(srv, "t")
	server := Node{ID: srv, Kind: KindServer, Server: &ServerProps{Transport: "http", Endpoint: "https://s"}}
	cases := map[string]*Graph{
		"version":      {Version: "v0"},
		"id kind":      {Version: Version, Nodes: []Node{{ID: "tool:x", Kind: KindServer, Server: &ServerProps{}}}},
		"bare prefix":  {Version: Version, Nodes: []Node{{ID: "server:", Kind: KindServer, Server: &ServerProps{}}}},
		"duplicate":    {Version: Version, Nodes: []Node{server, server}},
		"no props":     {Version: Version, Nodes: []Node{{ID: srv, Kind: KindServer}}},
		"two props":    {Version: Version, Nodes: []Node{{ID: srv, Kind: KindServer, Server: &ServerProps{}, Tool: &ToolProps{}}}},
		"wrong props":  {Version: Version, Nodes: []Node{{ID: tool, Kind: KindTool, Server: &ServerProps{}}}},
		"edge kind":    {Version: Version, Nodes: []Node{server}, Edges: []Edge{{From: srv, To: srv, Kind: "owns"}}},
		"missing from": {Version: Version, Nodes: []Node{server}, Edges: []Edge{{From: "agent:x", To: srv, Kind: Uses}}},
		"missing to":   {Version: Version, Nodes: []Node{server}, Edges: []Edge{{From: srv, To: tool, Kind: Exposes}}},
		"wrong ends":   {Version: Version, Nodes: []Node{server}, Edges: []Edge{{From: srv, To: srv, Kind: Uses}}},
		"stray scopes": {Version: Version, Nodes: []Node{server, {ID: tool, Kind: KindTool, Tool: &ToolProps{}}}, Edges: []Edge{{From: srv, To: tool, Kind: Exposes, Scopes: []string{"x"}}}},
		"stray ref":    {Version: Version, Nodes: []Node{server, {ID: tool, Kind: KindTool, Tool: &ToolProps{}}}, Edges: []Edge{{From: srv, To: tool, Kind: Exposes, Ref: &ConfigRef{}}}},
	}
	for name, g := range cases {
		if err := g.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := sample().Validate(); err != nil {
		t.Fatalf("the sample is invalid: %v", err)
	}
}

// Save and Load round-trip through a store directory; an absent store is an
// empty graph; a corrupt or invalid one is an error, and Save refuses an
// invalid graph rather than writing it.
func TestTheStoreRoundTrips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	g, err := Load(dir)
	must(t, err)
	if len(g.Nodes) != 0 {
		t.Fatal("an absent store is not empty")
	}
	must(t, Save(dir, sample()))
	back, err := Load(dir)
	must(t, err)
	a, _ := sample().Marshal()
	b, _ := back.Marshal()
	if !bytes.Equal(a, b) {
		t.Fatal("the store did not round-trip")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("the store left temporary files: %v", entries)
	}
	if err := Save(dir, &Graph{Version: "bad"}); err == nil {
		t.Fatal("saved an invalid graph")
	}
	must(t, os.WriteFile(filepath.Join(dir, FileName), []byte("{"), 0o600))
	if _, err := Load(dir); err == nil {
		t.Fatal("loaded a corrupt store")
	}
	blocked := filepath.Join(t.TempDir(), "file")
	must(t, os.WriteFile(blocked, nil, 0o600))
	if err := Save(filepath.Join(blocked, "sub"), sample()); err == nil {
		t.Fatal("saved under a file")
	}
	if _, err := Load(blocked); err == nil {
		t.Fatal("loaded a store whose directory is a file")
	}
}

// No type in the model has a field a secret could be stored in. SG-02 in
// passmcp-graph proves the ingest stores none; this proves the format cannot.
func TestTheModelHasNoFieldForASecret(t *testing.T) {
	banned := []string{"secret", "token", "password", "apikey", "api_key", "credential", "env", "value", "header"}
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(rt reflect.Type, path string) {
		for rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice {
			rt = rt.Elem()
		}
		if rt.Kind() == reflect.Map {
			t.Errorf("%s is a map: a free-form bag could hold anything", path)
			return
		}
		if rt.Kind() != reflect.Struct || seen[rt] {
			return
		}
		seen[rt] = true
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
			for _, b := range banned {
				if strings.Contains(name, b) {
					t.Errorf("%s.%s (%q) could hold a secret", path, f.Name, name)
				}
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeOf(Graph{}), "Graph")
}
