package messages

const progressNotificationMethodName = "notifications/progress"

func NewProgressNotification(progressToken string, progress float32, total float32, message string) *Notification {
	return &Notification{
		JsonRPC: "2.0",
		Method:  progressNotificationMethodName,
		Params: &map[string]interface{}{
			"progressToken": progressToken,
			"progress":      progress,
			"total":         total,
			"message":       message,
		},
	}
}

func WithProgress(request *Request, progressToken string) *Request {
	if request.Params == nil {
		request.Params = &JsonRPCParams{}
	}

	params := *request.Params
	meta, ok := params["_meta"].(map[string]interface{})
	if !ok {
		meta = make(map[string]interface{})
	}
	meta["progressToken"] = progressToken
	params["_meta"] = meta

	return request
}
