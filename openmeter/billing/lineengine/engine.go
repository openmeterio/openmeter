package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/clock"
)

var (
	_ billing.LineEngine     = (*Engine)(nil)
	_ billing.LineCalculator = (*Engine)(nil)
)

type Config struct {
	SplitLineGroupAdapter SplitLineGroupAdapter
	QuantitySnapshotter   QuantitySnapshotter
	RatingService         rating.Service
}

func (c Config) Validate() error {
	if c.SplitLineGroupAdapter == nil {
		return fmt.Errorf("split line group adapter is required")
	}

	if c.QuantitySnapshotter == nil {
		return fmt.Errorf("quantity snapshotter is required")
	}

	if c.RatingService == nil {
		return fmt.Errorf("rating service is required")
	}

	return nil
}

type Engine struct {
	adapter             SplitLineGroupAdapter
	quantitySnapshotter QuantitySnapshotter
	ratingService       rating.Service
}

func New(config Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Engine{
		adapter:             config.SplitLineGroupAdapter,
		quantitySnapshotter: config.QuantitySnapshotter,
		ratingService:       config.RatingService,
	}, nil
}

func (e *Engine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeInvoice
}

func (e *Engine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice is required")
	}

	if input.Invoice.QuantitySnapshotedAt != nil &&
		!input.Invoice.QuantitySnapshotedAt.Before(input.Invoice.DefaultCollectionAtForStandardInvoice()) {
		return input.Lines, nil
	}

	if input.Invoice.QuantitySnapshotedAt == nil &&
		input.Invoice.CollectionAt != nil &&
		clock.Now().Before(*input.Invoice.CollectionAt) {
		return input.Lines, nil
	}

	if err := e.quantitySnapshotter.SnapshotLineQuantities(ctx, input.Invoice, input.Lines); err != nil {
		if _, isInvalidDatabaseState := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); isInvalidDatabaseState {
			return nil, billing.ValidationIssue{
				Severity:  billing.ValidationIssueSeverityCritical,
				Code:      billing.ErrInvoiceLineSnapshotFailed.Code,
				Message:   err.Error(),
				Component: billing.ValidationComponentOpenMeterMetering,
			}
		}

		return nil, fmt.Errorf("snapshotting lines: %w", err)
	}

	return input.Lines, nil
}

func (e *Engine) OnMutableStandardLinesDeleted(_ context.Context, _ billing.OnMutableStandardLinesDeletedInput) error {
	return nil
}

func (e *Engine) OnUnsupportedCreditNote(_ context.Context, _ billing.OnUnsupportedCreditNoteInput) error {
	return nil
}

func (e *Engine) OnStandardInvoiceCreated(_ context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *Engine) OnInvoiceIssued(_ context.Context, _ billing.OnInvoiceIssuedInput) error {
	return nil
}

func (e *Engine) OnPaymentAuthorized(_ context.Context, _ billing.OnPaymentAuthorizedInput) error {
	return nil
}

func (e *Engine) OnPaymentSettled(_ context.Context, _ billing.OnPaymentSettledInput) error {
	return nil
}
