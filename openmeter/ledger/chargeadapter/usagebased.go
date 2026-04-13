package chargeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// usageBasedHandler maps usage-based credit lifecycle events to ledger transaction templates.
type usageBasedHandler struct {
	collector collector.Service
}

var _ usagebased.Handler = (*usageBasedHandler)(nil)

func NewUsageBasedHandler(collectorService collector.Service) usagebased.Handler {
	return &usageBasedHandler{
		collector: collectorService,
	}
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccrued(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.AmountToAllocate.IsZero() {
		return nil, nil
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued: %w", err)
	}

	realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:      input.Charge.Namespace,
		ChargeID:       input.Charge.ID,
		CustomerID:     input.Charge.Intent.CustomerID,
		At:             input.AllocateAt,
		Currency:       input.Charge.Intent.Currency,
		SettlementMode: input.Charge.Intent.SettlementMode,
		ServicePeriod:  input.Charge.Intent.ServicePeriod,
		Amount:         input.AmountToAllocate,
	})
	if err != nil {
		return nil, err
	}
	if len(realizations) == 0 {
		return nil, nil
	}

	return realizations, nil
}

func (h *usageBasedHandler) OnCreditsOnlyUsageAccruedCorrection(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Charge.Validate(); err != nil {
		return nil, fmt.Errorf("charge: %w", err)
	}

	if err := input.Run.Validate(); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}

	if input.AllocateAt.IsZero() {
		return nil, fmt.Errorf("allocate at is required")
	}

	if err := validateSettlementMode(
		input.Charge.Intent.SettlementMode,
		productcatalog.CreditOnlySettlementMode,
		productcatalog.CreditThenInvoiceSettlementMode,
	); err != nil {
		return nil, fmt.Errorf("credits only usage accrued correction: %w", err)
	}

	currencyCalculator, err := input.Charge.Intent.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("get currency calculator: %w", err)
	}

	if err := input.Corrections.ValidateWith(currencyCalculator); err != nil {
		return nil, fmt.Errorf("corrections: %w", err)
	}

	return h.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:                    input.Charge.Namespace,
		ChargeID:                     input.Charge.ID,
		CustomerID:                   input.Charge.Intent.CustomerID,
		AllocateAt:                   input.AllocateAt,
		Corrections:                  input.Corrections,
		LineageSegmentsByRealization: input.LineageSegmentsByRealization,
	})
}
