package server

import (
	"github.com/alwint3r/mcp2go/mcp/messages"
)

type Transport interface {
	Read() <-chan messages.JsonRPCMessage

	Write(msg messages.JsonRPCMessage) error

	Close() error
}
