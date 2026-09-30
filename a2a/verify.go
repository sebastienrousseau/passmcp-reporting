// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/internal/subjectdigest"
)

// Parse reads a statement and validates it. It refuses an MCP evaluation,
// or any other predicate, rather than reading it as an A2A one.
func Parse(b []byte) (*Statement, error) {
	var s Statement
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("a2a: not a statement: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Validate reports every way a statement is not interpretable, by the same
// rules as an MCP evaluation: the envelope, one subject whose name and
// digest match the target, a target and instrument that say what was
// evaluated and by what, a basis, and verdicts whose counts are their own.
func (s *Statement) Validate() error {
	var problems []string
	note := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	if s.Type != attestation.StatementType {
		note("_type is %q, want %q", s.Type, attestation.StatementType)
	}
	if s.PredicateType != PredicateType {
		note("predicateType is %q, want %q", s.PredicateType, PredicateType)
	}
	s.validateSubject(note)
	p := s.Predicate
	if p.SubjectKind != SubjectKindDescriptor {
		note("subjectKind is %q, want %q", p.SubjectKind, SubjectKindDescriptor)
	}
	validateTarget(p.Target, note)
	validateRun(p, note)
	validateCard(p.Card, note)
	validateVerdicts(p, note)

	if len(problems) > 0 {
		return fmt.Errorf("a2a: %s", strings.Join(problems, "; "))
	}
	return nil
}

// validateSubject checks that the statement is about exactly one agent,
// named as the predicate names it, with a digest that still covers it.
func (s *Statement) validateSubject(note func(string, ...any)) {
	switch len(s.Subject) {
	case 1:
		sub := s.Subject[0]
		want := strings.TrimSpace(s.Predicate.Target.Endpoint)
		if strings.TrimSpace(sub.Name) != want {
			note("the subject is named %q but the predicate is about %q", sub.Name, want)
		}
		switch found, mismatch := subjectdigest.Check(sub.Digest, descriptor(s.Predicate.Target)); {
		case !found:
			note("the subject has no %s digest", subjectdigest.Names)
		case mismatch != "":
			note("the subject %s digest does not cover the target it names: a statement whose predicate was edited after the fact", mismatch)
		}
	case 0:
		note("no subject: the statement is about nothing")
	default:
		note("%d subjects: a passmcp evaluation is about one agent", len(s.Subject))
	}
}

// validateTarget checks that the target is an absolute URL whose scheme is
// the transport it claims.
func validateTarget(t Target, note func(string, ...any)) {
	if t.Transport != "http" && t.Transport != "https" {
		note("the target transport is %q, want \"https\" or \"http\"", t.Transport)
	}
	u, err := url.Parse(strings.TrimSpace(t.Endpoint))
	switch {
	case strings.TrimSpace(t.Endpoint) == "":
		note("the target has no endpoint")
	case err != nil || u.Host == "":
		note("the target endpoint %q is not an absolute URL", t.Endpoint)
	case u.Scheme != t.Transport:
		note("the target endpoint is %s but the transport says %s", u.Scheme, t.Transport)
	}
}

// validateRun checks the instrument, the run time and the basis.
func validateRun(p Evaluation, note func(string, ...any)) {
	if p.Instrument.Name == "" || p.Instrument.Version == "" {
		note("the instrument is not identified")
	}
	if p.RanAt.IsZero() {
		note("no run time: a verdict about a live service is a verdict about a moment")
	}
	if p.JudgedAgainst.CheckInventory == "" {
		note("judgedAgainst.checkInventory is empty: the ids cannot be resolved to a catalogue")
	}
	if p.Score != nil && p.JudgedAgainst.Rubric == "" {
		note("a score with no rubric version, which is a number nothing can be compared to")
	}
}

// validateCard checks the card description, when there is one.
func validateCard(c *Card, note func(string, ...any)) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.URL) == "" {
		note("the card has no URL")
	}
	if c.Digest != "" && !hexDigest.MatchString(c.Digest) {
		note("the card digest %q is not a hex SHA-256", c.Digest)
	}
	if c.KeyID != "" && !c.Signed {
		note("the card names a verifying key but says it was not signed")
	}
}

// validateVerdicts checks that there are verdicts, each with an id and a
// known status, and that the counts are theirs.
func validateVerdicts(p Evaluation, note func(string, ...any)) {
	if len(p.Verdicts) == 0 {
		note("no verdicts")
		return
	}
	var counted attestation.Counts
	for i, v := range p.Verdicts {
		if v.ID == "" {
			note("verdict %d has no id", i)
			continue
		}
		if !tally(&counted, v.Status) {
			note("verdict %s has status %q", v.ID, v.Status)
		}
	}
	if counted != p.Counts {
		note("the counts disagree with the verdicts: says %+v, the verdicts are %+v", p.Counts, counted)
	}
}

// tally adds one verdict of status to c, and reports whether the status is
// known.
func tally(c *attestation.Counts, status string) bool {
	switch status {
	case "pass":
		c.Pass++
	case "warn":
		c.Warn++
	case "fail":
		c.Fail++
	case "skip":
		c.Skip++
	case "info":
		c.Info++
	default:
		return false
	}
	return true
}

// Covers reports whether the statement is about the agent at endpoint: the
// subject carries a sha256 or sha512 digest, and every one of those it
// carries is the digest of the agent's descriptor.
func (s *Statement) Covers(endpoint string) bool {
	if len(s.Subject) != 1 {
		return false
	}
	return subjectdigest.Covers(s.Subject[0].Digest, descriptor(Target{Endpoint: endpoint}))
}

// VerdictFor returns the first verdict with the given check id.
func (s *Statement) VerdictFor(id string) (attestation.Verdict, bool) {
	for _, v := range s.Predicate.Verdicts {
		if v.ID == id {
			return v, true
		}
	}
	return attestation.Verdict{}, false
}
