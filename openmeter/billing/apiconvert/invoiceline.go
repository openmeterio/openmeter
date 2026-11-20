package apiconvert

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	catalogconvert "github.com/openmeterio/openmeter/openmeter/productcatalog/apiconvert"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func mapDetailedLinesToAPI(lines billing.DetailedLines, currency currencyx.FiatCode, invoiceAt time.Time, taxConfig *productcatalog.TaxCodeConfig, taxCodes productcatalog.TaxCodesByID) (*[]apiv1models.InvoiceDetailedLine, error) {
	mappedLines, err := slicesx.MapWithErr(lines, func(line billing.DetailedLine) (apiv1models.InvoiceDetailedLine, error) {
		return mapDetailedLineToAPI(line, currency, invoiceAt, taxConfig, taxCodes)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map detailed lines: %w", err)
	}

	return lo.ToPtr(mappedLines), nil
}

func mapDetailedLineToAPI(line billing.DetailedLine, currency currencyx.FiatCode, invoiceAt time.Time, taxConfig *productcatalog.TaxCodeConfig, taxCodes productcatalog.TaxCodesByID) (apiv1models.InvoiceDetailedLine, error) {
	lineTaxConfig, err := ToAPITaxConfig(taxCodes, taxConfig)
	if err != nil {
		return apiv1models.InvoiceDetailedLine{}, fmt.Errorf("failed to map tax config: %w", err)
	}

	amountDiscountsAPI, err := mapInvoiceLineAmountDiscountsToAPI(line.AmountDiscounts)
	if err != nil {
		return apiv1models.InvoiceDetailedLine{}, fmt.Errorf("failed to map amount discounts: %w", err)
	}

	var discountsAPI *apiv1models.InvoiceLineDiscounts
	if amountDiscountsAPI != nil {
		discountsAPI = &apiv1models.InvoiceLineDiscounts{
			Amount: amountDiscountsAPI,
		}
	}

	return apiv1models.InvoiceDetailedLine{
		Type: apiv1models.InvoiceDetailedLineTypeFlatFee,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: invoiceAt,

		Currency: string(currency),
		Status:   apiv1models.InvoiceLineStatusDetailed,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   apiv1models.InvoiceLineManagedBySystem,

		Invoice: &apiv1models.InvoiceReference{
			Id: line.InvoiceID,
		},

		Period: apiv1models.Period{
			From: line.ServicePeriod.From,
			To:   line.ServicePeriod.To,
		},

		PerUnitAmount: lo.ToPtr(line.PerUnitAmount.String()),
		Quantity:      lo.ToPtr(line.Quantity.String()),
		Category:      lo.ToPtr(apiv1models.InvoiceDetailedLineCostCategory(line.Category)),
		TaxConfig:     lineTaxConfig,
		PaymentTerm:   lo.ToPtr(apiv1models.PricePaymentTerm(line.PaymentTerm)),

		RateCard: &apiv1models.InvoiceDetailedLineRateCard{
			TaxConfig: lineTaxConfig,
			Price: &apiv1models.FlatPriceWithPaymentTerm{
				Type:        apiv1models.FlatPriceWithPaymentTermTypeFlat,
				PaymentTerm: lo.ToPtr(apiv1models.PricePaymentTerm(line.PaymentTerm)),
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

func mapLineAppExternalIdsToAPI(externalIds externalid.LineExternalIDs) *apiv1models.InvoiceLineAppExternalIds {
	if lo.IsEmpty(externalIds) {
		return nil
	}

	return &apiv1models.InvoiceLineAppExternalIds{
		Invoicing: lo.EmptyableToPtr(externalIds.Invoicing),
	}
}

func mapInvoiceLineToAPI(line *billing.StandardLine, taxCodes productcatalog.TaxCodesByID) (apiv1models.InvoiceLine, error) {
	if line.UsageBased == nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("usage based line details are nil [line=%s]", line.ID)
	}

	if line.UsageBased.Price == nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("price is nil [line=%s]", line.ID)
	}

	price, err := catalogconvert.ToAPIRateCardUsageBasedPrice(*line.UsageBased.Price)
	if err != nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map price: %w", err)
	}

	children, err := mapDetailedLinesToAPI(line.DetailedLines, line.Currency, line.InvoiceAt, line.TaxConfig, taxCodes)
	if err != nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map children: %w", err)
	}

	discountsAPI, err := mapDiscountsToAPI(line.Discounts)
	if err != nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map discounts: %w", err)
	}

	lineTaxConfig, err := ToAPITaxConfig(taxCodes, line.TaxConfig)
	if err != nil {
		return apiv1models.InvoiceLine{}, fmt.Errorf("failed to map tax config: %w", err)
	}

	invoiceLine := apiv1models.InvoiceLine{
		Type: apiv1models.InvoiceLineTypeUsageBased,
		Id:   line.ID,

		CreatedAt: line.CreatedAt,
		DeletedAt: line.DeletedAt,
		UpdatedAt: line.UpdatedAt,
		InvoiceAt: line.InvoiceAt,

		// TODO: deprecation
		Currency: string(line.Currency),
		Status:   apiv1models.InvoiceLineStatusValid,

		Description: line.Description,
		Name:        line.Name,
		ManagedBy:   apiv1models.InvoiceLineManagedBy(line.ManagedBy),

		// TODO: deprecation
		Invoice: &apiv1models.InvoiceReference{
			Id: line.InvoiceID,
		},

		Metadata: convert.MapToPointer(line.Metadata),
		Period: apiv1models.Period{
			From: line.Period.From,
			To:   line.Period.To,
		},

		TaxConfig: lineTaxConfig,

		FeatureKey:                   lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		MeteredQuantity:              decimalPtrToStringPtrIfNotEqual(line.UsageBased.MeteredQuantity, line.UsageBased.Quantity),
		Quantity:                     decimalPtrToStringPtr(line.UsageBased.Quantity),
		PreLinePeriodQuantity:        decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.PreLinePeriodQuantity),
		MeteredPreLinePeriodQuantity: decimalPtrToStringPtrIgnoringZeroValue(line.UsageBased.MeteredPreLinePeriodQuantity),

		Price: lo.ToPtr(price),

		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			TaxConfig:  lineTaxConfig,
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

func ToAPIInvoiceLineSubscriptionReference(optSub *billing.SubscriptionReference) *apiv1models.InvoiceLineSubscriptionReference {
	if optSub == nil {
		return nil
	}

	out := &apiv1models.InvoiceLineSubscriptionReference{
		Item: apiv1models.IDResource{
			Id: optSub.ItemID,
		},
		Phase: apiv1models.IDResource{
			Id: optSub.PhaseID,
		},
		Subscription: apiv1models.IDResource{
			Id: optSub.SubscriptionID,
		},
	}

	out.BillingPeriod = apiv1models.Period{
		From: optSub.BillingPeriod.From,
		To:   optSub.BillingPeriod.To,
	}

	return out
}

func mapInvoiceLineCreditsToAPI(credits billing.CreditsApplied) *[]apiv1models.InvoiceLineCreditAllocation {
	if len(credits) == 0 {
		return nil
	}

	out := lo.Map(credits, func(credit billing.CreditApplied, _ int) apiv1models.InvoiceLineCreditAllocation {
		return apiv1models.InvoiceLineCreditAllocation{
			Amount:      credit.Amount.String(),
			Description: lo.EmptyableToPtr(credit.Description),
		}
	})

	return &out
}

func mapDiscountsToAPI(discounts billing.StandardLineDiscounts) (*apiv1models.InvoiceLineDiscounts, error) {
	if len(discounts.Usage) == 0 {
		return nil, nil
	}

	mappedUsageDiscounts, err := mapInvoiceLineUsageDiscountsToAPI(discounts.Usage)
	if err != nil {
		return nil, fmt.Errorf("failed to map usage discounts: %w", err)
	}

	return &apiv1models.InvoiceLineDiscounts{
		Usage: mappedUsageDiscounts,
	}, nil
}

func mapInvoiceLineAmountDiscountsToAPI(amountDiscounts billing.AmountLineDiscountsManaged) (*[]apiv1models.InvoiceLineAmountDiscount, error) {
	if len(amountDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(amountDiscounts, func(discount billing.AmountLineDiscountManaged) (apiv1models.InvoiceLineAmountDiscount, error) {
		reason, err := mapDiscountReasonToAPI(discount.Reason)
		if err != nil {
			return apiv1models.InvoiceLineAmountDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
		}

		return apiv1models.InvoiceLineAmountDiscount{
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

func mapInvoiceLineUsageDiscountsToAPI(usageDiscounts billing.UsageLineDiscountsManaged) (*[]apiv1models.InvoiceLineUsageDiscount, error) {
	if len(usageDiscounts) == 0 {
		return nil, nil
	}

	mapped, err := slicesx.MapWithErr(usageDiscounts, func(discount billing.UsageLineDiscountManaged) (apiv1models.InvoiceLineUsageDiscount, error) {
		reason, err := mapDiscountReasonToAPI(discount.Reason)
		if err != nil {
			return apiv1models.InvoiceLineUsageDiscount{}, fmt.Errorf("failed to map discount reason: %w", err)
		}

		return apiv1models.InvoiceLineUsageDiscount{
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

func mapDiscountReasonToAPI(reason billing.DiscountReason) (apiv1models.BillingDiscountReason, error) {
	out := apiv1models.BillingDiscountReason{}
	switch reason.Type() {
	case billing.MaximumSpendDiscountReason:
		if err := out.FromDiscountReasonMaximumSpend(apiv1models.DiscountReasonMaximumSpend{
			Type: apiv1models.DiscountReasonMaximumSpendTypeMaximumSpend,
		}); err != nil {
			return out, err
		}
	case billing.RatecardPercentageDiscountReason:
		reason, err := reason.AsRatecardPercentage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardPercentage(apiv1models.DiscountReasonRatecardPercentage{
			Percentage:    reason.Percentage,
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          apiv1models.DiscountReasonRatecardPercentageType(billing.RatecardPercentageDiscountReason),
		}); err != nil {
			return out, err
		}
	case billing.RatecardUsageDiscountReason:
		reason, err := reason.AsRatecardUsage()
		if err != nil {
			return out, err
		}

		if err := out.FromDiscountReasonRatecardUsage(apiv1models.DiscountReasonRatecardUsage{
			Quantity:      reason.Quantity.String(),
			CorrelationId: lo.EmptyableToPtr(reason.CorrelationID),
			Type:          apiv1models.DiscountReasonRatecardUsageType(billing.RatecardUsageDiscountReason),
		}); err != nil {
			return out, err
		}
	default:
		return apiv1models.BillingDiscountReason{}, fmt.Errorf("unknown discount reason type: %s", reason.Type())
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

// ToAPITaxConfig renders a line's tax config in the v1 shape. stripe.code comes from the
// referenced Tax Code's current Stripe mapping, not the code the line was created with, so a
// line renders what the Tax Code maps to today. taxCodes must hold the config's referenced
// Tax Code. A nil config renders no tax config.
func ToAPITaxConfig(taxCodes productcatalog.TaxCodesByID, cfg *productcatalog.TaxCodeConfig) (*apiv1models.TaxConfig, error) {
	if cfg == nil {
		return nil, nil
	}

	out, err := productcataloghttp.ToAPITaxConfig(*cfg, taxCodes)
	if err != nil {
		return nil, err
	}

	return &out, nil
}
