package events

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListNotificationEvents() ListNotificationEventsHandler
	GetNotificationEvent() GetNotificationEventHandler
	ResendNotificationEvent() ResendNotificationEventHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          notification.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service notification.Service,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(errorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          sharedOptions,
	}
}
