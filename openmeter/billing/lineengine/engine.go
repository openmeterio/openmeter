package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

func (e *Engine) OnMutableInvoiceLinesEditedViaAPI(ctx context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	if err := input.Validate(); err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("validating input: %w", err)
	}

	createdLines, err := slicesx.MapWithErr(input.Created, func(line billing.GenericInvoiceLine) (billing.GenericInvoiceLine, error) {
		lineID := line.GetID()

		line, err := e.snapshotManualStandardLineOverrideIfNeeded(ctx, input.Invoice, line)
		if err != nil {
			return nil, fmt.Errorf("snapshotting line[%s]: %w", lineID, err)
		}

		return line, nil
	})
	if err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("snapshotting created lines: %w", err)
	}

	updatedLines, err := slicesx.MapWithErr(input.Updated, func(override billing.InvoiceLineOverride) (billing.GenericInvoiceLine, error) {
		if err := validateSplitLineOverride(override); err != nil {
			return nil, err
		}

		line, err := override.ChangesToApply.Apply(override.ExistingLine)
		if err != nil {
			return nil, fmt.Errorf("applying changes to line[%s]: %w", override.ExistingLine.GetID(), err)
		}

		line, err = e.snapshotManualStandardLineOverrideIfNeeded(ctx, input.Invoice, line)
		if err != nil {
			return nil, fmt.Errorf("snapshotting line[%s]: %w", override.ExistingLine.GetID(), err)
		}

		return line, nil
	})
	if err != nil {
		return billing.OnMutableInvoiceUpdateResult{}, fmt.Errorf("snapshotting updated lines: %w", err)
	}

	return billing.OnMutableInvoiceUpdateResult{
		CreatedLines: createdLines,
		UpdatedLines: updatedLines,
	}, nil
}

func (e *Engine) snapshotManualStandardLineOverrideIfNeeded(ctx context.Context, invoice billing.GenericInvoiceReader, line billing.GenericInvoiceLine) (billing.GenericInvoiceLine, error) {
	if invoice.GetType() != billing.InvoiceTypeStandard {
		return line, nil
	}

	standardInvoice, err := invoice.AsInvoice().AsStandardInvoice()
	if err != nil {
		return nil, fmt.Errorf("getting standard invoice: %w", err)
	}

	if standardInvoice.Status == billing.StandardInvoiceStatusGathering {
		return line, nil
	}

	standardLine, err := line.AsInvoiceLine().AsStandardLine()
	if err != nil {
		return nil, fmt.Errorf("getting standard line: %w", err)
	}

	if err := e.quantitySnapshotter.SnapshotLineQuantities(ctx, standardInvoice, billing.StandardLines{&standardLine}); err != nil {
		return nil, fmt.Errorf("snapshotting line quantity: %w", err)
	}

	return standardLine.AsGenericLine(), nil
}

func validateSplitLineOverride(override billing.InvoiceLineOverride) error {
	if override.ExistingLine.GetSplitLineGroupID() == nil {
		return nil
	}

	if period, ok := override.ChangesToApply.Period.Get(); ok && !period.Equal(override.ExistingLine.GetServicePeriod()) {
		return billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", override.ExistingLine.GetID(), billing.ErrInvoiceLineNoPeriodChangeForSplitLine),
		}
	}

	if price, ok := override.ChangesToApply.Price.Get(); ok && !price.Equal(override.ExistingLine.GetPrice()) {
		return billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", override.ExistingLine.GetID(), billing.ErrInvoiceProgressiveBillingNotSupported),
		}
	}

	if featureKey, ok := override.ChangesToApply.FeatureKey.Get(); ok && featureKey != override.ExistingLine.GetFeatureKey() {
		return billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", override.ExistingLine.GetID(), billing.ErrInvoiceProgressiveBillingNotSupported),
		}
	}

	if discounts, ok := override.ChangesToApply.Discounts.Get(); ok && !equal.PtrEqual(discounts.Usage, override.ExistingLine.GetRateCardDiscounts().Usage) {
		return billing.ValidationError{
			Err: fmt.Errorf("line[%s]: %w", override.ExistingLine.GetID(), billing.ErrInvoiceLineProgressiveBillingUsageDiscountUpdateForbidden),
		}
	}

	return nil
}

func (e *Engine) OnMutableStandardLinesDeletedBySystem(_ context.Context, _ billing.OnMutableStandardLinesDeletedInput) error {
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
