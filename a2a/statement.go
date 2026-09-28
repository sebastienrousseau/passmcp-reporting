// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package a2a is the passmcp A2A evaluation attestation: the in-toto statement
// a run produces about one agent that speaks the Agent2Agent protocol, and
// everything needed to read and verify one offline.
//
// It is the A2A sibling of the attestation package's MCP evaluation, with
// the same envelope, the same verdict and score types and the same four
// properties: it states what it was judged against, its subject digest is
// over a target descriptor and says so, it carries every verdict and not
// only the failures, and it verifies offline. A consumer that already trusts
// an MCP evaluation can trust an A2A one the same way.
//
// The two are different predicate types rather than one type with a
// protocol field, because a consumer that recognises only MCP evaluations
// must refuse an A2A one instead of misreading its verdicts.
//
// The JSON Schema in spec/attestation/a2a-evaluation-v1.schema.json is
// generated from these types.
package a2a

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"satellion.com/passmcp-reporting/attestation"
)

// PredicateType identifies a passmcp A2A evaluation.
const PredicateType = "https://satellion.com/attestation/a2a-evaluation/v1"

// SubjectKindDescriptor is what the subject digest covers: a canonical
// descriptor of the agent's base URL, not an artifact.
const SubjectKindDescriptor = "a2a-target-descriptor"

// Statement is the in-toto envelope around an A2A evaluation.
type Statement struct {
	Type          string                `json:"_type"`
	Subject       []attestation.Subject `json:"subject"`
	PredicateType string                `json:"predicateType"`
	Predicate     Evaluation            `json:"predicate"`
}

// Evaluation is the A2A predicate.
type Evaluation struct {
	// SubjectKind says what the subject digest covers.
	SubjectKind string `json:"subjectKind"`
	// Target is the agent that was evaluated.
	Target Target `json:"target"`
	// JudgedAgainst is what the verdicts mean.
	JudgedAgainst Basis `json:"judgedAgainst"`
	// Instrument is what produced the statement.
	Instrument attestation.Instrument `json:"instrument"`
	// RanAt is when the run started, and Took how long it lasted.
	RanAt time.Time `json:"ranAt"`
	Took  string    `json:"took"`
	// Card describes the Agent Card the verdicts were drawn from.
	Card *Card `json:"card,omitempty"`
	// Verdicts is every check that ran, passes included.
	Verdicts []attestation.Verdict `json:"verdicts"`
	// Counts and Score are the summary a policy engine gates on.
	Counts attestation.Counts `json:"counts"`
	Score  *attestation.Score `json:"score,omitempty"`
	// Blocked is why the run stopped early, when it did.
	Blocked string `json:"blocked,omitempty"`
	// TraceID ties the statement to the telemetry the run recorded.
	TraceID string `json:"traceId,omitempty"`
}

// Target identifies the evaluated agent without disclosing credentials.
type Target struct {
	// Transport is "https", or "http" for a plain-HTTP agent.
	Transport string `json:"transport"`
	// Endpoint is the agent's base URL.
	Endpoint string `json:"endpoint"`
	// Agent is what the Agent Card said the agent was.
	Agent *AgentIdentity `json:"agent,omitempty"`
}

// AgentIdentity is the Agent Card's claim about the agent.
type AgentIdentity struct {
	Name            string `json:"name,omitempty"`
	Version         string `json:"version,omitempty"`
	ProtocolVersion string `json:"protocolVersion,omitempty"`
}

// Basis is what the verdicts were judged against.
type Basis struct {
	// ProtocolVersion is the A2A protocol version the card declared.
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	// Rubric is the scoring rubric behind Score; required with a score.
	Rubric string `json:"rubric,omitempty"`
	// CheckInventory is the version of the check catalogue the ids come
	// from.
	CheckInventory string `json:"checkInventory,omitempty"`
}

// Card describes the Agent Card as it was fetched.
type Card struct {
	// URL is where the card was fetched from.
	URL string `json:"url"`
	// Digest is the SHA-256 of the card's JCS canonical form, in hex, so
	// a later run can tell whether the card changed.
	Digest string `json:"digest,omitempty"`
	// Signed says whether the card carried a signature.
	Signed bool `json:"signed"`
	// KeyID names the key that verified the signature, when one did.
	KeyID string `json:"keyId,omitempty"`
}

// SubjectFor returns the one subject a statement about t carries: its
// endpoint as the name and the digest of its canonical descriptor.
func SubjectFor(t Target) attestation.Subject {
	return attestation.Subject{
		Name:   t.Endpoint,
		Digest: map[string]string{"sha256": digest(descriptor(t))},
	}
}

// descriptor is the canonical string the subject digest covers. The "a2a"
// prefix keeps an agent and an MCP server at the same URL from sharing a
// digest.
func descriptor(t Target) string {
	return "a2a\n" + strings.TrimSpace(t.Endpoint)
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Marshal renders the statement as the JSON that gets signed, indented as
// the MCP statement is.
func (s *Statement) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
