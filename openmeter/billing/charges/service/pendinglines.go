package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) CreatePendingInvoiceLines(ctx context.Context, input charges.CreatePendingInvoiceLinesInput) (*charges.CreatePendingInvoiceLinesResult, error) {
	input.Lines = slices.Clone(input.Lines)

	for i := range input.Lines {
		input.Lines[i].Namespace = input.Customer.Namespace
		input.Lines[i].Currency = input.Currency
		// The HTTP layer defaults lines to the invoice engine; charge-backed creation
		// re-derives the engine from the charge type.
		if input.Lines[i].Engine == billing.LineEngineTypeInvoice {
			input.Lines[i].Engine = ""
		}
	}

	if err := validateChargePendingInvoiceLinesInput(input); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	intents, err := mapPendingInvoiceLinesToChargeIntents(input)
	if err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*charges.CreatePendingInvoiceLinesResult, error) {
		result, err := s.create(ctx, charges.CreateInput{
			Namespace: input.Customer.Namespace,
			Intents:   intents,
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("create charges for pending invoice lines: result is nil")
		}

		if len(result.collectionAlignmentBypassedLines) > 0 {
			if err := s.invokeInvoiceNowOnCreate(ctx, result.collectionAlignmentBypassedLines); err != nil {
				return nil, fmt.Errorf("invoking invoice now on create: %w", err)
			}
		}

		if _, err := s.autoAdvanceCreatedCharges(ctx, result.charges); err != nil {
			return nil, err
		}

		if len(result.pendingLineResults) == 0 {
			return nil, fmt.Errorf("create charges for pending invoice lines: no gathering lines were created")
		}
		if len(result.pendingLineResults) > 1 {
			return nil, fmt.Errorf("create charges for pending invoice lines: expected one pending-line result, got %d", len(result.pendingLineResults))
		}

		pendingLineResult := result.pendingLineResults[0]
		orderedLines, err := orderPendingLinesByCreatedCharges(pendingLineResult.Lines, result.charges)
		if err != nil {
			return nil, fmt.Errorf("validating pending line results: %w", err)
		}
		pendingLineResult.Lines = orderedLines

		return pendingLineResult, nil
	})
}

func validateChargePendingInvoiceLinesInput(input charges.CreatePendingInvoiceLinesInput) error {
	var errs []error

	if err := input.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(input.Lines) == 0 {
		errs = append(errs, errors.New("no lines provided"))
	}

	for idx, line := range input.Lines {
		if line.ChargeID != nil {
			errs = append(errs, fmt.Errorf("line.%d: charge ID is not allowed for charge-backed pending line creation", idx))
		}

		if line.Engine != "" {
			errs = append(errs, fmt.Errorf("line.%d: engine is not allowed for charge-backed pending line creation", idx))
		}

		if line.ManagedBy != billing.ManuallyManagedLine {
			errs = append(errs, fmt.Errorf("line.%d: managed by must be %s for charge-backed pending line creation", idx, billing.ManuallyManagedLine))
		}

		if line.Subscription != nil {
			errs = append(errs, fmt.Errorf("line.%d: subscription is not allowed for charge-backed pending line creation", idx))
		}

		if line.Price.Type() == productcatalog.FlatPriceType {
			flatPrice, err := line.Price.AsFlat()
			if err != nil {
				errs = append(errs, fmt.Errorf("line.%d: converting price to flat: %w", idx, err))
				continue
			}

			// Zero-amount flat-fee charges do not materialize gathering lines, which this
			// flow cannot represent.
			if flatPrice.Amount.IsZero() {
				errs = append(errs, fmt.Errorf("line.%d: zero-amount flat fee is not supported for charge-backed pending line creation", idx))
			}

			if line.RateCardDiscounts.Usage != nil {
				errs = append(errs, fmt.Errorf("line.%d: usage discount is not supported for flat fee lines", idx))
			}
		}
	}

	return errors.Join(errs...)
}

func mapPendingInvoiceLinesToChargeIntents(input charges.CreatePendingInvoiceLinesInput) (charges.ChargeIntents, error) {
	intents := make(charges.ChargeIntents, 0, len(input.Lines))

	for idx, line := range input.Lines {
		intent, err := mapPendingInvoiceLineToChargeIntent(input.Customer.ID, input.Currency, line)
		if err != nil {
			return nil, fmt.Errorf("line.%d: %w", idx, err)
		}

		intents = append(intents, intent)
	}

	return intents, nil
}

