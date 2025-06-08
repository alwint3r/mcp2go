package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

const (
	ProtocolVersion20241105 = "2024-11-05"
	ProtocolVersion20250326 = "2025-03-26"
)

type CapabilityProperties struct {
	ListChanged *bool `json:"listChanged,omitempty"`
	Subscribe   *bool `json:"subscribe,omitempty"`
}

type Capabilities struct {
	Logging   *CapabilityProperties `json:"logging,omitempty"`
	Tools     *CapabilityProperties `json:"tools,omitempty"`
	Prompts   *CapabilityProperties `json:"prompts,omitempty"`
	Resources *CapabilityProperties `json:"resources,omitempty"`
}

type RequestError struct {
	Err         error
	ForResponse messages.ErrorResponse
}

type Server interface {
}

type RequestHandler func(context.Context, messages.Request) (*messages.JsonRPCResult, *RequestError)
type RequestHandlersMap map[string]RequestHandler

type CancellableRequestMap map[interface{}]context.CancelFunc

type DefaultServer struct {
	Name                string
	Version             string
	Capabilities        Capabilities
	ProtocolVersion     string
	Transport           Transport
	requestHandlers     RequestHandlersMap
	cancellableRequests CancellableRequestMap
	cancelMutex         sync.RWMutex // Protects access to cancellableRequests
	toolManager         ToolManager
	logger              *Logger
	config              ServerConfig
}

type ctxRequestIdKey struct{}

func (s *DefaultServer) handleInitializeRequest(ctx context.Context, request messages.Request) (*messages.JsonRPCResult, *RequestError) {
	result := messages.JsonRPCResult{
		"capabilities":    s.Capabilities,
		"protocolVersion": s.ProtocolVersion,
		"serverInfo": map[string]string{
			"name":    s.Name,
			"version": s.Version,
		},
	}

	return &result, nil
}

func (s *DefaultServer) handleRequest(ctx context.Context, request messages.Request) *messages.JsonRPCMessage {
	ctxWithValue := context.WithValue(ctx, ctxRequestIdKey{}, request.ID)
	message := messages.NewJsonRPCMessage()
	message.ID = request.ID

	handler, err := s.findRequestHandler(&request)
	if err != nil {
		if err.Error() == "method not found" {
			message.Error = &messages.ErrorResponse{
				Code:    messages.JsonRPCErrorMethodNotFound,
				Message: "Method not found",
			}
		} else {
			message.Error = &messages.ErrorResponse{
				Code:    messages.JsonRPCErrorInternalError,
				Message: fmt.Sprintf("Internal error: %v", err),
			}
		}
		return message
	}

	cancellableContext, cancel := context.WithCancel(ctxWithValue)
	defer cancel()

	s.cancelMutex.Lock()
	s.cancellableRequests[request.ID] = cancel
	s.cancelMutex.Unlock()

	handlerResult, requestErr := handler(cancellableContext, request)

	s.cancelMutex.Lock()
	delete(s.cancellableRequests, request.ID)
	s.cancelMutex.Unlock()

	if requestErr != nil {
		message.Error = &requestErr.ForResponse
		return message
	}

	message.Result = handlerResult
	return message
}

func (s *DefaultServer) handleNotification(message *messages.JsonRPCMessage) {
	if *message.Method == "notifications/cancelled" {
		if message.Params == nil {
			s.logger.Warn("Received cancellations notification with empty params")
			return
		}
		params := *message.Params
		if requestID, ok := params["requestId"]; ok {
			s.logger.Info("Cancellation request received for ID: %v", requestID)
			if cancelled := s.CancelRequest(requestID); cancelled {
				s.logger.Info("Successfully cancelled request ID: %v", requestID)
			} else {
				s.logger.Warn("Could not cancel request ID: %v (not found)", requestID)
			}
		}
	}
}

func (s *DefaultServer) findRequestHandler(request *messages.Request) (RequestHandler, error) {
	method := request.Method
	handler, exist := s.requestHandlers[method]
	if !exist {
		return nil, errors.New("method not found")
	}

	return handler, nil
}

// CancelRequest cancels a request that is in progress, thread-safe
func (s *DefaultServer) CancelRequest(id interface{}) bool {
	s.cancelMutex.Lock()
	defer s.cancelMutex.Unlock()

	cancel, exists := s.cancellableRequests[id]
	if exists {
		cancel()
		delete(s.cancellableRequests, id)
		return true
	}
	return false
}

// CancelAllRequests cancels all in-progress requests, thread-safe
func (s *DefaultServer) CancelAllRequests() {
	s.cancelMutex.Lock()
	defer s.cancelMutex.Unlock()

	for id, cancel := range s.cancellableRequests {
		cancel()
		delete(s.cancellableRequests, id)
	}
}

