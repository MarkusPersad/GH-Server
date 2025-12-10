package response

import (
	"GH-Server/pkg/exceptions"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Success(message string, data any) *Response {
	return &Response{
		Code:    0,
		Message: message,
		Data:    data,
	}
}

func Error(exception *exceptions.Exception, data any) *Response {
	return &Response{
		Code:    exception.Code,
		Message: exception.Msg,
		Data:    data,
	}
}
