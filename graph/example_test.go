// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package graph_test

import (
	"fmt"

	"satellion.com/passmcp-reporting/graph"
)

// A client configuration names a server; the same server upserted again from
// its attestation merges into one node, and the agent reaches it by a Uses
// edge that records where it was declared, not the declaration's secrets.
func Example() {
	g := graph.New()
	srv := graph.ServerID("http", "https://mcp.example.com/mcp")
	agent := graph.AgentID("cursor", "/home/u/.cursor/mcp.json", "default")

	g.Upsert(graph.Node{ID: agent, Kind: graph.KindAgent, Label: "cursor", Agent: &graph.AgentProps{Client: "cursor"}})
	g.Upsert(graph.Node{ID: srv, Kind: graph.KindServer, Label: "https://mcp.example.com/mcp",
		Server: &graph.ServerProps{Transport: "http", Endpoint: "https://mcp.example.com/mcp"}})
	g.Upsert(graph.Node{ID: srv, Kind: graph.KindServer, Server: &graph.ServerProps{Grade: "B"}})
	g.Link(graph.Edge{From: agent, To: srv, Kind: graph.Uses,
		Ref: &graph.ConfigRef{File: "/home/u/.cursor/mcp.json", Key: "mcpServers.example"}})

	n, _ := g.Node(srv)
	fmt.Println(len(g.Nodes), "nodes;", n.Server.Endpoint, "grade", n.Server.Grade)
	fmt.Println(g.Validate() == nil)
	// Output:
	// 2 nodes; https://mcp.example.com/mcp grade B
	// true
}
