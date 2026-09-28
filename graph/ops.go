// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// New returns an empty graph.
func New() *Graph {
	return &Graph{Version: Version, Nodes: []Node{}, Edges: []Edge{}}
}

// reindex rebuilds the lookup tables, which are not serialised.
func (g *Graph) reindex() {
	g.index = make(map[string]int, len(g.Nodes))
	for i, n := range g.Nodes {
		g.index[n.ID] = i
	}
	g.edges = make(map[string]int, len(g.Edges))
	for i, e := range g.Edges {
		g.edges[edgeKey(e)] = i
	}
}

func (g *Graph) ensureIndex() {
	if g.index == nil || g.edges == nil || len(g.index) != len(g.Nodes) || len(g.edges) != len(g.Edges) {
		g.reindex()
	}
}

func edgeKey(e Edge) string { return e.From + "\x00" + string(e.Kind) + "\x00" + e.To }

// Node returns the node with the given ID.
func (g *Graph) Node(nodeID string) (Node, bool) {
	g.ensureIndex()
	i, ok := g.index[nodeID]
	if !ok {
		return Node{}, false
	}
	return g.Nodes[i], true
}

// Upsert adds n, or merges it into the node with the same ID.
//
// Merging is field by field for the node's own properties: a field the new
// node sets replaces the old value, and a field it leaves empty keeps the old
// one. So evidence arriving from two sources — a server named in a client
// configuration, then attested — accumulates, and upserting the same node
// twice changes nothing.
func (g *Graph) Upsert(n Node) {
	g.ensureIndex()
	i, ok := g.index[n.ID]
	if !ok {
		// Stored as a merge into an empty node, so the graph never shares a
		// property struct with the caller.
		c := Node{ID: n.ID, Kind: n.Kind, Label: n.Label}
		mergeProps(&c, n)
		g.Nodes = append(g.Nodes, c)
		g.index[n.ID] = len(g.Nodes) - 1
		return
	}
	old := &g.Nodes[i]
	if n.Label != "" {
		old.Label = n.Label
	}
	mergeProps(old, n)
}

// mergeProps folds the property struct of n into old.
func mergeProps(old *Node, n Node) {
	old.Agent = mergeAgent(old.Agent, n.Agent)
	old.Server = mergeServer(old.Server, n.Server)
	old.Tool = mergeTool(old.Tool, n.Tool)
	if n.Identity != nil {
		old.Identity = n.Identity
	}
	if n.Source != nil {
		old.Source = n.Source
	}
	if n.Attestation != nil {
		old.Attestation = n.Attestation
	}
}

func mergeAgent(old, n *AgentProps) *AgentProps {
	if n == nil {
		return old
	}
	if old == nil {
		c := *n
		return &c
	}
	setString(&old.Client, n.Client)
	setString(&old.ConfigFile, n.ConfigFile)
	return old
}

func mergeServer(old, n *ServerProps) *ServerProps {
	if n == nil {
		return old
	}
	if old == nil {
		c := *n
		c.Failing = append([]Finding(nil), n.Failing...)
		return &c
	}
	for _, f := range []struct {
		dst *string
		src string
	}{
		{&old.Transport, n.Transport}, {&old.Endpoint, n.Endpoint}, {&old.Name, n.Name},
		{&old.Version, n.Version}, {&old.Auth, n.Auth}, {&old.Grade, n.Grade},
	} {
		setString(f.dst, f.src)
	}
	// Score, the attestation and its failing checks travel together: a newer
	// attestation replaces all three, so a fixed finding stops being carried.
	if n.Attestation != "" {
		old.Attestation, old.Score = n.Attestation, n.Score
		old.Failing = append([]Finding(nil), n.Failing...)
	} else if n.Score != nil {
		old.Score = n.Score
	}
	return old
}

func mergeTool(old, n *ToolProps) *ToolProps {
	if n == nil {
		return old
	}
	if old == nil {
		c := *n
		return &c
	}
	setString(&old.Name, n.Name)
	if n.ReadOnlyHint != nil {
		old.ReadOnlyHint = n.ReadOnlyHint
	}
	if n.DestructiveHint != nil {
		old.DestructiveHint = n.DestructiveHint
	}
	return old
}

