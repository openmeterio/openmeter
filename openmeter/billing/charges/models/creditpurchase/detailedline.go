package creditpurchase

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const CreditPurchaseChildUniqueReferenceID = "credit-purchase"

type newDetailedLineInput struct {
	Namespace     string
	InvoiceID     string
	Name          string
	ServicePeriod timeutil.ClosedPeriod

	CreditCurrency    currencies.Currency
	CreditAmount      alpacadecimal.Decimal
	ResolvedCostBasis alpacadecimal.Decimal

	FiatCurrency *currencyx.FiatCurrency
	FiatAmount   alpacadecimal.Decimal
}

func (i newDetailedLineInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice ID is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := i.CreditCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit currency: %w", err))
	}

	if i.CreditAmount.IsNegative() {
		errs = append(errs, errors.New("credit amount must be positive or zero"))
	}

	if !i.ResolvedCostBasis.IsPositive() {
		errs = append(errs, errors.New("resolved cost basis must be positive"))
	}

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if i.FiatAmount.IsNegative() {
		errs = append(errs, errors.New("fiat amount must be positive or zero"))
	}

	if i.FiatCurrency != nil && !i.FiatCurrency.IsRoundedToPrecision(i.FiatAmount) {
		errs = append(errs, errors.New("fiat amount must be rounded to fiat currency precision"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// newDetailedLine represents purchased credit as a fiat invoice line. The
// quantity preserves the credit amount, the unit amount preserves the exact
// resolved cost basis, and totals preserve the already-rounded fiat outcome.
func newDetailedLine(input newDetailedLineInput) (billing.DetailedLine, error) {
	if err := input.Validate(); err != nil {
		return billing.DetailedLine{}, err
	}

	detailedLine := billing.DetailedLine{
		DetailedLineBase: billing.DetailedLineBase{
			Base: stddetailedline.Base{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: input.Namespace,
					Name:      input.Name,
				}),
				Category:               stddetailedline.CategoryRegular,
				ChildUniqueReferenceID: CreditPurchaseChildUniqueReferenceID,
				Index:                  lo.ToPtr(0),
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				ServicePeriod:          input.ServicePeriod,
				PerUnitAmount:          input.ResolvedCostBasis,
				Quantity:               input.CreditCurrency.RoundToPrecision(input.CreditAmount),
				Totals: totals.Totals{
					Amount: input.FiatAmount,
					Total:  input.FiatAmount,
				},
			},
			InvoiceID: input.InvoiceID,
		},
	}

	if err := detailedLine.Validate(); err != nil {
		return billing.DetailedLine{}, fmt.Errorf("detailed line: %w", err)
	}

	if err := detailedLine.Totals.Validate(); err != nil {
		return billing.DetailedLine{}, fmt.Errorf("totals: %w", err)
	}

	calculatedAmount := input.FiatCurrency.RoundToPrecision(
		detailedLine.Quantity.Mul(detailedLine.PerUnitAmount),
	)
	if !detailedLine.Totals.Amount.Equal(calculatedAmount) {
		return billing.DetailedLine{}, fmt.Errorf(
			"totals amount does not match quantity and cost basis: expected %s, got %s",
			calculatedAmount,
			detailedLine.Totals.Amount,
		)
	}

	calculatedTotal := detailedLine.Totals.CalculateTotal()
	if !detailedLine.Totals.Total.Equal(calculatedTotal) {
		return billing.DetailedLine{}, fmt.Errorf(
			"totals total does not match its components: expected %s, got %s",
			calculatedTotal,
			detailedLine.Totals.Total,
		)
	}

	return detailedLine, nil
}

type WithDetailedLinesInput struct {
	Line *billing.StandardLine
	Name string

	CreditCurrency currencies.Currency
	CreditAmount   alpacadecimal.Decimal

	ResolvedCostBasis alpacadecimal.Decimal
	FiatCurrency      *currencyx.FiatCurrency
	FiatAmount        alpacadecimal.Decimal
}

func (i WithDetailedLinesInput) Validate() error {
	var errs []error

	if i.Line == nil {
		errs = append(errs, errors.New("line is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.CreditCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit currency: %w", err))
	}

	if i.CreditAmount.IsNegative() {
		errs = append(errs, errors.New("credit amount must be positive or zero"))
	}

	if !i.ResolvedCostBasis.IsPositive() {
		errs = append(errs, errors.New("resolved cost basis must be positive"))
	}

	if err := i.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if i.FiatAmount.IsNegative() {
		errs = append(errs, errors.New("fiat amount must be positive or zero"))
	}

	if i.FiatCurrency != nil && !i.FiatCurrency.IsRoundedToPrecision(i.FiatAmount) {
		errs = append(errs, errors.New("fiat amount must be rounded to fiat currency precision"))
	}

	if i.Line != nil && i.FiatCurrency != nil && i.Line.Currency != i.FiatCurrency.GetFiatCode() {
		errs = append(errs, fmt.Errorf(
			"line currency %s does not match fiat currency %s",
			i.Line.Currency,
			i.FiatCurrency.GetFiatCode(),
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// WithDetailedLines returns a cloned invoice line populated from an
// already-resolved credit valuation while preserving detailed-line identity.
func WithDetailedLines(input WithDetailedLinesInput) (*billing.StandardLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	line, err := input.Line.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning standard line: %w", err)
	}

	detailedLine, err := newDetailedLine(newDetailedLineInput{
		Namespace:         line.Namespace,
		InvoiceID:         line.InvoiceID,
		Name:              input.Name,
		ServicePeriod:     line.Period,
		CreditCurrency:    input.CreditCurrency,
		CreditAmount:      input.CreditAmount,
		ResolvedCostBasis: input.ResolvedCostBasis,
		FiatCurrency:      input.FiatCurrency,
		FiatAmount:        input.FiatAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("creating credit purchase detailed line: %w", err)
	}

	line.DetailedLines = line.DetailedLinesWithIDReuse(billing.DetailedLines{detailedLine})
	line.Totals = line.DetailedLines.SumTotals().RoundToPrecision(input.FiatCurrency)

	if !line.Totals.Total.Equal(input.FiatAmount) {
		return nil, fmt.Errorf(
			"line total does not match fiat amount: %s != %s",
			line.Totals.Total,
			input.FiatAmount,
		)
	}

	return line, nil
}
