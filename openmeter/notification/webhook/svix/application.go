package svix

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/idempotency"
)

func (h svixHandler) CreateApplication(ctx context.Context, id string) (*svix.ApplicationOut, error) {
	fn := func(ctx context.Context) (*svix.ApplicationOut, error) {
		input := svix.ApplicationIn{
			Name: id,
			Uid:  &id,
		}

		idempotencyKey, err := idempotency.Key()
		if err != nil {
			return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationName, input.Name),
			attribute.String(AnnotationApplicationUID, lo.FromPtr(input.Uid)),
			attribute.String("idempotency_key", idempotencyKey),
		}

		span.AddEvent("creating application", trace.WithAttributes(spanAttrs...))

		app, err := h.client.Application.GetOrCreate(ctx, input, &svix.ApplicationCreateOptions{
			IdempotencyKey: &idempotencyKey,
		})
		if err = internal.WrapSvixError(err); err != nil {
			return nil, fmt.Errorf("failed to get or create Svix application: %w", err)
		}

		return app, nil
	}

	return tracex.Start[*svix.ApplicationOut](ctx, h.tracer, "svix.create_application").Wrap(fn)
}
