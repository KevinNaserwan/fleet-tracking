package httpapi

import "github.com/gin-gonic/gin"

type ResponseMeta struct {
	RequestID string `json:"requestId"`
}

type successEnvelope struct {
	Data any          `json:"data"`
	Meta ResponseMeta `json:"meta"`
}

type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	RequestID string        `json:"requestId"`
	Details   []ErrorDetail `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func WriteSuccess(c *gin.Context, status int, data any) {
	c.JSON(status, successEnvelope{
		Data: data,
		Meta: ResponseMeta{RequestID: RequestID(c)},
	})
}

func WriteError(
	c *gin.Context,
	status int,
	code, message string,
	details ...ErrorDetail,
) {
	c.JSON(status, errorEnvelope{Error: errorBody{
		Code: code, Message: message, RequestID: RequestID(c), Details: details,
	}})
}
