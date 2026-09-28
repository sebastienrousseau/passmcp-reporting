// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultMaxBytes is the largest statement the loader reads. A statement
// with every verdict for a large server is tens of kilobytes; a megabyte
// is a bug or an attack.
const DefaultMaxBytes = 1 << 20

// DefaultTimeout bounds one https fetch.
const DefaultTimeout = 10 * time.Second

// Loader reads statement bytes from a file path or an https URL.
//
// It is the only part of the processor that touches the filesystem or the
// network, and it does so at load time, never on the request path.
type Loader struct {
	// Client makes https requests; nil uses http.DefaultClient. Tests
	// inject one that trusts their server's certificate.
	Client *http.Client
	// MaxBytes caps a statement's size; zero means DefaultMaxBytes.
	MaxBytes int64
	// Timeout bounds one fetch; zero means DefaultTimeout.
	Timeout time.Duration
}

// Load returns the bytes at source. An https URL is fetched; anything
// else with a scheme is refused; a bare path is read from disk. Both
// forms respect MaxBytes.
func (l *Loader) Load(ctx context.Context, source string) ([]byte, error) {
	switch {
	case strings.HasPrefix(source, "https://"):
		return l.fetch(ctx, source)
	case strings.Contains(source, "://"):
		return nil, errors.New("only https URLs and file paths are accepted")
	default:
		return l.read(source)
	}
}

func (l *Loader) maxBytes() int64 {
	if l.MaxBytes > 0 {
		return l.MaxBytes
	}
	return DefaultMaxBytes
}

func (l *Loader) read(path string) ([]byte, error) {
	f, err := os.Open(path) // #nosec G304 G703 -- the path the operator's configuration names
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return l.capped(f)
}

func (l *Loader) fetch(ctx context.Context, url string) ([]byte, error) {
	timeout := l.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	client := l.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: %s", resp.Status)
	}
	return l.capped(resp.Body)
}

// capped reads r in full, refusing anything past MaxBytes rather than
// silently truncating it — a truncated statement would fail to parse
// with a message that blames the wrong thing.
func (l *Loader) capped(r io.Reader) ([]byte, error) {
	limit := l.maxBytes()
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("statement exceeds %d bytes", limit)
	}
	return b, nil
}
