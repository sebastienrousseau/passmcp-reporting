// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package graph is the data model of the passmcp security graph: which agents
// reach which MCP servers and tools, under which identities, with each
// server's attested verdict attached.
//
// It is the format passmcp-graph stores and queries and passmcp's discovery
// writes into, published here under Apache-2.0 beside the attestation format
// so that gateways and GRC tools can read a graph without taking on either
// program. The JSON Schema in spec/graph is generated from these types.
//
// Three properties are deliberate:
//
//   - Node identifiers are derived from what a node is, never assigned. A
//     server's ID is the digest of the same target descriptor its attestation
//     subject covers, so ingesting the same evidence twice yields the same
//     nodes and edges, and a graph and an attestation name a server the same
//     way.
//   - There is no field for a secret. Tokens, API keys and environment values
//     a client configuration holds have nowhere to go: an agent keeps only a
//     reference to the file and key it came from. A store cannot leak what it
//     has no field for.
//   - Every property is typed. A free-form property bag would be a place
//     where a secret, or anything else, could be smuggled in unreviewed.
//
// A graph is stored as one JSON document, written deterministically: the
// same graph always encodes to the same bytes.
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"satellion.com/passmcp-reporting/attestation"
)

// Version identifies the graph format. It is versioned in the URL, as the
// attestation predicate is, so a reader that does not recognise it must not
// guess at the contents.
const Version = "https://satellion.com/graph/v1"

// Kind is what a node represents.
type Kind string

// The node kinds.
const (
	// KindAgent is an agent or an MCP client configured to use servers.
	KindAgent Kind = "agent"
	// KindServer is an MCP server.
	KindServer Kind = "server"
	// KindTool is a tool a server exposes.
	KindTool Kind = "tool"
	// KindIdentity is an OAuth client identity authorised on a server.
	KindIdentity Kind = "identity"
	// KindSource is where a server was discovered: a configuration file, a
	// gateway, a registry namespace or a target list.
	KindSource Kind = "source"
	// KindAttestation is a signed evaluation of a server.
	KindAttestation Kind = "attestation"
)

// EdgeKind is how two nodes relate.
type EdgeKind string

// The edge kinds, each with the node kinds it joins.
const (
	// Uses joins an agent to a server it is configured to use.
	Uses EdgeKind = "uses"
	// Exposes joins a server to one of its tools.
	Exposes EdgeKind = "exposes"
	// Authorizes joins an identity to a server it holds scopes on.
	Authorizes EdgeKind = "authorizes"
	// DiscoveredBy joins a server to the source it was discovered from.
	DiscoveredBy EdgeKind = "discovered_by"
	// AttestedBy joins a server to an attestation about it.
	AttestedBy EdgeKind = "attested_by"
)

// endpoints is the node kind each edge kind runs from and to.
var endpoints = map[EdgeKind][2]Kind{
	Uses:         {KindAgent, KindServer},
	Exposes:      {KindServer, KindTool},
	Authorizes:   {KindIdentity, KindServer},
	DiscoveredBy: {KindServer, KindSource},
	AttestedBy:   {KindServer, KindAttestation},
}

// Graph is a whole graph: its nodes and edges, in canonical order once
// sorted by Canonical or written by Marshal.
type Graph struct {
	// Version is always the Version constant.
	Version string `json:"version"`
	Nodes   []Node `json:"nodes"`
	Edges   []Edge `json:"edges"`

	index map[string]int
	edges map[string]int
}

// Node is one entity. Exactly one of the property structs is set, the one
// matching Kind.
type Node struct {
	// ID is derived from what the node is; see the ID functions.
	ID   string `json:"id"`
	Kind Kind   `json:"kind"`
	// Label is a human-readable name: an endpoint, a tool name, a client.
	Label string `json:"label"`

	Agent       *AgentProps       `json:"agent,omitempty"`
	Server      *ServerProps      `json:"server,omitempty"`
	Tool        *ToolProps        `json:"tool,omitempty"`
	Identity    *IdentityProps    `json:"identity,omitempty"`
	Source      *SourceProps      `json:"source,omitempty"`
	Attestation *AttestationProps `json:"attestation,omitempty"`
}

// AgentProps describes an agent or client. It holds no secret: only where
// its configuration came from.
type AgentProps struct {
	// Client is the client product, such as "claude-desktop" or "cursor".
	Client string `json:"client"`
	// ConfigFile is the file the agent's configuration was read from.
	ConfigFile string `json:"configFile,omitempty"`
}

