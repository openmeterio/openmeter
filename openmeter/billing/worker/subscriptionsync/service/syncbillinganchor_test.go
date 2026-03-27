package service

import (
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type BillingAnchorTestSuite struct {
	SuiteBase
}

func TestBillingAnchor(t *testing.T) {
	suite.Run(t, new(BillingAnchorTestSuite))
}

func (s *BillingAnchorTestSuite) SetupSuite() {
	s.SuiteBase.SetupSuite()
}

func (s *BillingAnchorTestSuite) BeforeTest(suiteName, testName string) {
	s.SuiteBase.BeforeTest(s.T().Context(), suiteName, testName)
}

func (s *BillingAnchorTestSuite) AfterTest(suiteName, testName string) {
	s.SuiteBase.AfterTest(s.T().Context(), suiteName, testName)
}

func (s *BillingAnchorTestSuite) TestBillingAnchorSinglePhase() {
	// Given we have a subscription:
	//  - with a single usage based item
	//  - an entitlement with a grant of 1000
	//  - started at 2025-07-10T15:00:00Z
	//  - billing anchor is at 2025-01-31T15:00:00Z
	// When synchronizing the subscription up to 2025-09-29T15:00:00Z
	// Then:
	//  - the entitlement should be set up to be active from 2025-07-10T15:00:00Z,
	//  - the first period should be 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z,
	// Then:
	//  - the gathering invoice should have the following service periods:
	//    - 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z
	//    - 2025-07-31T15:00:00Z - 2025-08-31T15:00:00Z
	//    - 2025-08-31T15:00:00Z - 2025-09-30T15:00:00Z

	ctx := s.T().Context()
	defer clock.UnFreeze()
	clock.FreezeTime(testutils.GetRFC3339Time(s.T(), "2025-06-30T15:00:00Z"))
	billingAnchor := testutils.GetRFC3339Time(s.T(), "2025-01-31T15:00:00Z")

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
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
					PhaseMeta: productcatalog.PhaseMeta{
						Name: "first-phase",
						Key:  "first-phase",
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:       "test-rate-card",
								Key:        s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(100),
								}),
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									IssueAfterReset: lo.ToPtr(1000.0),
									UsagePeriod:     datetime.MustParseDuration(s.T(), "P1M"),
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), plan)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	clock.FreezeTime(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"))
	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			Name: "subs-1",
		},
		Namespace:     s.Namespace,
		CustomerID:    s.Customer.ID,
		BillingAnchor: lo.ToPtr(billingAnchor),
	}, subscriptionPlan)
	s.NoError(err)
	s.NotNil(subsView)

	// Subscription retains the billing anchor
	s.Equal(billingAnchor, subsView.Subscription.BillingAnchor)

	// When synchronizing the subscription up to 2025-09-30T15:00:00Z
	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, testutils.GetRFC3339Time(s.T(), "2025-09-29T15:00:00Z")))

	// Then:
	//  - the entitlement should be set up to be active from 2025-07-10T15:00:00Z,
	//  - the first period should be 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z,
	ents, err := s.EntitlementConnector.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:  []string{s.Namespace},
		SubjectKeys: s.Customer.UsageAttribution.SubjectKeys,
		FeatureKeys: []string{s.APIRequestsTotalFeature.Key},
	})
	s.NoError(err)
	s.Equal(1, len(ents.Items))
	ent := ents.Items[0]

	// Assert the entitlement periods
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"), ent.ActiveFromTime())
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"), ent.MeasureUsageFrom.In(time.UTC))
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-01-31T15:00:00Z"), *ent.OriginalUsagePeriodAnchor)
	s.Equal(timeutil.ClosedPeriod{
		From: testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"),
		To:   testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"),
	}, *ent.CurrentUsagePeriod)

	//  - the gathering invoice should have the following service periods:
	//    - 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z
	//    - 2025-07-31T15:00:00Z - 2025-08-31T15:00:00Z
	//    - 2025-08-31T15:00:00Z - 2025-09-30T15:00:00Z

	invoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	s.expectLines(invoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Periods: []billing.Period{
				{
					Start: testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"),
					End:   testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"),
			}),
			AdditionalChecks: func(line billing.GenericInvoiceLine) {
				s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"), line.GetSubscriptionReference().BillingPeriod.From)
				s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"), line.GetSubscriptionReference().BillingPeriod.To)
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 1,
				PeriodMax: 2,
			},
			Periods: []billing.Period{
				{
					Start: testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"),
					End:   testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
				},
				{
					Start: testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
					End:   testutils.GetRFC3339Time(s.T(), "2025-09-30T15:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
				testutils.GetRFC3339Time(s.T(), "2025-09-30T15:00:00Z"),
			}),
		},
	})
}

