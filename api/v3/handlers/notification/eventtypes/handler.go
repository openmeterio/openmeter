package eventtypes

import (
	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListNotificationEventTypes() ListNotificationEventTypesHandler
	GetNotificationEventType() GetNotificationEventTypeHandler
}

// Event types are the same in every namespace, so the handlers read the
// registry without resolving the namespace.
type handler struct {
	registry *v4.EventRegistry
	options  []httptransport.HandlerOption
}

func New(registry *v4.EventRegistry, options ...httptransport.HandlerOption) Handler {
	return &handler{
		registry: registry,
		options:  options,
	}
}
