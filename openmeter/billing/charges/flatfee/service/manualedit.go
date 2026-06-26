package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// setManualOverrideIntent replaces the API-owned override layer with the edited
// invoice-line intent. The first manual edit creates the override row; later
// edits update only that override layer so subscription sync can keep owning
// the base layer.
func (s *CreditThenInvoiceStateMachine) setManualOverrideIntent(ctx context.Context, overrideFields flatfee.IntentMutableFields) error {
	if s.Charge.Intent.HasOverrideLayer() {
		if err := s.Charge.Intent.Mutate(meta.ChangeTargetOverride, func(_ flatfee.IntentMutableFields) (flatfee.IntentMutableFields, error) {
			return overrideFields, nil
		}); err != nil {
			return fmt.Errorf("mutating manual override intent: %w", err)
		}
	} else {
		// Manual edits first create an override layer. Subscription sync keeps
		// updating the base layer while API edits own the effective layer.
		base, err := s.Adapter.CreateChargeOverride(ctx, s.Charge.ChargeBase, overrideFields)
		if err != nil {
			return fmt.Errorf("creating manual intent override: %w", err)
		}

		s.Charge.ChargeBase = base
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) intentMutableFieldsFromManualLine(line billing.GenericInvoiceLineReader) (flatfee.IntentMutableFields, error) {
	if line == nil {
		return flatfee.IntentMutableFields{}, fmt.Errorf("line is required")
	}

	price := line.GetPrice()
	if price == nil {
		return flatfee.IntentMutableFields{}, fmt.Errorf("line[%s]: price is required", line.GetID())
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return flatfee.IntentMutableFields{}, fmt.Errorf("getting flat price from line[%s]: %w", line.GetID(), err)
	}

	out := s.Charge.Intent.GetEffectiveIntent().IntentMutableFields
	out.Name = line.GetName()
	out.Description = line.GetDescription()
	out.Metadata = line.GetMetadata().Clone()
	out.ServicePeriod = line.GetServicePeriod()
	out.InvoiceAt = line.GetInvoiceAt()
	out.FeatureKey = line.GetFeatureKey()
	out.PaymentTerm = flatPrice.PaymentTerm
	out.AmountBeforeProration = flatPrice.Amount

	taxConfig := line.GetTaxConfig()
	if taxConfig == nil {
		out.TaxConfig = productcatalog.TaxCodeConfig{}
	} else {
		out.TaxConfig = productcatalog.TaxCodeConfigFrom(taxConfig.ToProductCatalog())
	}

	if line.GetRateCardDiscounts().Percentage == nil {
		out.PercentageDiscounts = nil
	} else {
		out.PercentageDiscounts = lo.ToPtr(line.GetRateCardDiscounts().Percentage.PercentageDiscount.Clone())
	}

	out = out.Normalized(s.Charge.Intent.GetCurrency())
	if err := out.Validate(); err != nil {
		return flatfee.IntentMutableFields{}, err
	}

	return out, nil
}

// buildFlatFeeGatheringLineFromEffectiveIntent projects the persisted
// customer-facing charge intent back to the invoice line after a manual edit.
// The API edit is first normalized and persisted as an override, so returning
// the edited line directly could drift from the durable effective charge state.
func buildFlatFeeGatheringLineFromEffectiveIntent(charge flatfee.Charge, existing billing.GatheringLine) (billing.GatheringLine, error) {
	line, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
		Charge:        charge,
		ServicePeriod: charge.Intent.GetEffectiveServicePeriod(),
		InvoiceAt:     charge.Intent.GetEffectiveInvoiceAt(),
	})
	if err != nil {
		return billing.GatheringLine{}, err
	}

	line.ID = existing.ID
	line.CreatedAt = existing.CreatedAt
	line.UpdatedAt = existing.UpdatedAt
	line.DeletedAt = existing.DeletedAt
	line.InvoiceID = existing.InvoiceID
	line.UBPConfigID = existing.UBPConfigID
	line.DBState = existing.DBState

	return line, nil
}

func (s *CreditThenInvoiceStateMachine) UnsupportedManualEditOperation(_ context.Context, _ billing.InvoiceLineOverride) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot manually edit flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}
