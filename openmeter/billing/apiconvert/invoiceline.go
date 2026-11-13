package apiconvert

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	catalogconvert "github.com/openmeterio/openmeter/openmeter/productcatalog/apiconvert"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func mapDetailedLinesToAPI(lines billing.DetailedLines, currency currencyx.FiatCode, invoiceAt time.Time, taxConfig *productcatalog.TaxConfig) (*[]apimodels.InvoiceDetailedLine, error) {
	mappedLines, err := slicesx.MapWithErr(lines, func(line billing.DetailedLine) (apimodels.InvoiceDetailedLine, error) {
		return mapDetailedLineToAPI(line, currency, invoiceAt, taxConfig)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map detailed lines: %w", err)
	}

	return lo.ToPtr(mappedLines), nil
}

func mapDetailedLineToAPI(line billing.DetailedLine, currency currencyx.FiatCode, invoiceAt time.Time, taxConfig *productcatalog.TaxConfig) (apimodels.InvoiceDetailedLine, error) {
	amountDiscountsAPI, err := mapInvoiceLineAmountDiscountsToAPI(line.AmountDiscounts)
	if err != nil {
		return apimodels.InvoiceDetailedLine{}, fmt.Errorf("failed to map amount discounts: %w", err)
	}

	var discountsAPI *apimodels.InvoiceLineDiscounts
	if amountDiscountsAPI != nil {
		discountsAPI = &apimodels.InvoiceLineDiscounts{
			Amount: amountDiscountsAPI,
		}
	}

	return apimodels.InvoiceDetailedLine{
		Type: apimodels.InvoiceDetailedLineTypeFlatFee,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: invoiceAt,

		Currency: string(currency),
		Status:   apimodels.InvoiceLineStatusDetailed,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   apimodels.InvoiceLineManagedBySystem,

		Invoice: &apimodels.InvoiceReference{
			Id: line.InvoiceID,
		},

		Period: apimodels.Period{
			From: line.ServicePeriod.From,
			To:   line.ServicePeriod.To,
		},

		PerUnitAmount: lo.ToPtr(line.PerUnitAmount.String()),
		Quantity:      lo.ToPtr(line.Quantity.String()),
		Category:      lo.ToPtr(apimodels.InvoiceDetailedLineCostCategory(line.Category)),
		TaxConfig:     ToAPITaxConfig(taxConfig),
		PaymentTerm:   lo.ToPtr(apimodels.PricePaymentTerm(line.PaymentTerm)),

		RateCard: &apimodels.InvoiceDetailedLineRateCard{
			TaxConfig: ToAPITaxConfig(taxConfig),
			Price: &apimodels.FlatPriceWithPaymentTerm{
				Type:        apimodels.FlatPriceWithPaymentTermTypeFlat,
				PaymentTerm: lo.ToPtr(apimodels.PricePaymentTerm(line.PaymentTerm)),
				Amount:      line.PerUnitAmount.String(),
			},
			Quantity: lo.ToPtr(line.Quantity.String()),
		},

		CreditAllocations: mapInvoiceLineCreditsToAPI(line.CreditsApplied),
		Discounts:         discountsAPI,
		Totals:            mapTotalsToAPI(line.Totals),

		ExternalIds: mapLineAppExternalIdsToAPI(line.ExternalIDs),
	}, nil
}

func mapLineAppExternalIdsToAPI(externalIds externalid.LineExternalIDs) *apimodels.InvoiceLineAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &apimodels.InvoiceLineAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
	}
}

