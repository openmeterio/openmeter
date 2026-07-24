package creditpurchase

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const CreditPurchaseChildUniqueReferenceID = "credit-purchase"

type NewDetailedLineInput struct {
	Namespace     string
	InvoiceID     string
	Name          string
	ServicePeriod timeutil.ClosedPeriod

	CustomCurrency       currencies.Currency
	CustomCurrencyAmount alpacadecimal.Decimal
	ResolvedCostBasis    *costbasis.State

	FiatCurrency *currencyx.FiatCurrency
	FiatAmount   alpacadecimal.Decimal
}

func (i NewDetailedLineInput) Validate() error {
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

	if err := i.CustomCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("custom currency: %w", err))
	}

	if !i.CustomCurrency.IsCustom() {
		errs = append(errs, errors.New("custom currency must be custom"))
	}

	if i.CustomCurrencyAmount.IsNegative() {
		errs = append(errs, errors.New("custom currency amount must be positive or zero"))
	}

	if i.ResolvedCostBasis == nil {
		errs = append(errs, errors.New("resolved cost basis is required"))
	} else if err := i.ResolvedCostBasis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("resolved cost basis: %w", err))
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

// NewDetailedLine represents a custom-currency purchase as a fiat invoice
// line. The quantity preserves the custom-currency amount, the unit amount
// preserves the exact cost basis, and totals preserve the already-rounded
// fiat outcome.
func NewDetailedLine(input NewDetailedLineInput) (billing.DetailedLine, error) {
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
				PerUnitAmount:          input.ResolvedCostBasis.CostBasis,
				Quantity:               input.CustomCurrency.RoundToPrecision(input.CustomCurrencyAmount),
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
