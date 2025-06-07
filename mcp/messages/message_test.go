package messages_test

import (
	"encoding/json"
	"testing"

	"github.com/alwint3r/mcp2go/mcp/messages"
)

func TestValidRequestMessage(t *testing.T) {
	request := messages.NewJsonRPCMessage()
	request.ID = 1
	method := "initialize"
	request.Method = &method
	request.Params = &messages.JsonRPCParams{}

	if request.IsRequest() == false {
		t.Errorf("message should be a request")
	}

	if request.IsNotification() {
		t.Errorf("message should not be a notification")
	}

	if request.IsResponse() {
		t.Errorf("message should not be a response")
	}
}

func TestValidNotificationMessage(t *testing.T) {
	notification := messages.NewJsonRPCMessage()
	method := "textDocument/didChange"
	notification.Method = &method
	notification.Params = &messages.JsonRPCParams{
		"textDocument": map[string]interface{}{
			"uri": "file://example.txt",
		},
	}

	if !notification.IsNotification() {
		t.Errorf("message should be a notification")
	}

	if notification.IsRequest() {
		t.Errorf("message should not be a request")
	}

	if notification.IsResponse() {
		t.Errorf("message should not be a response")
	}
}

func TestValidSuccessResponseMessage(t *testing.T) {
	response := messages.NewJsonRPCMessage()
	response.ID = "request-1"
	result := messages.JsonRPCResult{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1,
		},
	}
	response.Result = &result

	if !response.IsResponse() {
		t.Errorf("message should be a response")
	}

	if response.IsRequest() {
		t.Errorf("message should not be a request")
	}

	if response.IsNotification() {
		t.Errorf("message should not be a notification")
	}

	if response.Error != nil {
		t.Errorf("error should be nil for success response")
	}
}

