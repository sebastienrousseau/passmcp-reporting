# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The README demo: examples/verify checking the two committed sample
# attestations offline, an MCP evaluation and an A2A one. Rendered by
# `make demo`, which compiles the example first so `go run` starts at once;
# nothing here reads the network.
Output .github/demo.gif
Require go
Set Shell "bash"
Set FontSize 13
Set Width 1280
Set Height 200
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "PS1='$ '; clear" Enter
Show

Type "go run ./examples/verify attestation/testdata/statement.json" Sleep 400ms Enter
Sleep 3s
Type "go run ./examples/verify a2a/testdata/statement.json" Sleep 400ms Enter
Sleep 4s
