package noop

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct{}

func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error {
	return nil
}

func (h Handler) Reconcile(_ context.Context) error {
	return nil
}

func (h Handler) Start() error {
	return nil
}

func (h Handler) Close() error {
	return nil
}

func New() (*Handler, error) {
	return &Handler{}, nil
}
