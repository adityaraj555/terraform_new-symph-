package error_handler

type RetriableError struct {
	Message string
}

func (err *RetriableError) Error() string {
	return err.Message
}
