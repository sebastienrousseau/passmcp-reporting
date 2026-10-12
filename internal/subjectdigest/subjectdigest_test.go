// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package subjectdigest

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

func TestCheck(t *testing.T) {
	const d = "http\nhttps://mcp.example.com/mcp"
	s256 := sha256.Sum256([]byte(d))
	s512 := sha512.Sum512([]byte(d))
	good256, good512 := hex.EncodeToString(s256[:]), hex.EncodeToString(s512[:])
	for name, tc := range map[string]struct {
		set      map[string]string
		found    bool
		mismatch string
	}{
		"empty":             {map[string]string{}, false, ""},
		"nil":               {nil, false, ""},
		"unknown only":      {map[string]string{"md5": "x"}, false, ""},
		"sha256":            {map[string]string{"sha256": good256}, true, ""},
		"sha512":            {map[string]string{"sha512": good512}, true, ""},
		"both":              {map[string]string{"sha256": good256, "sha512": good512}, true, ""},
		"sha256 wrong":      {map[string]string{"sha256": good512, "sha512": good512}, true, "sha256"},
		"sha512 wrong":      {map[string]string{"sha256": good256, "sha512": good256}, true, "sha512"},
		"both wrong":        {map[string]string{"sha256": "", "sha512": ""}, true, "sha256"},
		"uppercase refused": {map[string]string{"sha256": "AB" + good256[2:]}, true, "sha256"},
	} {
		t.Run(name, func(t *testing.T) {
			found, mismatch := Check(tc.set, d)
			if found != tc.found || mismatch != tc.mismatch {
				t.Fatalf("Check = (%v, %q), want (%v, %q)", found, mismatch, tc.found, tc.mismatch)
			}
			if Covers(tc.set, d) != (tc.found && tc.mismatch == "") {
				t.Fatalf("Covers disagrees with Check")
			}
		})
	}
}
