package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

type StdioTransport struct {
	reader        *bufio.Reader
	readerChannel chan messages.JsonRPCMessage
}

func NewStdioTransport() *StdioTransport {
	readerChannel := make(chan messages.JsonRPCMessage, 10)
	return &StdioTransport{
		reader:        bufio.NewReader(os.Stdin),
		readerChannel: readerChannel,
	}
}

func (s *StdioTransport) Read() <-chan messages.JsonRPCMessage {
	return s.readerChannel
}

func (s *StdioTransport) Write(msg messages.JsonRPCMessage) error {
	marshaled, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(marshaled)
	if err != nil {
		return err
	}

	return nil
}

func (s *StdioTransport) Close() error {
	close(s.readerChannel)
	return nil
}

func (s *StdioTransport) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			switch line, err := s.reader.ReadString('\n'); err {
			case nil:
				var msg messages.JsonRPCMessage
				err := json.Unmarshal([]byte(line), &msg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
					continue
				}
				s.readerChannel <- msg
			case io.EOF:
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	}
}
