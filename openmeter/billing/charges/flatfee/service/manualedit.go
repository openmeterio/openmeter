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
		if err := s.Charge.Intent.Mutate(meta.ChangeTargetOverride, func(fields *flatfee.IntentMutableFields) {
			*fields = overrideFields
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
	if invoiceAtAccessor, ok := line.(billing.InvoiceAtAccessor); ok {
		out.InvoiceAt = invoiceAtAccessor.GetInvoiceAt()
	} else {
		// Standard invoice lines do not carry their own invoice-at value, so
		// keep the current effective charge intent's invoice-at for standard-line edits.
		out.InvoiceAt = s.Charge.Intent.GetEffectiveInvoiceAt()
	}
	out.FeatureKey = line.GetFeatureKey()
	out.PaymentTerm = flatPrice.PaymentTerm
	out.AmountBeforeProration = flatPrice.Amount

	var taxConfig *billing.TaxConfig
	switch invoiceLine := line.AsInvoiceLine(); invoiceLine.Type() {
	case billing.InvoiceLineTypeStandard:
		standardLine, err := invoiceLine.AsStandardLine()
		if err != nil {
			return flatfee.IntentMutableFields{}, fmt.Errorf("getting standard line[%s]: %w", line.GetID(), err)
		}

		taxConfig = standardLine.TaxConfig
	case billing.InvoiceLineTypeGathering:
		gatheringLine, err := invoiceLine.AsGatheringLine()
		if err != nil {
			return flatfee.IntentMutableFields{}, fmt.Errorf("getting gathering line[%s]: %w", line.GetID(), err)
		}

		taxConfig = billing.FromProductCatalog(gatheringLine.TaxConfig)
	}
	// Flat-fee override intents require a tax code ID, but gathering-line edits
	// may omit tax config or carry legacy tax data without the normalized tax
	// code ID. Treat missing tax state as unchanged, and preserve the current
	// effective tax code ID when only the line tax behavior was provided.
	if taxConfig != nil {
		out.TaxConfig = productcatalog.TaxCodeConfigFrom(taxConfig.ToProductCatalog())
		if out.TaxConfig.TaxCodeID == "" {
			out.TaxConfig.TaxCodeID = s.Charge.Intent.GetEffectiveTaxConfig().TaxCodeID
		}
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

func (s *CreditThenInvoiceStateMachine) UnsupportedManualEditOperation(_ context.Context, _ billing.InvoiceLineOverride) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot manually edit flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}
