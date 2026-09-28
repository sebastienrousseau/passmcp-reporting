<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Governance

passmcp-reporting is one repository in the passmcp family and is governed the
way passmcp is. This document says what is specific to this repository and
points at passmcp's for the rest.

## Roles

**Maintainer:** Sebastien Rousseau (<sebastian.rousseau@gmail.com>,
GitHub `@sebastienrousseau`), with commit access, responsible for the
format's stability, its releases and its security response.

**Contributor:** anyone who opens an issue or a pull request. Mechanics
are in [CONTRIBUTING.md](CONTRIBUTING.md).

## What is decided here, and what is not

This repository decides **the format**: what a statement carries, what
`Validate` accepts, and the schema. A change to any of those is a change
other people's software sees, so it follows the rule in
[CONTRIBUTING.md](CONTRIBUTING.md): a tracking issue open for at least a
week before it lands, and a record in [docs/adr/](docs/adr/README.md) when
the decision will be questioned later.

It does not decide **the verdicts**: which checks passmcp runs and how it
scores them are passmcp's, recorded in passmcp's ADRs. The rubric stays there
for that reason.

## Version and release

The version is passmcp's. This repository never chooses its own; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md). Releases are signed tags,
cut by the Maintainer.

## Continuity

The single-Maintainer model is a real bus-factor risk, stated rather than
hidden. The succession procedure — hand-off, community fork after six
months of unresponsiveness, and compromise response — is passmcp's, in
[passmcp's GOVERNANCE.md](https://github.com/sebastienrousseau/passmcp/blob/main/GOVERNANCE.md),
and applies to this repository as one of the family. Apache-2.0 lets
anyone fork without further permission.

## Changes to this document

Through the usual pull request process, with the same one-week notice as
a substantial code change.