func mapInvoiceLineToAPI(line *billing.StandardLine) (apimodels.InvoiceLine, error) {
	if line.UsageBased == nil {
		return apimodels.InvoiceLine{}, fmt.Errorf("usage based line details are nil [line=%s]", line.ID)
	}

	if line.UsageBased.Price == nil {
		return apimodels.InvoiceLine{}, fmt.Errorf("price is nil [line=%s]", line.ID)
	}

	price, err := catalogconvert.ToAPIRateCardUsageBasedPrice(*line.UsageBased.Price)
	if err != nil {
		return apimodels.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	children, err := mapDetailedLinesToAPI(line.DetailedLines, line.Currency, line.InvoiceAt, line.TaxConfig.ToProductCatalog())
	if err != nil {
		return apimodels.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
	}

	discountsAPI, err := mapDiscountsToAPI(line.Discounts)
	if err != nil {
		return apimodels.InvoiceLine{}, fmt.Errorf("failed to map discounts: %w", err)
	}

	invoiceLine := apimodels.InvoiceLine{
		Type: apimodels.InvoiceLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		// TODO: deprecation
		Currency: string(line.Currency),
		Status:   apimodels.InvoiceLineStatusValid,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   apimodels.InvoiceLineManagedBy(line.ManagedBy),

		// TODO: deprecation
		Invoice: &apimodels.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: apimodels.Period{
			From: line.Period.From,
			To:   line.Period.To,
		},

		TaxConfig: ToAPITaxConfig(line.TaxConfig.ToProductCatalog()),

		FeatureKey:                   lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		MeteredQuantity:              decimalPtrToStringPtrIfNotEqual(line.UsageBased.MeteredQuantity, line.UsageBased.Quantity),
		Quantity:                     decimalPtrToStringPtr(line.UsageBased.Quantity),
		PreLinePeriodQuantity:        decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.PreLinePeriodQuantity),
		MeteredPreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.MeteredPreLinePeriodQuantity),

		Price: lo.ToPtr(price),

		RateCard: &apimodels.InvoiceUsageBasedRateCard{
			TaxConfig:  ToAPITaxConfig(line.TaxConfig.ToProductCatalog()),
			Price:      lo.ToPtr(price),
			FeatureKey: lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		},

		CreditAllocations: mapInvoiceLineCreditsToAPI(line.CreditsApplied),
		Discounts:         discountsAPI,
		Children:          children,
		Totals:            mapTotalsToAPI(line.Totals),

		ExternalIds:  mapLineAppExternalIdsToAPI(line.ExternalIDs),
		Subscription: ToAPIInvoiceLineSubscriptionReference(line.Subscription),
	}

	return invoiceLine, nil
}

func ToAPIInvoiceLineSubscriptionReference(optSub *billing.SubscriptionReference) *apimodels.InvoiceLineSubscriptionReference {
	if optSub == nil {
		return nil
	}

	out := &apimodels.InvoiceLineSubscriptionReference{
		Item: apimodels.IDResource{
			Id: optSub.ItemID,
		},
		Phase: apimodels.IDResource{
			Id: optSub.PhaseID,
		},
		Subscription: apimodels.IDResource{
			Id: optSub.SubscriptionID,
		},
	}

	out.BillingPeriod = apimodels.Period{
		From: optSub.BillingPeriod.From,
		To:   optSub.BillingPeriod.To,
	}

	return out
}

func mapInvoiceLineCreditsToAPI(credits billing.CreditsApplied) *[]apimodels.InvoiceLineCreditAllocation {
	if len(credits) == 0 {
		return nil
	}

	out := lo.Map(credits, func(credit billing.CreditApplied, _ int) apimodels.InvoiceLineCreditAllocation {
		return apimodels.InvoiceLineCreditAllocation{
			Amount:      credit.Amount.String(),
			Description: lo.EmptyableToPtr(credit.Description),
		}
	})

	return &out
}

func mapDiscountsToAPI(discounts billing.StandardLineDiscounts) (*apimodels.InvoiceLineDiscounts, error) {
	if len(discounts.Usage) == 0 {
		return nil, nil
	}

	mappedUsageDiscounts, err := mapInvoiceLineUsageDiscountsToAPI(discounts.Usage)
	if err != nil {
		return nil, fmt.Errorf("failed to map usage discounts: %w", err)
	}

	return &apimodels.InvoiceLineDiscounts{
		Usage: mappedUsageDiscounts,
	}, nil
}

