// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileName is the one file a graph store holds, inside the store directory.
//
// The store is a directory rather than a file so that tools sharing it can
// keep their own files beside the graph (passmcp-graph keeps the evidence it
// ingested), while the graph itself stays a single canonical document: easy
// to diff, to commit, and to read with nothing but a JSON parser.
const FileName = "graph.json"

// Load reads the graph stored in dir. A directory with no graph yet, or no
// directory at all, is an empty graph.
func Load(dir string) (*Graph, error) {
	b, err := os.ReadFile(filepath.Join(dir, FileName)) // #nosec G304 -- the store directory the operator named
	if errors.Is(err, fs.ErrNotExist) {
		// Absent means empty only when dir itself is not something else.
		// Windows reports a path through a regular file as "not found"
		// where Unix says "not a directory", so the check is made here
		// rather than left to the error ReadFile returns.
		if info, serr := os.Stat(dir); serr == nil && !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	g, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, FileName), err)
	}
	return g, nil
}

// Save validates g and writes it canonically to dir, creating the directory
// if needed. The write goes to a temporary file first and is renamed into
// place, so a reader never sees a half-written graph.
func Save(dir string, g *Graph) error {
	if err := g.Validate(); err != nil {
		return err
	}
	b, err := g.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, FileName+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, FileName))
}
