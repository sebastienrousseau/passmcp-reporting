// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package subjectdigest checks an in-toto DigestSet against the target
// descriptor it claims to cover, in every algorithm the verifier knows.
//
// The attestation and a2a verifiers share it so that the two formats
// accept the same digest sets: every known algorithm a subject carries is
// recomputed and must match, at least one must be present, and an
// algorithm the verifier does not know is ignored, as in-toto allows.
package subjectdigest

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

// algorithm is one digest the verifier can recompute.
type algorithm struct {
	name string
	sum  func([]byte) []byte
}

// known lists the recomputable algorithms in the order a mismatch is
// reported. SubjectFor writes the first; the rest are accepted beside it or
// instead of it.
var known = []algorithm{
	{"sha256", func(b []byte) []byte { s := sha256.Sum256(b); return s[:] }},
	{"sha512", func(b []byte) []byte { s := sha512.Sum512(b); return s[:] }},
}

// Names is the known algorithms, for messages: "sha256 or sha512".
const Names = "sha256 or sha512"

// Check compares set with the digests of descriptor. found reports whether
// set carries any known algorithm; mismatch names the first known algorithm
// present whose value is not the digest of descriptor, or is "" when every
// one matches.
func Check(set map[string]string, descriptor string) (found bool, mismatch string) {
	for _, a := range known {
		got, ok := set[a.name]
		if !ok {
			continue
		}
		found = true
		if got != hex.EncodeToString(a.sum([]byte(descriptor))) && mismatch == "" {
			mismatch = a.name
		}
	}
	return found, mismatch
}

// Covers reports whether set carries at least one known algorithm and every
// known algorithm it carries is the digest of descriptor.
func Covers(set map[string]string, descriptor string) bool {
	found, mismatch := Check(set, descriptor)
	return found && mismatch == ""
}
