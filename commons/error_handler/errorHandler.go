package error_handler

type ICodedError interface {
	error
	GetErrorCode() int
}

type ServiceError struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorDescription"`
}

func (err ServiceError) Error() string {
	return err.ErrorMessage
}

func (err ServiceError) GetErrorCode() int {
	return err.ErrorCode
}

func NewServiceError(code int, message string) *ServiceError {
	return &ServiceError{ErrorCode: code, ErrorMessage: message}
}

type RetriableError struct {
	ServiceError
}

func NewRetriableError(code int, message string) *RetriableError {
	return &RetriableError{ServiceError: ServiceError{ErrorCode: code, ErrorMessage: message}}
}

func (err RetriableError) Error() string {
	return err.ErrorMessage
}
func (err RetriableError) GetErrorCode() int {
	return err.ErrorCode
}
