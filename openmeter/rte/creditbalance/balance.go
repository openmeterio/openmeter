package creditbalance

import (
	"encoding/json"
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
	CustomerID    customer.CustomerID                      `json:"customerId"`
	Currency      currencies.CurrencyReference             `json:"currency"`
	FeatureFilter mo.Option[creditpurchase.FeatureFilters] `json:"featureFilter"`
}

func (k Key) MarshalJSON() ([]byte, error) {
	// A present empty filter means unrestricted credits. Encode it as [] so it
	// survives a round-trip separately from the absent filter's null (all credits).
	if k.FeatureFilter.IsPresent() && len(k.FeatureFilter.OrEmpty()) == 0 {
		k.FeatureFilter = mo.Some(creditpurchase.FeatureFilters{})
	}

	type wireKey Key

	return json.Marshal(wireKey(k))
}

func (k *Key) UnmarshalJSON(data []byte) error {
	type wireKey Key
	var decoded wireKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	// The calculator represents unrestricted credits as Some(nil), not Some([]).
	if decoded.FeatureFilter.IsPresent() && len(decoded.FeatureFilter.OrEmpty()) == 0 {
		decoded.FeatureFilter = customerbalance.NewUnrestrictedFeatureFilter()
	}

	*k = Key(decoded)

	return nil
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
