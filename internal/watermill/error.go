package watermill

import "github.com/ThreeDotsLabs/watermill/message"

type RetryableError interface {
	error

	RetryMessages() []*message.Message
}

func NewRetryableError(messages []*message.Message, err error) RetryableError {
	return &retryableError{retryMessages: messages, err: err}
}

type retryableError struct {
	retryMessages []*message.Message
	err           error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) RetryMessages() []*message.Message {
	return e.retryMessages
}
