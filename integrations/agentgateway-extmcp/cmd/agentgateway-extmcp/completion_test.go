// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// completionFor runs the command with -completion and returns what it printed.
func completionFor(t *testing.T, shell string) (string, error) {
	t.Helper()
	var out, errw bytes.Buffer
	saved := stdout
	stdout = &out
	t.Cleanup(func() { stdout = saved })
	err := run(context.Background(), []string{"-completion", shell}, &errw, nil)
	return out.String(), err
}

func TestCompletionCoversEveryFlagInEveryShell(t *testing.T) {
	flags := []string{"config", "listen", "reload-interval", "max-bytes", "fetch-timeout", "log-level", "completion"}
	for _, shell := range completionShells {
		script, err := completionFor(t, shell)
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		for _, f := range flags {
			if !strings.Contains(script, f) {
				t.Errorf("%s completion does not mention -%s", shell, f)
			}
		}
		for _, want := range map[string][]string{
			"bash": {"complete -F _agentgateway_extmcp agentgateway-extmcp", "-config) COMPREPLY=($(compgen -f", `-log-level) COMPREPLY=($(compgen -W "debug info warn error"`},
			"zsh":  {"#compdef agentgateway-extmcp", "'-config[", ":path:_files", ":value:(debug info warn error)"},
			"fish": {"-o config", "-r -F", `-o log-level -d "debug, info, warn or error" -x -a "debug info warn error"`},
		}[shell] {
			if !strings.Contains(script, want) {
				t.Errorf("%s completion lacks %q:\n%s", shell, want, script)
			}
		}
	}
}

func TestCompletionNeedsNoConfig(t *testing.T) {
	// -config is required to serve, not to print a completion script.
	if _, err := completionFor(t, "bash"); err != nil {
		t.Errorf("-completion without -config: %v", err)
	}
}

func TestCompletionRefusesAnUnknownShell(t *testing.T) {
	script, err := completionFor(t, "powershell")
	if err == nil || !strings.Contains(err.Error(), "bash, fish, zsh") {
		t.Errorf("err = %v, want a refusal naming the supported shells", err)
	}
	if script != "" {
		t.Errorf("wrote %q for an unknown shell", script)
	}
}

func TestZshEscape(t *testing.T) {
	if got := zshEscape("a [b]: it's"); got != `a \[b\]\: it'\''s` {
		t.Errorf("zshEscape = %q", got)
	}
}
