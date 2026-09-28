// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package spec publishes the JSON Schemas of passmcp's formats — the MCP and
// A2A evaluation attestations and the security graph — as data a Go program
// can embed, for a consumer that validates against the schema
// rather than importing the verifier — a policy engine with its own schema
// validator, or a test that a statement written by something else still
// conforms.
//
// The files are generated from the attestation, a2a and graph packages'
// types by scripts/specgen, and CI fails when one drifts; the bytes here are
// the same bytes as the files under spec/ in the repository.
package spec

import _ "embed"

// AttestationSchema is JSON Schema 2020-12 for an in-toto Statement carrying
// the mcp-evaluation/v1 predicate.
//
//go:embed attestation/mcp-evaluation-v1.schema.json
var AttestationSchema []byte

// A2AAttestationSchema is JSON Schema 2020-12 for an in-toto Statement
// carrying the a2a-evaluation/v1 predicate.
//
//go:embed attestation/a2a-evaluation-v1.schema.json
var A2AAttestationSchema []byte

// GraphSchema is JSON Schema 2020-12 for a security graph document.
//
//go:embed graph/graph-v1.schema.json
var GraphSchema []byte
