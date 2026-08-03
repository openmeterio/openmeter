package flatfee

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ billingrating.StandardLineAccessor = (*RateableIntent)(nil)

// RateableIntent is the effective charge intent for one flat-fee realization
// period. It keeps charge valuation independent from the invoice line that may
// eventually represent the realized amount.
type RateableIntent struct {
	Intent

	AmountAfterProration alpacadecimal.Decimal
}

func (r RateableIntent) Validate() error {
	var errs []error

	if err := r.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if r.AmountAfterProration.IsNegative() {
		errs = append(errs, errors.New("amount after proration cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetRateableIntent returns the normalized effective intent used for rating.
// The effective intent is the sole source of both the realized service period
// and the full period used as the proration reference.
func (c Charge) GetRateableIntent() (RateableIntent, error) {
	intent := c.Intent.GetEffectiveIntent()
	intent = intent.Normalized()

	if err := intent.Validate(); err != nil {
		return RateableIntent{}, fmt.Errorf("validating effective intent: %w", err)
	}

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return RateableIntent{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	rateableIntent := RateableIntent{
		Intent:               intent,
		AmountAfterProration: intent.Currency.RoundToPrecision(amountAfterProration),
	}
	if err := rateableIntent.Validate(); err != nil {
		return RateableIntent{}, err
	}

	return rateableIntent, nil
}

func (r RateableIntent) GetMeteredQuantity() (*alpacadecimal.Decimal, error) {
	return nil, nil
}

func (r RateableIntent) GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error) {
	return nil, nil
}

func (r RateableIntent) GetPrice() *productcatalog.Price {
	return productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      r.AmountAfterProration,
		PaymentTerm: r.PaymentTerm,
	})
}

func (r RateableIntent) GetServicePeriod() timeutil.ClosedPeriod {
	return r.ServicePeriod
}

func (r RateableIntent) GetFeatureKey() string {
	if r.FeatureKey == nil {
		return ""
	}

	return *r.FeatureKey
}

func (r RateableIntent) GetCurrencyCalculator() (currencyx.Currency, error) {
	return r.Currency, nil
}

func (r RateableIntent) GetName() string {
	return r.Name
}

func (r RateableIntent) GetRateCardDiscounts() billing.Discounts {
	if r.PercentageDiscounts == nil {
		return billing.Discounts{}
	}

	return billing.Discounts{
		Percentage: r.PercentageDiscounts.CloneOrNil(),
	}
}

func (r RateableIntent) GetUnitConfig() *productcatalog.UnitConfig {
	return nil
}

func (r RateableIntent) GetStandardLineDiscounts() billing.StandardLineDiscounts {
	return billing.StandardLineDiscounts{}
}

func (r RateableIntent) IsProgressivelyBilled() bool {
	return false
}

func (r RateableIntent) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	return r.ServicePeriod, nil
}

func (r RateableIntent) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}

func (r RateableIntent) GetCreditsApplied() billing.CreditsApplied {
	return nil
}
