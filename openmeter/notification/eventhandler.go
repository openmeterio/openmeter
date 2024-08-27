package notification

import (
	"context"
	"time"
)

const (
	DefaultReconcileInterval = 15 * time.Second
	DefaultDispatchTimeout   = 30 * time.Second
)

type EventHandler interface {
	EventDispatcher
	EventReconciler

	Start() error
	Close() error
}

type EventReconciler interface {
	Reconcile(ctx context.Context) error
}

type EventDispatcher interface {
	Dispatch(*Event) error
}
