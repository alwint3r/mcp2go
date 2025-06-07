package messages

type JsonRPCResult map[string]interface{}
type JsonRPCParams map[string]interface{}

type JsonRPCMessage struct {
	JsonRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id,omitempty"`
	Method  *string        `json:"method,omitempty"`
	Params  *JsonRPCParams `json:"params,omitempty"`
	Result  *JsonRPCResult `json:"result,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

func NewJsonRPCMessage() *JsonRPCMessage {
	return &JsonRPCMessage{
		JsonRPC: "2.0",
	}
}

func (j *JsonRPCMessage) IsNotification() bool {
	return j.ID == nil && j.Method != nil
}

func (j *JsonRPCMessage) IsRequest() bool {
	return j.ID != nil && j.Method != nil
}

func (j *JsonRPCMessage) IsResponse() bool {
	return j.ID != nil && j.Method == nil && (j.Error != nil || j.Result != nil)
}
