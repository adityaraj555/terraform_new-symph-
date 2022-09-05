package error_handler

import (
	"encoding/json"
	"fmt"
)

type ICodedError interface {
	error
	GetErrorCode() int
}

type ServiceError struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorDescription"`
}
type ErrorMessage struct {
	Message     string      `json:"message"`
	MessageCode interface{} `json:"messageCode"`
}

func (err ServiceError) Error() string {
	return err.ErrorMessage
}

func (err ServiceError) GetErrorCode() int {
	return err.ErrorCode
}

func NewServiceError(code int, messages ...string) *ServiceError {
	var Message ErrorMessage
	err := json.Unmarshal([]byte(messages[0]), &Message)
	if (err != nil) || (Message.Message == "" && Message.MessageCode == nil) {
		Message = ErrorMessage{
			Message:     messages[0],
			MessageCode: code,
		}
	}
	for i := 1; i < len(messages); i++ {
		Message.Message += fmt.Sprintf(" %s", messages[i])
	}
	errorByteData, _ := json.Marshal(Message)
	return &ServiceError{ErrorCode: code, ErrorMessage: string(errorByteData)}
}

type RetriableError struct {
	ServiceError
}

func NewRetriableError(code int, messages ...string) *RetriableError {
	var Message ErrorMessage
	err := json.Unmarshal([]byte(messages[0]), &Message)
	if (err != nil) || (Message.Message == "" && Message.MessageCode == nil) {
		Message = ErrorMessage{
			Message:     messages[0],
			MessageCode: code,
		}
	}
	for i := 1; i < len(messages); i++ {
		Message.Message += fmt.Sprintf(" %s", messages[i])
	}
	errorByteData, _ := json.Marshal(Message)
	return &RetriableError{ServiceError: ServiceError{ErrorCode: code, ErrorMessage: string(errorByteData)}}
}

func (err RetriableError) Error() string {
	return err.ErrorMessage
}
func (err RetriableError) GetErrorCode() int {
	return err.ErrorCode
}
