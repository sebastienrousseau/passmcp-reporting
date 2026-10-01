# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The agentgateway ExtMcp processor as a container, built from this
# directory alone. There is no go.work in the context, so the build sees
# the module exactly as `go install` does: against the passmcp-reporting
# release go.mod requires.
#
#   docker build -t agentgateway-extmcp integrations/agentgateway-extmcp
#   docker run --rm -p 4400:4400 -v "$PWD/config.json:/etc/extmcp/config.json:ro" agentgateway-extmcp
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
# In a container the listener must not be loopback-only.
ENTRYPOINT ["/usr/local/bin/agentgateway-extmcp", "-listen", "0.0.0.0:4400"]
CMD ["-config", "/etc/extmcp/config.json"]
