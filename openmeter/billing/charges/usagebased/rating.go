package usagebased

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ rating.StandardLineAccessor = (*RateableIntent)(nil)

type RateableIntent struct {
	Intent

	ServicePeriod  timeutil.ClosedPeriod
	MeterValue     alpacadecimal.Decimal
	CreditsApplied billing.CreditsApplied
}

func (r RateableIntent) GetMeteredQuantity() (*alpacadecimal.Decimal, error) {
	return lo.ToPtr(r.MeterValue), nil
}

func (r RateableIntent) GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error) {
	return lo.ToPtr(alpacadecimal.Zero), nil
}

func (r RateableIntent) GetPrice() *productcatalog.Price {
	return r.Price.Clone()
}

func (r RateableIntent) GetServicePeriod() timeutil.ClosedPeriod {
	return r.ServicePeriod
}

func (r RateableIntent) GetFeatureKey() string {
	return r.FeatureKey
}

func (r RateableIntent) GetCurrencyCalculator() (currencyx.Currency, error) {
	return r.Intent.Intent.Currency, nil
}

func (r RateableIntent) GetName() string {
	return r.Name
}

func (r RateableIntent) GetRateCardDiscounts() billing.Discounts {
	return r.Discounts.Clone()
}

func (r RateableIntent) GetUnitConfig() *productcatalog.UnitConfig {
	if r.UnitConfig == nil {
		return nil
	}

	return lo.ToPtr(r.UnitConfig.Clone())
}

func (r RateableIntent) GetStandardLineDiscounts() billing.StandardLineDiscounts {
	// A charge is never associated with user defined line discounts
	return billing.StandardLineDiscounts{}
}

func (r RateableIntent) IsProgressivelyBilled() bool {
	// A charge is never progressively billed
	return false
}

func (r RateableIntent) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	return r.ServicePeriod, nil
}

func (r RateableIntent) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}

func (r RateableIntent) GetCreditsApplied() billing.CreditsApplied {
	return r.CreditsApplied
}
