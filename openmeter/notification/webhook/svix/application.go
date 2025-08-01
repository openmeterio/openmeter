package svix

import (
	"context"
	"fmt"
	"time"

	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
)

func (h svixHandler) CreateApplication(ctx context.Context, id string) (*svix.ApplicationOut, error) {
	input := svix.ApplicationIn{
		Name: id,
		Uid:  &id,
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	app, err := h.client.Application.GetOrCreate(ctx, input, &svix.ApplicationCreateOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err = internal.WrapSvixError(err); err != nil {
		return nil, fmt.Errorf("failed to get or create Svix application: %w", err)
	}

	return app, nil
}
