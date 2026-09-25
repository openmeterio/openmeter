package service

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func (s *SubscriptionHandlerTestSuite) TestReorderedLifecycleEventsUseCurrentSubscription() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given an edit after creation, while the created event still carries the old price.
	createdView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{{
		PhaseMeta: s.phaseMeta("first-phase", ""),
		RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  "fee",
				Name: "fee",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(5),
					PaymentTerm: productcatalog.InArrearsPaymentTerm,
				}),
			},
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}},
	}})
	id := createdView.Subscription.NamespacedID
	clock.SetTime(start.AddDate(0, 0, 1))
	_, err := s.SubscriptionWorkflowService.EditRunning(ctx, id, []subscription.Patch{
		patch.PatchRemoveItem{PhaseKey: "first-phase", ItemKey: "fee"},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "fee",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.Require().NoError(err)

	// When the updated event is handled before the delayed created event.
	s.Require().NoError(s.Service.HandleSubscriptionChange(ctx, id))
	before := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID).Lines.OrEmpty()
	s.Require().NoError(s.Service.HandleSubscriptionChange(ctx, id))

	// Then the delayed event cannot restore older billing state or replace lines.
	after := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID).Lines.OrEmpty()
	s.Require().Equal(len(before), len(after))
	var currentPriceFound bool
	for i := range before {
		s.Equal(before[i].ID, after[i].ID)
		s.Equal(before[i].Price, after[i].Price)
		s.Equal(before[i].Period, after[i].Period)
		price, err := after[i].Price.AsFlat()
		s.Require().NoError(err)
		currentPriceFound = currentPriceFound || price.Amount.InexactFloat64() == 10
	}
	s.True(currentPriceFound, "the committed edit must be reflected in billing")
}
