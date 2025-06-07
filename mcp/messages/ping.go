package messages

const pingRequestMethodName = "ping"

type PingRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
}

type PingResponse struct {
	JsonRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Result  map[string]interface{} `json:"result"`
}

func NewPingResponse(requestID interface{}) *PingResponse {
	return &PingResponse{
		JsonRPC: "2.0",
		ID:      requestID,
		Result:  make(map[string]interface{}),
	}
}

func NewPingRequest(requestID interface{}) *PingRequest {
	return &PingRequest{
		JsonRPC: "2.0",
		ID:      requestID,
		Method:  pingRequestMethodName,
	}
}
