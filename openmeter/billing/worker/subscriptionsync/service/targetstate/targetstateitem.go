package targetstate

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type StateItem struct {
	SubscriptionItemWithPeriods

	CurrencyCalculator currencyx.Calculator
	Subscription       subscription.Subscription
}

// IsBillable returns true if the item is billable (e.g. we can create a gathering line for it even if it's value is 0 or create a charge for it)
//
// Note: GetExpectedLine might return nil in other cases, e.g. we don't want to create a flat fee line when pro-rating is in effect and the service period
// and the currency rounding results in a 0 amount.
func (r StateItem) IsBillable() bool {
	// If the rate card has no price, it is not billable
	price := r.Spec.RateCard.AsMeta().Price
	if price == nil {
		return false
	}

	// If the subscription item is an arrears item billed once at the end of the phase (flat fee only), it is not billable until
	// the phase has an activeTo time set.
	if price := r.Spec.RateCard.AsMeta().Price; price != nil && price.GetPaymentTerm() == productcatalog.InArrearsPaymentTerm {
		if r.FullServicePeriod.Duration() == time.Duration(0) {
			return false
		}
	}

	return true
}

func (r StateItem) GetServicePeriod() timeutil.ClosedPeriod {
	return r.ServicePeriod.ToClosedPeriod()
}

func (r StateItem) GetExpectedLine() (*billing.GatheringLine, error) {
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   r.Subscription.Namespace,
				Name:        r.Spec.RateCard.AsMeta().Name,
				Description: r.Spec.RateCard.AsMeta().Description,
			}),
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               r.CurrencyCalculator.Currency,
			ChildUniqueReferenceID: &r.UniqueID,
			TaxConfig:              r.Spec.RateCard.AsMeta().TaxConfig,
			ServicePeriod:          r.GetServicePeriod(),
			InvoiceAt:              r.GetInvoiceAt(),
			RateCardDiscounts:      discountsToBillingDiscounts(r.Spec.RateCard.AsMeta().Discounts),
			Subscription: &billing.SubscriptionReference{
				SubscriptionID: r.Subscription.ID,
				PhaseID:        r.PhaseID,
				ItemID:         r.SubscriptionItem.ID,
				BillingPeriod: timeutil.ClosedPeriod{
					From: r.BillingPeriod.Start,
					To:   r.BillingPeriod.End,
				},
			},
		},
	}

	price := r.Spec.RateCard.AsMeta().Price
	if price != nil && price.GetPaymentTerm() == productcatalog.InArrearsPaymentTerm {
		if r.FullServicePeriod.Duration() == time.Duration(0) {
			return nil, nil
		}
	}

	if price == nil {
		return nil, fmt.Errorf("price must be defined for usage based price")
	}

	switch price.Type() {
	case productcatalog.FlatPriceType:
		price, err := price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		perUnitAmount := r.CurrencyCalculator.RoundToPrecision(price.Amount)
		if !r.ServicePeriod.IsEmpty() && r.shouldProrate() {
			perUnitAmount = r.CurrencyCalculator.RoundToPrecision(price.Amount.Mul(r.PeriodPercentage()))
		}

		if perUnitAmount.IsZero() {
			return nil, nil
		}

		line.Price = lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      perUnitAmount,
			PaymentTerm: price.PaymentTerm,
		}))
		line.FeatureKey = lo.FromPtr(r.SubscriptionItem.RateCard.AsMeta().FeatureKey)
	default:
		if r.SubscriptionItem.RateCard.AsMeta().Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.Price = lo.FromPtr(r.SubscriptionItem.RateCard.AsMeta().Price)
		line.FeatureKey = lo.FromPtr(r.SubscriptionItem.RateCard.AsMeta().FeatureKey)
	}

	return &line, nil
}

func discountsToBillingDiscounts(discounts productcatalog.Discounts) billing.Discounts {
	out := billing.Discounts{}

	if discounts.Usage != nil {
		out.Usage = &billing.UsageDiscount{UsageDiscount: *discounts.Usage}
	}

	if discounts.Percentage != nil {
		out.Percentage = &billing.PercentageDiscount{PercentageDiscount: *discounts.Percentage}
	}

	return out
}

func (r StateItem) shouldProrate() bool {
	if !r.Subscription.ProRatingConfig.Enabled {
		return false
	}

	if r.SubscriptionItem.RateCard.AsMeta().Price.Type() != productcatalog.FlatPriceType {
		return false
	}

	if r.Subscription.ActiveTo != nil && !r.Subscription.ActiveTo.After(r.ServicePeriod.End) {
		return false
	}

	switch r.Subscription.ProRatingConfig.Mode {
	case productcatalog.ProRatingModeProratePrices:
		return true
	default:
		return false
	}
}

var ErrExpectedLineIsEmpty = errors.New("expected line is empty")

func (r StateItem) GetExpectedLineOrErr() (billing.GatheringLine, error) {
	line, err := r.GetExpectedLine()
	if err != nil {
		return billing.GatheringLine{}, err
	}

	if line == nil {
		return billing.GatheringLine{}, fmt.Errorf("%w [child_unique_id: %s]", ErrExpectedLineIsEmpty, r.UniqueID)
	}

	return *line, nil
}
