// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package processor is an agentgateway ExtMcp policy processor that gates
// MCP traffic on passmcp attestations.
//
// agentgateway calls the processor's CheckRequest before forwarding an MCP
// method to a backend, naming the backend(s) the call targets. The
// processor answers from a table of attestations it verified at startup,
// one per backend name, against a policy: a minimum score and a set of
// check categories in which a failing verdict is disqualifying. A backend
// with no usable attestation is denied unless the policy says otherwise.
//
// Verification is the attestation package from the root module — the
// same code any Go consumer embeds — and it is offline: the processor
// never calls passmcp, or the server it is deciding about, to make a
// decision. Statements are read from a file or fetched over https once,
// at startup and on reload, never per request.
package processor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Config is the processor's configuration file.
type Config struct {
	// Targets maps an agentgateway backend name — what arrives in
	// McpRequest.service_names — to the attestation that vouches for it.
	Targets map[string]Target `json:"targets"`
	// Policy is what a statement must show for its target to pass.
	Policy Policy `json:"policy"`
	// PassthroughMethods are JSON-RPC methods answered with Pass without
	// consulting any attestation, such as "initialize". Empty gates every
	// method the gateway sends; the gateway's own `methods` allowlist on
	// the processor is the better place to narrow that, and this exists
	// for operators who configured `*` there.
	PassthroughMethods []string `json:"passthroughMethods,omitempty"`
}

// Target is one gated backend.
type Target struct {
	// Attestation is where the statement comes from: a file path, or an
	// https URL. Nothing else is accepted.
	Attestation string `json:"attestation"`
	// Endpoint is the MCP endpoint the statement must cover — the address
	// passmcp evaluated. It is compared through the statement's subject
	// digest, not as a string.
	Endpoint string `json:"endpoint"`
	// Transport is "http" or "stdio"; empty means "http".
	Transport string `json:"transport,omitempty"`
}

// Policy is what an attestation must show for its target to pass.
type Policy struct {
	// MinScore is the lowest acceptable score, 0–100. Zero disables the
	// score check. A statement with no score fails a non-zero MinScore,
	// because "not scored" is not "scored high enough".
	MinScore float64 `json:"minScore"`
	// DenyFailIn lists check categories (the verdict's phase, such as
	// "auth" or "protocol") in which any failing verdict denies the
	// target, whatever the score.
	DenyFailIn []string `json:"denyFailIn,omitempty"`
	// RequireAttestation denies a target that has no usable statement:
	// one not configured, unreadable, invalid, or covering a different
	// endpoint. Absent means true. False lets such targets through, with
	// a log line, which is a rollout mode rather than a policy.
	RequireAttestation *bool `json:"requireAttestation,omitempty"`
}

// Requires reports whether a target without a usable statement is denied.
func (p Policy) Requires() bool {
	return p.RequireAttestation == nil || *p.RequireAttestation
}

// ReadConfig reads and validates a configuration file.
func ReadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path) // #nosec G304 G703 -- the path the operator named on the command line
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return ParseConfig(b)
}

// ParseConfig parses and validates configuration from JSON.
func ParseConfig(b []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate reports every way the configuration is unusable.
func (c *Config) Validate() error {
	var errs []error
	if len(c.Targets) == 0 {
		errs = append(errs, errors.New("config: no targets"))
	}
	names := make([]string, 0, len(c.Targets))
	for name := range c.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t := c.Targets[name]
		if strings.TrimSpace(name) == "" {
			errs = append(errs, errors.New("config: a target has an empty name"))
		}
		if t.Attestation == "" {
			errs = append(errs, fmt.Errorf("config: target %q has no attestation", name))
		} else if strings.Contains(t.Attestation, "://") && !strings.HasPrefix(t.Attestation, "https://") {
			errs = append(errs, fmt.Errorf("config: target %q: attestation must be a file path or an https URL", name))
		}
		if t.Endpoint == "" {
			errs = append(errs, fmt.Errorf("config: target %q has no endpoint", name))
		}
		switch t.Transport {
		case "", "http", "stdio":
		default:
			errs = append(errs, fmt.Errorf("config: target %q: transport must be http or stdio", name))
		}
	}
	if c.Policy.MinScore < 0 || c.Policy.MinScore > 100 {
		errs = append(errs, errors.New("config: policy.minScore must be between 0 and 100"))
	}
	for _, cat := range c.Policy.DenyFailIn {
		if strings.TrimSpace(cat) == "" {
			errs = append(errs, errors.New("config: policy.denyFailIn has an empty category"))
		}
	}
	return errors.Join(errs...)
}

// transport returns the target's transport with the default applied.
func (t Target) transport() string {
	if t.Transport == "" {
		return "http"
	}
	return t.Transport
}
