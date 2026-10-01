// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package attestation

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// These tests use only this package and the standard library, so they move
// with it when it leaves passmcp's repository. The statements are written by
// hand for that reason: building one from a report is the engine's job.

func valid() *Statement {
	t := Target{Transport: "http", Endpoint: "https://mcp.example.com/mcp"}
	return &Statement{
		Type:          StatementType,
		PredicateType: PredicateType,
		Subject:       []Subject{SubjectFor(t)},
		Predicate: Evaluation{
			SubjectKind:   SubjectKindDescriptor,
			Target:        t,
			JudgedAgainst: Basis{SpecRevision: "2026-07-28", Rubric: "1", CheckInventory: "1"},
			Instrument:    Instrument{Name: "passmcp", Version: "0.0.3", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
			Took:          "3.2s",
			Verdicts: []Verdict{
				{ID: "auth.unauthenticated_tools", Phase: "auth", Status: "pass"},
				{ID: "protocol.origin", Phase: "protocol", Status: "fail", Severity: "major", Evidence: []string{"req#12"}},
			},
			Counts: Counts{Pass: 1, Fail: 1},
			Score:  &Score{Total: 88, Grade: "B", Assessed: 6, Of: 6},
		},
	}
}

func TestAWellFormedStatementValidates(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestValidateNamesEveryProblem. Each mutation breaks one property the
// format guarantees; the error must say which.
func TestValidateNamesEveryProblem(t *testing.T) {
	for want, mutate := range map[string]func(s *Statement){
		"_type is":                   func(s *Statement) { s.Type = "x" },
		"predicateType is":           func(s *Statement) { s.PredicateType = "x" },
		"no subject":                 func(s *Statement) { s.Subject = nil },
		"2 subjects":                 func(s *Statement) { s.Subject = append(s.Subject, s.Subject[0]) },
		"subject has no name":        func(s *Statement) { s.Subject[0].Name = " " },
		"named":                      func(s *Statement) { s.Subject[0].Name = "https://trusted.example.com/mcp" },
		"no sha256 or sha512 digest": func(s *Statement) { s.Subject[0].Digest = map[string]string{} },
		"does not cover the target":  func(s *Statement) { s.Subject[0].Digest["sha256"] = strings.Repeat("0", 64) },
		"subjectKind is":             func(s *Statement) { s.Predicate.SubjectKind = "artifact" },
		"no endpoint":                func(s *Statement) { s.Predicate.Target.Endpoint = "" },
		"no transport":               func(s *Statement) { s.Predicate.Target.Transport = "" },
		"unknown transport":          func(s *Statement) { s.Predicate.Target.Transport = "carrier-pigeon" },
		"not identified":             func(s *Statement) { s.Predicate.Instrument.Version = "" },
		"no run time":                func(s *Statement) { s.Predicate.RanAt = time.Time{} },
		"no verdicts":                func(s *Statement) { s.Predicate.Verdicts = nil; s.Predicate.Counts = Counts{} },
		"checkInventory is empty":    func(s *Statement) { s.Predicate.JudgedAgainst.CheckInventory = "" },
		"no rubric version":          func(s *Statement) { s.Predicate.JudgedAgainst.Rubric = "" },
		"has no id":                  func(s *Statement) { s.Predicate.Verdicts[0].ID = "" },
		"has status":                 func(s *Statement) { s.Predicate.Verdicts[0].Status = "probably" },
		"counts disagree":            func(s *Statement) { s.Predicate.Counts.Pass = 7 },
	} {
		t.Run(want, func(t *testing.T) {
			s := valid()
			mutate(s)
			err := s.Validate()
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("err = %v, want it to mention %q", err, want)
			}
		})
	}
}

// TestEveryStatusIsCounted. Counts are what a policy gates on, so each
// status has to land in its own bucket.
func TestEveryStatusIsCounted(t *testing.T) {
	s := valid()
	s.Predicate.Verdicts = []Verdict{
		{ID: "a", Status: "pass"}, {ID: "b", Status: "warn"}, {ID: "c", Status: "fail"},
		{ID: "d", Status: "skip"}, {ID: "e", Status: "info"},
	}
	s.Predicate.Counts = Counts{Pass: 1, Warn: 1, Fail: 1, Skip: 1, Info: 1}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAStatementWithoutAScoreNeedsNoRubric(t *testing.T) {
	s := valid()
	s.Predicate.Score = nil
	s.Predicate.JudgedAgainst.Rubric = ""
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestParseRoundTripsWhatMarshalWrites(t *testing.T) {
	b, err := valid().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(b), "\n") || !strings.Contains(string(b), "\n  \"subject\"") {
		t.Errorf("not indented and newline-terminated:\n%s", b)
	}
	got, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Predicate.Target.Endpoint != "https://mcp.example.com/mcp" || len(got.Predicate.Verdicts) != 2 {
		t.Errorf("round trip lost data: %+v", got.Predicate)
	}
}

func TestParseRefusesWhatIsNotAValidStatement(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil || !strings.Contains(err.Error(), "not a statement") {
		t.Errorf("malformed JSON: %v", err)
	}
	b, _ := json.Marshal(map[string]string{"_type": StatementType})
	if _, err := Parse(b); err == nil {
		t.Error("an empty statement parsed")
	}
}

func TestVerdictForSeparatesFailedFromNeverMeasured(t *testing.T) {
	s := valid()
	v, err := s.VerdictFor("protocol.origin")
	if err != nil || v.Status != "fail" {
		t.Fatalf("VerdictFor = %+v, %v", v, err)
	}
	if _, err := s.VerdictFor("never.measured"); !errors.Is(err, ErrNoSuchCheck) {
		t.Errorf("err = %v, want ErrNoSuchCheck", err)
	}
}

// TestCoversRecomputesRatherThanCompares. A gateway asks "is this
// statement about the server I am routing to"; the answer must come from
// the digest, so a rewritten predicate cannot pass.
func TestCoversRecomputesRatherThanCompares(t *testing.T) {
	s := valid()
	if !s.Covers("http", "https://mcp.example.com/mcp") {
		t.Error("the statement does not cover its own target")
	}
	if !s.Covers("http", "  https://mcp.example.com/mcp  ") {
		t.Error("surrounding whitespace changed the answer")
	}
	if s.Covers("stdio", "https://mcp.example.com/mcp") {
		t.Error("another transport was covered")
	}
	if s.Covers("http", "https://other.example.com/mcp") {
		t.Error("another server was covered")
	}
	s.Subject = nil
	if s.Covers("http", "https://mcp.example.com/mcp") {
		t.Error("a statement with no subject covers something")
	}
}

// sha512Of is the SHA-512 of the descriptor of t, in hex: the digest a
// producer whose policy requires SHA-512 writes beside, or instead of,
// the SHA-256 SubjectFor writes.
func sha512Of(t Target) string {
	sum := sha512.Sum512([]byte(descriptor(t)))
	return hex.EncodeToString(sum[:])
}

// TestSubjectDigestAlgorithms. in-toto's DigestSet is a map so a subject
// can carry more than one algorithm. Every known algorithm present must
// recompute, at least one must be present, and an unknown one alone is no
// digest at all.
func TestSubjectDigestAlgorithms(t *testing.T) {
	target := valid().Predicate.Target
	good256 := SubjectFor(target).Digest["sha256"]
	good512 := sha512Of(target)
	for name, tc := range map[string]struct {
		digest map[string]string
		want   string // "" for accepted, else a fragment of the error
	}{
		"sha256 and sha512, both correct": {map[string]string{"sha256": good256, "sha512": good512}, ""},
		"sha512 only, correct":            {map[string]string{"sha512": good512}, ""},
		"sha256 correct, sha512 wrong":    {map[string]string{"sha256": good256, "sha512": strings.Repeat("0", 128)}, "sha512 digest does not cover"},
		"sha512 correct, sha256 wrong":    {map[string]string{"sha256": strings.Repeat("0", 64), "sha512": good512}, "sha256 digest does not cover"},
		"sha512 empty beside sha256":      {map[string]string{"sha256": good256, "sha512": ""}, "sha512 digest does not cover"},
		"an unknown algorithm only":       {map[string]string{"md5": "d41d8cd98f00b204e9800998ecf8427e"}, "no sha256 or sha512 digest"},
		"an unknown one beside sha256":    {map[string]string{"md5": "x", "sha256": good256}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			s := valid()
			s.Subject[0].Digest = tc.digest
			err := s.Validate()
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("refused: %v", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
			if covers := s.Covers(target.Transport, target.Endpoint); covers != (tc.want == "") {
				t.Fatalf("Covers = %v, want %v", covers, tc.want == "")
			}
		})
	}
}

// TestSubjectForWritesSHA256Only. What passmcp produces does not change
// because the verifier learned another algorithm.
func TestSubjectForWritesSHA256Only(t *testing.T) {
	d := SubjectFor(valid().Predicate.Target).Digest
	if len(d) != 1 || d["sha256"] == "" {
		t.Fatalf("SubjectFor wrote %v, want sha256 alone", d)
	}
}

func TestSubjectForIsStableAndDistinct(t *testing.T) {
	a := SubjectFor(Target{Transport: "http", Endpoint: "https://a.example/mcp"})
	b := SubjectFor(Target{Transport: "http", Endpoint: "https://a.example/mcp"})
	c := SubjectFor(Target{Transport: "stdio", Endpoint: "https://a.example/mcp"})
	if a.Digest["sha256"] != b.Digest["sha256"] {
		t.Error("one target, two digests")
	}
	if a.Digest["sha256"] == c.Digest["sha256"] {
		t.Error("two transports, one digest")
	}
	if len(a.Digest["sha256"]) != 64 {
		t.Errorf("digest %q is not hex SHA-256", a.Digest["sha256"])
	}
}

// bigStatement is valid() with n verdicts, marshalled: the shape of a real
// run, which carries every check that ran.
func bigStatement(b *testing.B, n int) []byte {
	b.Helper()
	st := valid()
	for len(st.Predicate.Verdicts) < n {
		i := len(st.Predicate.Verdicts)
		st.Predicate.Verdicts = append(st.Predicate.Verdicts, Verdict{
			ID: fmt.Sprintf("catalog.check_%d", i), Phase: "catalog", Status: "pass",
			Evidence: []string{fmt.Sprintf("req#%d", i+3)}, Doc: "https://satellion.com/passmcp/docs/checks/#check-catalog",
		})
		st.Predicate.Counts.Pass++
	}
	raw, err := st.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	return raw
}

// BenchmarkValidate is what a gateway pays per admission decision: parsing
// and validating a statement, digest included, at the size of the fixture
// and at the size of a full run.
func BenchmarkValidate(b *testing.B) {
	for _, n := range []int{2, 120} {
		raw := bigStatement(b, n)
		b.Run(fmt.Sprintf("verdicts=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				st, err := Parse(raw)
				if err != nil {
					b.Fatal(err)
				}
				if err := st.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
