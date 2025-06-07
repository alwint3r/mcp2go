package messages

type Request struct {
	JsonRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Method  string         `json:"method"`
	Params  *JsonRPCParams `json:"params,omitempty"`
}

func NewRequestFromJsonRPCMessage(message JsonRPCMessage) Request {
	return Request{
		JsonRPC: message.JsonRPC,
		ID:      message.ID,
		Method:  *message.Method,
		Params:  message.Params,
	}
}
