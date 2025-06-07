package server

import (
	"context"
	"errors"
	"fmt"

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
}

type ctxRequestIdKey struct{}

func (s *DefaultServer) handleInitializeRequest(ctx context.Context, request messages.Request) (*messages.JsonRPCResult, *RequestError) {
	result := messages.JsonRPCResult{
		"capabilities": s.Capabilities,
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
	if err != nil && err.Error() == "method not found" {
		message.Error = &messages.ErrorResponse{
			Code:    messages.JsonRPCErrorMethodNotFound,
			Message: "Method not found",
		}

		return message
	}

	cancellableContext, cancel := context.WithCancel(ctxWithValue)
	s.cancellableRequests[request.ID] = cancel
	handlerResult, requestErr := handler(cancellableContext, request)
	if requestErr != nil {
		message.Error = &requestErr.ForResponse
		return message
	}

	cancel()
	delete(s.cancellableRequests, request.ID)

	message.Result = handlerResult
	return message
}

func (s *DefaultServer) findRequestHandler(request *messages.Request) (RequestHandler, error) {
	method := request.Method
	handler, exist := s.requestHandlers[method]
	if !exist {
		return nil, errors.New("method not found")
	}

	return handler, nil
}

func (s *DefaultServer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-s.Transport.Read():
			if !ok {
				return fmt.Errorf("type error: %T", msg)
			}

			if msg.JsonRPC != "2.0" {
				return fmt.Errorf("invalid JSON-RPC protocol")
			}

			if msg.IsRequest() {
				request := messages.NewRequestFromJsonRPCMessage(msg)
				response := s.handleRequest(ctx, request)
				s.Transport.Write(*response)
			} else if msg.IsNotification() {

			} else if msg.IsResponse() {

			}
		}
	}
}

func NewDefaultServer(transport Transport, protocolVersion string, version string, name string) *DefaultServer {
	server := &DefaultServer{
		Version:             version,
		Name:                name,
		Transport:           transport,
		ProtocolVersion:     protocolVersion,
		requestHandlers:     make(RequestHandlersMap),
		cancellableRequests: make(CancellableRequestMap),
		Capabilities:        Capabilities{},
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
