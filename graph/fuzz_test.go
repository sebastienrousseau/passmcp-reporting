// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"bytes"
	"testing"
)

// FuzzParse feeds Parse arbitrary bytes, since a store on disk is edited by
// people and written by other tools. Parse must never panic, and a graph it
// accepts must be one Marshal encodes canonically: the encoding parses back
// to an accepted graph that encodes to the same bytes, and every edge still
// resolves to nodes Node can find. A store whose bytes changed on a no-op
// load and save would diff on every ingest, which is what the canonical
// order exists to prevent.
func FuzzParse(f *testing.F) {
	if b, err := sample().Marshal(); err == nil {
		f.Add(b)
	}
	if b, err := New().Marshal(); err == nil {
		f.Add(b)
	}
	for _, s := range []string{
		``, `null`, `{}`, `[]`,
		`{"version":"https://satellion.com/graph/v1","nodes":[{"id":"tool:","kind":"tool","tool":{}}]}`,
		`{"version":"https://satellion.com/graph/v1","nodes":[],"edges":[{"from":"a","to":"b","kind":"uses"}]}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		g, err := Parse(b)
		if err != nil {
			if g != nil {
				t.Fatalf("Parse returned a graph with an error: %v", err)
			}
			return
		}
		assertCanonical(t, g)
	})
}

// assertCanonical checks the properties an accepted graph keeps.
func assertCanonical(t *testing.T, g *Graph) {
	t.Helper()
	first, err := g.Marshal()
	if err != nil {
		t.Fatalf("Marshal of an accepted graph: %v", err)
	}
	again, err := Parse(first)
	if err != nil {
		t.Fatalf("an accepted graph is refused after Marshal: %v\n%s", err, first)
	}
	second, err := again.Marshal()
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("Marshal is not a fixed point:\n%s\n---\n%s", first, second)
	}
	for _, e := range again.Edges {
		if _, ok := again.Node(e.From); !ok {
			t.Fatalf("edge %+v: from is not a node after the round trip", e)
		}
		if _, ok := again.Node(e.To); !ok {
			t.Fatalf("edge %+v: to is not a node after the round trip", e)
		}
	}
}
