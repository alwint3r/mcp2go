package messages

const cancellationMethodName = "notifications/cancelled"

func NewCancellationNotification(requestID interface{}, reason string) *Notification {
	return &Notification{
		JsonRPC: "2.0",
		Method:  cancellationMethodName,
		Params: &map[string]interface{}{
			"requestId": requestID,
			"reason":    reason,
		},
	}
}

func NewCancellationOfRequest(request *Request, reason string) *Notification {
	return &Notification{
		JsonRPC: request.JsonRPC,
		Method:  cancellationMethodName,
		Params: &map[string]interface{}{
			"requestId": request.ID,
			"reason":    reason,
		},
	}
}
