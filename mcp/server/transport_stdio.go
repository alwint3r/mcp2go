package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

type StdioTransport struct {
	readerChannel chan messages.JsonRPCMessage
	logger        *Logger
	writerChannel chan messages.JsonRPCMessage
}

func NewStdioTransport() *StdioTransport {
	const channelCapacity = 100
	readerChannel := make(chan messages.JsonRPCMessage, channelCapacity)
	writerChannel := make(chan messages.JsonRPCMessage, channelCapacity)
	logger := NewLogger("StdioTransport")
	logger.Out = os.Stderr
	logger.ErrOut = os.Stderr
	return &StdioTransport{
		readerChannel: readerChannel,
		logger:        logger,
		writerChannel: writerChannel,
	}
}

func (s *StdioTransport) write(msg messages.JsonRPCMessage) error {
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

	return nil
}

func (s *StdioTransport) Read() <-chan messages.JsonRPCMessage {
	return s.readerChannel
}

func (s *StdioTransport) Write(msg messages.JsonRPCMessage, ctx context.Context) error {
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

	// Try to write to channel non-blocking first
	select {
	case s.writerChannel <- msg:
		return nil
	default:
		// Channel is full, try with timeout
		select {
		case s.writerChannel <- msg:
			return nil
		case <-ctx.Done():
			// Context deadline exceeded, fall back to direct write
			s.logger.Warn("Writer channel full, falling back to direct write: %v", ctx.Err())
			return s.write(msg) // Direct write as fallback
		}
	}
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

	scannerCtx, cancelScanner := context.WithCancel(context.Background())
	defer cancelScanner()

	scanErrCh := make(chan error, 1)
	lineCh := make(chan string, 10)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		const maxScannerBuffer = 1024 * 1024 // 1MB
		buffer := make([]byte, maxScannerBuffer)
		scanner.Buffer(buffer, maxScannerBuffer)

		go func() {
			<-scannerCtx.Done()
			// Force stdin to close/unblock by closing file descriptor
			// This is a hack to unblock scanner.Scan() when context is cancelled
			// Note that this won't work on Windows, but it's a Unix-specific solution
			f, _ := os.Open("/dev/null")
			os.Stdin.Close() // This will cause scanner.Scan() to return false
			os.Stdin = f     // Replace with a dummy file
		}()

		for scanner.Scan() {
			select {
			case <-scannerCtx.Done():
				return
			default:
				line := scanner.Text()
				if line == "" {
					continue
				}

				select {
				case lineCh <- line:
				case <-scannerCtx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			scanErrCh <- fmt.Errorf("scanner error: %w", err)
		}

		close(lineCh)
	}()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping transport: %v", ctx.Err())
			cancelScanner()
			return fmt.Errorf("transport stopped: %w", ctx.Err())

		case err := <-scanErrCh:
			s.logger.Error("Scanner error: %v", err)
			cancelScanner()
			return err

		case line, ok := <-lineCh:
			if !ok {
				s.logger.Info("End of input reached, transport stopping normally")
				return nil
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
					withTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if writeErr := s.Write(*errorMsg, withTimeout); writeErr != nil {
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
				cancelScanner()
				return fmt.Errorf("transport stopped while sending message: %w", ctx.Err())
			}

		case outgoing := <-s.writerChannel:
			err := s.write(outgoing)
			if err != nil {
				s.logger.Error("Failed to write response: %v", err)
			}
		}

	}
}
