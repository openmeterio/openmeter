package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type newDetailedLineInput struct {
	Name                   string                `json:"name"`
	Quantity               alpacadecimal.Decimal `json:"quantity"`
	PerUnitAmount          alpacadecimal.Decimal `json:"perUnitAmount"`
	ChildUniqueReferenceID string                `json:"childUniqueReferenceID"`
	Period                 *billing.Period       `json:"period,omitempty"`
	// PaymentTerm is the payment term for the detailed line, defaults to arrears
	PaymentTerm productcatalog.PaymentTermType `json:"paymentTerm,omitempty"`
	Category    billing.FlatFeeCategory        `json:"category,omitempty"`

	AmountDiscounts billing.AmountLineDiscountsManaged `json:"amountDiscounts,omitempty"`
	CreditsApplied  billing.CreditsApplied             `json:"creditsApplied,omitempty"`
}

func (i newDetailedLineInput) Validate() error {
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

func (i newDetailedLineInput) TotalAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
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

type addDiscountInput struct {
	BilledAmountBeforeLine alpacadecimal.Decimal
	MaxSpend               alpacadecimal.Decimal
	Currency               currencyx.Calculator
}

func (i newDetailedLineInput) AddDiscountForOverage(in addDiscountInput) newDetailedLineInput {
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

func newDetailedLines(line *billing.StandardLine, inputs ...newDetailedLineInput) (billing.DetailedLines, error) {
	return slicesx.MapWithErr(inputs, func(in newDetailedLineInput) (billing.DetailedLine, error) {
		if err := in.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		period := line.Period
		if in.Period != nil {
			period = *in.Period
		}

		if in.Category == "" {
			in.Category = billing.FlatFeeCategoryRegular
		}

		line := billing.DetailedLine{
			DetailedLineBase: billing.DetailedLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: line.Namespace,
					Name:      in.Name,
				}),

				ServicePeriod:          period,
				InvoiceID:              line.InvoiceID,
				Currency:               line.Currency,
				ChildUniqueReferenceID: &in.ChildUniqueReferenceID,
				TaxConfig:              line.TaxConfig,

				PaymentTerm:   lo.CoalesceOrEmpty(in.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				PerUnitAmount: in.PerUnitAmount,
				Quantity:      in.Quantity,
				Category:      in.Category,
			},
			AmountDiscounts: in.AmountDiscounts,
		}

		if err := line.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		return line, nil
	})
}

type newDetailedLinesInput []newDetailedLineInput

func (i newDetailedLinesInput) Sum(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero

	for _, in := range i {
		sum = sum.Add(in.TotalAmount(currency))
	}

	return sum
}

func mergeDetailedLines(parentLine *billing.StandardLine, in newDetailedLinesInput) error {
	detailedLines, err := newDetailedLines(parentLine, in...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	// The lines are generated in order, so we can just persist the index
	for idx := range detailedLines {
		detailedLines[idx].Index = lo.ToPtr(idx)
	}

	parentLine.DetailedLines = parentLine.DetailedLinesWithIDReuse(detailedLines)

	return nil
}

func calculateDetailedLineTotals(line billing.DetailedLine) (billing.Totals, error) {
	// Calculate the line totals
	calc, err := line.Currency.Calculator()
	if err != nil {
		return billing.Totals{}, err
	}

	// Calculate the line totals
	totals := billing.Totals{
		DiscountsTotal: line.AmountDiscounts.SumAmount(calc),
		CreditsTotal:   line.CreditsApplied.SumAmount(calc),

		// TODO[OM-979]: implement taxes
		TaxesInclusiveTotal: alpacadecimal.Zero,
		TaxesExclusiveTotal: alpacadecimal.Zero,
		TaxesTotal:          alpacadecimal.Zero,
	}

	amount := calc.RoundToPrecision(line.PerUnitAmount.Mul(line.Quantity))

	switch line.Category {
	case billing.FlatFeeCategoryCommitment:
		totals.ChargesTotal = amount
	default:
		totals.Amount = amount
	}

	totals.Total = totals.CalculateTotal()

	return totals, nil
}
