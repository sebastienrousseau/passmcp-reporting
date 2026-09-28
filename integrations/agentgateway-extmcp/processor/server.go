// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"satellion.com/passmcp-reporting/integrations/agentgateway-extmcp/gen/extmcp"
)

// Server implements agentgateway's ExtMcp service over a Store.
type Server struct {
	extmcp.UnimplementedExtMcpServer
	store *Store
	log   *slog.Logger
}

// NewServer returns a service that decides from store's current snapshot.
func NewServer(store *Store, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{store: store, log: log}
}

// CheckRequest gates one MCP call. Every backend the request names must
// pass the policy; the first that does not denies the whole call with a
// reason naming it. A request that names no backend has nothing to gate
// and passes.
//
// A store with no snapshot answers with a gRPC error rather than a
// decision, so the gateway's failureMode — not this code — chooses
// between failing open and failing closed.
func (s *Server) CheckRequest(_ context.Context, req *extmcp.McpRequest) (*extmcp.McpRequestResult, error) {
	snap := s.store.Snapshot()
	if snap == nil {
		return nil, status.Error(codes.Unavailable, ErrNotLoaded.Error())
	}
	method := req.GetMethod()
	targets := req.GetServiceNames()
	if snap.Passthrough(method) {
		s.log.Debug("pass-through", "method", method, "targets", targets)
		return pass(nil), nil
	}
	meta := make(map[string]any, len(targets))
	for _, name := range targets {
		d := snap.Decide(name)
		if !d.Allow {
			s.log.Info("denied", "method", method, "target", name, "reason", d.Reason)
			return &extmcp.McpRequestResult{
				Result: &extmcp.McpRequestResult_Error{Error: &extmcp.AuthorizationError{
					Code:   extmcp.AuthorizationError_PERMISSION_DENIED,
					Reason: d.Reason,
				}},
			}, nil
		}
		s.log.Debug("allowed", "method", method, "target", name, "reason", d.Reason)
		if d.Statement != nil && d.Statement.Predicate.Score != nil {
			meta[name] = map[string]any{
				"score": d.Statement.Predicate.Score.Total,
				"grade": d.Statement.Predicate.Score.Grade,
			}
		}
	}
	return pass(meta), nil
}

// CheckResponse passes every response. The attestation is about the
// server, not about what it said this time; response inspection is
// another processor's job.
func (s *Server) CheckResponse(_ context.Context, _ *extmcp.McpResponse) (*extmcp.McpResponseResult, error) {
	return &extmcp.McpResponseResult{Result: &extmcp.McpResponseResult_Pass{Pass: &extmcp.Pass{}}}, nil
}

// pass builds a Pass result. When scores are known they ride along under
// the "passmcp" key of the result's metadata bag, one entry per target, so
// a later filter can read them from CEL.
func pass(scores map[string]any) *extmcp.McpRequestResult {
	res := &extmcp.McpRequestResult{Result: &extmcp.McpRequestResult_Pass{Pass: &extmcp.Pass{}}}
	if len(scores) == 0 {
		return res
	}
	// Only floats and strings go in, so this cannot fail; the error is
	// checked anyway because a silent nil would be a wrong answer.
	if md, err := structpb.NewStruct(map[string]any{"passmcp": scores}); err == nil {
		res.Metadata = md
	}
	return res
}
