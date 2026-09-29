// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const profile = `mode: set
satellion.com/m/a/a.go:1.1,2.2 6 1
satellion.com/m/a/a.go:3.1,4.2 2 0
satellion.com/m/examples/x/main.go:1.1,9.2 10 0
`

func writeProfile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunWritesTheEndpointDocumentToStdout(t *testing.T) {
	var out, errs bytes.Buffer
	code := run([]string{"-profile", writeProfile(t, profile), "-exclude", "/examples/"}, &out, &errs)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs.String())
	}
	if errs.Len() != 0 {
		t.Errorf("stderr should be empty on success, got %q", errs.String())
	}
	var b Badge
	if err := json.Unmarshal(out.Bytes(), &b); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	want := Badge{SchemaVersion: 1, Label: "coverage", Message: "75.0%", Color: "yellow"}
	if b != want {
		t.Errorf("got %+v, want %+v", b, want)
	}
}

func TestExcludedFilesAreNotCounted(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"-profile", writeProfile(t, profile)}, &out, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	// 6 of 18 statements without the exclusion.
	if !strings.Contains(out.String(), `"message":"33.3%"`) {
		t.Errorf("without -exclude the example should count, got %s", out.String())
	}
}

func TestABlockListedTwiceCountsOnceAndIsCoveredIfEitherRanIt(t *testing.T) {
	blocks, err := parseProfile(strings.NewReader("mode: count\nf.go:1.1,2.2 4 0\nf.go:1.1,2.2 4 3\ng.go:1.1,2.2 4 0\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	pct, err := percent(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if pct != 50 {
		t.Errorf("got %v%%, want 50%%", pct)
	}
}

func TestUsageAndInputErrorsGoToStderr(t *testing.T) {
	cases := map[string]struct {
		args []string
		code int
		want string
	}{
		"no profile":        {nil, 2, "-profile is required"},
		"unknown flag":      {[]string{"-nope"}, 2, "flag provided but not defined"},
		"missing file":      {[]string{"-profile", filepath.Join(t.TempDir(), "absent")}, 1, "coveragebadge:"},
		"no statements":     {[]string{"-profile", writeProfile(t, "mode: set\n")}, 1, "counts no statements"},
		"malformed line":    {[]string{"-profile", writeProfile(t, "mode: set\nnonsense\n")}, 1, "line 2: not a cover profile block"},
		"bad stmt count":    {[]string{"-profile", writeProfile(t, "f.go:1.1,2.2 x 1\n")}, 1, "bad statement count"},
		"negative stmts":    {[]string{"-profile", writeProfile(t, "f.go:1.1,2.2 -1 1\n")}, 1, "bad statement count"},
		"bad exec count":    {[]string{"-profile", writeProfile(t, "f.go:1.1,2.2 1 y\n")}, 1, "bad execution count"},
		"negative exec":     {[]string{"-profile", writeProfile(t, "f.go:1.1,2.2 1 -4\n")}, 1, "bad execution count"},
		"no colon in block": {[]string{"-profile", writeProfile(t, "f.go 1 1\n")}, 1, "not a cover profile block"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var out, errs bytes.Buffer
			if code := run(tc.args, &out, &errs); code != tc.code {
				t.Errorf("exit %d, want %d", code, tc.code)
			}
			if out.Len() != 0 {
				t.Errorf("stdout should be empty on failure, got %q", out.String())
			}
			if !strings.Contains(errs.String(), tc.want) {
				t.Errorf("stderr %q does not mention %q", errs.String(), tc.want)
			}
		})
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestAnUnwritableStdoutFails(t *testing.T) {
	var errs bytes.Buffer
	if code := run([]string{"-profile", writeProfile(t, profile)}, failWriter{}, &errs); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errs.String(), "closed") {
		t.Errorf("stderr %q should carry the write error", errs.String())
	}
}

func TestColourThresholds(t *testing.T) {
	cases := []struct {
		pct   float64
		msg   string
		color string
	}{
		{100, "100.0%", "brightgreen"},
		{90, "90.0%", "brightgreen"},
		{89.96, "90.0%", "brightgreen"}, // rounded first, so message and colour agree
		{89.9, "89.9%", "green"},
		{85, "85.0%", "green"},
		{84.9, "84.9%", "yellow"},
		{70, "70.0%", "yellow"},
		{69.9, "69.9%", "red"},
		{0, "0.0%", "red"},
	}
	for _, tc := range cases {
		b := badgeFor(tc.pct)
		if b.Message != tc.msg || b.Color != tc.color {
			t.Errorf("badgeFor(%v) = %q %q, want %q %q", tc.pct, b.Message, b.Color, tc.msg, tc.color)
		}
	}
}

func TestSplitListDropsBlanks(t *testing.T) {
	got := splitList(" /examples/ ,, /scripts/ ")
	if len(got) != 2 || got[0] != "/examples/" || got[1] != "/scripts/" {
		t.Errorf("got %q", got)
	}
	if splitList("") != nil {
		t.Error("an empty list should be nil")
	}
}
