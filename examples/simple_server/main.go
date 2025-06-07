package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alwint3r/mcp2go/mcp/server"
)

func main() {
	transport := server.NewStdioTransport()
	fmt.Fprintln(os.Stderr, "Initializing server!")
	defer transport.Close()

	toolManager := server.NewToolManager()
	toolManager.AddTool(server.Tool{
		Name:        "get_weather",
		Description: "Get the current weather of a location",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "City name or state name",
				},
			},
			"required": []string{"location"},
		},
	}, func(ctx context.Context, name string, arguments map[string]interface{}) server.ToolResult {
		responseText := fmt.Sprintf("Current weather in %v is 27 degree Celsius", arguments["location"])
		return server.ToolResult{
			Content: []server.ToolCallContent{
				{
					Type: "text",
					Text: &responseText,
				},
			},
			IsError: false,
		}
	})

	// Claude app has yet to support the 2025-03-26 protocol version
	s := server.NewDefaultServer(transport, server.ProtocolVersion20241105, "1.0.0", "SimpleMCPServer")
	s = server.WithToolsCapability(s, false, false)
	s = server.WithToolManager(s, &toolManager)

	ctx := context.Background()
	fmt.Fprintln(os.Stderr, "Launching server loop thread")
	go func() {
		err := s.Run(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v", err)
		}

		fmt.Fprintln(os.Stderr, "Server exiting")
	}()

	fmt.Fprintln(os.Stderr, "Starting server transport")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := transport.Start(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error start transport: %v\n", err)
			sigChan <- os.Interrupt // Signal main goroutine to exit on error
		}

		fmt.Fprintln(os.Stderr, "Transport exiting")
	}()

	<-sigChan

	fmt.Fprintln(os.Stderr, "Exiting server")
}

// example request
// {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"roots":{"listChanged":true}}}}

// example response
// {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}},"serverInfo":{"name":"SimpleMCPServer","version":"1.0.0"}}}
