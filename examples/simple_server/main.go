package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alwint3r/mcp2go/mcp/server"
)

func main() {
	transport := server.NewStdioTransport()
	defer transport.Close()

	s := server.NewDefaultServer(transport, server.ProtocolVersion20250326, "1.0.0", "SimpleMCPServer")
	s = server.WithToolsCapability(s, false, false)

	ctx := context.Background()
	go s.Run(ctx)
	err := transport.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error start transport: %v\n", err)
	}
}

// example request
// {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"roots":{"listChanged":true}}}}

// example response
// {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}},"serverInfo":{"name":"SimpleMCPServer","version":"1.0.0"}}}
