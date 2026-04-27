package notification

import (
	"context"
	"runtime"
	"time"
)

const (
	DefaultReconcileInterval           = 15 * time.Second
	DefaultDispatchTimeout             = 30 * time.Second
	DefaultDeliveryStatePendingTimeout = 3 * time.Hour
	DefaultDeliveryStateSendingTimeout = 48 * time.Hour
)

var DefaultReconcilerWorkers = runtime.GOMAXPROCS(0)

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
	Dispatch(ctx context.Context, event *Event) error
}
