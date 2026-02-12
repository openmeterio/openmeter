package subscription_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscription "github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestBillingOnFirstOfMonth(t *testing.T) {
	// Let's declare our variables
	// note: this namespace is hardcoded in the test framework (TuriP: why on earth is it hardcoded in the test framework?)
	namespace := "test-namespace"

	startOfSub := testutils.GetRFC3339Time(t, "2025-06-15T12:00:00Z")
	currentTime := startOfSub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)

	clock.SetTime(currentTime)

	// 1st, let's create the features
	feats := tDeps.FeatureConnector.CreateExampleFeatures(t)
	require.Len(t, feats, 3)

	// 2nd, let's create the plan
	p, err := tDeps.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test_plan",
				Currency:       "USD",
				BillingCadence: datetime.MustParseDuration(t, "P1M"), // Let's do monthly billing
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:      "default",
						Name:     "Default Phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						// Let's have an in-arrears monthly entitlement ratecard
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        feats[0].Key,
								Name:       "Test Rate Card",
								FeatureKey: lo.ToPtr(feats[0].Key),
								FeatureID:  lo.ToPtr(feats[0].ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromInt(100),
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1M"), // compatible with the billing cadence
									IssueAfterReset: lo.ToPtr(10.0),
								}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1M"),
						},
						// Let's have an in-advance monthly ratecard
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        feats[1].Key,
								Name:       "Test Rate Card 2",
								FeatureKey: lo.ToPtr(feats[1].Key),
								FeatureID:  lo.ToPtr(feats[1].ID),
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(10),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
						// Let's have an in arrears daily ratecard
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        feats[2].Key,
								Name:       "Test Rate Card 3",
								FeatureKey: lo.ToPtr(feats[2].Key),
								FeatureID:  lo.ToPtr(feats[2].ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromInt(1),
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1D"), // compatible with the billing cadence
									IssueAfterReset: lo.ToPtr(10.0),
								}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1D"),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	p, err = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(currentTime),
		},
	})
	require.NoError(t, err)

	// 3rd, let's create the billing profile
	_, err = tDeps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()))
	require.NoError(t, err)

	// Then create the customer
	c, err := tDeps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Test Customer",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_1"},
			},
		},
	})
	require.NoError(t, err)

	pi := &pcsubscription.PlanInput{}
	pi.FromRef(&pcsubscription.PlanRefInput{
		Key:     p.Key,
		Version: &p.Version,
	})

	// First of month
	firstOfMonth := time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, currentTime.Location())
	startOfDay := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())

	// 4th, let's create the subscription
	s, err := tDeps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  namespace,
			CustomerID: c.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &startOfSub,
				},
				Name: "Test Subscription",
			},
			BillingAnchor: &firstOfMonth, // We align the billing anchor to the first of the month
		},
		PlanInput: *pi,
	})
	require.NoError(t, err)
	require.NotNil(t, s)

	view, err := tDeps.SubscriptionService.GetView(ctx, s.NamespacedID)
	require.NoError(t, err)
	require.NotNil(t, view)

	t.Run("entitlements", func(t *testing.T) {
		// Let's check the UsagePeriods are aligned
		ent1 := view.Phases[0].ItemsByKey[feats[0].Key][0].Entitlement
		require.Equal(t, datetime.MustParseDuration(t, "P1M"), ent1.Entitlement.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Interval.ISODuration)

		require.Equal(t, firstOfMonth, ent1.Entitlement.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Anchor)
		require.Equal(t, startOfSub, ent1.Entitlement.MeasureUsageFrom.UTC())

		ent2 := view.Phases[0].ItemsByKey[feats[2].Key][0].Entitlement
		require.Equal(t, datetime.MustParseDuration(t, "P1D"), ent2.Entitlement.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Interval.ISODuration)

		require.Equal(t, firstOfMonth, ent2.Entitlement.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Anchor)
		require.Equal(t, startOfSub, ent2.Entitlement.MeasureUsageFrom.UTC())
	})

	// Let's pass some time
	clock.SetTime(currentTime.Add(time.Minute))

	// 5th, let's synchronize the invoice
	require.NoError(t, tDeps.subscriptionSyncService.SynchronizeSubscription(ctx, view, firstOfMonth.AddDate(0, 1, 0)))

	// 6th, let's check the invoice
	invoices, err := tDeps.billingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{c.ID},
		Expand:     billing.GatheringInvoiceExpandAll,
	})

	require.NoError(t, err)
	require.Len(t, invoices.Items, 1)

	invoice := invoices.Items[0]

	lns, ok := invoice.Lines.Get()
	require.True(t, ok)

	linesByFeature := lo.GroupBy(lns, func(l billing.GatheringLine) string {
		if l.FeatureKey != "" {
			return l.FeatureKey
		}

		if l.ChildUniqueReferenceID != nil {
			return *l.ChildUniqueReferenceID
		}

		return ""
	})

	endOfMonth := firstOfMonth.AddDate(0, 1, 0)

	t.Run("lines for test-feature-1", func(t *testing.T) {
		lines, ok := linesByFeature[feats[0].Key]
		require.True(t, ok)

		// We expect a single in arrears line
		require.Len(t, lines, 1)

		line := lines[0]

		require.Equal(t, startOfSub, line.ServicePeriod.From)
		require.Equal(t, endOfMonth, line.ServicePeriod.To)
		require.Equal(t, endOfMonth, line.InvoiceAt)
	})

	t.Run("lines for test-feature-2", func(t *testing.T) {
		// As these are not usagebasedlines, we'll use filtering here
		var lines []billing.GatheringLine

		for k, v := range linesByFeature {
			if strings.Contains(k, feats[1].Key) {
				lines = append(lines, v...)
			}
		}

		// We should have two lines, 1 for the current month and 1 for the next
		require.Len(t, lines, 2)

		// Let's sort by line.ChildUniqueReferenceID
		slices.SortFunc(lines, func(i, j billing.GatheringLine) int {
			return strings.Compare(*i.ChildUniqueReferenceID, *j.ChildUniqueReferenceID)
		})

		line1 := lines[0]
		require.Equal(t, startOfSub, line1.ServicePeriod.From)
		require.Equal(t, endOfMonth, line1.ServicePeriod.To)
		require.Equal(t, startOfSub, line1.InvoiceAt)

		line2 := lines[1]
		require.Equal(t, endOfMonth, line2.ServicePeriod.From)
		require.Equal(t, endOfMonth.AddDate(0, 1, 0), line2.ServicePeriod.To)
		require.Equal(t, endOfMonth, line2.InvoiceAt)
	})

	t.Run("lines for test-feature-3", func(t *testing.T) {
		lines, ok := linesByFeature[feats[2].Key]
		require.True(t, ok)

		// We expect 16 lines (15 to 30)
		require.Len(t, lines, 16)

		// Let's sort the lines by the period start ascending
		slices.SortFunc(lines, func(i, j billing.GatheringLine) int {
			return i.ServicePeriod.From.Compare(j.ServicePeriod.From)
		})

		// The first line will be partial for the half day
		line1 := lines[0]
		require.Equal(t, startOfSub, line1.ServicePeriod.From)
		require.Equal(t, startOfDay.AddDate(0, 0, 1), line1.ServicePeriod.To)
		require.Equal(t, endOfMonth, line1.InvoiceAt)

		for idx, line := range lines[1:] {
			require.Equal(t, startOfDay.AddDate(0, 0, idx+1), line.ServicePeriod.From)
			require.Equal(t, startOfDay.AddDate(0, 0, idx+2), line.ServicePeriod.To)
			require.Equal(t, endOfMonth, line.InvoiceAt)
		}
	})
}

