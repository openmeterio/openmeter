package eventbus

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AfterCommitEnqueuer registers a best-effort enqueue callback on the current
// transaction. The event must remain immutable after registration.
type AfterCommitEnqueuer func(context.Context, marshaler.Event)

// NewAfterCommitEnqueuer calls enqueue only after the outer transaction commits.
// The callback runs on the writer and must return without waiting for delivery or
// retrying a failed enqueue. Registration failures are logged and the event is
// skipped. Neither registration nor delivery errors propagate to the writer.
func NewAfterCommitEnqueuer(enqueue func(marshaler.Event), logger *slog.Logger) AfterCommitEnqueuer {
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
