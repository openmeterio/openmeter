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

	Discounts billing.LineDiscounts `json:"discounts,omitempty"`
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
		Currency:      currency,
		PerUnitAmount: i.PerUnitAmount,
		Quantity:      i.Quantity,
		Discounts:     i.Discounts,
	})
}

type getTotalAmountInput struct {
	Currency      currencyx.Calculator
	PerUnitAmount alpacadecimal.Decimal
	Quantity      alpacadecimal.Decimal
	Discounts     billing.LineDiscounts
}

func TotalAmount(in getTotalAmountInput) alpacadecimal.Decimal {
	total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))

	total = total.Sub(in.Discounts.Amount.SumAmount(in.Currency))

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
		i.Discounts.Amount = append(i.Discounts.Amount, billing.AmountLineDiscountManaged{
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
	i.Discounts.Amount = append(i.Discounts.Amount, billing.AmountLineDiscountManaged{
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

func newDetailedLines(line *billing.Line, inputs ...newDetailedLineInput) ([]*billing.Line, error) {
	return slicesx.MapWithErr(inputs, func(in newDetailedLineInput) (*billing.Line, error) {
		if err := in.Validate(); err != nil {
			return nil, err
		}

		period := line.Period
		if in.Period != nil {
			period = *in.Period
		}

		if in.Category == "" {
			in.Category = billing.FlatFeeCategoryRegular
		}

		line := &billing.Line{
			LineBase: billing.LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: line.Namespace,
					Name:      in.Name,
				}),
				Type:                   billing.InvoiceLineTypeFee,
				Status:                 billing.InvoiceLineStatusDetailed,
				Period:                 period,
				ManagedBy:              billing.SystemManagedLine,
				InvoiceAt:              line.InvoiceAt,
				InvoiceID:              line.InvoiceID,
				Currency:               line.Currency,
				ChildUniqueReferenceID: &in.ChildUniqueReferenceID,
				ParentLineID:           lo.ToPtr(line.ID),
				TaxConfig:              line.TaxConfig,
			},
			FlatFee: &billing.FlatFeeLine{
				PaymentTerm:   lo.CoalesceOrEmpty(in.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				PerUnitAmount: in.PerUnitAmount,
				Quantity:      in.Quantity,
				Category:      in.Category,
			},
			Discounts: in.Discounts,
		}

		if err := line.Validate(); err != nil {
			return nil, err
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

func mergeDetailedLines(line *billing.Line, in newDetailedLinesInput) error {
	detailedLines, err := newDetailedLines(line, in...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	// The lines are generated in order, so we can just persist the index
	for idx := range detailedLines {
		detailedLines[idx].FlatFee.Index = lo.ToPtr(idx)
	}

	childrenWithIDReuse, err := line.ChildrenWithIDReuse(detailedLines)
	if err != nil {
		return fmt.Errorf("failed to reuse child IDs: %w", err)
	}

	line.Children = childrenWithIDReuse

	return nil
}
