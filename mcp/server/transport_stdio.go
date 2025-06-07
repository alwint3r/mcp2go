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
	logger        *Logger
}

func NewStdioTransport() *StdioTransport {
	readerChannel := make(chan messages.JsonRPCMessage, 10)
	logger := NewLogger("StdioTransport")
	logger.Out = os.Stderr
	logger.ErrOut = os.Stderr
	return &StdioTransport{
		readerChannel: readerChannel,
		logger:        logger,
	}
}

func (s *StdioTransport) Read() <-chan messages.JsonRPCMessage {
	return s.readerChannel
}

func (s *StdioTransport) Write(msg messages.JsonRPCMessage) error {
	marshaled, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to marshal message: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	withNewLine := string(marshaled) + "\n"
	_, err = os.Stdout.WriteString(withNewLine)
	if err != nil {
		s.logger.Error("Failed to write to stdout: %v", err)
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	if msg.IsResponse() {
		if msg.Error != nil {
			s.logger.Debug("Sent error response: id=%v, code=%d", msg.ID, msg.Error.Code)
		} else {
			s.logger.Debug("Sent success response: id=%v", msg.ID)
		}
	} else if msg.IsRequest() {
		s.logger.Debug("Sent request: method=%s, id=%v", *msg.Method, msg.ID)
	} else if msg.IsNotification() {
		s.logger.Debug("Sent notification: method=%s", *msg.Method)
	}

	return nil
}

func (s *StdioTransport) Close() error {
	s.logger.Info("Closing transport")

	func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warn("Attempted to close already closed channel: %v", r)
			}
		}()
		close(s.readerChannel)
	}()

	s.logger.Info("Transport closed")
	return nil
}

func (s *StdioTransport) Start(ctx context.Context) error {
	s.logger.Info("Starting StdIO transport")
	defer s.logger.Info("StdIO transport stopped")

	scanner := bufio.NewScanner(os.Stdin)

	const maxScannerBuffer = 1024 * 1024 // 1MB
	buffer := make([]byte, maxScannerBuffer)
	scanner.Buffer(buffer, maxScannerBuffer)

	// Main scanning loop
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping transport: %v", ctx.Err())
			return fmt.Errorf("transport stopped: %w", ctx.Err())
		default:
			line := scanner.Text()
			if line == "" {
				continue
			}

			s.logger.Debug("Received input line: %d bytes", len(line))
			var msg messages.JsonRPCMessage
			err := json.Unmarshal([]byte(line), &msg)
			if err != nil {
				s.logger.Error("Error parsing JSON: %v", err)
				errorMsg := messages.NewJsonRPCMessage()
				errorMsg.Error = &messages.ErrorResponse{
					Code:    messages.JsonRPCErrorParse,
					Message: fmt.Sprintf("Failed to parse JSON: %v", err),
				}

				go func() {
					if writeErr := s.Write(*errorMsg); writeErr != nil {
						s.logger.Error("Failed to write error response: %v", writeErr)
					}
				}()
				continue
			}

			select {
			case s.readerChannel <- msg:
				if msg.IsRequest() {
					s.logger.Debug("Queued request message: method=%s, id=%v", *msg.Method, msg.ID)
				} else if msg.IsNotification() {
					s.logger.Debug("Queued notification message: method=%s", *msg.Method)
				} else {
					s.logger.Debug("Queued response message: id=%v", msg.ID)
				}
			case <-ctx.Done():
				s.logger.Warn("Context cancelled while sending message")
				return fmt.Errorf("transport stopped while sending message: %w", ctx.Err())
			}
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("Scanner error: %v", err)
		return fmt.Errorf("scanner error: %w", err)
	}

	s.logger.Info("End of input reached, transport stopping normally")
	return nil
}
