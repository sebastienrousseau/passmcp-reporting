<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Roadmap

What passmcp-reporting intends to do over the next twelve months,
October 2026 to September 2027, and what it will not do.

**This is the maintainer's current intent, not a commitment.** The
project has one maintainer ([GOVERNANCE.md](GOVERNANCE.md)) and no
funded schedule, so nothing below carries a date. Every line is drawn
from a document in this repository or in passmcp's, linked beside it; a
plan that is not written down there is not on this page. The page
changes like any document here: a pull request, with the same one-week
notice as a substantial change ([GOVERNANCE.md](GOVERNANCE.md)).

## What passmcp-reporting intends to do

### Release with passmcp, in lockstep

- Carry passmcp's version and release with every passmcp release,
  tagging first because passmcp imports this module; a release with
  nothing in it is the version rule working
  ([CHANGELOG.md](CHANGELOG.md), passmcp's
  [ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).
- Keep the patch digit moving for everything until 1.0, and treat a
  change to what `Validate` accepts, in either direction, as breaking:
  a tracking issue open for a week, and a CHANGELOG entry that says so
  ([Stability guarantees](README.md#stability-guarantees),
  [CONTRIBUTING.md](CONTRIBUTING.md)).

### Keep the format embeddable

- Keep the root module on the standard library alone, so a consumer
  adding a verifier adds one line to its `go.sum`
  ([ADR 0001](docs/adr/0001-verifier-in-its-own-module.md),
  [AGENTS.md](AGENTS.md)).
- Keep the format under Apache-2.0 so that anyone can implement or embed
  it without passmcp's engine (passmcp's
  [ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md)).
- Keep the published schemas generated from the types, so a consumer
  validating by schema and one validating with this package agree
  ([DEVELOPMENT.md](DEVELOPMENT.md#generated-artefacts)).

### Get the attestation used where admission is decided

- Keep the agentgateway processor working against the current
  agentgateway `ExtMcp` contract, as the reference consumer of the
  verifier ([integrations/agentgateway-extmcp](integrations/agentgateway-extmcp/README.md)).
  Gateway integrations are what the format was relicensed for (passmcp's
  [ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md)).

### Move what passmcp still holds, when there is a reason

- **The rubric**, when passmcp's scorer reads it as data rather than
  generating it from code; until then a copy here would drift
  ([ADR 0001](docs/adr/0001-verifier-in-its-own-module.md)).
- **The report schema and the renderers**, when a consumer outside
  passmcp needs them; extracting them before would design an API for a
  user who does not exist ([ADR 0001](docs/adr/0001-verifier-in-its-own-module.md),
  passmcp's [ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).

Neither has a trigger yet, and naming one is not a promise to act.

### Keep the project's assurance current

- Re-review the [security model](docs/security-model.md) when a trust
  boundary changes, after a reported vulnerability, or by 2027-03-31,
  and resolve its open findings.
- Meet the OpenSSF Best Practices criteria that can be met from the
  repository ([project 15109](https://www.bestpractices.dev/projects/15109)).

## What passmcp-reporting will not do

Each of these is a recorded decision, not an omission.

| passmcp-reporting will not | Why | Record |
|---|---|---|
| Add a dependency to the root module | A consumer adding a verifier must not inherit a dependency graph | [ADR 0001](docs/adr/0001-verifier-in-its-own-module.md), [AGENTS.md](AGENTS.md) |
| Check signatures | Who wrote a statement is the envelope's job (DSSE, cosign, `gh attestation`); this package checks what it says | [SECURITY.md](SECURITY.md), [docs/verify.md](docs/verify.md) |
| Produce statements | Only passmcp writes one; `SubjectFor` exists so a producer digests a target the way a verifier recomputes it | [When not to use](README.md#when-not-to-use-passmcp-reporting) |
| Score, or decide what a score means | The rubric and the verdicts are passmcp's; a consumer's policy decides | [GOVERNANCE.md](GOVERNANCE.md), [ADR 0001](docs/adr/0001-verifier-in-its-own-module.md) |
| Fetch, open a file or follow a reference on a statement's say-so | Everything in a statement is untrusted | [AGENTS.md](AGENTS.md), [security model](docs/security-model.md) |
| Inspect responses or mutate requests in the processor | The attestation is about the server, not one response | [processor README](integrations/agentgateway-extmcp/README.md#out-of-scope) |

[ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md)
names what would change the format's direction: an evaluation predicate
standardised by the MCP project itself, which this format would then
align with. [ADR 0001](docs/adr/0001-verifier-in-its-own-module.md)
names what would move the rubric here sooner: a second producer of
statements.

## Suggesting a change

Open an [issue](https://github.com/sebastienrousseau/passmcp-reporting/issues)
that says what you need and why. A request that runs into one of the
decisions above is still worth filing: each record says what would make
it wrong, and evidence of that is how a decision changes.
