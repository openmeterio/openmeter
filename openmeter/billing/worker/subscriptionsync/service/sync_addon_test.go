package service

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// A seat add on a period that is already paid bills only the added seat for the rest of
// the period: the seats on the paid invoice are already covered until the period ends.
func (s *CreditThenInvoiceTestSuite) TestInAdvanceAddonQuantityIncreaseAfterPaidInvoiceBillsOnlyAddedSeatProrated() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	seatAddedAt := s.mustParseTime("2024-01-02T00:00:00Z")
	periodEnd := s.mustParseTime("2024-02-01T00:00:00Z")

	addonPublishedAt := start.Add(-time.Hour)

	clock.FreezeTime(addonPublishedAt)
	defer clock.UnFreeze()
	s.enableProrating()

	seatPrice := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(6),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	seatRateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:   "seats",
			Name:  "Seats",
			Price: seatPrice,
		},
		BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
	}

	flatAmount := func(price *productcatalog.Price) float64 {
		flat, err := price.AsFlat()
		s.Require().NoError(err)

		return flat.Amount.InexactFloat64()
	}

	var subsView subscription.SubscriptionView
	var extraSeatsID models.NamespacedID
	var paidInvoice billing.StandardInvoice

	s.Run("subscribe with two seats", func() {
		// given:
		// - a plan whose monthly in-advance fee is the first seat
		// - an add-on that adds the same fee for every extra seat
		seatPlan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Seat Plan",
					Key:            "seat-plan",
					Version:        1,
					Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{
					{
						PhaseMeta: s.phaseMeta("first-phase", ""),
						RateCards: productcatalog.RateCards{seatRateCard.Clone()},
					},
				},
			},
		})
		s.Require().NoError(err)

		extraSeats, err := s.AddonService.CreateAddon(ctx, addon.CreateAddonInput{
			NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
			Addon: productcatalog.Addon{
				AddonMeta: productcatalog.AddonMeta{
					Name:         "Extra Seats",
					Key:          "extra-seats",
					Version:      1,
					Currency:     currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
					InstanceType: productcatalog.AddonInstanceTypeMultiple,
				},
				RateCards: productcatalog.RateCards{seatRateCard.Clone()},
			},
		})
		s.Require().NoError(err)

		extraSeats, err = s.AddonService.PublishAddon(ctx, addon.PublishAddonInput{
			NamespacedID:    extraSeats.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(addonPublishedAt)},
		})
		s.Require().NoError(err)

		// An add-on counts as active only strictly after its effective date, and only an
		// active add-on can be linked to a plan.
		clock.FreezeTime(start)

		_, err = s.PlanAddonService.CreatePlanAddon(ctx, planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
			PlanID:          seatPlan.ID,
			AddonID:         extraSeats.ID,
			FromPlanPhase:   "first-phase",
		})
		s.Require().NoError(err)

		_, err = s.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
			NamespacedID:    seatPlan.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(start)},
		})
		s.Require().NoError(err)

		subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
			Key:     seatPlan.Key,
			Version: lo.ToPtr(1),
		})
		s.Require().NoError(err)

		// when:
		// - the customer subscribes and takes one extra seat from the start
		subsView, err = s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: lo.ToPtr(start)},
				Name:   "seat-subscription",
			},
			Namespace:  s.Namespace,
			CustomerID: s.Customer.ID,
		}, subscriptionPlan)
		s.Require().NoError(err)

		viewWithAddon, subscriptionAddon, err := s.SubscriptionWorkflowService.AddAddon(ctx, subsView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         extraSeats.ID,
			InitialQuantity: 1,
			Timing:          s.timingImmediate(),
		})
		s.Require().NoError(err)
		subsView = viewWithAddon
		extraSeatsID = subscriptionAddon.NamespacedID

		// then:
		// - both seats are billed through a single $12 item
		seatItems := s.getPhaseByKey(s.T(), subsView, "first-phase").ItemsByKey["seats"]
		s.Require().Len(seatItems, 1)
		s.Equal(12.0, flatAmount(seatItems[0].Spec.RateCard.AsMeta().Price))
	})

	s.Run("pay january in advance", func() {
		// when:
		// - the first period is invoiced and paid
		s.Require().NoError(s.Service.SyncByView(ctx, subsView, periodEnd))

		clock.FreezeTime(seatAddedAt)
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.Require().NoError(err)
		s.Require().Len(draftInvoices, 1)

		paidInvoice, err = s.BillingService.ApproveInvoice(ctx, draftInvoices[0].GetInvoiceID())
		s.Require().NoError(err)

		// then:
		// - the customer paid $12 for two seats for all of January
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(12.0, paidInvoice.Totals.Total.InexactFloat64())
	})

	s.Run("add a third seat on january 2", func() {
		// when:
		// - the add-on quantity goes from one extra seat to two
		// - the resulting pending lines are invoiced
		var err error
		subsView, _, err = s.SubscriptionWorkflowService.ChangeAddonQuantity(ctx, subsView.Subscription.NamespacedID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: extraSeatsID,
			Quantity:            2,
			Timing:              s.timingImmediate(),
		})
		s.Require().NoError(err)

		s.Require().NoError(s.Service.SyncByView(ctx, subsView, periodEnd))

		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.Require().NoError(err)
		s.Require().Len(draftInvoices, 1)

		// then:
		// - the customer owes only the added seat for the rest of January: $6 * 30/31 = $5.81
		s.Equal(5.81, draftInvoices[0].Totals.Total.InexactFloat64())

		// - the $12 already paid for the first two seats stands
		paidInvoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: paidInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(12.0, paidInvoice.Totals.Total.InexactFloat64())
	})
}
