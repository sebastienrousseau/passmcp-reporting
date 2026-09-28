<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Verifying a statement

Three questions, in order. The first two are this package's; the third is
the envelope's.

## 1. Is it well-formed and intact?

```go
st, err := attestation.Parse(b)   // the in-toto envelope and the predicate type
if err != nil { … }
if err := st.Validate(); err != nil { … }
```

`Validate` checks the statement and predicate types, that there is exactly
one subject named as the target, that the subject's digest equals the
digest recomputed from the predicate's target descriptor, that every
required field is present, that a score carries its rubric version, and
that every verdict's status is one of the five. It reads nothing but the
bytes it was given.

Without Go, the same structural check is the schema:

```sh
# any JSON Schema 2020-12 validator; here, check-jsonschema
check-jsonschema --schemafile spec/attestation/mcp-evaluation-v1.schema.json statement.json
```

The schema cannot recompute the digest. A consumer validating by schema
alone should compute `sha256(transport + "\n" + endpoint)` from the
predicate's target and compare it with `subject[0].digest.sha256` itself;
that is the check that catches a renamed subject.

## 2. Is it about the server in front of me?

```go
if !st.Covers("http", "https://mcp.example.com/mcp") { … }
```

A valid statement about a different server is not evidence about this
one. `Covers` compares the transport and the endpoint, after the same
normalisation passmcp applies when it writes them.

## 3. Who wrote it?

Neither of the above says the bytes came from a passmcp run rather than a
text editor. That is the job of a signature over the statement, and passmcp
leaves the choice of envelope to the consumer because the consumer already
has one:

- **Sigstore.** `cosign sign-blob statement.json` and `cosign verify-blob`
  with the signer's identity; keyless from a workflow, or with a key. passmcp's
  manual has the worked commands.
- **DSSE.** Wrap the statement in a DSSE envelope with `payloadType`
  `application/vnd.in-toto+json`; every in-toto verifier reads one.
- **GitHub artifact attestations.** A statement produced in a workflow
  can be attested with `actions/attest` and checked with
  `gh attestation verify`.

Verify the signature first, then `Validate`, then `Covers`. A statement
that passes all three is a claim by a known party about this server, and
the verdicts inside it are what to gate on.

## Gating

The statement carries every verdict, so a gate can be exactly as specific
as a policy needs:

```go
v, err := st.VerdictFor("auth.unauthenticated_tools")
switch {
case errors.Is(err, attestation.ErrNoSuchCheck): // not checked: decide what that means
case v.Status == "fail":                         // checked, and wrong
}
```

`Compare(before, after)` names what got worse between two statements about
the same server, which is what a re-evaluation policy wants: not "is the
score above N" but "did anything regress since the version we admitted".
