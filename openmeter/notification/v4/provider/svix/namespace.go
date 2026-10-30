package svix

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/v2/go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	tracexv2 "github.com/openmeterio/openmeter/pkg/framework/tracex/v2"
)

const providerDataApplicationID = "svix.application_id"

func (p *Provider) GetNamespace(ctx context.Context, input notificationprovider.GetNamespaceInput) (*notificationprovider.GetNamespaceResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.GetNamespaceResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid get application input [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		app, err := p.client.Application().Get(ctx, input.ID)
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to get application [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		result, err := fromAPIApplicationOut(app)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid get application response [svix.application.id=%s svix.application.uid=%s]: %w",
				lo.FromPtr(app).Id,
				input.ID,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.application.id", app.Id),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.get_namespace",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.ID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) CreateNamespace(ctx context.Context, input notificationprovider.CreateNamespaceInput) (*notificationprovider.CreateNamespaceResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.CreateNamespaceResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid create application input [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		metadata := input.Metadata.ToMap()
		if metadata == nil {
			metadata = map[string]string{}
		}

		// UID-based recovery also works after Svix's idempotency window expires.
		app, err := p.client.Application().GetOrCreate(ctx, svix.ApplicationIn{
			Uid:      &input.ID,
			Name:     input.Name,
			Metadata: &metadata,
		}, &svix.ApplicationCreateOptions{IdempotencyKey: input.IdempotencyKey})
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to get or create application [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		result, err := fromAPIApplicationOut(app)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid get or create application response [svix.application.id=%s svix.application.uid=%s]: %w",
				lo.FromPtr(app).Id,
				input.ID,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.application.id", app.Id),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.create_namespace",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.ID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) UpdateNamespace(ctx context.Context, input notificationprovider.UpdateNamespaceInput) (*notificationprovider.UpdateNamespaceResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.UpdateNamespaceResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid update application input [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		patch := svix.ApplicationPatch{Name: input.Name}
		if input.Metadata != nil {
			metadata := input.Metadata.ToMap()
			if metadata == nil {
				metadata = map[string]string{}
			}

			patch.Metadata = &metadata
		}

		var app *svix.ApplicationOut
		var err error

		// Empty updates still verify existence without making a remote mutation.
		switch {
		case input.Name == nil && input.Metadata == nil:
			app, err = p.client.Application().Get(ctx, input.ID)
		default:
			app, err = p.client.Application().Patch(ctx, input.ID, patch)
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to update application [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		result, err := fromAPIApplicationOut(app)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid update application response [svix.application.id=%s svix.application.uid=%s]: %w",
				lo.FromPtr(app).Id,
				input.ID,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.application.id", app.Id),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.update_namespace",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.ID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) DeleteNamespace(ctx context.Context, input notificationprovider.DeleteNamespaceInput) error {
	fn := func(ctx context.Context) error {
		if err := input.Validate(); err != nil {
			return fmt.Errorf(
				"invalid delete application input [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		err := p.client.Application().Delete(ctx, input.ID)

		var svixErr *svix.Error

		if errors.As(err, &svixErr) && svixErr.Status() == http.StatusNotFound {
			return nil
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return fmt.Errorf(
				"failed to delete application [svix.application.uid=%s]: %w",
				input.ID,
				err,
			)
		}

		return nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.delete_namespace",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.ID),
		),
	).Do(ctx, fn)
}

func fromAPIApplicationOut(app *svix.ApplicationOut) (*notificationprovider.Namespace, error) {
	if app == nil {
		return nil, errors.New("invalid application response: nil response")
	}

	// Never replace Notification identity with a generated provider ID, including
	// when a malformed response omits the UID.
	if app.Id == "" || lo.FromPtr(app.Uid) == "" {
		return nil, &notificationprovider.ProviderError{
			Kind: notificationprovider.ProviderErrorKindTerminal,
			Code: notificationprovider.ProviderErrorCodeRequestFailed,
			Err:  errors.New("invalid application response: missing ID or UID"),
		}
	}

	return &notificationprovider.Namespace{
		ID:       *app.Uid,
		Name:     app.Name,
		Metadata: app.Metadata,
		ProviderData: map[string]string{
			providerDataApplicationID: app.Id,
		},
	}, nil
}
