package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

type StdioTransport struct {
	readerChannel chan messages.JsonRPCMessage
}

func NewStdioTransport() *StdioTransport {
	readerChannel := make(chan messages.JsonRPCMessage, 10)
	return &StdioTransport{
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

	withNewLine := string(marshaled) + "\n"
	_, err = os.Stdout.WriteString(withNewLine)
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
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg messages.JsonRPCMessage
			err := json.Unmarshal([]byte(line), &msg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
				continue
			}

			s.readerChannel <- msg
		}
	}

	return nil
}
