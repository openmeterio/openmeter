package webhook

import (
	"errors"
	"fmt"
	"time"
)

var ErrNotImplemented = errors.New("not implemented")

func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}

func IgnoreNotImplemented(err error) error {
	if IsNotImplemented(err) {
		return nil
	}

	return err
}

var _ error = (*ValidationError)(nil)

type ValidationError struct {
	err error
}

func (e ValidationError) Error() string {
	return e.err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.err
}

func NewValidationError(err error) error {
	if err == nil {
		return nil
	}

	return ValidationError{err: err}
}

func IsValidationError(err error) bool {
	return isError[ValidationError](err)
}

type NotFoundError struct {
	err error
}

func (e NotFoundError) Error() string {
	return e.err.Error()
}

func (e NotFoundError) Unwrap() error {
	return e.err
}

func NewNotFoundError(err error) error {
	if err == nil {
		return nil
	}

	return NotFoundError{err: err}
}

func IsNotFoundError(err error) bool {
	return isError[NotFoundError](err)
}

func isError[T error](err error) bool {
	var t T

	return errors.As(err, &t)
}

type RetryableError struct {
	err error

	retryAfter time.Duration
}

func (e RetryableError) Error() string {
	return e.err.Error()
}

func (e RetryableError) Unwrap() error {
	return e.err
}

func (e RetryableError) RetryAfter() time.Duration {
	return e.retryAfter
}

func IsRetryableError(err error) bool {
	return isError[RetryableError](err)
}

const DefaultRetryAfter = 15 * time.Second

func NewRetryableError(err error, after time.Duration) error {
	if err == nil {
		return nil
	}

	if after == 0 {
		after = DefaultRetryAfter
	}

	return RetryableError{
		err:        err,
		retryAfter: after,
	}
}

type MessageAlreadyExistsError struct {
	namespace string
	eventID   string
}

func (e MessageAlreadyExistsError) Error() string {
	return fmt.Sprintf("message already exists [namespace=%s eventID=%s]", e.namespace, e.eventID)
}

func NewMessageAlreadyExistsError(namespace string, eventID string) error {
	return MessageAlreadyExistsError{
		namespace: namespace,
		eventID:   eventID,
	}
}

func IsMessageAlreadyExistsError(err error) bool {
	return isError[MessageAlreadyExistsError](err)
}

type UnrecoverableError struct {
	err error
}

func (e UnrecoverableError) Error() string {
	return e.err.Error()
}

func (e UnrecoverableError) Unwrap() error {
	return e.err
}

func IsUnrecoverableError(err error) bool {
	return isError[UnrecoverableError](err)
}

func NewUnrecoverableError(err error) error {
	if err == nil {
		return nil
	}

	return UnrecoverableError{
		err: err,
	}
}

var ErrMaxChannelsPerWebhookExceeded = fmt.Errorf("maximum number of channels (%d) per webhook exceeded", MaxChannelsPerWebhook)

func IsMaxChannelsPerWebhookExceededError(err error) bool {
	return errors.Is(err, ErrMaxChannelsPerWebhookExceeded)
}
