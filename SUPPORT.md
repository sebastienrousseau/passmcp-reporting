<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Support

Where to take a question, in the order most likely to get you an answer.

## Documentation

| You want | Read |
|---|---|
| What a statement carries and why | [docs/format.md](docs/format.md) |
| Verifying one, in Go or without it | [docs/verify.md](docs/verify.md) |
| Package reference | <https://pkg.go.dev/satellion.com/passmcp-reporting> |
| Producing a statement | [passmcp's manual](https://satellion.com/passmcp/docs/) — `passmcp attest` and `--output attestation` |
| Contributing and toolchain | [DEVELOPMENT.md](DEVELOPMENT.md) |
| Why the format is licensed this way | passmcp's [ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md) |

## Questions and discussion

[GitHub Issues](https://github.com/sebastienrousseau/passmcp-reporting/issues)
for "how do I", "should the format carry X", and anything open-ended:
open one with the Question template. The repository has no Discussions. A
question about a verdict itself — why passmcp judged a server the way it
did — belongs in [passmcp's issues](https://github.com/sebastienrousseau/passmcp/issues),
because the verifier only reads what passmcp wrote.

## Bugs and feature requests

[GitHub Issues](https://github.com/sebastienrousseau/passmcp-reporting/issues),
using the templates. A good report includes the statement (redact the
endpoint if you must), the version of this module and of the passmcp that
wrote it, and what `Validate` said.

## Security

**Do not open a public issue for a vulnerability.** Follow the private
reporting process in [SECURITY.md](SECURITY.md).

## Response expectations

passmcp-reporting is maintained by one person alongside other work. Issues
are read within a week; security reports are prioritised per the SLA in
SECURITY.md. There is no commercial support offering.