func mapPendingInvoiceLineToChargeIntent(customerID string, currency currencyx.Code, line billing.GatheringLine) (charges.ChargeIntent, error) {
	baseIntent, err := chargeIntentBaseFromPendingInvoiceLine(customerID, currency, line)
	if err != nil {
		return charges.ChargeIntent{}, err
	}

	switch line.Price.Type() {
	case productcatalog.FlatPriceType:
		flatPrice, err := line.Price.AsFlat()
		if err != nil {
			return charges.ChargeIntent{}, fmt.Errorf("converting price to flat: %w", err)
		}

		return charges.NewChargeIntent(flatfee.Intent{
			Intent:                baseIntent,
			InvoiceAt:             line.InvoiceAt,
			SettlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			PaymentTerm:           flatPrice.PaymentTerm,
			FeatureKey:            line.FeatureKey,
			PercentageDiscounts:   billingPercentageDiscountToProductCatalog(line.RateCardDiscounts.Percentage),
			AmountBeforeProration: flatPrice.Amount,
		}), nil
	default:
		return charges.NewChargeIntent(usagebased.Intent{
			Intent:         baseIntent,
			InvoiceAt:      line.InvoiceAt,
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			FeatureKey:     line.FeatureKey,
			Price:          line.Price,
			Discounts:      billingDiscountsToProductCatalog(line.RateCardDiscounts),
		}), nil
	}
}

func chargeIntentBaseFromPendingInvoiceLine(customerID string, currency currencyx.Code, line billing.GatheringLine) (meta.Intent, error) {
	annotations, err := line.Annotations.Clone()
	if err != nil {
		return meta.Intent{}, fmt.Errorf("cloning annotations: %w", err)
	}

	return meta.Intent{
		Name:              line.Name,
		Description:       line.Description,
		Metadata:          line.Metadata.Clone(),
		Annotations:       annotations,
		ManagedBy:         billing.ManuallyManagedLine,
		CustomerID:        customerID,
		Currency:          currency,
		ServicePeriod:     line.ServicePeriod,
		FullServicePeriod: line.ServicePeriod,
		BillingPeriod:     line.ServicePeriod,
		TaxConfig:         productcatalog.TaxCodeConfigFrom(line.TaxConfig),
		UniqueReferenceID: line.ChildUniqueReferenceID,
	}, nil
}

func billingDiscountsToProductCatalog(discounts billing.Discounts) productcatalog.Discounts {
	return productcatalog.Discounts{
		Percentage: billingPercentageDiscountToProductCatalog(discounts.Percentage),
		Usage:      billingUsageDiscountToProductCatalog(discounts.Usage),
	}
}

func billingPercentageDiscountToProductCatalog(discount *billing.PercentageDiscount) *productcatalog.PercentageDiscount {
	if discount == nil {
		return nil
	}

	return lo.ToPtr(discount.PercentageDiscount.Clone())
}

func billingUsageDiscountToProductCatalog(discount *billing.UsageDiscount) *productcatalog.UsageDiscount {
	if discount == nil {
		return nil
	}

	return lo.ToPtr(discount.UsageDiscount.Clone())
}

func orderPendingLinesByCreatedCharges(lines []billing.GatheringLine, createdCharges charges.Charges) ([]billing.GatheringLine, error) {
	linesByChargeID := make(map[string]billing.GatheringLine, len(lines))
	var errs []error

	for idx, line := range lines {
		if line.ChargeID == nil || *line.ChargeID == "" {
			errs = append(errs, fmt.Errorf("line.%d: charge ID is required on charge-backed pending line result", idx))
			continue
		}

		chargeID := *line.ChargeID
		if _, ok := linesByChargeID[chargeID]; ok {
			errs = append(errs, fmt.Errorf("line.%d: duplicate charge ID %s in pending line result", idx, chargeID))
			continue
		}

		linesByChargeID[chargeID] = line
	}

	createdChargeIDs := make(map[string]struct{}, len(createdCharges))
	out := make([]billing.GatheringLine, 0, len(createdCharges))
	for idx, createdCharge := range createdCharges {
		chargeID, err := createdCharge.GetChargeID()
		if err != nil {
			errs = append(errs, fmt.Errorf("created charge.%d: resolving charge ID: %w", idx, err))
			continue
		}
		createdChargeIDs[chargeID.ID] = struct{}{}

		line, ok := linesByChargeID[chargeID.ID]
		if !ok {
			errs = append(errs, fmt.Errorf("created charge.%d[%s]: gathering line was not created", idx, chargeID.ID))
			continue
		}

		expectedEngine, ok := lineEngineTypeForChargeType(createdCharge.Type())
		if !ok {
			errs = append(errs, fmt.Errorf("created charge.%d[%s]: unsupported charge type %s", idx, chargeID.ID, createdCharge.Type()))
			continue
		}

		if line.Engine != expectedEngine {
			errs = append(errs, fmt.Errorf("created charge.%d[%s]: expected line engine %s, got %s", idx, chargeID.ID, expectedEngine, line.Engine))
			continue
		}

		out = append(out, line)
	}

	for idx, line := range lines {
		if line.ChargeID == nil {
			continue
		}

		if _, ok := createdChargeIDs[*line.ChargeID]; !ok {
			errs = append(errs, fmt.Errorf("line.%d: pending line result references unexpected charge ID %s", idx, *line.ChargeID))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return out, nil
}

func lineEngineTypeForChargeType(chargeType meta.ChargeType) (billing.LineEngineType, bool) {
	switch chargeType {
	case meta.ChargeTypeFlatFee:
		return billing.LineEngineTypeChargeFlatFee, true
	case meta.ChargeTypeUsageBased:
		return billing.LineEngineTypeChargeUsageBased, true
	default:
		return "", false
	}
}
