// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"satellion.com/passmcp-reporting/attestation"
)

// The reason an attestation is unusable reaches every MCP client whose
// call to that target is denied. It must not carry the attestation URL,
// whose query can be a credential (a presigned object URL), and it must
// not grow with the statement, which can be as large as MaxBytes.

func TestAFailedFetchDoesNotRepeatTheURL(t *testing.T) {
	const secret = "X-Amz-Signature=0123456789abcdef"
	// Port 1 on loopback refuses the connection at once.
	_, err := (&Loader{}).Load(context.Background(), "https://127.0.0.1:1/statement.json?"+secret)
	if err == nil {
		t.Fatal("a fetch from a closed port succeeded")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "127.0.0.1:1/statement.json") {
		t.Errorf("the error repeats the URL: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "fetch: ") {
		t.Errorf("err = %v, want it to say the fetch failed", err)
	}
}

func TestAnUnusableReasonIsBounded(t *testing.T) {
	dir := t.TempDir()
	st := valid()
	// A statement about an endpoint with a very long name, configured
	// against another endpoint: the refusal quotes the statement's.
	long := testEndpoint + "/" + strings.Repeat("é", 4000)
	st.Predicate.Target.Endpoint = long
	st.Subject = []attestation.Subject{attestation.SubjectFor(st.Predicate.Target)}
	path := writeStatement(t, dir, "long.json", st)
	s := loadedStore(t, dir, Config{Targets: map[string]Target{"mcp": {Attestation: path, Endpoint: testEndpoint}}})

	reason := s.Snapshot().Unusable["mcp"]
	if reason == "" {
		t.Fatal("the statement was accepted for another endpoint")
	}
	if len(reason) > maxReason+len("…") {
		t.Errorf("reason is %d bytes, want at most %d", len(reason), maxReason+len("…"))
	}
	if !utf8.ValidString(reason) || !strings.HasSuffix(reason, "…") {
		t.Errorf("reason was not cut on a rune boundary with a marker: %q", reason[len(reason)-8:])
	}
	if d := s.Snapshot().Decide("mcp"); d.Allow || len(d.Reason) > maxReason+200 {
		t.Errorf("decision = %v, %d-byte reason", d.Allow, len(d.Reason))
	}
}
