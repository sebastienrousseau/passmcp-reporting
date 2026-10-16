// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"fmt"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/attestation"
)

// Decision is the outcome for one target.
type Decision struct {
	// Allow is whether the target may be called.
	Allow bool
	// Reason says why, in a sentence that names the target. On a denial
	// it is what the MCP client sees; on a pass it is what the log sees.
	Reason string
	// Statement is the attestation the decision rests on, when there is
	// one. A pass with no statement is only possible when the policy does
	// not require one.
	Statement *attestation.Statement
}

// entry is what the store holds for one configured target: a verified
// statement, or the reason there is none.
type entry struct {
	statement *attestation.Statement
	reason    string
}

// Evaluate decides one target against the policy. A nil entry is a target
// the configuration does not know; an entry with no statement is one whose
// attestation could not be loaded or verified, with the reason recorded.
func (p Policy) Evaluate(target string, e *entry) Decision {
	if e == nil || e.statement == nil {
		return p.withoutStatement(target, e)
	}
	st := e.statement
	pred := st.Predicate
	if reason, denied := p.scoreDenial(target, pred.Score); denied {
		return Decision{Statement: st, Reason: reason}
	}
	if failing := p.failingIn(pred.Verdicts); len(failing) > 0 {
		return Decision{Statement: st, Reason: fmt.Sprintf("target %q: failing checks in category %s", target, describeFailing(failing))}
	}
	reason := fmt.Sprintf("target %q: attestation passes policy", target)
	if pred.Score != nil {
		reason = fmt.Sprintf("target %q: score %g (%s) passes policy", target, pred.Score.Total, pred.Score.Grade)
	}
	return Decision{Allow: true, Reason: reason, Statement: st}
}

// withoutStatement decides a target that has no verified statement: one the
// configuration does not know (a nil entry), or one whose attestation could
// not be used. It is denied when the policy requires an attestation.
func (p Policy) withoutStatement(target string, e *entry) Decision {
	if e == nil {
		if p.Requires() {
			return Decision{Reason: fmt.Sprintf("target %q: no attestation configured", target)}
		}
		return Decision{Allow: true, Reason: fmt.Sprintf("target %q: no attestation configured; policy does not require one", target)}
	}
	if p.Requires() {
		return Decision{Reason: fmt.Sprintf("target %q: attestation unusable: %s", target, e.reason)}
	}
	return Decision{Allow: true, Reason: fmt.Sprintf("target %q: attestation unusable (%s); policy does not require one", target, e.reason)}
}

// scoreDenial returns the reason a statement's score fails the policy's
// minimum, and whether it does. A minimum that is not positive accepts any
// score, or none; a positive one refuses a statement that carries no score.
func (p Policy) scoreDenial(target string, score *attestation.Score) (string, bool) {
	switch {
	case !(p.MinScore > 0):
		return "", false
	case score == nil:
		return fmt.Sprintf("target %q: attestation carries no score; policy requires at least %g", target, p.MinScore), true
	case score.Total < p.MinScore:
		return fmt.Sprintf("target %q: score %g (%s) is below the minimum %g", target, score.Total, score.Grade, p.MinScore), true
	}
	return "", false
}

// describeFailing renders failing checks per category, categories sorted, as
// "auth (a, b); protocol (c)".
func describeFailing(failing map[string][]string) string {
	cats := make([]string, 0, len(failing))
	for cat := range failing {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	parts := make([]string, 0, len(cats))
	for _, cat := range cats {
		parts = append(parts, fmt.Sprintf("%s (%s)", cat, strings.Join(failing[cat], ", ")))
	}
	return strings.Join(parts, "; ")
}

// failingIn returns, per denied category, the ids of the checks that fail
// in it. Only "fail" counts: a warning is a finding, not a disqualification,
// and a skipped check was not measured.
func (p Policy) failingIn(verdicts []attestation.Verdict) map[string][]string {
	if len(p.DenyFailIn) == 0 {
		return nil
	}
	denied := make(map[string]bool, len(p.DenyFailIn))
	for _, cat := range p.DenyFailIn {
		denied[cat] = true
	}
	var out map[string][]string
	for _, v := range verdicts {
		if v.Status != "fail" || !denied[v.Phase] {
			continue
		}
		if out == nil {
			out = make(map[string][]string)
		}
		out[v.Phase] = append(out[v.Phase], v.ID)
	}
	return out
}
