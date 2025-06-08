package server

import (
	"context"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

type Transport interface {
	Read() <-chan messages.JsonRPCMessage

	Write(msg messages.JsonRPCMessage, ctx context.Context) error

	Close() error
}
