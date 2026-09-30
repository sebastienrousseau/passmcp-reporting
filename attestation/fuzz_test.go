// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package attestation

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse feeds Parse arbitrary bytes, since a statement can come from
// anyone. Parse must never panic, and a statement it accepts must survive
// the round trip a consumer makes: Marshal it, Parse the result, and get
// an accepted statement that marshals to the same bytes and still covers
// the target it names. A statement that verified once and not after being
// re-encoded, or that verified but did not cover its own target, would be
// evidence that changes meaning in transit.
func FuzzParse(f *testing.F) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "statement.json"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(fixture)
	if b, err := valid().Marshal(); err == nil {
		f.Add(b)
	}
	for _, s := range []string{
		``, `null`, `{}`, `[]`, `"statement"`,
		`{"_type":"https://in-toto.io/Statement/v1","subject":[{},{}]}`,
		`{"predicate":{"verdicts":[{"id":"x","status":"nope"}],"counts":{"pass":-1}}}`,
		`{"predicate":{"score":{"total":1e308},"ranAt":"0001-01-01T00:00:00Z"}}`,
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
		assertRoundTrip(t, s)
	})
}

// assertRoundTrip checks the properties an accepted statement keeps.
func assertRoundTrip(t *testing.T, s *Statement) {
	t.Helper()
	if !s.Covers(s.Predicate.Target.Transport, s.Predicate.Target.Endpoint) {
		t.Fatalf("an accepted statement does not cover its own target %+v", s.Predicate.Target)
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
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("Marshal is not a fixed point:\n%s\n---\n%s", first, second)
	}
	d := Compare(s, again)
	changed := len(d.Regressed) + len(d.Improved) + len(d.SeverityChanged) + len(d.Unassessed) + len(d.Added)
	if !d.Comparable || changed > 0 {
		t.Fatalf("a statement compared with its own round trip changed: %+v", d)
	}
}
