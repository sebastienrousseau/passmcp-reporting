// The tools CI runs that are not the module's own code, pinned. It is a
// module of its own so that neither the root module, which must keep no
// require directive, nor the processor, whose dependency graph ships in
// an image, inherits these. Dependabot watches this directory; bump by
// hand with: cd tools && GOWORK=off go get -tool <path>@<version>.

module satellion.com/passmcp-reporting/tools

go 1.26.8

tool (
	golang.org/x/exp/cmd/gorelease
	golang.org/x/vuln/cmd/govulncheck
)

require (
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/telemetry v0.0.0-20260908163034-4bcc4b2ee518 // indirect
	golang.org/x/tools v0.50.0 // indirect
	golang.org/x/vuln v1.8.0 // indirect
)
