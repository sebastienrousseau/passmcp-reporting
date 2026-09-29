<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# agentgateway-extmcp

An [agentgateway](https://github.com/agentgateway/agentgateway) `mcpGuardrails`
processor that gates MCP backends on passmcp attestations.

agentgateway can hand every MCP method to an external policy server over
gRPC before forwarding it (the `ExtMcp` service in `proto/ext_mcp.proto`).
This processor answers those calls from a table of passmcp attestations —
one in-toto statement per backend — verified offline at startup against a
policy. A backend whose statement is missing, invalid, about a different
endpoint, below the score floor, or failing a check in a disqualifying
category is denied with a reason that names it; everything else passes.

It is an example integration, small on purpose: a few hundred lines around
the [`attestation`](../../attestation) package, which is the same code any
Go consumer embeds to verify a statement. Nothing here re-implements
verification.

## What it decides

For each backend named in `McpRequest.service_names`, in order:

| Condition | Answer |
|---|---|
| Method is in `passthroughMethods` | `Pass`, no table lookup |
| Backend not in `targets` | deny `no attestation configured` (or pass when `requireAttestation: false`) |
| Statement unreadable, fails `Parse`/`Validate`, or does not `Cover` the configured endpoint | deny `attestation unusable: …` (or pass when `requireAttestation: false`) |
| `minScore > 0` and the statement has no score, or a lower one | deny `score N is below the minimum M` |
| Any verdict with `status: fail` whose phase is in `denyFailIn` | deny `failing checks in category X (ids)` |
| Otherwise | `Pass`; scores ride along in the result's `metadata` under `passmcp.<target>` |

Denials are `AuthorizationError` with code `PERMISSION_DENIED`, which the
gateway turns into JSON-RPC error `-32001` carrying the reason.
`CheckResponse` always passes: the attestation is about the server, not
about one response.

A processor with no configuration loaded answers with gRPC `UNAVAILABLE`
rather than a decision, so the gateway's `failureMode` decides what
happens, not this code.

## Running it

Install a release, or run it from a checkout of this
directory:

```sh
go install satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/cmd/agentgateway-extmcp@v0.0.2
agentgateway-extmcp -config example/config.json -listen 127.0.0.1:4400

go run ./cmd/agentgateway-extmcp -config example/config.json -listen 127.0.0.1:4400
```

Or as a container. Every release publishes a multi-arch image with
SLSA build provenance, and this directory builds the same image. It
listens on all interfaces inside the container and reads its
configuration from `/etc/extmcp/config.json`:

```sh
docker run --rm -p 4400:4400 -v "$PWD/example:/etc/extmcp:ro" \
  ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp:0.0.2
gh attestation verify oci://ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp:0.0.2 --owner sebastienrousseau

docker build -t agentgateway-extmcp .   # or build it from this directory
```

Shell completions come from the flag set:

```sh
agentgateway-extmcp -completion bash > /etc/bash_completion.d/agentgateway-extmcp
agentgateway-extmcp -completion zsh > "${fpath[1]}/_agentgateway-extmcp"
agentgateway-extmcp -completion fish > ~/.config/fish/completions/agentgateway-extmcp.fish
```

Flags: `-config` (required), `-listen` (default `127.0.0.1:4400`; use
`0.0.0.0:…` in a container), `-reload-interval` (default off),
`-max-bytes` (default 1 MiB per statement), `-fetch-timeout` (default
10s), `-log-level`, and `-completion` (print a shell completion script). Diagnostics are JSON lines on stderr.

Statements are read once at startup. `SIGHUP` re-reads the configuration
and every statement; `-reload-interval` does the same on a timer. A
reload whose configuration does not parse is logged and the previous
table stays in service.

### Configuration

[`example/config.json`](example/config.json):

```json
{
  "targets": {
    "github": {
      "attestation": "https://attestations.example.com/github-mcp.intoto.json",
      "endpoint": "https://api.githubcopilot.com/mcp/",
      "transport": "http"
    }
  },
  "policy": {
    "minScore": 70,
    "denyFailIn": ["auth", "protocol"],
    "requireAttestation": true
  },
  "passthroughMethods": ["initialize", "ping"]
}
```

- `targets` keys are agentgateway backend names, exactly as they appear
  in the gateway's `mcp.targets[].name`.
- `attestation` is a file path or an `https://` URL. `http://` is
  refused; a statement that could be swapped in transit is not evidence.
- `endpoint` and `transport` describe the server passmcp evaluated. They
  are checked through the statement's subject digest (`Covers`), not by
  string comparison, so a statement about one server cannot be pointed
  at another.
- `denyFailIn` names verdict phases, such as `auth`, `protocol`,
  `tools`; see passmcp's check inventory for the list.

### Pointing agentgateway at it

[`example/agentgateway.yaml`](example/agentgateway.yaml) is a complete
local configuration. The relevant part:

```yaml
mcp:
  policies:
    mcpGuardrails:
      processors:
        - kind: remote
          host: 127.0.0.1:4400
          failureMode: failClosed
          methods:
            tools/call: request
            resources/read: request
            prompts/get: request
```

`methods` is the gateway's allowlist; methods it does not match never
reach the processor. `failClosed` is the setting that makes the gate a
gate.

- **Put `mcpGuardrails` under `mcp.policies` or a route, not on a
  target.** agentgateway rejects it on an `mcp.targets[]` entry with
  ``unknown field `mcpGuardrails` ``, because MCP policies apply to the
  whole target set. Gating is still per backend: the gateway names the
  backends a request touches in `service_names`, and the processor
  decides for each.
- **Leave `tools/list` out of `methods` unless every target is
  attested.** One listing spans every target, so its `service_names`
  carries all of them, and a single unattested backend denies the whole
  listing. The agent then sees no tools at all, from attested backends
  included. Calls to an unattested backend are denied either way.
- **Check `denyFailIn` against what your servers actually fail.**
  `protocol` covers checks such as `protocol.origin`, which a local
  server that accepts any `Origin` fails. Such a server is denied under
  the example policy even when its score clears `minScore`. That is the
  intended reading of the policy, not a processor fault.

## Out of scope

- **Signature verification.** The processor checks that a statement is
  well-formed, self-consistent and about the configured server; it does
  not check who signed it. Verify a DSSE envelope with your existing
  tooling (cosign, in-toto) before placing the statement where this
  processor reads it, or serve it from a location only your pipeline can
  write. The `attestation` package is the payload half of that pipeline
  on purpose.
- **TLS between the gateway and the processor.** The gRPC listener is
  plaintext; put it on loopback or a private network, or terminate TLS
  in front of it and configure `backendTLS` on the processor entry.
- **Response inspection and request mutation.** Both are in the wire
  contract and neither is used.

## Development

```sh
make            # vet, lint, test
make test-race
make generate   # regenerate gen/extmcp from proto/ext_mcp.proto
```

The module is nested: it has its own `go.mod` so the root module stays
free of dependencies. Its `require` names the passmcp-reporting release it
builds against when installed. Inside this checkout, the repository's
`go.work` makes it build against the verifier at the same commit
instead. A `replace` directive would do the same, but `go install`
refuses any module that carries one, and `make integrations` and CI
fail if one appears. A change to the verifier that the processor needs
therefore ships in two releases: the verifier first, then the processor
requiring it. Lint uses the
repository's `.golangci.yml`. Generated bindings are committed; see
[`proto/README.md`](proto/README.md) for their provenance.
