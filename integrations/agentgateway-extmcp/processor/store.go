// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"unicode/utf8"

	"satellion.com/passmcp-reporting/attestation"
)

// Store holds the current configuration and the statements verified
// against it, and replaces both atomically on Reload.
//
// A request reads one immutable Snapshot, so a reload in progress never
// shows a request half of the old table and half of the new.
type Store struct {
	configPath string
	loader     *Loader
	log        *slog.Logger

	mu   sync.RWMutex
	snap *Snapshot
}

// Snapshot is one loaded configuration with its verified statements.
type Snapshot struct {
	// Config is the configuration the snapshot was built from.
	Config *Config
	// Statements maps each configured target to its verified statement;
	// a target whose attestation could not be used is absent.
	Statements map[string]*attestation.Statement
	// Unusable maps each configured target without a statement to why.
	Unusable map[string]string

	entries     map[string]*entry
	passthrough map[string]bool
}

// NewStore returns a store that reads configPath on every Reload. It
// holds nothing until the first Reload succeeds.
func NewStore(configPath string, loader *Loader, log *slog.Logger) *Store {
	if loader == nil {
		loader = &Loader{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Store{configPath: configPath, loader: loader, log: log}
}

// Snapshot returns the current table, or nil before the first successful
// Reload.
func (s *Store) Snapshot() *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

// Reload re-reads the configuration and every attestation it names. A
// configuration that does not parse leaves the previous snapshot in place
// and is returned as an error; an attestation that cannot be loaded or
// verified is logged, recorded in Unusable, and treated as absent, which
// the policy then judges.
func (s *Store) Reload(ctx context.Context) error {
	cfg, err := ReadConfig(s.configPath)
	if err != nil {
		return err
	}
	snap, err := Build(ctx, cfg, s.loader, s.log)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.snap = snap
	s.mu.Unlock()
	s.log.Info("attestations loaded",
		"targets", len(cfg.Targets),
		"verified", len(snap.Statements),
		"unusable", len(snap.Unusable))
	return nil
}

// Build loads and verifies every attestation cfg names. It fails only when
// ctx ends; a target's own problems are recorded, not returned, so one
// bad statement does not take the others down with it.
func Build(ctx context.Context, cfg *Config, loader *Loader, log *slog.Logger) (*Snapshot, error) {
	if loader == nil {
		loader = &Loader{}
	}
	if log == nil {
		log = slog.Default()
	}
	snap := &Snapshot{
		Config:      cfg,
		Statements:  make(map[string]*attestation.Statement),
		Unusable:    make(map[string]string),
		entries:     make(map[string]*entry, len(cfg.Targets)),
		passthrough: make(map[string]bool, len(cfg.PassthroughMethods)),
	}
	for _, m := range cfg.PassthroughMethods {
		snap.passthrough[m] = true
	}
	names := make([]string, 0, len(cfg.Targets))
	for name := range cfg.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t := cfg.Targets[name]
		st, err := verify(ctx, loader, t)
		if err != nil {
			reason := bounded(err.Error())
			log.Warn("attestation unusable", "target", name, "source", t.Attestation, "reason", reason)
			snap.Unusable[name] = reason
			snap.entries[name] = &entry{reason: reason}
			continue
		}
		log.Info("attestation verified",
			"target", name,
			"endpoint", t.Endpoint,
			"score", scoreOf(st),
			"instrument", st.Predicate.Instrument.Name+" "+st.Predicate.Instrument.Version,
			"ranAt", st.Predicate.RanAt)
		snap.Statements[name] = st
		snap.entries[name] = &entry{statement: st}
	}
	return snap, nil
}

// verify loads a target's statement and checks it is a valid passmcp
// attestation about that target. Each refusal names its stage, because
// "unreadable", "not a statement" and "a statement about a different
// server" call for different fixes.
func verify(ctx context.Context, loader *Loader, t Target) (*attestation.Statement, error) {
	b, err := loader.Load(ctx, t.Attestation)
	if err != nil {
		return nil, fmt.Errorf("unreadable: %w", err)
	}
	st, err := attestation.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("invalid: %w", err)
	}
	if err := st.Validate(); err != nil {
		return nil, fmt.Errorf("invalid: %w", err)
	}
	if !st.Covers(t.transport(), t.Endpoint) {
		return nil, fmt.Errorf("does not cover %s %s (it is about %s %s)",
			t.transport(), t.Endpoint, st.Predicate.Target.Transport, st.Predicate.Target.Endpoint)
	}
	return st, nil
}

// ErrNotLoaded is returned when a decision is asked of a store that has
// no snapshot yet.
var ErrNotLoaded = errors.New("processor: no configuration loaded")

// Decide evaluates one target against the snapshot's policy.
func (sn *Snapshot) Decide(target string) Decision {
	return sn.Config.Policy.Evaluate(target, sn.entries[target])
}

// Passthrough reports whether method is answered without a decision.
func (sn *Snapshot) Passthrough(method string) bool {
	return sn.passthrough[method]
}

func scoreOf(st *attestation.Statement) string {
	if st.Predicate.Score == nil {
		return "none"
	}
	return fmt.Sprintf("%g (%s)", st.Predicate.Score.Total, st.Predicate.Score.Grade)
}

// maxReason bounds the reason recorded for an unusable attestation. The
// reason can quote the statement, which may be as large as the loader's
// MaxBytes, and it is sent in every denial for that target and logged on
// every reload.
const maxReason = 512

// bounded cuts s to maxReason bytes on a rune boundary, marking the cut.
func bounded(s string) string {
	if len(s) <= maxReason {
		return s
	}
	cut := maxReason
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
