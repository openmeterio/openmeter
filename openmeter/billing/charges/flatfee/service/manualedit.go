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

func (s *CreditThenInvoiceStateMachine) UnsupportedLineManualEditOperation(_ context.Context, _ meta.PatchLineManualEdit) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot manually edit flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

func (s *CreditThenInvoiceStateMachine) intentMutableFieldsFromLineManualEdit(line billing.GenericInvoiceLineReader) (flatfee.IntentMutableFields, error) {
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
	out.PaymentTerm = flatPrice.PaymentTerm
	out.AmountBeforeProration = flatPrice.Amount
	out.PercentageDiscounts = line.GetRateCardDiscounts().Percentage.CloneOrNil()

	out = out.Normalized(s.Charge.Intent.GetCurrency())
	if err := out.Validate(); err != nil {
		return flatfee.IntentMutableFields{}, err
	}

	return out, nil
}

func intentFromManualCreatedLine(
	ctx context.Context,
	invoice billing.GenericInvoiceReader,
	line billing.GenericInvoiceLineReader,
	defaultInvoicingTaxCodeResolver billing.DefaultTaxCodeResolver,
) (flatfee.Intent, error) {
	if invoice == nil {
		return flatfee.Intent{}, fmt.Errorf("invoice is required")
	}

	if line == nil {
		return flatfee.Intent{}, fmt.Errorf("line is required")
	}

	if line.GetID() == "" {
		return flatfee.Intent{}, fmt.Errorf("line id is required")
	}

	if chargeID := line.GetChargeID(); chargeID != nil && *chargeID != "" {
		return flatfee.Intent{}, fmt.Errorf("line[%s]: charge id must be empty for manual create", line.GetID())
	}

	price := line.GetPrice()
	if price == nil {
		return flatfee.Intent{}, fmt.Errorf("line[%s]: price is required", line.GetID())
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return flatfee.Intent{}, fmt.Errorf("getting flat price from line[%s]: %w", line.GetID(), err)
	}

	annotations, err := line.GetAnnotations().Clone()
	if err != nil {
		return flatfee.Intent{}, fmt.Errorf("cloning line[%s] annotations: %w", line.GetID(), err)
	}

	servicePeriod := line.GetServicePeriod()
	invoiceAt := line.GetCreatedAt()
	if invoiceAtAccessor, ok := line.(billing.InvoiceAtAccessor); ok {
		invoiceAt = invoiceAtAccessor.GetInvoiceAt()
	} else {
		// New standard lines do not expose invoice-at as generic scheduling
		// input. For charge-backed manual creates, derive the intent schedule
		// from the flat-fee payment term instead of the line's display-only
		// StandardLine.InvoiceAt field.
		switch flatPrice.PaymentTerm {
		case productcatalog.InAdvancePaymentTerm:
			invoiceAt = servicePeriod.From
		case productcatalog.InArrearsPaymentTerm:
			invoiceAt = servicePeriod.To
		}
	}

	taxConfig := productcatalog.TaxCodeConfig{}
	if lineTaxConfig := line.GetTaxConfig(); lineTaxConfig != nil {
		taxConfig = productcatalog.TaxCodeConfigFrom(lineTaxConfig.ToProductCatalog())
	}

	intent := flatfee.Intent{
		Intent: meta.Intent{
			ManagedBy:   billing.ManuallyManagedLine,
			CustomerID:  invoice.GetCustomerID().ID,
			Annotations: annotations,
			Currency:    line.GetCurrency(),
			TaxConfig:   taxConfig,
		},
		IntentMutableFields: flatfee.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              line.GetName(),
				Description:       line.GetDescription(),
				Metadata:          line.GetMetadata().Clone(),
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
			},
			InvoiceAt:             invoiceAt,
			PaymentTerm:           flatPrice.PaymentTerm,
			PercentageDiscounts:   nil,
			ProRating:             productcatalog.ProRatingConfig{},
			AmountBeforeProration: flatPrice.Amount,
		},
		FeatureKey:     lo.EmptyableToPtr(line.GetFeatureKey()),
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
	}

	if line.GetRateCardDiscounts().Percentage != nil {
		intent.PercentageDiscounts = lo.ToPtr(line.GetRateCardDiscounts().Percentage.Clone())
	}

	intent = intent.Normalized()
	if intent.TaxConfig.TaxCodeID == "" {
		if defaultInvoicingTaxCodeResolver == nil {
			return flatfee.Intent{}, fmt.Errorf("line[%s]: default invoicing tax code resolver is required", line.GetID())
		}

		defaultTaxCodeID, err := defaultInvoicingTaxCodeResolver(ctx)
		if err != nil {
			return flatfee.Intent{}, fmt.Errorf("resolving default invoicing tax code: %w", err)
		}

		intent.TaxConfig.TaxCodeID = defaultTaxCodeID
	}

	if err := intent.Validate(); err != nil {
		return flatfee.Intent{}, err
	}

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return flatfee.Intent{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	if amountAfterProration.IsZero() {
		return flatfee.Intent{}, billing.ErrInvoiceLineZeroAmountCreate
	}

	return intent, nil
}
