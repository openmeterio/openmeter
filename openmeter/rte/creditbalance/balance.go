package creditbalance

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Key identifies a current balance, not a historical query.
type Key struct {
	CustomerID    customer.CustomerID
	Currency      currencies.CurrencyReference
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
}

func (k Key) Validate() error {
	var errs []error

	if err := k.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := k.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if k.Currency.IsCustom() && (k.Currency.CustomCurrencyID == nil || *k.Currency.CustomCurrencyID == "") {
		errs = append(errs, errors.New("custom currency ID is required"))
	}

	if err := customerbalance.ValidateFeatureFilter(k.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// FeatureScope preserves the difference between the whole portfolio,
// unrestricted credits, and credits eligible for one feature.
func (k Key) FeatureScope() string {
	if k.FeatureFilter.IsAbsent() {
		return "all"
	}

	features := k.FeatureFilter.OrEmpty().Normalize()
	if len(features) == 0 {
		return "unrestricted"
	}

	return "feature:" + features[0]
}

type Balance struct {
	Booked    alpacadecimal.Decimal `json:"booked"`
	Spendable alpacadecimal.Decimal `json:"spendable"`
	Live      alpacadecimal.Decimal `json:"live"`
	Pending   alpacadecimal.Decimal `json:"pending"`
}
