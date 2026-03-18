package pricer

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type DetailedLine struct {
	Name                   string                 `json:"name"`
	Quantity               alpacadecimal.Decimal  `json:"quantity"`
	PerUnitAmount          alpacadecimal.Decimal  `json:"perUnitAmount"`
	ChildUniqueReferenceID string                 `json:"childUniqueReferenceID"`
	Period                 *timeutil.ClosedPeriod `json:"period,omitempty"`
	// PaymentTerm is the payment term for the detailed line, defaults to arrears
	PaymentTerm productcatalog.PaymentTermType `json:"paymentTerm,omitempty"`
	Category    billing.FlatFeeCategory        `json:"category,omitempty"`

	AmountDiscounts billing.AmountLineDiscountsManaged `json:"amountDiscounts,omitempty"`
	CreditsApplied  billing.CreditsApplied             `json:"creditsApplied,omitempty"`

	Totals totals.Totals `json:"totals,omitempty"`
}

func (i DetailedLine) Validate() error {
	if i.Quantity.IsNegative() {
		return fmt.Errorf("quantity must be zero or positive")
	}

	if i.PerUnitAmount.IsNegative() {
		return fmt.Errorf("amount must be zero or positive")
	}

	if i.ChildUniqueReferenceID == "" {
		return fmt.Errorf("child unique ID is required")
	}

	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

func (i DetailedLine) TotalAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
	return TotalAmount(getTotalAmountInput{
		Currency:        currency,
		PerUnitAmount:   i.PerUnitAmount,
		Quantity:        i.Quantity,
		AmountDiscounts: i.AmountDiscounts,
		CreditsApplied:  i.CreditsApplied,
	})
}

type getTotalAmountInput struct {
	Currency        currencyx.Calculator
	PerUnitAmount   alpacadecimal.Decimal
	Quantity        alpacadecimal.Decimal
	AmountDiscounts billing.AmountLineDiscountsManaged
	CreditsApplied  billing.CreditsApplied
}

func TotalAmount(in getTotalAmountInput) alpacadecimal.Decimal {
	total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))

	total = total.Sub(in.AmountDiscounts.SumAmount(in.Currency))

	total = total.Sub(in.CreditsApplied.SumAmount(in.Currency))

	return total
}

type AddDiscountInput struct {
	BilledAmountBeforeLine alpacadecimal.Decimal
	MaxSpend               alpacadecimal.Decimal
	Currency               currencyx.Calculator
}

func (i DetailedLine) AddDiscountForOverage(in AddDiscountInput) DetailedLine {
	normalizedPreUsage := in.Currency.RoundToPrecision(in.BilledAmountBeforeLine)

	lineTotal := i.TotalAmount(in.Currency)

	totalBillableAmount := normalizedPreUsage.Add(lineTotal)

	normalizedMaxSpend := in.Currency.RoundToPrecision(in.MaxSpend)

	if totalBillableAmount.LessThanOrEqual(normalizedMaxSpend) {
		// Nothing to do here
		return i
	}

	if totalBillableAmount.GreaterThanOrEqual(normalizedMaxSpend) && in.BilledAmountBeforeLine.GreaterThanOrEqual(normalizedMaxSpend) {
		// 100% discount
		i.AmountDiscounts = append(i.AmountDiscounts, billing.AmountLineDiscountManaged{
			AmountLineDiscount: billing.AmountLineDiscount{
				Amount: lineTotal,
				LineDiscountBase: billing.LineDiscountBase{
					Description:            formatMaximumSpendDiscountDescription(normalizedMaxSpend),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
				},
			},
		})
		return i
	}

	discountAmount := totalBillableAmount.Sub(normalizedMaxSpend)
	i.AmountDiscounts = append(i.AmountDiscounts, billing.AmountLineDiscountManaged{
		AmountLineDiscount: billing.AmountLineDiscount{
			Amount: discountAmount,
			LineDiscountBase: billing.LineDiscountBase{
				Description:            formatMaximumSpendDiscountDescription(normalizedMaxSpend),
				ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
				Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
			},
		},
	})

	return i
}

func formatMaximumSpendDiscountDescription(amount alpacadecimal.Decimal) *string {
	// TODO[OM-1019]: currency formatting!
	return lo.ToPtr(fmt.Sprintf("Maximum spend discount for charges over %s", amount))
}

type DetailedLines []DetailedLine

func (i DetailedLines) Sum(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero

	for _, in := range i {
		sum = sum.Add(in.TotalAmount(currency))
	}

	return sum
}