func setString(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// Link adds e, or merges it into the edge with the same endpoints and kind:
// scopes are unioned and a new Ref replaces the old one.
func (g *Graph) Link(e Edge) {
	g.ensureIndex()
	k := edgeKey(e)
	i, ok := g.edges[k]
	if !ok {
		e.Scopes = sortedUnique(append([]string{}, e.Scopes...))
		if e.Ref != nil {
			r := *e.Ref
			e.Ref = &r
		}
		g.Edges = append(g.Edges, e)
		g.edges[k] = len(g.Edges) - 1
		return
	}
	old := &g.Edges[i]
	old.Scopes = sortedUnique(append(append([]string{}, old.Scopes...), e.Scopes...))
	if e.Ref != nil {
		r := *e.Ref
		old.Ref = &r
	}
}

// Unlink removes the edge with the given endpoints and kind, if present. A
// re-ingest uses it to drop a relation the evidence no longer shows.
func (g *Graph) Unlink(from string, kind EdgeKind, to string) {
	g.ensureIndex()
	k := edgeKey(Edge{From: from, Kind: kind, To: to})
	i, ok := g.edges[k]
	if !ok {
		return
	}
	g.Edges = append(g.Edges[:i], g.Edges[i+1:]...)
	g.reindex()
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	sort.Strings(in)
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}

// EdgesFrom returns the edges of kind that leave the node nodeID, in
// canonical order.
func (g *Graph) EdgesFrom(nodeID string, kind EdgeKind) []Edge {
	var out []Edge
	for _, e := range g.Edges {
		if e.From == nodeID && e.Kind == kind {
			out = append(out, e)
		}
	}
	sortEdges(out)
	return out
}

// EdgesTo returns the edges of kind that arrive at the node nodeID, in
// canonical order.
func (g *Graph) EdgesTo(nodeID string, kind EdgeKind) []Edge {
	var out []Edge
	for _, e := range g.Edges {
		if e.To == nodeID && e.Kind == kind {
			out = append(out, e)
		}
	}
	sortEdges(out)
	return out
}

// Merge upserts every node and links every edge of other into g.
func (g *Graph) Merge(other *Graph) {
	for _, n := range other.Nodes {
		g.Upsert(n)
	}
	for _, e := range other.Edges {
		g.Link(e)
	}
}

// Canonical sorts the nodes by ID and the edges by source, kind and target,
// which is the order Marshal writes.
func (g *Graph) Canonical() {
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sortEdges(g.Edges)
	g.reindex()
}

func sortEdges(es []Edge) {
	sort.Slice(es, func(i, j int) bool { return edgeKey(es[i]) < edgeKey(es[j]) })
}

// Marshal renders the graph canonically: the same graph always encodes to
// the same bytes, so a store under version control diffs cleanly and two
// ingests of the same evidence are byte-identical.
func (g *Graph) Marshal() ([]byte, error) {
	g.Canonical()
	if g.Version == "" {
		g.Version = Version
	}
	if g.Nodes == nil {
		g.Nodes = []Node{}
	}
	if g.Edges == nil {
		g.Edges = []Edge{}
	}
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Parse reads a graph and validates it.
func Parse(b []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, fmt.Errorf("graph: not a graph: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	g.reindex()
	return &g, nil
}

// Validate reports every way the graph breaks the model: an unknown version
// or kind, a node whose properties do not match its kind or whose ID does
// not carry its kind, a duplicate node, and an edge that joins nodes of the
// wrong kinds or names a node that does not exist.
func (g *Graph) Validate() error {
	var problems []string
	note := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }
	if g.Version != Version {
		note("version is %q, want %q", g.Version, Version)
	}
	kinds := make(map[string]Kind, len(g.Nodes))
	for i, n := range g.Nodes {
		validateNode(i, n, kinds, note)
	}
	for i, e := range g.Edges {
		validateEdge(i, e, kinds, note)
	}
	if len(problems) > 0 {
		return fmt.Errorf("graph: %s", strings.Join(problems, "; "))
	}
	return nil
}

// validateNode checks one node and records its kind for the edge checks.
func validateNode(i int, n Node, kinds map[string]Kind, note func(string, ...any)) {
	if !strings.HasPrefix(n.ID, string(n.Kind)+":") || len(n.ID) == len(n.Kind)+1 {
		note("node %d: id %q does not carry its kind %q", i, n.ID, n.Kind)
	}
	if _, dup := kinds[n.ID]; dup {
		note("node %d: duplicate id %q", i, n.ID)
	}
	kinds[n.ID] = n.Kind
	set := propsSet(n)
	switch {
	case len(set) != 1:
		note("node %d (%s): %d property sets %v, want exactly the one for its kind", i, n.ID, len(set), set)
	case set[0] != n.Kind:
		note("node %d (%s): kind %q carries %s properties", i, n.ID, n.Kind, set[0])
	}
}

// propsSet lists the kinds whose property struct the node carries.
func propsSet(n Node) []Kind {
	var set []Kind
	for _, p := range []struct {
		k  Kind
		ok bool
	}{
		{KindAgent, n.Agent != nil}, {KindServer, n.Server != nil}, {KindTool, n.Tool != nil},
		{KindIdentity, n.Identity != nil}, {KindSource, n.Source != nil}, {KindAttestation, n.Attestation != nil},
	} {
		if p.ok {
			set = append(set, p.k)
		}
	}
	return set
}

// validateEdge checks that an edge's kind is known and that it joins
// existing nodes of the kinds its kind allows.
func validateEdge(i int, e Edge, kinds map[string]Kind, note func(string, ...any)) {
	want, ok := endpoints[e.Kind]
	if !ok {
		note("edge %d: unknown kind %q", i, e.Kind)
		return
	}
	from, fok := kinds[e.From]
	to, tok := kinds[e.To]
	switch {
	case !fok:
		note("edge %d (%s): from %q is not a node", i, e.Kind, e.From)
	case !tok:
		note("edge %d (%s): to %q is not a node", i, e.Kind, e.To)
	case from != want[0] || to != want[1]:
		note("edge %d: %s joins %s to %s, not %s to %s", i, e.Kind, want[0], want[1], from, to)
	}
	if len(e.Scopes) > 0 && e.Kind != Authorizes {
		note("edge %d: only %s edges carry scopes", i, Authorizes)
	}
	if e.Ref != nil && e.Kind != Uses {
		note("edge %d: only %s edges carry a configuration reference", i, Uses)
	}
}
