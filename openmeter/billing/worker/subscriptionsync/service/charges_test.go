package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargesTestSuite struct {
	SuiteBase
}

func (s *ChargesTestSuite) SetupSuite() {
	s.SuiteBase.SetupSuite()
}

func (s *ChargesTestSuite) BeforeTest(suiteName, testName string) {
	s.SuiteBase.BeforeTest(s.T().Context(), suiteName, testName)
}

func (s *ChargesTestSuite) AfterTest(suiteName, testName string) {
	s.SuiteBase.AfterTest(s.T().Context(), suiteName, testName)
}

func TestCharges(t *testing.T) {
	suite.Run(t, new(ChargesTestSuite))
}

func (s *ChargesTestSuite) mustParseTime(t string) time.Time {
	s.T().Helper()
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *ChargesTestSuite) TestBackfillCharges() {
	ctx := s.T().Context()

	// Given the charges service is not active on subscription
	// Given a customer with a subscription started on 2025-01-01 that has usage-based lines and flat fees
	// Given a the subscription has synced up to 2025-03-01
	// When the subscription service gets the charges enabled
	// When canceling the subscription as of 2025-02-15
	// Then:
	// - The sync completes successfully
	// - The lines receive charge IDs
	// - The charges are created with the correct intent and realizations

	clock.FreezeTime(s.mustParseTime("2025-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	s.enableProgressiveBilling()
	s.MockStreamingConnector.AddSimpleEvent(
		*s.APIRequestsTotalFeature.MeterSlug,
		10,
		s.mustParseTime("2025-01-20T00:00:00Z"),
	)

	s.True(s.Service.backfillCharges)
	defer func() {
		s.Service.backfillCharges = true
	}()

	s.Service.backfillCharges = false

	var (
		subView        subscription.SubscriptionView
		invoices       billing.ListInvoicesResponse
		progressiveAt  time.Time
		createdInvoice billing.StandardInvoice
		err            error
	)

	s.Run("setup subscription and sync with charges disabled", func() {
		subView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: s.Namespace,
			},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Test Plan",
					Key:            "test-plan",
					Version:        1,
					Currency:       currency.USD,
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{
					{
						PhaseMeta: s.phaseMeta("default", ""),
						RateCards: productcatalog.RateCards{
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "in-advance-flat",
									Name: "in-advance-flat",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromFloat(5),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
							},
							&productcatalog.UsageBasedRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:        s.APIRequestsTotalFeature.Key,
									Name:       s.APIRequestsTotalFeature.Key,
									FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
									FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
									Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
										Amount: alpacadecimal.NewFromFloat(10),
									}),
								},
								BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
							},
						},
					},
				},
			},
		})

		s.NoError(s.Service.SynchronizeSubscription(ctx, subView, s.mustParseTime("2025-03-01T00:00:00Z")))
	})

	s.Run("progressive invoice has no charge ids while disabled", func() {
		progressiveAt = s.mustParseTime("2025-01-10T00:00:00Z")
		clock.FreezeTime(progressiveAt)
		createdInvoices, createErr := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.CustomerID{
				Namespace: s.Namespace,
				ID:        s.Customer.ID,
			},
			AsOf: &progressiveAt,
		})
		s.NoError(createErr)
		s.Len(createdInvoices, 1)
		createdInvoice = createdInvoices[0]

		for _, line := range createdInvoice.Lines.OrEmpty() {
			s.Nil(line.ChargeID)
		}
	})

	s.Run("re-enable charges then cancel and resync", func() {
		s.Service.backfillCharges = true

		cancelAt := s.mustParseTime("2025-02-15T00:00:00Z")
		clock.FreezeTime(cancelAt)
		sub, cancelErr := s.SubscriptionService.Cancel(ctx, subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(cancelErr)

		subView, err = s.SubscriptionService.GetView(ctx, sub.NamespacedID)
		s.NoError(err)

		// Event delivery is async, so we need to advance the clock a bit.
		clock.FreezeTime(clock.Now().Add(time.Second))
		s.NoError(s.Service.SynchronizeSubscription(ctx, subView, clock.Now()))
	})

	s.Run("collect invoices for assertion", func() {
		invoices, err = s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{s.Namespace},
			Customers:  []string{s.Customer.ID},
			Expand:     billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.NotEmpty(invoices.Items)
	})

	chargeIDs := map[string]struct{}{}
	standardLineIDs := map[string]struct{}{}
	firstPeriodDraftLineIDByIntentType := map[charges.IntentType]string{}

	s.Run("lines receive charge ids after backfill", func() {
		for _, invoice := range invoices.Items {
			switch invoice.Type() {
			case billing.InvoiceTypeStandard:
				stdInvoice, convErr := invoice.AsStandardInvoice()
				s.NoError(convErr)
				s.Len(stdInvoice.Lines.OrEmpty(), 2, "expected exactly two lines in standard invoice")

				for _, line := range stdInvoice.Lines.OrEmpty() {
					if line.Subscription == nil || line.Subscription.SubscriptionID != subView.Subscription.ID {
						continue
					}

					intentType := charges.IntentTypeFlatFee
					if line.UsageBased.Price.Type() != productcatalog.FlatPriceType {
						intentType = charges.IntentTypeUsageBased
					}

					firstPeriodDraftLineIDByIntentType[intentType] = line.ID

					if line.ChargeID == nil {
						continue
					}

					chargeIDs[*line.ChargeID] = struct{}{}
					standardLineIDs[line.ID] = struct{}{}
				}
			case billing.InvoiceTypeGathering:
				gatheringInvoice, convErr := invoice.AsGatheringInvoice()
				s.NoError(convErr)

				for _, line := range gatheringInvoice.Lines.OrEmpty() {
					if line.Subscription == nil || line.Subscription.SubscriptionID != subView.Subscription.ID {
						continue
					}

					if line.ChargeID == nil {
						continue
					}

					chargeIDs[*line.ChargeID] = struct{}{}
				}
			}
		}

		s.NotEmpty(chargeIDs, "expected at least one charge id on subscription lines")
		s.NotEmpty(standardLineIDs, "expected at least one standard line linked to a charge")
		s.NotEmpty(firstPeriodDraftLineIDByIntentType[charges.IntentTypeFlatFee], "expected first-period draft flat-fee line")
		s.NotEmpty(firstPeriodDraftLineIDByIntentType[charges.IntentTypeUsageBased], "expected first-period draft usage-based line")
	})

	s.Run("charges have intent and realization data", func() {
		foundFirstPeriodChargeByIntentType := map[charges.IntentType]bool{}

		for chargeID := range chargeIDs {
			charge, getErr := s.Service.chargesService.GetChargeByID(ctx, models.NamespacedID{
				Namespace: s.Namespace,
				ID:        chargeID,
			})
			s.NoError(getErr)

			s.Equal(s.Customer.ID, charge.Intent.CustomerID)
			s.NotNil(charge.Intent.UniqueReferenceID)

			expectedLineID, ok := firstPeriodDraftLineIDByIntentType[charge.Intent.IntentType]
			if !ok {
				continue
			}

			foundFirstPeriodChargeByIntentType[charge.Intent.IntentType] = true
			s.Len(charge.Realizations.StandardInvoice, 1, "expected exactly one realization for first-period %s charge", charge.Intent.IntentType)
			s.Equal(expectedLineID, charge.Realizations.StandardInvoice[0].LineID)
			s.Equal(charges.StandardInvoiceRealizationStatusDraft, charge.Realizations.StandardInvoice[0].Status)
		}

		s.True(foundFirstPeriodChargeByIntentType[charges.IntentTypeFlatFee], "expected first-period flat-fee charge to be backfilled")
		s.True(foundFirstPeriodChargeByIntentType[charges.IntentTypeUsageBased], "expected first-period usage-based charge to be backfilled")
	})
}
