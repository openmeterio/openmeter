package flushhandler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

type FlushEventHandler interface {
	OnFlushSuccess(ctx context.Context, events []models.SinkMessage) error
	Start(context.Context) error
	WaitForDrain(context.Context) error
	Close() error
}

type (
	FlushCallback func(context.Context, []models.SinkMessage) error
)
