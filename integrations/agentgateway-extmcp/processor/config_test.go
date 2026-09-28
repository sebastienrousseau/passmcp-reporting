// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"strings"
	"testing"
)

func TestParseConfigAcceptsTheExampleShape(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{
	  "targets": {"github": {"attestation": "github.json", "endpoint": "https://mcp.example.com/mcp"}},
	  "policy": {"minScore": 70, "denyFailIn": ["auth"]},
	  "passthroughMethods": ["initialize"]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Targets["github"].transport(); got != "http" {
		t.Errorf("default transport = %q, want http", got)
	}
	if !cfg.Policy.Requires() {
		t.Error("requireAttestation should default to true")
	}
	if cfg.Policy.MinScore != 70 || len(cfg.PassthroughMethods) != 1 {
		t.Errorf("parsed %+v", cfg)
	}
}

// TestParseConfigNamesEveryProblem: each broken input must be refused with
// a message that says what is wrong, because a processor that starts on a
// misread configuration gates nothing.
func TestParseConfigNamesEveryProblem(t *testing.T) {
	for want, in := range map[string]string{
		"unexpected EOF":           `{`,
		"unknown field":            `{"targets": {}, "unknownField": {}}`,
		"no targets":               `{"targets": {}}`,
		"has no attestation":       `{"targets": {"a": {"endpoint": "https://x"}}}`,
		"has no endpoint":          `{"targets": {"a": {"attestation": "a.json"}}}`,
		"file path or an https":    `{"targets": {"a": {"attestation": "http://x/a.json", "endpoint": "https://x"}}}`,
		"transport must be":        `{"targets": {"a": {"attestation": "a.json", "endpoint": "https://x", "transport": "carrier-pigeon"}}}`,
		"minScore must be":         `{"targets": {"a": {"attestation": "a.json", "endpoint": "https://x"}}, "policy": {"minScore": 101}}`,
		"denyFailIn has an empty":  `{"targets": {"a": {"attestation": "a.json", "endpoint": "https://x"}}, "policy": {"denyFailIn": [" "]}}`,
		"a target has an empty na": `{"targets": {" ": {"attestation": "a.json", "endpoint": "https://x"}}}`,
	} {
		t.Run(want, func(t *testing.T) {
			_, err := ParseConfig([]byte(in))
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("err = %v, want it to mention %q", err, want)
			}
		})
	}
}

func TestReadConfigReportsAMissingFile(t *testing.T) {
	_, err := ReadConfig(t.TempDir() + "/absent.json")
	if err == nil || !strings.Contains(err.Error(), "config:") {
		t.Fatalf("err = %v", err)
	}
}
