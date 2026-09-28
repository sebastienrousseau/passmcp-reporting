// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package attestation

import "sort"

// A score says how a server did; a delta says how it changed. Drift is the
// product: a server that passed review and then changed what it does is the
// one that gets through a one-off gate, and the change is visible only by
// comparing two statements about it. The comparison is by check, not by
// score, because a score that did not move can hide a check that went from
// pass to fail beside another that went the other way.

// Change is one check whose outcome differs between two statements. From or
// To is empty when the check appears in only one of them.
type Change struct {
	ID           string `json:"id"`
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	FromSeverity string `json:"fromSeverity,omitempty"`
	ToSeverity   string `json:"toSeverity,omitempty"`
}

// Delta is how a later statement differs from an earlier one about the same
// target.
type Delta struct {
	// Comparable is false when the two were judged against different
	// rubrics or check inventories. The verdicts still compare, since ids
	// are stable within an inventory version, but the score does not.
	Comparable bool `json:"comparable"`
	// Regressed are checks whose outcome got worse: pass or info to warn
	// or fail, or warn to fail. This is what a drift gate fails on.
	Regressed []Change `json:"regressed,omitempty"`
	// Improved are checks whose outcome got better.
	Improved []Change `json:"improved,omitempty"`
	// SeverityChanged are checks failing in both, with a different
	// severity.
	SeverityChanged []Change `json:"severityChanged,omitempty"`
	// Unassessed are checks measured in the earlier statement and skipped
	// or absent in the later one: coverage the later run lost, which is
	// not the same as the server getting worse.
	Unassessed []Change `json:"unassessed,omitempty"`
	// Added are checks the earlier statement did not measure.
	Added []Change `json:"added,omitempty"`
	// ScoreFrom and ScoreTo are the totals, when both carry one and the
	// two are comparable.
	ScoreFrom *float64 `json:"scoreFrom,omitempty"`
	ScoreTo   *float64 `json:"scoreTo,omitempty"`
}

// Regression reports whether any check got worse.
func (d Delta) Regression() bool { return len(d.Regressed) > 0 }

// rank orders outcomes by how bad they are. Skip is not ranked: a check
// that did not run says nothing about the server.
var rank = map[string]int{"pass": 1, "info": 1, "warn": 2, "fail": 3}

// Compare returns how after differs from before. It assumes the caller has
// already established that both are about the same target; Covers is how.
func Compare(before, after *Statement) Delta {
	b, a := verdictIndex(before), verdictIndex(after)
	d := Delta{
		Comparable: before.Predicate.JudgedAgainst.Rubric == after.Predicate.JudgedAgainst.Rubric &&
			before.Predicate.JudgedAgainst.CheckInventory == after.Predicate.JudgedAgainst.CheckInventory,
	}

	for _, id := range unionIDs(b, a) {
		was, inBefore := b[id]
		now, inAfter := a[id]
		d.classify(Change{ID: id, From: was.Status, To: now.Status, FromSeverity: was.Severity, ToSeverity: now.Severity},
			inBefore, inAfter)
	}

	if d.Comparable && before.Predicate.Score != nil && after.Predicate.Score != nil {
		from, to := before.Predicate.Score.Total, after.Predicate.Score.Total
		d.ScoreFrom, d.ScoreTo = &from, &to
	}
	return d
}

// unionIDs returns every check id in either index, sorted, so a delta is
// reported in the same order however the verdicts were written.
func unionIDs(b, a map[string]Verdict) []string {
	ids := make([]string, 0, len(b)+len(a))
	for id := range b {
		ids = append(ids, id)
	}
	for id := range a {
		if _, ok := b[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// classify files one check's change under the list it belongs to. A check
// absent from either side, or unranked there (skip), is an addition or an
// unassessed check rather than a movement; a change within the same rank
// matters only when a failure's severity moved.
func (d *Delta) classify(c Change, inBefore, inAfter bool) {
	wasRanked, nowRanked := rank[c.From] > 0, rank[c.To] > 0
	switch {
	case !inBefore || !wasRanked:
		if nowRanked {
			d.Added = append(d.Added, c)
		}
	case !inAfter || !nowRanked:
		d.Unassessed = append(d.Unassessed, c)
	case rank[c.To] > rank[c.From]:
		d.Regressed = append(d.Regressed, c)
	case rank[c.To] < rank[c.From]:
		d.Improved = append(d.Improved, c)
	case c.From == "fail" && c.FromSeverity != c.ToSeverity:
		d.SeverityChanged = append(d.SeverityChanged, c)
	}
}

// verdictIndex keeps the first verdict per id, which is what VerdictFor
// returns: a family id produces several, canonically ordered.
func verdictIndex(s *Statement) map[string]Verdict {
	out := map[string]Verdict{}
	for _, v := range s.Predicate.Verdicts {
		if _, seen := out[v.ID]; !seen {
			out[v.ID] = v
		}
	}
	return out
}
