// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command apidoc fails when an exported identifier in a public package has
// no doc comment. pkgsite renders these comments as the API reference, so
// an undocumented identifier is a hole in the published documentation.
//
//	go run ./scripts/apidoc ./attestation ./spec
package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	missing, err := check(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "apidoc:", err)
		os.Exit(1)
	}
	os.Exit(report(os.Stderr, missing))
}

// report writes the verdict and returns the exit status: 0 when nothing is
// missing, 1 otherwise, and 1 as well if the verdict could not be written.
func report(w io.Writer, missing []string) int {
	var b strings.Builder
	status := 0
	if len(missing) == 0 {
		b.WriteString("apidoc: every exported identifier is documented\n")
	} else {
		status = 1
		fmt.Fprintf(&b, "apidoc: %d exported identifiers have no doc comment:\n", len(missing))
		for _, m := range missing {
			b.WriteString("  " + m + "\n")
		}
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return 1
	}
	return status
}

// check returns "dir: Name" for every undocumented exported identifier in
// the given package directories, test files excluded.
func check(dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("name at least one package directory")
	}
	var missing []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		var files []*ast.File
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("%s: no Go package", dir)
		}
		d, err := doc.NewFromFiles(fset, files, dir)
		if err != nil {
			return nil, err
		}
		missing = append(missing, undocumented(filepath.Clean(dir), d)...)
	}
	sort.Strings(missing)
	return missing, nil
}

func undocumented(dir string, d *doc.Package) []string {
	var out []string
	note := func(name, text string) {
		if strings.TrimSpace(text) == "" {
			out = append(out, dir+": "+name)
		}
	}
	if strings.TrimSpace(d.Doc) == "" {
		out = append(out, dir+": package "+d.Name)
	}
	values(d.Consts, note)
	values(d.Vars, note)
	for _, f := range d.Funcs {
		note(f.Name, f.Doc)
	}
	for _, t := range d.Types {
		note(t.Name, t.Doc)
		values(t.Consts, note)
		values(t.Vars, note)
		for _, f := range t.Funcs {
			note(f.Name, f.Doc)
		}
		for _, m := range t.Methods {
			note(t.Name+"."+m.Name, m.Doc)
		}
	}
	return out
}

// values reports each exported constant or variable in a declaration list,
// taking the comment from the group or, in a grouped declaration, from the
// name's own line.
func values(vs []*doc.Value, note func(name, text string)) {
	for _, v := range vs {
		for _, n := range v.Names {
			if !ast.IsExported(n) {
				continue
			}
			text := v.Doc
			if text == "" {
				text = specDoc(v.Decl, n)
			}
			note(n, text)
		}
	}
}

// specDoc is the comment on one name inside a grouped declaration, where
// each constant or variable carries its own comment rather than the group.
func specDoc(decl *ast.GenDecl, name string) string {
	for _, s := range decl.Specs {
		vs, ok := s.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, n := range vs.Names {
			if n.Name == name {
				if vs.Doc != nil {
					return vs.Doc.Text()
				}
				if vs.Comment != nil {
					return vs.Comment.Text()
				}
			}
		}
	}
	return ""
}