func (s *DefaultServer) Run(ctx context.Context) error {
	s.logger.Info("Server started")
	defer s.logger.Info("Server stopping")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, server stopping: %v", ctx.Err())
			return fmt.Errorf("server stopped: %w", ctx.Err())
		case msg, ok := <-s.Transport.Read():
			if !ok {
				s.logger.Error("Transport channel closed unexpectedly")
				return fmt.Errorf("transport channel closed unexpectedly")
			}

			if msg.JsonRPC != "2.0" {
				s.logger.Warn("Received message with invalid JSON-RPC version: %s", msg.JsonRPC)
				if msg.ID != nil {
					errResponse := messages.NewJsonRPCMessage()
					errResponse.ID = msg.ID
					errResponse.Error = &messages.ErrorResponse{
						Code:    messages.JsonRPCErrorInvalidRequest,
						Message: "Invalid JSON-RPC protocol version",
					}
					withTimeoutCtx, cancel := context.WithTimeout(ctx, s.config.OutgoingMessageTimeoutSeconds*time.Second)
					defer cancel()
					if err := s.Transport.Write(*errResponse, withTimeoutCtx); err != nil {
						s.logger.Error("Failed to write error response: %v", err)
					}
				}
				continue
			}

			if msg.IsRequest() {
				request := messages.NewRequestFromJsonRPCMessage(msg)
				s.logger.Debug("Handling request: %s (ID: %v)", request.Method, request.ID)
				go func(request messages.Request, ctx context.Context) {
					withTimeoutCtx, cancel := context.WithTimeout(ctx, s.config.OutgoingMessageTimeoutSeconds*time.Second)
					defer cancel()
					response := s.handleRequest(ctx, request)
					if err := s.Transport.Write(*response, withTimeoutCtx); err != nil {
						s.logger.Error("Failed to write response: %v", err)
					}
				}(request, ctx)

			} else if msg.IsNotification() {
				s.logger.Debug("Received notification: %s", *msg.Method)
				go s.handleNotification(&msg)
			} else if msg.IsResponse() {
				s.logger.Debug("Received response message with ID: %v", msg.ID)
			} else {
				s.logger.Warn("Received invalid message type")
				if msg.ID != nil {
					errResponse := messages.NewJsonRPCMessage()
					errResponse.ID = msg.ID
					errResponse.Error = &messages.ErrorResponse{
						Code:    messages.JsonRPCErrorInvalidRequest,
						Message: "Invalid message type",
					}
					withTimeoutCtx, cancel := context.WithTimeout(ctx, s.config.OutgoingMessageTimeoutSeconds*time.Second)
					defer cancel()
					if err := s.Transport.Write(*errResponse, withTimeoutCtx); err != nil {
						s.logger.Error("Failed to write error response: %v", err)
					}
				}
			}
		}
	}
}

func NewDefaultServer(transport Transport, protocolVersion string, version string, name string) *DefaultServer {
	return NewDefaultServerWithConfig(transport, protocolVersion, version, name, NewDefaultConfig())
}

func NewDefaultServerWithConfig(transport Transport, protocolVersion string, version string, name string, config ServerConfig) *DefaultServer {
	logger := NewLogger(name)
	logger.MinLevel = config.LogLevel
	logger.ShowTime = config.ShowTimestamps

	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v, using stderr\n", err)
		} else {
			logger.ErrOut = logFile
			logger.Out = logFile
		}
	} else {
		logger.ErrOut = os.Stderr
		logger.Out = os.Stderr
	}

	server := &DefaultServer{
		Version:             version,
		Name:                name,
		Transport:           transport,
		ProtocolVersion:     protocolVersion,
		requestHandlers:     make(RequestHandlersMap),
		cancellableRequests: make(CancellableRequestMap),
		Capabilities:        Capabilities{},
		logger:              logger,
		config:              config,
	}

	server.requestHandlers["initialize"] = server.handleInitializeRequest

	return server
}

func WithLoggingCapability(server *DefaultServer) *DefaultServer {
	server.Capabilities.Logging = &CapabilityProperties{}
	return server
}

func WithToolsCapability(server *DefaultServer, listChanged, subscribe bool) *DefaultServer {
	server.Capabilities.Tools = &CapabilityProperties{}
	if listChanged {
		server.Capabilities.Tools.ListChanged = &listChanged
	}

	if subscribe {
		server.Capabilities.Tools.Subscribe = &subscribe
	}

	return server
}

func WithPromptsCapability(server *DefaultServer, listChanged, subscribe bool) *DefaultServer {
	server.Capabilities.Prompts = &CapabilityProperties{}
	if listChanged {
		server.Capabilities.Prompts.ListChanged = &listChanged
	}

	if subscribe {
		server.Capabilities.Prompts.Subscribe = &subscribe
	}

	return server
}

func WithResourcesCapability(server *DefaultServer, listChanged, subscribe bool) *DefaultServer {
	server.Capabilities.Resources = &CapabilityProperties{}
	if listChanged {
		server.Capabilities.Resources.ListChanged = &listChanged
	}

	if subscribe {
		server.Capabilities.Resources.Subscribe = &subscribe
	}

	return server
}

func WithToolManager(server *DefaultServer, toolManager *ToolManager) *DefaultServer {
	server.toolManager = *toolManager
	server.requestHandlers["tools/list"] = func(ctx context.Context, r messages.Request) (*messages.JsonRPCResult, *RequestError) {
		tools := server.toolManager.ListAllTools()
		response := &messages.JsonRPCResult{
			"tools": tools,
		}

		return response, nil
	}

	server.requestHandlers["tools/call"] = func(ctx context.Context, r messages.Request) (*messages.JsonRPCResult, *RequestError) {
		params := *r.Params
		toolName, ok := params["name"].(string)
		if !ok {
			return nil, &RequestError{
				Err: nil,
				ForResponse: messages.ErrorResponse{
					Code:    messages.JsonRPCErrorInvalidParams,
					Message: "invalid tool name",
				},
			}
		}
		arguments, ok := params["arguments"].(map[string]interface{})
		if !ok {
			return nil, &RequestError{
				Err: nil,
				ForResponse: messages.ErrorResponse{
					Code:    messages.JsonRPCErrorInvalidParams,
					Message: "invalid tool arguments",
				},
			}
		}

		toolCallResult := server.toolManager.CallTool(ctx, toolName, arguments)
		response := messages.JsonRPCResult{
			"content": toolCallResult.Content,
			"isError": toolCallResult.IsError,
		}
		return &response, nil
	}

	return server
}
