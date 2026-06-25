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
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *service) CreatePendingInvoiceLines(ctx context.Context, input charges.CreatePendingInvoiceLinesInput) (*charges.CreatePendingInvoiceLinesResult, error) {
	for i := range input.Lines {
		input.Lines[i].Namespace = input.Customer.Namespace
		input.Lines[i].Currency = input.Currency
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
		orderedLines, err := orderPendingLinesByCreatedCharges(orderPendingLinesByCreatedChargesInput{
			lines:          pendingLineResult.Lines,
			createdCharges: result.charges,
		})
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
	baseIntent, mutableFields, annotations, err := chargeIntentBaseFromPendingInvoiceLine(customerID, currency, line)
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
			IntentMutableFields:   mutableFields,
			Annotations:           annotations,
			InvoiceAt:             line.InvoiceAt,
			SettlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			PaymentTerm:           flatPrice.PaymentTerm,
			FeatureKey:            line.FeatureKey,
			PercentageDiscounts:   billingPercentageDiscountToProductCatalog(line.RateCardDiscounts.Percentage),
			AmountBeforeProration: flatPrice.Amount,
		}), nil
	default:
		return charges.NewChargeIntent(usagebased.Intent{
			Intent:              baseIntent,
			IntentMutableFields: mutableFields,
			Annotations:         annotations,
			InvoiceAt:           line.InvoiceAt,
			SettlementMode:      productcatalog.CreditThenInvoiceSettlementMode,
			FeatureKey:          line.FeatureKey,
			Price:               line.Price,
			Discounts:           billingDiscountsToProductCatalog(line.RateCardDiscounts),
		}), nil
	}
}

func chargeIntentBaseFromPendingInvoiceLine(customerID string, currency currencyx.Code, line billing.GatheringLine) (meta.Intent, meta.IntentMutableFields, models.Annotations, error) {
	annotations, err := line.Annotations.Clone()
	if err != nil {
		return meta.Intent{}, meta.IntentMutableFields{}, nil, fmt.Errorf("cloning annotations: %w", err)
	}

	return meta.Intent{
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currency,
			UniqueReferenceID: line.ChildUniqueReferenceID,
		}, meta.IntentMutableFields{
			Name:              line.Name,
			Description:       line.Description,
			Metadata:          line.Metadata.Clone(),
			ServicePeriod:     line.ServicePeriod,
			FullServicePeriod: line.ServicePeriod,
			BillingPeriod:     line.ServicePeriod,
			TaxConfig:         productcatalog.TaxCodeConfigFrom(line.TaxConfig),
		},
		annotations,
		nil
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

type orderPendingLinesByCreatedChargesInput struct {
	lines          []billing.GatheringLine
	createdCharges charges.Charges
}

func (i orderPendingLinesByCreatedChargesInput) Validate() error {
	if err := errors.Join(lo.Map(i.lines, func(line billing.GatheringLine, idx int) error {
		if line.ChargeID == nil || *line.ChargeID == "" {
			return fmt.Errorf("line.%d: charge ID is required on charge-backed pending line result", idx)
		}

		return nil
	})...); err != nil {
		return err
	}

	lineChargeIDs := lo.Map(i.lines, func(line billing.GatheringLine, _ int) string {
		return *line.ChargeID
	})
	if !slices.Equal(lineChargeIDs, lo.Uniq(lineChargeIDs)) {
		return fmt.Errorf("duplicate charge IDs found in pending line result")
	}

	return nil
}

func orderPendingLinesByCreatedCharges(input orderPendingLinesByCreatedChargesInput) ([]billing.GatheringLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	linesByChargeID := lo.SliceToMap(input.lines, func(line billing.GatheringLine) (string, billing.GatheringLine) {
		return *line.ChargeID, line
	})

	indexedCreatedCharges := lo.Map(input.createdCharges, func(charge charges.Charge, idx int) createdChargeWithIndex {
		return createdChargeWithIndex{
			index:  idx,
			charge: charge,
		}
	})

	out, err := slicesx.MapWithErr(indexedCreatedCharges, func(createdCharge createdChargeWithIndex) (billing.GatheringLine, error) {
		empty := billing.GatheringLine{}

		chargeID, err := createdCharge.charge.GetChargeID()
		if err != nil {
			return empty, fmt.Errorf("created charge.%d: resolving charge ID: %w", createdCharge.index, err)
		}

		line, ok := linesByChargeID[chargeID.ID]
		if !ok {
			return empty, fmt.Errorf("created charge.%d[%s]: gathering line was not created", createdCharge.index, chargeID.ID)
		}

		expectedEngine, ok := lineEngineTypeForChargeType(createdCharge.charge.Type())
		if !ok {
			return empty, fmt.Errorf("created charge.%d[%s]: unsupported charge type %s", createdCharge.index, chargeID.ID, createdCharge.charge.Type())
		}

		if line.Engine != expectedEngine {
			return empty, fmt.Errorf("created charge.%d[%s]: expected line engine %s, got %s", createdCharge.index, chargeID.ID, expectedEngine, line.Engine)
		}

		return line, nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

type createdChargeWithIndex struct {
	index  int
	charge charges.Charge
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