func mapInvoiceLineAmountDiscountsToAPI(amountDiscounts billing.AmountLineDiscountsManaged) (*[]apimodels.InvoiceLineAmountDiscount, error) {
	if len(amountDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(amountDiscounts, func(discount billing.AmountLineDiscountManaged) (apimodels.InvoiceLineAmountDiscount, error) {
		reason, err := mapDiscountReasonToAPI(discount.Reason)
		if err != nil {
			return apimodels.InvoiceLineAmountDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
		}

		return apimodels.InvoiceLineAmountDiscount{
			Id:          discount.ID,
			Amount:      discount.Amount.String(),
			CreatedAt:   discount.CreatedAt,
			DeletedAt:   discount.DeletedAt,
			UpdatedAt:   discount.UpdatedAt,
			Description: discount.Description,
			ExternalIds: mapLineAppExternalIdsToAPI(discount.ExternalIDs),
			Reason:      reason,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map amount discounts: %w", err)
	}

	return lo.ToPtr(mapped), nil
}

func mapInvoiceLineUsageDiscountsToAPI(usageDiscounts billing.UsageLineDiscountsManaged) (*[]apimodels.InvoiceLineUsageDiscount, error) {
	if len(usageDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(usageDiscounts, func(discount billing.UsageLineDiscountManaged) (apimodels.InvoiceLineUsageDiscount, error) {
		reason, err := mapDiscountReasonToAPI(discount.Reason)
		if err != nil {
			return apimodels.InvoiceLineUsageDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
		}

		return apimodels.InvoiceLineUsageDiscount{
			Id:                    discount.ID,
			Quantity:              discount.Quantity.String(),
			PreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(discount.PreLinePeriodQuantity),
			CreatedAt:             discount.CreatedAt,
			DeletedAt:             discount.DeletedAt,
			UpdatedAt:             discount.UpdatedAt,
			Description:           discount.Description,
			ExternalIds:           mapLineAppExternalIdsToAPI(discount.ExternalIDs),
			Reason:                reason,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map usage discounts: %w", err)
	}

	return lo.ToPtr(mapped), nil
}

func mapDiscountReasonToAPI(reason billing.DiscountReason) (apimodels.BillingDiscountReason, error) {
	out := apimodels.BillingDiscountReason{}
	switch reason.Type() {
	case billing.MaximumSpendDiscountReason:
		if err := out.FromDiscountReasonMaximumSpend(apimodels.DiscountReasonMaximumSpend{
			Type: apimodels.DiscountReasonMaximumSpendTypeMaximumSpend,
		}); err != nil {
			return out, err
		}
	case billing.RatecardPercentageDiscountReason:
		reason, err := reason.AsRatecardPercentage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardPercentage(apimodels.DiscountReasonRatecardPercentage{
			Percentage:    reason.Percentage,
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          apimodels.DiscountReasonRatecardPercentageType(billing.RatecardPercentageDiscountReason),
		}); err != nil {
			return out, err
		}
	case billing.RatecardUsageDiscountReason:
		reason, err := reason.AsRatecardUsage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardUsage(apimodels.DiscountReasonRatecardUsage{
			Quantity:      reason.Quantity.String(),
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          apimodels.DiscountReasonRatecardUsageType(billing.RatecardUsageDiscountReason),
		}); err != nil {
			return out, err
		}
	default:
		return apimodels.BillingDiscountReason{}, fmt.Errorf("unknown discount reason type: %s", reason.Type())
	}

	return out, nil
}

func decimalPtrToStringPtr(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	return lo.ToPtr(d.String())
}

func decimalPtrToStringPtrIgnoringZeroValue(d *alpacadecimal.Decimal) *string {
	if d == nil {
		return nil
	}

	if d.IsZero() {
		return nil
	}

	return lo.ToPtr(d.String())
}

// decimalPtrToStringPtrIfNotEqual returns a pointer to the string representation of the decimal if it is not equal to the other decimal.
func decimalPtrToStringPtrIfNotEqual(value *alpacadecimal.Decimal, other *alpacadecimal.Decimal) *string {
	if value == nil {
		return nil
	}

	if other == nil {
		return lo.ToPtr(value.String())
	}

	if value.Equal(lo.FromPtr(other)) {
		return nil
	}

	return lo.ToPtr(value.String())
}

func ToAPITaxConfig(to *productcatalog.TaxConfig) *apimodels.TaxConfig {
	if to == nil {
		return nil
	}

	return lo.ToPtr(catalogconvert.ToAPITaxConfig(*to))
}
