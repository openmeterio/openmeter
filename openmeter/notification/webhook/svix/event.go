package svix

import (
	"context"
	"fmt"

	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

func (h svixHandler) RegisterEventTypes(ctx context.Context, params webhook.RegisterEventTypesInputs) error {
	fn := func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)

		for _, eventType := range params.EventTypes {
			input := svix.EventTypeUpdate{
				Description: eventType.Description,
				FeatureFlag: nil,
				GroupName:   &eventType.GroupName,
				Schemas:     &eventType.Schemas,
				Deprecated:  &eventType.Deprecated,
			}

			spanAttrs := []attribute.KeyValue{
				attribute.String(AnnotationEventType, eventType.Name),
				attribute.String(AnnotationEventGroupName, eventType.GroupName),
			}

			span.AddEvent("upserting schema(s) for event type", trace.WithAttributes(spanAttrs...))

			_, err := h.client.EventType.Update(ctx, eventType.Name, input)
			if err = internal.WrapSvixError(err); err != nil {
				return fmt.Errorf("failed to create event type: %w", err)
			}
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "svix.register_event_types").Wrap(fn)
}
