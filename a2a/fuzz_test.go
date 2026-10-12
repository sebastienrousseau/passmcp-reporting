// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse feeds Parse arbitrary bytes, as attestation.FuzzParse does for
// the MCP predicate: an A2A statement comes from anyone too. Parse must
// never panic, and a statement it accepts must still cover its own agent
// and must marshal to bytes that Parse accepts and that marshal the same
// way again.
func FuzzParse(f *testing.F) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "statement.json"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(fixture)
	if b, err := valid().Marshal(); err == nil {
		f.Add(b)
	}
	// A subject carrying only sha512, which Validate accepts.
	only512 := valid()
	sum := sha512.Sum512([]byte(descriptor(only512.Predicate.Target)))
	only512.Subject[0].Digest = map[string]string{"sha512": hex.EncodeToString(sum[:])}
	if b, err := only512.Marshal(); err == nil {
		f.Add(b)
	}
	for _, s := range []string{
		``, `null`, `{}`, `[]`, `"statement"`,
		`{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://satellion.com/attestation/mcp-evaluation/v1"}`,
		`{"predicate":{"agentCard":{"digest":"zz"},"verdicts":[{"id":"x","status":"nope"}]}}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := Parse(b)
		if err != nil {
			if s != nil {
				t.Fatalf("Parse returned a statement with an error: %v", err)
			}
			return
		}
		if !s.Covers(s.Predicate.Target.Endpoint) {
			t.Fatalf("an accepted statement does not cover its own agent %q", s.Predicate.Target.Endpoint)
		}
		first, err := s.Marshal()
		if err != nil {
			t.Fatalf("Marshal of an accepted statement: %v", err)
		}
		again, err := Parse(first)
		if err != nil {
			t.Fatalf("an accepted statement is refused after Marshal: %v\n%s", err, first)
		}
		second, err := again.Marshal()
		if err != nil || !bytes.Equal(first, second) {
			t.Fatalf("Marshal is not a fixed point (err %v):\n%s\n---\n%s", err, first, second)
		}
	})
}
