package notification

import "context"

type validator interface {
	Validate(context.Context, Service) error
}
