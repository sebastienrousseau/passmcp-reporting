<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Verifying a release

A release of passmcp-reporting is two signed git tags and one container
image:

| Artefact | Where | What proves it |
|---|---|---|
| `vX.Y.Z`, the verifier module | git, and the Go module proxy | an SSH signature on the annotated tag, and the Go checksum database |
| `integrations/agentgateway-extmcp/vX.Y.Z`, the processor module | git, and the Go module proxy | an SSH signature on the annotated tag |
| `ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp:X.Y.Z` | GitHub Container Registry | SLSA build provenance attested to the image digest |

Every command on this page was run against v0.0.4 on 2026-09-30, with
git 2.55 and GitHub CLI 2.101, and produced the output shown. They need
git 2.34 or later (the first with SSH signatures), `curl`, `jq`,
`ssh-keygen`, and for the image a GitHub CLI with `gh attestation`.

This page is about the releases. Signing and verifying an attestation
statement, the file passmcp writes about a server, is passmcp's
[signing guide](https://satellion.com/passmcp/docs/signing/); the
verifier in this repository checks the statement's contents, not who
signed it ([Verifying a statement](verify.md)).

## The release tags

Tags are annotated and signed with the maintainer's SSH key. GitHub
publishes the SSH keys an account has registered for signing at
`https://api.github.com/users/<user>/ssh_signing_keys`; that list is
where the public key comes from. (`https://github.com/<user>.keys` is
the account's authentication keys, a different list.)

Turn the list into an `allowed_signers` file for git, then verify:

```sh
git clone https://github.com/sebastienrousseau/passmcp-reporting
cd passmcp-reporting

curl -fsSL https://api.github.com/users/sebastienrousseau/ssh_signing_keys \
  | jq -r '.[] | "sebastian.rousseau@gmail.com namespaces=\"git\" \(.key)"' \
  > ../allowed_signers

git -c gpg.ssh.allowedSignersFile=../allowed_signers verify-tag v0.0.4
git -c gpg.ssh.allowedSignersFile=../allowed_signers verify-tag integrations/agentgateway-extmcp/v0.0.4
```

Each `verify-tag` prints, and exits 0:

```text
Good "git" signature for sebastian.rousseau@gmail.com with ED25519 key SHA256:f6FG+guRNtT3R36oQFS4oWCx1d10nm+BoaIL3Tkh1r4
```

Check the fingerprint as well as the word "Good". The API lists every
key the account may sign with, one per machine, so a signature from any
of them verifies against that file. Every release from v0.0.1 to v0.0.4
was signed by the key above. A release signed by another key of the
account is not necessarily wrong, but it is worth asking about in an
issue before relying on it.

To accept that key and no other, keep only its line:

```sh
curl -fsSL https://api.github.com/users/sebastienrousseau/ssh_signing_keys \
  | jq -r '.[].key' | while read -r key; do
      fp=$(ssh-keygen -lf - <<<"${key}" | cut -d' ' -f2)
      if [ "${fp}" = "SHA256:f6FG+guRNtT3R36oQFS4oWCx1d10nm+BoaIL3Tkh1r4" ]; then
        echo "sebastian.rousseau@gmail.com namespaces=\"git\" ${key}"
      fi
    done > ../allowed_signers
```

A tag that is unsigned, signed by a key not in the file, or changed
after signing fails `verify-tag` with a non-zero exit; with an empty
`allowed_signers` the same tag reports `No principal matched.` and
exits 1. The tag message is the first thing to read after the
signature: it is `passmcp-reporting vX.Y.Z`, or
`agentgateway-extmcp vX.Y.Z` for the processor.

## The Go module

`go get` checks every module it downloads against the Go checksum
database (`sum.golang.org`), which records the first hash anyone saw for
a version and never changes it. A module proxy that served different
bytes for v0.0.4 than it served everyone else fails the build.

```sh
go get satellion.com/passmcp-reporting@v0.0.4
go mod verify     # all modules verified
```

The checksum database proves the bytes are the ones everyone gets, not
that they came from the signed tag. To tie the two, compare the commit
the proxy says it built the version from with the commit the signed tag
names:

```sh
go mod download -json satellion.com/passmcp-reporting@v0.0.4 | jq -r .Origin.Hash
git rev-parse 'v0.0.4^{commit}'
```

For v0.0.4 both print `9151a7b2cbcf5780174e1de81af83c57cc8ca17b`.

## The processor image

The release workflow builds the processor image for linux/amd64 and
linux/arm64, pushes it, and attests SLSA build provenance to its digest
with GitHub's artifact attestations. Verify it with the GitHub CLI:

```sh
gh attestation verify oci://ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp:0.0.4 \
  --repo sebastienrousseau/passmcp-reporting \
  --signer-workflow sebastienrousseau/passmcp-reporting/.github/workflows/release.yml \
  --source-ref refs/tags/v0.0.4
```

It exits 0 only when the provenance was signed by this repository's
release workflow, running for the tag `v0.0.4`, and fails for any other
repository (`--repo sebastienrousseau/passmcp` answers HTTP 404, since
no attestation for that digest exists there). With `--format json`, the
certificate shows what the image was built from:

```sh
gh attestation verify oci://ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp:0.0.4 \
  --repo sebastienrousseau/passmcp-reporting --format json \
  | jq '.[0].verificationResult.signature.certificate | {sourceRepositoryRef, sourceRepositoryDigest, buildSignerURI}'
```

For 0.0.4, `sourceRepositoryDigest` is
`9151a7b2cbcf5780174e1de81af83c57cc8ca17b`, the commit the signed tag
`v0.0.4` names, and the image digest is
`sha256:eba48111e6e21e07631ced5e5883dd8af2dbb76740b5566c78229a96af2c83fa`,
the digest the [release page](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/integrations%2Fagentgateway-extmcp%2Fv0.0.4)
lists. Adding `--source-digest` with the commit from `git rev-parse`
makes the command fail unless the image was built from exactly that
commit.

A tag can be moved to another image; a digest cannot. Having verified
it, run the image by digest:

```sh
docker run --rm -p 4400:4400 -v "$PWD/example:/etc/extmcp:ro" \
  ghcr.io/sebastienrousseau/passmcp-agentgateway-extmcp@sha256:eba48111e6e21e07631ced5e5883dd8af2dbb76740b5566c78229a96af2c83fa
```

That is the v0.0.4 image, which serves plaintext. From 0.0.6 the image
serves TLS only and refuses to start without a key pair: the mounted
directory holds `config.json`, `tls.crt` and `tls.key`, as the
[processor's README](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/integrations/agentgateway-extmcp/README.md#tls)
describes.

## What else is and is not signed

- **Nothing else is the release check.** Commits are signed too, the
  maintainer's with the keys above and the merge commits on `main` with
  GitHub's own key, but the signed tag is the statement that a commit is
  a release.
- **Release pages.** They are composed by the release workflow and carry
  no files, only the image digest above.
- **The manual.** It is published to GitHub Pages from `main` and is not
  signed.
