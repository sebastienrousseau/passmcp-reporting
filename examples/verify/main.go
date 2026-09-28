// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command verify is the smallest consumer of a passmcp attestation: it reads
// one from a file, checks it offline, and prints the score and every
// failing check. It is what a gateway's admission hook does before it
// decides anything, and it needs nothing but the standard library and the
// attestation packages.
//
// It reads the statement's predicate type first and hands the statement to
// the verifier for that type: an MCP evaluation to the attestation package,
// an A2A evaluation to the a2a package. A predicate it does not recognise is
// refused, not guessed at.
//
//	go run ./examples/verify attestation/testdata/statement.json
//	go run ./examples/verify a2a/testdata/statement.json
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"satellion.com/passmcp-reporting/a2a"
	"satellion.com/passmcp-reporting/attestation"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: verify <statement.json>")
		os.Exit(2)
	}
	b, err := os.ReadFile(os.Args[1]) // #nosec G304 G703 -- the file the operator named on the command line
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var head struct {
		PredicateType string `json:"predicateType"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		fmt.Fprintln(os.Stderr, "not a statement:", err)
		os.Exit(1)
	}
	switch head.PredicateType {
	case a2a.PredicateType:
		err = printA2A(b)
	default:
		err = printMCP(b)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "not a passmcp attestation:", err)
		os.Exit(1)
	}
}

func printMCP(b []byte) error {
	st, err := attestation.Parse(b)
	if err != nil {
		return err
	}
	p := st.Predicate
	fmt.Printf("%s %s — %s; judged against MCP %s by %s %s; %d pass, %d warn, %d fail, %d skip\n",
		p.Target.Transport, p.Target.Endpoint, score(p.Score, p.JudgedAgainst.Rubric), p.JudgedAgainst.SpecRevision,
		p.Instrument.Name, p.Instrument.Version, p.Counts.Pass, p.Counts.Warn, p.Counts.Fail, p.Counts.Skip)
	printFailures(p.Verdicts)
	return nil
}

func printA2A(b []byte) error {
	st, err := a2a.Parse(b)
	if err != nil {
		return err
	}
	p := st.Predicate
	fmt.Printf("a2a %s — %s; judged against A2A %s by %s %s; %d pass, %d warn, %d fail, %d skip\n",
		p.Target.Endpoint, score(p.Score, p.JudgedAgainst.Rubric), p.JudgedAgainst.ProtocolVersion,
		p.Instrument.Name, p.Instrument.Version, p.Counts.Pass, p.Counts.Warn, p.Counts.Fail, p.Counts.Skip)
	printFailures(p.Verdicts)
	return nil
}

func score(s *attestation.Score, rubric string) string {
	if s == nil {
		return "no score"
	}
	return fmt.Sprintf("score %.0f/100 (%s) under rubric %s", s.Total, s.Grade, rubric)
}

func printFailures(vs []attestation.Verdict) {
	for _, v := range vs {
		if v.Status == "fail" {
			fmt.Printf("  fail  %-8s %-40s %s\n", v.Severity, v.ID, v.Doc)
		}
	}
}
