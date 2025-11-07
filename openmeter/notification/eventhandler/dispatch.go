package eventhandler

import (
	"context"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

func (h *Handler) Dispatch(ctx context.Context, event *notification.Event) error {
	spanLink := trace.LinkFromContext(ctx)

	fn := func(ctx context.Context) error {
		return h.reconcileEvent(ctx, event)
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("notification event handler panicked",
					"error", err,
					"code.stacktrace", string(debug.Stack()))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout)
		defer cancel()

		tracerOpts := []trace.SpanStartOption{
			trace.WithNewRoot(),
			trace.WithLinks(spanLink),
		}

		err := tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", tracerOpts...).Wrap(fn)
		if err != nil {
			h.logger.WarnContext(ctx, "failed to dispatch event", "eventID", event.ID, "error", err)
		}
	}()

	return nil
}
