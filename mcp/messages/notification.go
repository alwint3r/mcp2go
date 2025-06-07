package messages

type Notification struct {
	JsonRPC string                  `json:"jsonrpc"`
	Method  string                  `json:"method"`
	Params  *map[string]interface{} `json:"params"`
}
