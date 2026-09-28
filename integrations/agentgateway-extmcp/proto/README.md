<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# proto

`ext_mcp.proto` is a verbatim copy of
`crates/protos/proto/ext_mcp.proto` from
[agentgateway](https://github.com/agentgateway/agentgateway) at commit
`38fc45b`, licensed under Apache-2.0 by the agentgateway authors. The
upstream file carries no header of its own; this note is its provenance.
It is the wire contract an `mcpGuardrails` processor speaks, and it is
copied rather than fetched so the module builds from a checkout with no
network access.

The Go bindings in `../gen/extmcp` are generated from it by
`make generate` (buf as the compiler, `protoc-gen-go` and
`protoc-gen-go-grpc` as local plugins) and committed, so a consumer
needs no protobuf toolchain. The only change on the way through is the
`go_package` option, which buf's managed mode points at this module
instead of agentgateway's tree; the proto file itself is not edited.

To pick up a newer upstream revision: replace the file, update the
commit in this note and in the header of the generated files, and run
`make generate`.