func (s *BillingAnchorTestSuite) TestBillingAnchorMultiPhase() {
	// Given we have a subscription:
	//  - phases:
	//    - first phase:
	//		- not billable, 1 month long
	//      - an entitlement with a grant of 1000
	//    - second phase:
	//		- billable, 1 month long billing cadence
	//		- an entitlement with a grant of 1000
	//  - started at 2025-07-10T15:00:00Z
	//  - billing anchor is at 2025-01-31T15:00:00Z
	// When synchronizing the subscription up to 2025-09-29T15:00:00Z
	// Then:
	//  - there are two entitlements for each phase
	//  	- free phase entitlement:
	// 			- from 2025-07-10T15:00:00Z to 2025-08-10T15:00:00Z
	// 			- the first period should be 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z
	//  	- billed phase entitlement:
	// 			- from 2025-08-10T15:00:00Z
	// 			- the first period should be 2025-08-10T15:00:00Z - 2025-08-31T15:00:00Z
	// Then:
	//  - the gathering invoice should have the following service periods:
	//    - 2025-08-10T15:00:00Z - 2025-08-31T15:00:00Z
	//    - 2025-08-31T15:00:00Z - 2025-09-30T15:00:00Z

	ctx := s.T().Context()
	defer clock.UnFreeze()
	clock.FreezeTime(testutils.GetRFC3339Time(s.T(), "2025-06-30T15:00:00Z"))
	billingAnchor := testutils.GetRFC3339Time(s.T(), "2025-01-31T15:00:00Z")

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
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
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "not-billable-phase",
						Key:      "not-billable-phase",
						Duration: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:       "test-rate-card",
								Key:        s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									IssueAfterReset: lo.ToPtr(1000.0),
									UsagePeriod:     datetime.MustParseDuration(s.T(), "P1M"),
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name: "billable-phase",
						Key:  "billable-phase",
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:       "test-rate-card",
								Key:        s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(100),
								}),
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									IssueAfterReset: lo.ToPtr(1000.0),
									UsagePeriod:     datetime.MustParseDuration(s.T(), "P1M"),
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), plan)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	clock.FreezeTime(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"))
	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			Name: "subs-1",
		},
		Namespace:     s.Namespace,
		CustomerID:    s.Customer.ID,
		BillingAnchor: lo.ToPtr(billingAnchor),
	}, subscriptionPlan)
	s.NoError(err)
	s.NotNil(subsView)

	// Subscription retains the billing anchor
	s.Equal(billingAnchor, subsView.Subscription.BillingAnchor)

	// When synchronizing the subscription up to 2025-09-30T15:00:00Z
	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, testutils.GetRFC3339Time(s.T(), "2025-09-29T15:00:00Z")))

	// Then:
	//  - the entitlement should be set up to be active from 2025-07-10T15:00:00Z,
	//  - the first period should be 2025-07-10T15:00:00Z - 2025-07-31T15:00:00Z,
	ents, err := s.EntitlementConnector.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:  []string{s.Namespace},
		SubjectKeys: s.Customer.UsageAttribution.SubjectKeys,
		FeatureKeys: []string{s.APIRequestsTotalFeature.Key},
	})
	s.NoError(err)
	slices.SortFunc(ents.Items, func(a, b entitlement.Entitlement) int {
		return a.ActiveFromTime().Compare(b.ActiveFromTime())
	})
	s.Equal(2, len(ents.Items))
	freePhaseEntitlement := ents.Items[0]
	billedPhaseEntitlement := ents.Items[1]

	// Assert the entitlement periods
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"), freePhaseEntitlement.ActiveFromTime())
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-08-10T15:00:00Z"), *freePhaseEntitlement.ActiveToTime())
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"), freePhaseEntitlement.MeasureUsageFrom.In(time.UTC))
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-01-31T15:00:00Z"), *freePhaseEntitlement.OriginalUsagePeriodAnchor)
	s.Equal(timeutil.ClosedPeriod{
		From: testutils.GetRFC3339Time(s.T(), "2025-07-10T15:00:00Z"),
		To:   testutils.GetRFC3339Time(s.T(), "2025-07-31T15:00:00Z"),
	}, *freePhaseEntitlement.CurrentUsagePeriod)

	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-08-10T15:00:00Z"), billedPhaseEntitlement.ActiveFromTime())
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-08-10T15:00:00Z"), billedPhaseEntitlement.MeasureUsageFrom.In(time.UTC))
	s.Equal(testutils.GetRFC3339Time(s.T(), "2025-01-31T15:00:00Z"), *billedPhaseEntitlement.OriginalUsagePeriodAnchor)
	s.Equal(timeutil.ClosedPeriod{
		From: testutils.GetRFC3339Time(s.T(), "2025-08-10T15:00:00Z"),
		To:   testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
	}, *billedPhaseEntitlement.CurrentUsagePeriod)

	//  - the gathering invoice should have the following service periods:
	//    - 2025-08-10T15:00:00Z - 2025-08-31T15:00:00Z
	//    - 2025-08-31T15:00:00Z - 2025-09-30T15:00:00Z

	invoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	s.expectLines(invoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "billable-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Periods: []billing.Period{
				{
					Start: testutils.GetRFC3339Time(s.T(), "2025-08-10T15:00:00Z"),
					End:   testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
			}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "billable-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Periods: []billing.Period{
				{
					Start: testutils.GetRFC3339Time(s.T(), "2025-08-31T15:00:00Z"),
					End:   testutils.GetRFC3339Time(s.T(), "2025-09-30T15:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				testutils.GetRFC3339Time(s.T(), "2025-09-30T15:00:00Z"),
			}),
		},
	})
}