func TestValidErrorResponseMessage(t *testing.T) {
	response := messages.NewJsonRPCMessage()
	response.ID = 42
	errorResponse := messages.ErrorResponse{
		Code:    -32601,
		Message: "Method not found",
	}
	response.Error = &errorResponse

	if !response.IsResponse() {
		t.Errorf("message should be a response")
	}

	if response.IsRequest() {
		t.Errorf("message should not be a request")
	}

	if response.IsNotification() {
		t.Errorf("message should not be a notification")
	}

	if response.Error == nil {
		t.Errorf("error should not be nil for error response")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with different ID types (number, string, null)
	t.Run("WithStringID", func(t *testing.T) {
		msg := messages.NewJsonRPCMessage()
		msg.ID = "abc123"
		method := "test"
		msg.Method = &method

		if !msg.IsRequest() {
			t.Errorf("message with string ID should be a request")
		}
	})

	t.Run("WithNumberID", func(t *testing.T) {
		msg := messages.NewJsonRPCMessage()
		msg.ID = 123
		method := "test"
		msg.Method = &method

		if !msg.IsRequest() {
			t.Errorf("message with number ID should be a request")
		}
	})

	t.Run("InvalidMessage", func(t *testing.T) {
		msg := messages.NewJsonRPCMessage()
		// Not setting any fields, should not match any message type

		if msg.IsRequest() {
			t.Errorf("empty message should not be a request")
		}

		if msg.IsNotification() {
			t.Errorf("empty message should not be a notification")
		}

		if msg.IsResponse() {
			t.Errorf("empty message should not be a response")
		}
	})
}

func TestJsonSerialization(t *testing.T) {
	t.Run("RequestSerialization", func(t *testing.T) {
		request := messages.NewJsonRPCMessage()
		request.ID = 1
		method := "initialize"
		request.Method = &method
		params := messages.JsonRPCParams{
			"processId": 1234,
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		}
		request.Params = &params

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		var unmarshaledRequest messages.JsonRPCMessage
		err = json.Unmarshal(jsonData, &unmarshaledRequest)
		if err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		// Verify all fields were preserved
		if unmarshaledRequest.JsonRPC != "2.0" {
			t.Errorf("jsonrpc should be 2.0, got %s", unmarshaledRequest.JsonRPC)
		}

		if unmarshaledRequest.ID != float64(1) {
			t.Errorf("ID should be 1, got %v (type: %T)", unmarshaledRequest.ID, unmarshaledRequest.ID)
		}

		if *unmarshaledRequest.Method != "initialize" {
			t.Errorf("method should be initialize, got %s", *unmarshaledRequest.Method)
		}

		if (*unmarshaledRequest.Params)["processId"] != float64(1234) {
			t.Errorf("processId should be 1234, got %v", (*unmarshaledRequest.Params)["processId"])
		}

		if !unmarshaledRequest.IsRequest() {
			t.Errorf("unmarshalledRequest should be a request")
		}
	})

	t.Run("NotificationSerialization", func(t *testing.T) {
		notification := messages.NewJsonRPCMessage()
		method := "textDocument/didChange"
		notification.Method = &method
		params := messages.JsonRPCParams{
			"textDocument": map[string]interface{}{
				"uri":     "file://example.txt",
				"version": 2,
			},
			"contentChanges": []interface{}{
				map[string]interface{}{
					"text": "new content",
				},
			},
		}
		notification.Params = &params

		jsonData, err := json.Marshal(notification)
		if err != nil {
			t.Fatalf("failed to marshal notification: %v", err)
		}

		var unmarshaledNotification messages.JsonRPCMessage
		err = json.Unmarshal(jsonData, &unmarshaledNotification)
		if err != nil {
			t.Fatalf("failed to unmarshal notification: %v", err)
		}

		// Verify notification properties
		if !unmarshaledNotification.IsNotification() {
			t.Errorf("unmarshaled message should be a notification")
		}

		textDocumentMap := (*unmarshaledNotification.Params)["textDocument"].(map[string]interface{})
		if textDocumentMap["uri"] != "file://example.txt" {
			t.Errorf("uri should be file://example.txt, got %v", textDocumentMap["uri"])
		}
	})

	t.Run("ErrorResponseSerialization", func(t *testing.T) {
		errorResponse := messages.NewJsonRPCMessage()
		errorResponse.ID = "request-1"
		var errorData interface{} = "Additional error details"
		errorObj := messages.ErrorResponse{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    &errorData,
		}
		errorResponse.Error = &errorObj

		jsonData, err := json.Marshal(errorResponse)
		if err != nil {
			t.Fatalf("failed to marshal error response: %v", err)
		}

		var unmarshaledResponse messages.JsonRPCMessage
		err = json.Unmarshal(jsonData, &unmarshaledResponse)
		if err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		// Verify error response properties
		if !unmarshaledResponse.IsResponse() {
			t.Errorf("unmarshaled message should be a response")
		}

		if unmarshaledResponse.ID != "request-1" {
			t.Errorf("ID should be request-1, got %v", unmarshaledResponse.ID)
		}

		if unmarshaledResponse.Error == nil {
			t.Fatalf("error should not be nil")
		}

		if unmarshaledResponse.Error.Code != -32600 {
			t.Errorf("error code should be -32600, got %d", unmarshaledResponse.Error.Code)
		}

		if unmarshaledResponse.Error.Message != "Invalid Request" {
			t.Errorf("error message should be 'Invalid Request', got %s", unmarshaledResponse.Error.Message)
		}
	})
}

func TestRawJsonDeserialization(t *testing.T) {
	t.Run("DeserializeRequest", func(t *testing.T) {
		jsonStr := `{
			"jsonrpc": "2.0", 
			"id": 42, 
			"method": "textDocument/formatting", 
			"params": {
				"textDocument": {"uri": "file:///example.go"},
				"options": {"tabSize": 4, "insertSpaces": true}
			}
		}`

		var msg messages.JsonRPCMessage
		err := json.Unmarshal([]byte(jsonStr), &msg)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if !msg.IsRequest() {
			t.Errorf("message should be a request")
		}

		if msg.ID != float64(42) {
			t.Errorf("ID should be 42, got %v", msg.ID)
		}

		if *msg.Method != "textDocument/formatting" {
			t.Errorf("method should be textDocument/formatting, got %s", *msg.Method)
		}

		options, ok := (*msg.Params)["options"].(map[string]interface{})
		if !ok {
			t.Fatalf("params.options should be a map")
		}

		if options["tabSize"] != float64(4) {
			t.Errorf("options.tabSize should be 4, got %v", options["tabSize"])
		}
	})
}
