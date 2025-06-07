package messages

import "errors"

type ErrorResponse struct {
	Code    int64        `json:"code"`
	Message string       `json:"message"`
	Data    *interface{} `json:"data"`
}

type Response struct {
	JsonRPC string                  `json:"jsonrpc"`
	ID      interface{}             `json:"id"`
	Result  *map[string]interface{} `json:"result"`
	Error   *ErrorResponse          `json:"error"`
}

func (r *Response) IsValid() error {
	if r.Result != nil && r.Error != nil {
		return errors.New("response must not have a result and an error")
	}

	return nil
}
