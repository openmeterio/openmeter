package eventbus

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AfterCommitPublisher observes committed source changes for a best-effort consumer.
// The event must be immutable once submitted.
type AfterCommitPublisher func(context.Context, marshaler.Event)

// NewAfterCommitPublisher waits for the outer transaction to commit before handing
// the event to enqueue. enqueue must be nonblocking because it runs on the writer.
func NewAfterCommitPublisher(enqueue func(marshaler.Event), logger *slog.Logger) AfterCommitPublisher {
	return func(ctx context.Context, event marshaler.Event) {
		tx, err := entutils.GetDriverFromContext(ctx)
		if err == nil {
			err = tx.AfterCommit(func() { enqueue(event) })
		}

		if err != nil {
			logger.WarnContext(ctx, "skipping after-commit event", "event", event.EventName(), "error", err)
		}
	}
}
