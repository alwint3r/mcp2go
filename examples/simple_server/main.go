package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alwint3r/mcp2go/mcp/server"
)

func main() {
	logger := server.NewLogger("Main")
	logger.ErrOut = os.Stderr
	logger.Out = os.Stderr
	logger.Info("Initializing server!")

	transport := server.NewStdioTransport()
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

	config := server.ServerConfig{
		LogLevel:         server.LogDebug,
		ShowTimestamps:   true,
		MaxRequestActive: 10,
		LogFile:          "",
	}

	// Claude app has yet to support the 2025-03-26 protocol version
	s := server.NewDefaultServerWithConfig(transport, server.ProtocolVersion20241105, "1.0.0", "SimpleMCPServer", config)
	s = server.WithToolsCapability(s, false, false)
	s = server.WithToolManager(s, &toolManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrCh := make(chan error, 1)
	transportErrCh := make(chan error, 1)

	logger.Info("Launching server loop thread")
	go func() {
		err := s.Run(ctx)
		if err != nil {
			logger.Error("Server error: %v", err)
			serverErrCh <- err
		}
		logger.Info("Server exiting")
		close(serverErrCh)
	}()

	logger.Info("Starting server transport")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := transport.Start(ctx)
		if err != nil {
			logger.Error("Transport error: %v", err)
			transportErrCh <- err
		}
		logger.Info("Transport exiting")
		close(transportErrCh)
	}()

	// Wait for a signal to exit or an error from any goroutine
	select {
	case <-sigChan:
		logger.Info("Received termination signal")
	case err := <-serverErrCh:
		if err != nil {
			logger.Error("Server terminated due to error: %v", err)
		}
	case err := <-transportErrCh:
		if err != nil {
			logger.Error("Transport terminated due to error: %v", err)
		}
	}

	s.CancelAllRequests()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	cancel()

	logger.Info("Gracefully shutting down...")

	waitCh := make(chan struct{})
	go func() {
		<-serverErrCh
		<-transportErrCh
		close(waitCh)
	}()

	select {
	case <-waitCh:
		logger.Info("All goroutines exited successfully")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timed out, forcing exit")
	}

	logger.Info("Exiting server")
}

// example request
// {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"roots":{"listChanged":true}}}}

// example response
// {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}},"serverInfo":{"name":"SimpleMCPServer","version":"1.0.0"}}}
