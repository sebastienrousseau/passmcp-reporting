# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The agentgateway ExtMcp processor as a container, built from this
# directory alone. There is no go.work in the context, so the build sees
# the module exactly as `go install` does: against the passmcp-reporting
# release go.mod requires.
#
#   docker build -t agentgateway-extmcp integrations/agentgateway-extmcp
#   docker run --rm -p 4400:4400 -v "$PWD/extmcp:/etc/extmcp:ro" agentgateway-extmcp
#
# The mounted directory holds config.json and the key pair, tls.crt and
# tls.key, readable by uid 65532. Without the key pair the processor
# refuses to start. A sidecar sharing the gateway's network namespace
# (a Kubernetes pod) may serve without TLS on loopback instead, by
# replacing the arguments:
#
#   docker run ... agentgateway-extmcp -config /etc/extmcp/config.json \
#     -listen 127.0.0.1:4400 -plaintext
#
# Both bases are pinned by digest: a tag can be moved under a build, and a
# digest cannot.
FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOWORK=off GOFLAGS=-trimpath
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w" -o /out/agentgateway-extmcp ./cmd/agentgateway-extmcp

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /out/agentgateway-extmcp /usr/local/bin/agentgateway-extmcp
LABEL org.opencontainers.image.source="https://github.com/sebastienrousseau/passmcp-reporting" \
      org.opencontainers.image.description="agentgateway ExtMcp processor that gates MCP backends on passmcp attestations" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.title="agentgateway-extmcp" \
      org.opencontainers.image.vendor="Sebastien Rousseau"
USER 65532:65532
EXPOSE 4400
# In a container the listener must not be loopback-only, and off loopback
# it serves TLS only: the default arguments name the key pair. A -listen
# in the arguments overrides the one here, since the last one given wins.
ENTRYPOINT ["/usr/local/bin/agentgateway-extmcp", "-listen", "0.0.0.0:4400"]
CMD ["-config", "/etc/extmcp/config.json", "-tls-cert", "/etc/extmcp/tls.crt", "-tls-key", "/etc/extmcp/tls.key"]
