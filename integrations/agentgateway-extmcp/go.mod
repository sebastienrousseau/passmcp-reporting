module satellion.com/passmcp-reporting/integrations/agentgateway-extmcp

go 1.26.8

require (
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.11
	satellion.com/passmcp-reporting v0.0.1
)

require (
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

// Until satellion.com/passmcp-reporting v0.0.1 is tagged, the verifier comes
// from this checkout. The release removes this line before tagging this
// module, since go install refuses a module with a replace directive.
replace satellion.com/passmcp-reporting => ../..
