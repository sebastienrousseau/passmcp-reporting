// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"encoding/json"
	"testing"

	"satellion.com/passmcp-reporting/a2a"
	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/graph"
)

// The embedded schema is the published one: it parses, it is the 2020-12
// dialect, and its $id names the predicate the attestation package writes.
func TestEmbeddedSchemaIsThePublishedOne(t *testing.T) {
	var s struct {
		Schema string         `json:"$schema"`
		ID     string         `json:"$id"`
		Ref    string         `json:"$ref"`
		Defs   map[string]any `json:"$defs"`
	}
	if err := json.Unmarshal(AttestationSchema, &s); err != nil {
		t.Fatalf("the embedded schema is not JSON: %v", err)
	}
	if s.Schema != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("dialect %q", s.Schema)
	}
	if s.ID != attestation.PredicateType+"/schema.json" {
		t.Errorf("$id %q does not name the predicate %q", s.ID, attestation.PredicateType)
	}
	if s.Ref != "#/$defs/Statement" || s.Defs["Statement"] == nil || s.Defs["Evaluation"] == nil {
		t.Errorf("the root is not the Statement: ref %q, %d definitions", s.Ref, len(s.Defs))
	}
}

// The A2A and graph schemas are embedded too, each naming its own format and
// rooted at its own document type.
func TestTheOtherSchemasAreEmbedded(t *testing.T) {
	for _, c := range []struct {
		name, id, root string
		b              []byte
	}{
		{"a2a", a2a.PredicateType + "/schema.json", "#/$defs/Statement", A2AAttestationSchema},
		{"graph", graph.Version + "/schema.json", "#/$defs/Graph", GraphSchema},
	} {
		var s struct {
			Schema string `json:"$schema"`
			ID     string `json:"$id"`
			Ref    string `json:"$ref"`
		}
		if err := json.Unmarshal(c.b, &s); err != nil {
			t.Fatalf("%s: not JSON: %v", c.name, err)
		}
		if s.Schema != "https://json-schema.org/draft/2020-12/schema" || s.ID != c.id || s.Ref != c.root {
			t.Errorf("%s: dialect %q, $id %q, $ref %q", c.name, s.Schema, s.ID, s.Ref)
		}
	}
}
