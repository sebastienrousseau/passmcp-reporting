<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Benchmarks

What an admission decision costs: parsing a statement and validating it,
subject digest included. There is one benchmark, `BenchmarkValidate` in
[`attestation/attestation_test.go`](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/attestation/attestation_test.go),
run at two sizes — the two-verdict fixture, and a statement carrying the
verdict of every check in a full passmcp run.

```sh
go test ./attestation -run xxx -bench Validate -benchmem -count=3
```

## Results

| Scenario | Input | Time per operation | Allocated | Allocations |
| :--- | ---: | ---: | ---: | ---: |
| Parse + Validate, 2 verdicts | 1.3 KB | 3.8 µs | 1.4 KB | 17 |
| Parse + Validate, 120 verdicts | 28 KB | 66 µs | 38.6 KB | 390 |

Environment: Apple A18 Pro (6 cores), macOS, Go 1.26.8 (the version
`go.mod` pins), measured 2026-09-23 over three runs; the spread across
runs was under 2%.

## Reading them

- The cost is linear in the number of verdicts and dominated by
  `encoding/json`: about three allocations per verdict, which is the
  verdict struct and its evidence and doc strings.
- The digest is one SHA-256 over a few dozen bytes and does not register.
- A gateway making one decision per new server, or per re-evaluation,
  pays tens of microseconds. There is no reason to cache the result of
  `Validate`; cache the decision if anything.

## What is not measured

Signature verification, which is the envelope's job and dwarfs this; and
fetching the statement, which this package does not do.