// ServerProps describes an MCP server and its latest attested verdict.
type ServerProps struct {
	// Transport is "http" or "stdio".
	Transport string `json:"transport"`
	// Endpoint is the URL, or the command line for a child process.
	Endpoint string `json:"endpoint"`
	// Name and Version are what the server said it was.
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	// Auth is how the server admits clients, as passmcp recorded it: "none",
	// "oauth", "bearer" or "unknown".
	Auth string `json:"auth,omitempty"`
	// Score and Grade are from the latest attestation, when there is one.
	Score *float64 `json:"score,omitempty"`
	Grade string   `json:"grade,omitempty"`
	// Attestation is the digest of the latest attestation's statement.
	Attestation string `json:"attestation,omitempty"`
	// Failing lists the checks that failed in the latest attestation, with
	// their severity, so risk can be explained without re-reading it.
	Failing []Finding `json:"failing,omitempty"`
}

// Finding is one failing check carried onto a server node.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity,omitempty"`
}

// ToolProps describes a tool and the annotations its server declared.
type ToolProps struct {
	Name string `json:"name"`
	// ReadOnlyHint and DestructiveHint are the declared annotations; nil
	// means the server did not declare one, which the MCP specification
	// reads as not read-only and possibly destructive.
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
}

// IdentityProps describes an OAuth client identity. A client ID is public;
// no secret is recorded.
type IdentityProps struct {
	Issuer   string `json:"issuer"`
	ClientID string `json:"clientId,omitempty"`
}

// SourceProps describes where servers were discovered.
type SourceProps struct {
	// Type is "config", "gateway", "registry" or "targets".
	Type string `json:"type"`
	// Ref is the file, URL or registry namespace the source names.
	Ref string `json:"ref"`
}

// AttestationProps describes an attestation.
type AttestationProps struct {
	// Digest is the SHA-256 of the statement's bytes.
	Digest        string `json:"digest"`
	PredicateType string `json:"predicateType"`
	// RanAt is when the evaluated run started, in RFC 3339.
	RanAt string `json:"ranAt,omitempty"`
	// File is where the statement was read from, when it was a file.
	File string `json:"file,omitempty"`
}

// Edge relates two nodes.
type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Kind EdgeKind `json:"kind"`
	// Scopes are the scopes an Authorizes edge grants.
	Scopes []string `json:"scopes,omitempty"`
	// Ref is where a Uses edge was declared: the configuration file and the
	// key under which the server appears. The key's value is never kept.
	Ref *ConfigRef `json:"ref,omitempty"`
}

// ConfigRef points at a place in a configuration file without copying what
// is there.
type ConfigRef struct {
	File string `json:"file"`
	Key  string `json:"key"`
}

// id derives a stable identifier from the kind and the parts that make a
// node what it is.
func id(k Kind, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return string(k) + ":" + hex.EncodeToString(sum[:])
}

// ServerID is the ID of the server at transport and endpoint. It uses the
// same descriptor the attestation subject digest covers, so the ID's hex part
// equals that digest.
func ServerID(transport, endpoint string) string {
	sub := attestation.SubjectFor(attestation.Target{Transport: transport, Endpoint: endpoint})
	return string(KindServer) + ":" + sub.Digest["sha256"]
}

// ToolID is the ID of the tool called name on the server serverID.
func ToolID(serverID, name string) string { return id(KindTool, serverID, name) }

// AgentID is the ID of the agent called name in the client's configuration
// file.
func AgentID(client, configFile, name string) string {
	return id(KindAgent, client, configFile, name)
}

// IdentityID is the ID of the client identity clientID at issuer.
func IdentityID(issuer, clientID string) string { return id(KindIdentity, issuer, clientID) }

// SourceID is the ID of a discovery source.
func SourceID(sourceType, ref string) string { return id(KindSource, sourceType, ref) }

// AttestationID is the ID of the attestation whose statement has digest.
func AttestationID(digest string) string { return string(KindAttestation) + ":" + digest }

// Digest returns the SHA-256 of statement bytes, as AttestationProps.Digest
// and ServerProps.Attestation record it.
func Digest(statement []byte) string {
	sum := sha256.Sum256(statement)
	return hex.EncodeToString(sum[:])
}
