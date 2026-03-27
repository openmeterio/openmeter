package targetstate

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

type StateItem struct {
	SubscriptionItemWithPeriods

	Subscription       subscription.Subscription
	CurrencyCalculator currencyx.Calculator
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

func (r StateItem) GetExpectedLine() (*billing.GatheringLine, error) {
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   r.Subscription.Namespace,
				Name:        r.Spec.RateCard.AsMeta().Name,
				Description: r.Spec.RateCard.AsMeta().Description,
			}),
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               r.Subscription.Currency,
			ChildUniqueReferenceID: &r.UniqueID,
			TaxConfig:              r.Spec.RateCard.AsMeta().TaxConfig,
			ServicePeriod:          r.ServicePeriod.ToClosedPeriod(),
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

	if price := r.Spec.RateCard.AsMeta().Price; price != nil && price.GetPaymentTerm() == productcatalog.InArrearsPaymentTerm {
		if r.FullServicePeriod.Duration() == time.Duration(0) {
			return nil, nil
		}
	}

	switch r.Spec.RateCard.AsMeta().Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := r.Spec.RateCard.AsMeta().Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		perUnitAmount := r.CurrencyCalculator.RoundToPrecision(price.Amount)
		if !r.ServicePeriod.IsEmpty() && shouldProrate(r, r.Subscription) {
			perUnitAmount = r.CurrencyCalculator.RoundToPrecision(price.Amount.Mul(r.PeriodPercentage()))
		}

		if perUnitAmount.IsZero() {
			return nil, nil
		}

		line.Price = lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      perUnitAmount,
			PaymentTerm: price.PaymentTerm,
		}))
		line.FeatureKey = lo.FromPtr(r.Spec.RateCard.AsMeta().FeatureKey)
	default:
		if r.Spec.RateCard.AsMeta().Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.Price = lo.FromPtr(r.Spec.RateCard.AsMeta().Price)
		line.FeatureKey = lo.FromPtr(r.Spec.RateCard.AsMeta().FeatureKey)
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

func shouldProrate(item StateItem, subs subscription.Subscription) bool {
	if !subs.ProRatingConfig.Enabled {
		return false
	}

	if item.Spec.RateCard.AsMeta().Price.Type() != productcatalog.FlatPriceType {
		return false
	}

	if subs.ActiveTo != nil && !subs.ActiveTo.After(item.ServicePeriod.End) {
		return false
	}

	switch subs.ProRatingConfig.Mode {
	case productcatalog.ProRatingModeProratePrices:
		return true
	default:
		return false
	}
}