func TestAnchoredAlignment_MidMonthStart_EarlyCancel_IssueNextAnchor(t *testing.T) {
	// Namespace used by the test framework
	namespace := "test-namespace"

	startOfSub := testutils.GetRFC3339Time(t, "2025-06-15T12:00:00Z")
	currentTime := startOfSub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)

	clock.SetTime(currentTime)

	// Create minimal plan with one in-arrears monthly item so it produces an end-of-month line
	feats := tDeps.FeatureConnector.CreateExampleFeatures(t)
	require.NotEmpty(t, feats)

	p, err := tDeps.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Anchored Plan",
				Key:             "anchored_plan",
				Currency:        "USD",
				BillingCadence:  datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
				RateCards: productcatalog.RateCards{
					&productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:        feats[0].Key,
							Name:       "FLAT_PRICE",
							FeatureKey: lo.ToPtr(feats[0].Key),
							FeatureID:  lo.ToPtr(feats[0].ID),
							Price:      productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(1), PaymentTerm: productcatalog.InArrearsPaymentTerm}),
						},
						BillingCadence: datetime.MustParseDuration(t, "P1M"),
					},
				},
			}},
		},
	})
	require.NoError(t, err)
	p, err = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{NamespacedID: p.NamespacedID, EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(currentTime)}})
	require.NoError(t, err)

	// Create billing profile with anchored alignment: first day of month
	profInput := minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID())
	profInput.Namespace = namespace
	profInput.WorkflowConfig.Collection.Alignment = billing.AlignmentKindAnchored
	anchor := time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	profInput.WorkflowConfig.Collection.AnchoredAlignmentDetail = lo.ToPtr(billing.AnchoredAlignmentDetail{
		Interval: datetime.MustParseDuration(t, "P1M"),
		Anchor:   anchor,
	})
	_, err = tDeps.billingService.CreateProfile(ctx, profInput)
	require.NoError(t, err)

	// Create customer
	c, err := tDeps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name:             "Test Customer",
			UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"subject_1"}},
		},
	})
	require.NoError(t, err)

	// Create subscription with billing anchor = first of month
	pi := &pcsubscription.PlanInput{}
	pi.FromRef(&pcsubscription.PlanRefInput{Key: p.Key, Version: &p.Version})
	firstOfNextMonth := testutils.GetRFC3339Time(t, "2025-07-01T00:00:00Z")
	beforeFirstOfMonth := testutils.GetRFC3339Time(t, "2025-06-28T00:00:00Z")
	s, err := tDeps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  namespace,
			CustomerID: c.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: &startOfSub},
				Name:   "Anchored Sub",
			},
			BillingAnchor: &beforeFirstOfMonth,
		},
		PlanInput: *pi,
	})
	require.NoError(t, err)

	view, err := tDeps.SubscriptionService.GetView(ctx, s.NamespacedID)
	require.NoError(t, err)

	// Cancel effective before end of month (e.g., on 20th)
	cancelAt := beforeFirstOfMonth
	_, err = tDeps.subscriptionService.Cancel(ctx, s.NamespacedID, subscription.Timing{Custom: &cancelAt})
	require.NoError(t, err)

	// Let's advance time until after
	clock.SetTime(cancelAt.Add(time.Hour * 1))

	// Sync up to the next anchor (July 1st). Lines should remain on gathering invoice until then.
	require.NoError(t, tDeps.subscriptionSyncService.SynchronizeSubscriptionAndInvoiceCustomer(ctx, view, firstOfNextMonth))

	invoices, err := tDeps.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{c.ID},
		Expand:     billing.InvoiceExpandAll,
		Statuses:   []string{},
	})
	require.NoError(t, err)

	// Gathering and Standard, let's get the standard
	require.Len(t, invoices.Items, 2)
	invoice, ok := lo.Find(invoices.Items, func(i billing.Invoice) bool {
		return i.Type() != billing.InvoiceTypeGathering
	})
	require.True(t, ok)
	gatheringInvoice, err := invoice.AsGatheringInvoice()
	require.NoError(t, err)

	// We should have a gathering invoice with lines invoiceAt at end of June and collectionAt at July 1st (due to anchored alignment)
	require.Equal(t, firstOfNextMonth, gatheringInvoice.NextCollectionAt)

	lns, ok := gatheringInvoice.Lines.Get()
	require.True(t, ok)
	for _, l := range lns {
		require.True(t, l.InvoiceAt.Before(gatheringInvoice.NextCollectionAt))
	}
}
