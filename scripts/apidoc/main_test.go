// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writePkg(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	// A test file's identifiers are not API and must not be reported.
	if err := os.WriteFile(filepath.Join(dir, "p_test.go"), []byte("package p\n\nfunc Undocumented() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCheckNamesEveryUndocumentedExportedIdentifier(t *testing.T) {
	dir := writePkg(t, `package p

const (
	// Good is documented.
	Good = 1
	Bad  = 2
	Trailing = 3 // Trailing is documented on its line.
)

var Loose int

func Exported() {}

// Documented is.
func Documented() {}

func unexported() {}

type T struct{}

func (T) M() {}

// N is documented.
func (T) N() {}

func NewT() T { return T{} }
`)
	got, err := check([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	clean := filepath.Clean(dir)
	want := []string{clean + ": Bad", clean + ": Exported", clean + ": Loose", clean + ": NewT", clean + ": T", clean + ": T.M", clean + ": package p"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCheckPassesADocumentedPackage(t *testing.T) {
	dir := writePkg(t, "// Package p is documented.\npackage p\n\n// F is documented.\nfunc F() {}\n")
	got, err := check([]string{dir})
	if err != nil || len(got) != 0 {
		t.Errorf("got %q, %v; want nothing", got, err)
	}
}

func TestCheckRefusesNoPackages(t *testing.T) {
	if _, err := check(nil); err == nil {
		t.Error("no directories accepted")
	}
	if _, err := check([]string{t.TempDir()}); err == nil || !strings.Contains(err.Error(), "no Go package") {
		t.Errorf("empty directory: %v", err)
	}
	if _, err := check([]string{filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Error("a missing directory was accepted")
	}
}

func TestReport(t *testing.T) {
	var b bytes.Buffer
	if report(&b, nil) != 0 || !strings.Contains(b.String(), "every exported identifier") {
		t.Errorf("clean report: %q", b.String())
	}
	b.Reset()
	if report(&b, []string{"p: F"}) != 1 || !strings.Contains(b.String(), "1 exported identifiers") || !strings.Contains(b.String(), "  p: F") {
		t.Errorf("failing report: %q", b.String())
	}
}

func TestThisRepositorysPublicPackagesAreDocumented(t *testing.T) {
	got, err := check([]string{"../../attestation", "../../spec"})
	if err != nil || len(got) != 0 {
		t.Errorf("undocumented: %q, %v", got, err)
	}
}
