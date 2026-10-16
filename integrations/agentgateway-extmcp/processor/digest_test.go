// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"satellion.com/passmcp-reporting/attestation"
)

// TestVerifyAcceptsASha512Subject: the verifier the processor links
// decides which subject digests it recognises. From passmcp-reporting
// v0.0.6 a statement may carry sha512 instead of sha256 (ADR 0003), so a
// processor that still linked v0.0.5 would refuse a valid attestation
// and deny its server.
func TestVerifyAcceptsASha512Subject(t *testing.T) {
	st := valid()
	sum := sha512.Sum512([]byte("http\n" + testEndpoint))
	st.Subject = []attestation.Subject{{Name: testEndpoint, Digest: map[string]string{"sha512": hex.EncodeToString(sum[:])}}}
	path := writeStatement(t, t.TempDir(), "s.json", st)
	if _, err := verify(context.Background(), &Loader{}, Target{Attestation: path, Endpoint: testEndpoint}); err != nil {
		t.Fatalf("a statement whose only subject digest is a correct sha512 was refused: %v", err)
	}
}
