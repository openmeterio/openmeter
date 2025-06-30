package billingworkersubscription

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type SubscriptionHandlerTestSuite struct {
	billingtest.BaseSuite

	billingtest.SubscriptionMixin

	Namespace               string
	Customer                *customer.Customer
	APIRequestsTotalFeature feature.Feature
	Context                 context.Context

	Handler *Handler
}

func (s *SubscriptionHandlerTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.SubscriptionMixin.SetupSuite(s.T(), s.GetSubscriptionMixInDependencies())

	handler, err := New(Config{
		BillingService:      s.BillingService,
		Logger:              slog.Default(),
		Tracer:              noop.NewTracerProvider().Tracer("test"),
		TxCreator:           s.BillingAdapter,
		SubscriptionService: s.SubscriptionService,
	})
	s.NoError(err)

	s.Handler = handler
}

func (s *SubscriptionHandlerTestSuite) BeforeTest(suiteName, testName string) {
	s.Namespace = "test-subs-update-" + ulid.Make().String()
	// TODO: go 1.24, let's use T()'s context
	s.Context = context.Background()

	ctx := s.Context

	appSandbox := s.InstallSandboxApp(s.T(), s.Namespace)

	s.ProvisionBillingProfile(ctx, s.Namespace, appSandbox.GetID())

	apiRequestsTotalMeterSlug := "api-requests-total"

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: s.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "API Requests Total",
			},
			Key:           apiRequestsTotalMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "Replacing meters must not return error")

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: s.Namespace,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterSlug: lo.ToPtr("api-requests-total"),
	})
	s.NoError(err)
	s.APIRequestsTotalFeature = apiRequestsTotalFeature

	customerEntity := s.CreateTestCustomer(s.Namespace, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.Customer = customerEntity
}

func (s *SubscriptionHandlerTestSuite) AfterTest(suiteName, testName string) {
	clock.UnFreeze()
	clock.ResetTime()

	err := s.MeterAdapter.ReplaceMeters(s.Context, []meter.Meter{})
	s.NoError(err, "Replacing meters must not return error")

	s.MockStreamingConnector.Reset()
	s.Handler.featureFlags = FeatureFlags{}
}

func TestSubscriptionHandlerScenarios(t *testing.T) {
	suite.Run(t, new(SubscriptionHandlerTestSuite))
}

func (s *SubscriptionHandlerTestSuite) mustParseTime(t string) time.Time {
	s.T().Helper()
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *SubscriptionHandlerTestSuite) TestSubscriptionHappyPath() {
	ctx := s.Context
	namespace := s.Namespace
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()
	defer s.MockStreamingConnector.Reset()

	_ = s.InstallSandboxApp(s.T(), namespace)

	s.enableProgressiveBilling()

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "free trial",
						Key:      "free-trial",
						Duration: lo.ToPtr(testutils.GetISODuration(s.T(), "P1M")),
					},
					// TODO[OM-1031]: let's add discount handling (as this could be a 100% discount for the first month)
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        s.APIRequestsTotalFeature.Key,
								Name:       s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "discounted phase",
						Key:      "discounted-phase",
						Duration: lo.ToPtr(testutils.GetISODuration(s.T(), "P2M")),
					},
					// TODO[OM-1031]: 50% discount
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        s.APIRequestsTotalFeature.Key,
								Name:       s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "final phase",
						Key:      "final-phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
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
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	s.NoError(err)
	s.NotNil(plan)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(start),
			},
			Name: "subs-1",
		},
		Namespace:  namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	freeTierPhase := getPhaseByKey(s.T(), subsView, "free-trial")
	s.Equal(lo.ToPtr(isodate.MustParse(s.T(), "P1M")), freeTierPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].Spec.RateCard.GetBillingCadence())

	discountedPhase := getPhaseByKey(s.T(), subsView, "discounted-phase")
	var gatheringInvoiceID billing.InvoiceID

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{s.Customer.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		// then there should be a gathering invoice
		invoice := s.gatheringInvoice(ctx, namespace, s.Customer.ID)
		invoiceUpdatedAt := invoice.UpdatedAt

		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		// 1 month free tier + in arrears billing with 1 month cadence
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// When we advance the clock the invoice doesn't get changed
		clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

		gatheringInvoice := s.gatheringInvoice(ctx, namespace, s.Customer.ID)
		s.NoError(err)
		gatheringInvoiceID = gatheringInvoice.InvoiceID()

		s.DebugDumpInvoice("gathering invoice - 2nd update", gatheringInvoice)

		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(invoiceUpdatedAt, gatheringInvoice.UpdatedAt)
		s.Equal(billing.InvoiceStatusGathering, gatheringInvoice.Status)
		s.Equal(line.UpdatedAt, gatheringLine.UpdatedAt)
	})

	s.NoError(gatheringInvoiceID.Validate())

	// Progressive billing updates
	s.Run("progressive billing updates", func() {
		s.MockStreamingConnector.AddSimpleEvent(
			*s.APIRequestsTotalFeature.MeterSlug,
			100,
			s.mustParseTime("2024-02-02T00:00:00Z"))
		clock.FreezeTime(s.mustParseTime("2024-02-15T00:00:01Z"))

		// we invoice the customer
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.CustomerID{
				ID:        s.Customer.ID,
				Namespace: namespace,
			},
			AsOf: lo.ToPtr(s.mustParseTime("2024-02-15T00:00:00Z")),
		})
		if err != nil {
			fmt.Printf("current time: %s\n", clock.Now().Format(time.RFC3339))
		}
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]

		s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.Equal(float64(5*100), invoice.Totals.Total.InexactFloat64())

		s.Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-02-15T00:00:00Z"))
		s.Equal(line.Period, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-02-15T00:00:00Z"),
		})

		// let's fetch the gathering invoice
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]
		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))
		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})

		// TODO[OM-1037]: let's add/change some items of the subscription then expect that the new item appears on the gathering
		// invoice, but the draft invoice is untouched.
	})

	s.Run("subscription cancellation", func() {
		clock.FreezeTime(s.mustParseTime("2024-02-20T00:00:00Z"))

		cancelAt := s.mustParseTime("2024-03-01T00:00:00Z")
		subs, err := s.SubscriptionService.Cancel(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		}, subscription.Timing{
			Custom: lo.ToPtr(cancelAt),
		})
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		// Subscription has set the cancellation date, and the view's subscription items are updated to have the cadence
		// set properly up to the cancellation date.

		// If we are now resyncing the subscription, the gathering invoice should be updated to reflect the new cadence.

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   cancelAt,
		})
		s.Equal(gatheringLine.InvoiceAt, cancelAt)

		// split group
		s.NotNil(gatheringLine.SplitLineHierarchy)
		splitLineGroup := gatheringLine.SplitLineHierarchy.Group

		s.Equal(splitLineGroup.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(splitLineGroup.ServicePeriod, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
	})

	s.Run("continue subscription", func() {
		clock.FreezeTime(s.mustParseTime("2024-02-21T00:00:00Z"))

		subs, err := s.SubscriptionService.Continue(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		})
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		// If we are now resyncing the subscription, the gathering invoice should be updated to reflect the original cadence

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// split group
		s.NotNil(gatheringLine.SplitLineHierarchy)
		splitLineGroup := gatheringLine.SplitLineHierarchy.Group

		s.Equal(splitLineGroup.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(splitLineGroup.ServicePeriod, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
	})
}

func (s *SubscriptionHandlerTestSuite) TestUncollectableCollection() {
	// Test that the InvoicePendingLines returns the correct error when there are no lines to invoice,
	// as sync depends on this.

	// Given
	//  a customer with a gathering invoice, that is not collectible
	// When
	//  invoice pending lines is called
	// Then
	//  ErrInvoiceCreateNoLines is returned

	namespace := "ns-uncollectable-collection"
	ctx := context.Background()

	appSandbox := s.InstallSandboxApp(s.T(), namespace)

	customer := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customer)

	s.ProvisionBillingProfile(ctx, namespace, appSandbox.GetID())

	// Test no gathering invoice state
	s.Run("no gathering invoice", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
		})
		s.Error(err)
		s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)
		s.Len(invoices, 0)
	})

	apiRequestsTotalFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	defer apiRequestsTotalFeature.Cleanup()

	lineServicePeriod := billing.Period{
		Start: lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")),
		End:   lo.Must(time.Parse(time.RFC3339, "2025-01-02T00:00:00Z")),
	}

	clock.SetTime(lineServicePeriod.Start)
	defer clock.ResetTime()

	pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customer.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []*billing.Line{
			{
				LineBase: billing.LineBase{
					Period:    lineServicePeriod,
					InvoiceAt: lineServicePeriod.End,
					ManagedBy: billing.ManuallyManagedLine,
					Type:      billing.InvoiceLineTypeUsageBased,
					Name:      "UBP - unit",
				},
				UsageBased: &billing.UsageBasedLine{
					FeatureKey: apiRequestsTotalFeature.Feature.Key,
					Price: productcatalog.NewPriceFrom(
						productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(1),
						},
					),
				},
			},
		},
	})

	s.NoError(err)
	s.Len(pendingLines.Lines, 1)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.GetID(),
	})
	s.Error(err)
	s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)
	s.Len(invoices, 0)
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsProrating() {
	ctx := context.Background()
	namespace := "test-subs-pro-rating"
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()
	s.enableProrating()

	appSandbox := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, appSandbox.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "first-phase",
						Key:      "first-phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1D"),
						},
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears-3m",
								Name: "in-arrears-3m",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(9),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P3M"),
						},
					},
				},
			},
		},
	})

	s.NoError(err)
	s.NotNil(plan)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(start),
			},
			Name: "subs-1",
		},
		Namespace:  namespace,
		CustomerID: customerEntity.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		oneDayLines := lo.Filter(lines, func(line *billing.Line, _ int) bool {
			return line.Period.End.Sub(line.Period.Start) == time.Hour*24
		})
		s.Len(oneDayLines, 31) // january is 31 days long, and we generate lines for each daily for in arrears price

		for _, line := range oneDayLines {
			s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID, "failed for line %v", line.ID)
			s.Equal(line.Subscription.PhaseID, subsView.Phases[0].SubscriptionPhase.ID, "failed for line %v", line.ID)
			s.Equal(line.Subscription.ItemID, subsView.Phases[0].ItemsByKey["in-arrears"][0].SubscriptionItem.ID, "failed for line %v", line.ID)
			s.Equal(line.InvoiceAt, s.mustParseTime("2024-02-01T00:00:00Z"), "failed for line %v", line.ID)
			s.Equal(line.Period, billing.Period{
				Start: s.mustParseTime("2024-01-01T00:00:00Z").AddDate(0, 0, line.Period.Start.Day()-1),
				End:   s.mustParseTime("2024-01-01T00:00:00Z").AddDate(0, 0, line.Period.Start.Day()),
			}, "failed for line %v", line.ID)
			s.Equal(line.FlatFee.PerUnitAmount.InexactFloat64(), 5.0, "failed for line %v", line.ID)
			s.Equal(line.FlatFee.Quantity.InexactFloat64(), 1.0, "failed for line %v", line.ID)
		}
	})

	s.Run("canceling the subscription DOES NOT cause the existing item to be pro-rated", func() {
		// this test needs items longer than subscription.BillingCadence
		clock.SetTime(s.mustParseTime("2024-01-01T10:00:00Z"))

		cancelAt := s.mustParseTime("2024-02-01T00:00:00Z")
		subs, err := s.SubscriptionService.Cancel(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		}, subscription.Timing{
			Custom: lo.ToPtr(cancelAt),
		})
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		threeMonthLines := lo.Filter(lines, func(line *billing.Line, _ int) bool {
			return line.Period.End.Sub(line.Period.Start) != time.Hour*24 // all other lines will be 1 dqy
		})
		s.Len(threeMonthLines, 1)

		flatFeeLine := threeMonthLines[0]
		s.Equal(flatFeeLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(flatFeeLine.InvoiceAt, cancelAt)
		s.Equal(flatFeeLine.Period, billing.Period{
			Start: s.mustParseTime("2024-01-01T00:00:00Z"),
			End:   cancelAt,
		})
		s.Equal(9.0, flatFeeLine.FlatFee.PerUnitAmount.InexactFloat64())
		s.Equal(1.0, flatFeeLine.FlatFee.Quantity.InexactFloat64())
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncNonBillableAmountProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will only contain the new version of the fee, as the old one was
	//  pro-rated and the total amount is 0

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 1, // as its in-advance, we'll generate the item for the next month too
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:40Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-02-01T00:00:00Z"),
			}),
			// Periods:   s.generatePeriods("2024-01-01T00:00:40Z", "2024-02-01T00:00:40Z", "P1M", 1),
			// InvoiceAt: s.generateDailyTimestamps("2024-01-01T00:00:40Z", 6),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncNonBillableAmount() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will contain both versions of the fee as we are not
	//  doing any pro-rating logic

	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: false,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-advance",
								Name: "in-advance",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}

	subsView := s.createSubscriptionFromPlan(planInput)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
			}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 1,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:40Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-02-01T00:00:00Z"),
			}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsGatheringSyncNonBillableAmount() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single static fee in arrears
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will contain both versions of the fee as we are not
	//  doing any pro-rating logic

	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: false,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}

	subsView := s.createSubscriptionFromPlan(planInput)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			// We'll wait till the end of the billing cadence of the item
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:40Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			// We'll wait till the end of the billing cadence of the item
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncBillableAmountProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription later
	// Then
	//  the gathering invoice will contain the pro-rated previous fee and the new fee

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(10),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(20),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](0.32), // 10 * 1 / 31
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
			}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](19.35), // 20 * 30 / 31
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
			}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 1,
				PeriodMax: 1,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](20),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-02-01T00:00:00Z"),
			}),
			// Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 5),
			// InvoiceAt: s.generateDailyTimestamps("2024-01-01T12:00:00Z", 5),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncDraftInvoiceProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding draft invoice and we edit the subscription later
	// Then
	//  then the draft invoice gets updated with the new pro-rated fee and the new fee
	//  item will be available as a gathering invoice

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))

	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	s.DebugDumpInvoice("draft invoice", draftInvoices[0])

	draftInvoice := draftInvoices[0]
	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](9.68), // 10 * 30 / 31
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 1,
				PeriodMax: 1,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	// draft invoice
	draftInvoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](0.19), // 6 * 1 / 31
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncIssuedInvoiceProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding invoice that has been already finalized and we edit the subscription later
	// Then
	//  the finalized invoice doesn't get updated with the new pro-rated fee, but we
	//  add a warning to the invoice

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))

	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)

	approvedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, approvedInvoice.Status)

	s.expectLines(approvedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](9.68), // 10 * 30 / 31
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 1,
				PeriodMax: 1,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	// issued invoice
	approvedInvoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.expectLines(approvedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
		},
	})
	s.Len(approvedInvoice.ValidationIssues, 1)

	s.expectValidationIssueForLine(approvedInvoice.Lines.OrEmpty()[0], approvedInvoice.ValidationIssues[0])
}

func (s *SubscriptionHandlerTestSuite) TestDefactoZeroPrices() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single FlatFee price that is zero
	// When
	//  we provision the lines
	// Then
	//  No lines should be invoiced

	// Let's create the initial subscription
	subView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-advance",
								Name: "in-advance",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(0),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1D")),
						},
					},
				},
			},
		},
	})

	// Now let's synchronize the subscription

	asOf := s.mustParseTime("2024-01-03T12:00:00Z")
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, asOf))

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{s.Namespace},
		Customers:  []string{s.Customer.ID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})
	require.NoError(s.T(), err)

	// Now let's assert that there are no lines
	require.Len(s.T(), invoices.Items, 0)
}

func (s *SubscriptionHandlerTestSuite) TestAlignedSubscriptionInvoicing() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//	a subscription with a single phase with a single item with multiple versions of it
	// When
	//  we provision the lines
	// Then
	//  in-arrears lines should be invoiced aligned
	//  in-advance lines should be invoiced immediately aligned

	// Let's create the initial subscription
	subView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P4W"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: false,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-advance",
								Name: "in-advance",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1W")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1W")),
						},
					},
				},
			},
		},
	})

	// Let's advance a day and make some edits
	clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))

	subView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subView.Subscription.NamespacedID, []subscription.Patch{
		// Let's update in-advance item
		&patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		&patch.PatchAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "first-phase",
						ItemKey:  "in-advance",
						RateCard: &productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "in-advance",
								Key:  "in-advance",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(8), // changed price 5 -> 8
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1W")),
						},
					},
				},
			},
		},
		// Let's update in-arrears item
		&patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
		},
		&patch.PatchAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "first-phase",
						ItemKey:  "in-arrears",
						RateCard: &productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "in-arrears",
								Key:  "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(7), // changed price 5 -> 7
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1W")),
						},
					},
				},
			},
		},
	}, s.timingImmediate())
	s.NoError(err)

	// Now let's synchronize the subscription

	asOf := s.mustParseTime("2024-01-03T12:00:00Z")
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, asOf))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 7,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](8),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-01-08T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-08T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-01-22T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-22T00:00:00Z"),
					End:   s.mustParseTime("2024-01-29T00:00:00Z"),
				},
				// As these are in advance items, we also generate them for the next Billing Period (from 2024-01-29 to 2024-02-26)
				{
					Start: s.mustParseTime("2024-01-29T00:00:00Z"),
					End:   s.mustParseTime("2024-02-05T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-05T00:00:00Z"),
					End:   s.mustParseTime("2024-02-12T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-12T00:00:00Z"),
					End:   s.mustParseTime("2024-02-19T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-19T00:00:00Z"),
					End:   s.mustParseTime("2024-02-26T00:00:00Z"),
				},
			},
			// in-advance items are invoiced immediately when change happens
			InvoiceAt: mo.Some([]time.Time{
				// In Advance Items are invoicable at the start of the Billing Period (even if thats before the start of their creation / service period)
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
			}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-29T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](7),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-01-08T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-08T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-01-22T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-22T00:00:00Z"),
					End:   s.mustParseTime("2024-01-29T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
				s.mustParseTime("2024-01-29T00:00:00Z"),
			}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestAlignedSubscriptionCancellation() {
	ctx := s.Context
	startTime := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(startTime)
	defer clock.UnFreeze()

	// Given
	//	a subscription with two phases, first is a trial, second is a regular phase, that has been already sinced
	// When
	//  we cancel said subscription during the trial phase
	// Then
	//  items of future phases should be removed

	// Let's create the initial subscription
	subView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "trial",
						Key:      "trial",
						Duration: lo.ToPtr(testutils.GetISODuration(s.T(), "P1M")),
					},
					// TODO[OM-1031]: let's add discount handling (as this could be a 100% discount for the first month)
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        s.APIRequestsTotalFeature.Key,
								Name:       s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "default",
						Key:      "default",
						Duration: nil,
					},
					// TODO[OM-1031]: 50% discount
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        s.APIRequestsTotalFeature.Key,
								Name:       s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	// Let's advane the clock a minute
	clock.FreezeTime(clock.Now().Add(time.Minute))

	// Let's synchronize the subscription until well into the second phase
	syncUntil := startTime.AddDate(0, 3, 0) // 3 months should suffice
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, syncUntil))

	// Let's check the invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	// Trial isn't synchronized as its a free trial...
	// Let's check the default phase
	s.expectLines(gatheringInvoice, subView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "default",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(5)})),
			Periods: []billing.Period{
				{
					Start: startTime.AddDate(0, 1, 0),
					End:   startTime.AddDate(0, 2, 0),
				},
				{
					Start: startTime.AddDate(0, 2, 0),
					End:   startTime.AddDate(0, 3, 0),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				startTime.AddDate(0, 2, 0),
				startTime.AddDate(0, 3, 0),
			}),
		},
	})

	// Let's cancel the subscription a day later
	cancelAt := clock.Now().Add(time.Hour * 24)

	clock.FreezeTime(cancelAt)
	sub, err := s.SubscriptionService.Cancel(ctx, subView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)

	subView, err = s.SubscriptionService.GetView(ctx, sub.NamespacedID)
	s.NoError(err)

	// Let's synchronize the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, syncUntil))

	// Let's validate that every line was canceled
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
}

func (s *SubscriptionHandlerTestSuite) TestAlignedSubscriptionProgressiveBillingCancellation() {
	ctx := s.Context
	startTime := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(startTime)
	defer clock.UnFreeze()

	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing = billing.InvoicingConfig{
			AutoAdvance:        true,
			DraftPeriod:        isodate.MustParse(s.T(), "P0D"),
			ProgressiveBilling: true,
		}

		s.True(profile.Default)
	})
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2023-01-01T00:00:00Z"))

	// Given
	//	a subscription with one phase, with an usage-based rate card that has been already sinced
	//  we have already progressively billed the line for a day
	// When
	//  we cancel said subscription during the first billing period
	// Then
	//  The remaining part of the billing period should be invoiced
	//  The gathering invoice should be deleted

	testPrice := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
			{
				UpToAmount: nil,
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
		},
	})

	// Let's create the initial subscription
	subView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "default",
						Key:      "default",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        s.APIRequestsTotalFeature.Key,
								Name:       s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
								Price:      testPrice,
							},
							BillingCadence: isodate.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	// Let's synchronize the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, clock.Now().Add(time.Minute))) // time is frozen to start time (syncing in arrears upto which would sync nothing)

	// Let's check the invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	// Trial isn't synchronized as its a free trial...
	// Let's check the default phase
	s.expectLines(gatheringInvoice, subView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "default",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Price: mo.Some(testPrice),
			Periods: []billing.Period{
				{
					Start: startTime,
					End:   startTime.AddDate(0, 1, 0),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				startTime.AddDate(0, 1, 0),
			}),
		},
	})

	// Given we already have a progressively billed line/invoice for a day
	// Let's advane the clock a day
	progressiveBilledAt := startTime.Add(time.Hour * 24)
	clock.FreezeTime(progressiveBilledAt)

	createdInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.CustomerID{
			Namespace: s.Namespace,
			ID:        s.Customer.ID,
		},
		AsOf: &progressiveBilledAt,
	})
	s.NoError(err)
	s.Len(createdInvoices, 1)
	createdInvoice := createdInvoices[0]

	// Let's check the invoice
	s.populateChildIDsFromParents(&createdInvoice)
	s.DebugDumpInvoice("partial invoice", createdInvoice)

	s.expectLines(createdInvoice, subView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "default",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Price: mo.Some(testPrice),
			Periods: []billing.Period{
				{
					Start: startTime,
					End:   startTime.AddDate(0, 0, 1),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				startTime.AddDate(0, 0, 1),
			}),
		},
	})

	// Let's fetch the gathering invoice again
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice - after progressive billing", gatheringInvoice)

	s.expectLines(gatheringInvoice, subView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "default",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Price: mo.Some(testPrice),
			Periods: []billing.Period{
				{
					Start: startTime.AddDate(0, 0, 1),
					End:   startTime.AddDate(0, 1, 0),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				startTime.AddDate(0, 1, 0),
			}),
		},
	})

	// When canceling the subscription, only the remaining part of the billing period should be invoiced
	// Let's cancel the subscription a few ms later, to make sure that the remaining gathering line is empty
	// (this tests if we are fast enought we are still handling the deletion gracefully)
	cancelAt := progressiveBilledAt.Add(10 * time.Millisecond)

	clock.FreezeTime(cancelAt)
	sub, err := s.SubscriptionService.Cancel(ctx, subView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)

	subView, err = s.SubscriptionService.GetView(ctx, sub.NamespacedID)
	s.NoError(err)

	// Event delivery is async, so we need to advance the clock a bit
	clock.FreezeTime(clock.Now().Add(time.Second))
	// Let's synchronize the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, clock.Now()))

	// Let's validate that the gathering invoice is gone too
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceOneTimeFeeSyncing() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single one-time fee in advance
	// When
	//  we we provision the lines
	// Then
	//  the gathering invoice will contain the generated item

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
				Version:  0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsOneTimeFeeSyncing() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single one-time fee in arrears with a shifted billing anchor
	// When
	//  we we provision the lines
	// Then
	//  there will be no gathering invoice, as we don't know what is in arrears

	// When
	//  we cancel the subscription
	// Then
	//  the gathering invoice will contain the generated item schedule to the cancellation's timestamp

	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
						},
					},
				},
			},
		},
	}

	plan, err := s.PlanService.CreatePlan(ctx, planInput)
	s.NoError(err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(clock.Now()),
			},
			Name: "subs-1",
		},
		BillingAnchor: lo.ToPtr(s.mustParseTime("2023-12-15T00:00:00Z")),
		Namespace:     s.Namespace,
		CustomerID:    s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	// let's cancel the subscription
	cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")

	subs, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Custom: &cancelAt,
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subs.NamespacedID)
	s.NoError(err)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears",
				Version:  0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdate() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	// When
	//  when we add a new phase, that disrupts the period of previous items with a new usage based price for the same feature
	// Then
	//  then the gathering invoice is updated, the period of the previous items are updated accordingly

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: isodate.MustParse(s.T(), "P2D"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		// we'll have the single line in the first phase truncated to its 2 day length
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-03T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-03T00:00:00Z")}),
		},
		// We'll have one line for the second phase that gets aligned to the billing anchor
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-03T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdateDraftInvoice() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  a draft invoice has been created.
	// When
	//  we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for the same feature
	// Then
	//  the gathering invoice is updated, the period of the previous items are updated accordingly in the draft invoice
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	// we sync two months so we have lines on gathering
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// Some time has passed, we're syncing the draft invoice
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](11),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-03-01T00:00:00Z")}),
		},
	})

	// To simulate late subscription events (the events not being processed in time by the billing worker)
	// we'll do a time-travel here to work around otherwise system limitations.
	// This is fine and accurate.

	clock.FreezeTime(s.mustParseTime("2024-01-30T00:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: isodate.MustParse(s.T(), "P30D"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Now the time-travel is over, let's reset back to the "present"
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-31T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-03-01T00:00:00Z")}),
		},
	})

	updatedDraftInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - 2nd sync", updatedDraftInvoice)

	s.expectLines(updatedDraftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](11),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-31T00:00:00Z"),
				},
			},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdateIssuedInvoice() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  an issued invoice has been created.
	// When
	//  when we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for
	//  the same feature
	// Then
	//  then the gathering invoice is updated, the finalized invoice doesn't get updated with the periods, but a validation issue is added
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario
	//
	// NOTE: This is variant of the TestUsageBasedGatheringUpdateDraftInvoice so we are keeping the checks at a minimum here

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))
	// We need usage at the period change to trigger the validation issue
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-31T12:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)

	issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, issuedInvoice.Status)
	s.Len(issuedInvoice.ValidationIssues, 0)
	s.DebugDumpInvoice("issued invoice", issuedInvoice)
	s.expectLines(issuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](12),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	// Now lets travel back in time
	clock.FreezeTime(s.mustParseTime("2024-01-30T00:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: isodate.MustParse(s.T(), "P30D"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// gathering invoice
	s.DebugDumpInvoice("gathering invoice - 2nd sync", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	updatedIssuedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: issuedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("issued invoice - 2nd sync", updatedIssuedInvoice)

	s.expectLines(updatedIssuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](12),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"), // This is not updated, which is what we want
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedUpdateWithLineSplits() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have progressive billing enalbed
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  invoice1 has been created for 2024-01-01T00:00:00Z - 2024-01-15T00:00:00Z, gets issued
	//  invoice2 has been created for 2024-01-15T00:00:00Z - 2024-01-18T00:00:00Z, remains in draft state
	// When
	//  when we add a new phase at 2024-01-10T00:00:00Z, that disrupts the period of previous items with a
	// new usage based qty due to the period changes for the same feature
	// Then
	//  then the gathering invoice is updated, the period of the previous items are updated accordingly in the draft invoice
	//  invoice1 remains the same, but a validation error has been added
	//  invoice2's line gets deleted, and the invoice goes to deleted state, as it doesn't have any line items
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), but smaller differences
	// (minutes) can happen due to async nature of processing, thus we need to handle these scenarios

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-12T09:30:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-15T11:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 7, s.mustParseTime("2024-01-18T12:30:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 11, s.mustParseTime("2024-01-29T00:00:00Z"))

	s.enableProgressiveBilling()

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// invoice 1: issued invoice creation
	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
	draftInvoices1, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
	})
	s.NoError(err)
	s.Len(draftInvoices1, 1)

	invoice1, err := s.BillingService.ApproveInvoice(ctx, draftInvoices1[0].InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, invoice1.Status)

	s.populateChildIDsFromParents(&invoice1)
	s.DebugDumpInvoice("issued invoice1", invoice1)

	s.expectLines(invoice1, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](2),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
	})

	clock.FreezeTime(s.mustParseTime("2024-01-18T00:00:00Z"))

	// invoice 2: draft invoice creation
	draftInvoices2, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-18T00:00:00Z")),
	})
	s.NoError(err)
	s.Len(draftInvoices2, 1)

	draftInvoice2 := draftInvoices2[0]
	s.populateChildIDsFromParents(&draftInvoice2)
	s.DebugDumpInvoice("draft invoice2", draftInvoice2)
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice2.Status)

	s.expectLines(draftInvoice2, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](3),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-01-18T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-18T00:00:00Z")}),
		},
	})

	// gathering invoice checks
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-18T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-02-01T00:00:00Z"),
				s.mustParseTime("2024-03-01T00:00:00Z"),
			}),
		},
	})
	clock.FreezeTime(s.mustParseTime("2024-01-09T12:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: isodate.MustParse(s.T(), "P10D"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())

	s.NoError(err)
	s.NotNil(updatedSubsView)

	// THEN
	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-01-18T00:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-11T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-02-01T00:00:00Z"),
				s.mustParseTime("2024-03-01T00:00:00Z"),
			}),
		},
	})

	// invoice 1 (issued) checks
	updatedIssuedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoice1.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedIssuedInvoice)
	s.DebugDumpInvoice("invoice1 (issued) - 2nd sync", updatedIssuedInvoice)

	// remains the same
	s.expectLines(updatedIssuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](2),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
	})

	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])

	// invoice 2 (draft) checks
	updatedDraftInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice2.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedDraftInvoice)
	s.DebugDumpInvoice("draft invoice2 - 2nd sync", updatedDraftInvoice)
	s.Len(updatedDraftInvoice.Lines.OrEmpty(), 0)
	s.Equal(billing.InvoiceStatusDeleted, updatedDraftInvoice.Status)
}

func (s *SubscriptionHandlerTestSuite) TestGatheringManualEditSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	// When
	//  we have the gathering invoice created, and update an item (manually)
	// Then
	//  resyncing the subscription would not cause the item to be upserted again

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: gatheringInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			line := s.getLineByChildID(*invoice, fmt.Sprintf("%s/first-phase/in-advance/v[0]/period[0]", subsView.Subscription.ID))

			line.FlatFee.PaymentTerm = productcatalog.InArrearsPaymentTerm
			line.Period = billing.Period{
				Start: line.Period.Start.Add(time.Hour),
				End:   line.Period.End.Add(time.Hour),
			}
			line.InvoiceAt = line.Period.End
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

	// Then the line should not be updated
	invoiceLine := s.getLineByChildID(gatheringInvoice, *updatedLine.ChildUniqueReferenceID)
	s.True(invoiceLine.LineBase.Equal(updatedLine.LineBase), "line should not be updated")
}

func (s *SubscriptionHandlerTestSuite) TestSplitLineManualEditSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProgressiveBilling()

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2024-01-01T10:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	//  we have the gathering invoice created
	//  we have a draft invoice with a split line
	// When
	//  the item on the draft invoice gets updated (manually)
	// Then
	//  editing the subscription will update fields, but period will be managed by the sync to ensure consistency between line and parent

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  s.APIRequestsTotalFeature.Key,
						Name: "ubp",
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						}),
						FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
						FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	// lets sync for 2 months so we have lines on gathering
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice - pre invoicing", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)
	draftInvoice := draftInvoices[0]

	s.DebugDumpInvoice("draft invoice", draftInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: draftInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Len(lines, 1)

			line := lines[0]

			line.Name = "test"
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.NotNil(updatedLine)

	clock.FreezeTime(s.mustParseTime("2024-01-10T00:00:00Z"))
	_, err = s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subsView.Subscription.NamespacedID)
	s.NoError(err)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.T().Log("-> Subscription canceled")

	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	resyncedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: editedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - after sync", resyncedInvoice)

	// Then the line should not be updated
	s.Len(resyncedInvoice.Lines.OrEmpty(), 1)
	resyncedInvoiceLine := resyncedInvoice.Lines.OrEmpty()[0]

	// Field updates are supported for manually managed lines
	s.Equal(resyncedInvoiceLine.LineBase.Name, updatedLine.Name)
	// Period however is managed by the sync to ensure consistency between line and parent (update endpoint does the filtering)
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-10T00:00:00Z"),
	}, resyncedInvoiceLine.Period)
}

func (s *SubscriptionHandlerTestSuite) TestGatheringManualDeleteSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	// When
	//  we have the gathering invoice created, and delete an item (manually)
	// Then
	//  resyncing the subscription would not cause the item to be upserted again

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	var updatedLine *billing.Line

	childUniqueReferenceID := fmt.Sprintf("%s/first-phase/in-advance/v[0]/period[0]", subsView.Subscription.ID)

	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: gatheringInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			line := s.getLineByChildID(*invoice, childUniqueReferenceID)

			line.DeletedAt = lo.ToPtr(clock.Now())
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
		IncludeDeletedLines: true,
	})

	updatedLineFromEditedInvoice := s.getLineByChildID(editedInvoice, childUniqueReferenceID)
	s.NotNil(updatedLineFromEditedInvoice.DeletedAt)
	s.Equal(billing.ManuallyManagedLine, updatedLineFromEditedInvoice.ManagedBy)

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

	// Then the line should not be recreated
	s.expectNoLineWithChildID(gatheringInvoice, *updatedLine.ChildUniqueReferenceID)
}

func (s *SubscriptionHandlerTestSuite) TestSplitLineManualDeleteSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProgressiveBilling()

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2024-01-01T10:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	//  we have the gathering invoice created
	//  we have a draft invoice with a split line
	// When
	//  the item on the draft invoice gets deleted (manually)
	// Then
	//  editing the subscription would not cause the item to be recreated, but periods are updated

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  s.APIRequestsTotalFeature.Key,
						Name: "ubp",
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						}),
						FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
						FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice - pre invoicing", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)
	draftInvoice := draftInvoices[0]

	s.DebugDumpInvoice("draft invoice", draftInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: draftInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Len(lines, 1)

			line := lines[0]

			line.DeletedAt = lo.ToPtr(clock.Now())

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.NotNil(updatedLine)

	clock.FreezeTime(s.mustParseTime("2024-01-10T00:00:00Z"))
	_, err = s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subsView.Subscription.NamespacedID)
	s.NoError(err)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.T().Log("-> Subscription canceled")

	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	resyncedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: editedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll.SetDeletedLines(true),
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - after sync", resyncedInvoice)

	// The line should still be deleted
	s.Len(resyncedInvoice.Lines.OrEmpty(), 1)

	line := resyncedInvoice.Lines.OrEmpty()[0]
	s.NotNil(line.DeletedAt)
	// Period is updated
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-10T00:00:00Z"),
	}, line.Period)

	s.NotNil(line.SplitLineHierarchy)
	parentGroup := line.SplitLineHierarchy.Group
	// Parent's period is in sync with the child
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-10T00:00:00Z"),
	}, parentGroup.ServicePeriod)
	s.Equal(fmt.Sprintf("%s/first-phase/api-requests-total/v[0]/period[0]", subsView.Subscription.ID), *parentGroup.UniqueReferenceID)
}

func (s *SubscriptionHandlerTestSuite) TestRateCardTaxSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have tax information set in the rate card
	// When
	//  we synchronize the subscription phases
	// Then
	//  the gathering invoice will contain the tax details

	taxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_10000000",
		},
	}

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-arrears",
						Name: "in-arrears",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InArrearsPaymentTerm,
						}),
						TaxConfig: taxConfig,
					},
					BillingCadence: isodate.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	lines := gatheringInvoice.Lines.OrEmpty()
	for _, line := range lines {
		s.Equal(taxConfig, line.TaxConfig)
	}

	// Given we edit the subscription the tax config is carried over to the lines

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			TaxConfig:      taxConfig,
			BillingCadence: lo.ToPtr(isodate.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after edit", gatheringInvoice)

	lines = gatheringInvoice.Lines.OrEmpty()
	for _, line := range lines {
		s.Equal(taxConfig, line.TaxConfig)
	}
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceInstantBillingOnSubscriptionCreation() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an in advance fee
	// When
	//  we start the subscription
	// Then
	//  the gathering invoice will automatically be invoiced so that the in advance fee is billed (those are always flat fees)
	//
	// Note that the UBP line is not synced because the subscription is not active yet

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, subsView, s.mustParseTime("2024-01-01T00:00:00Z")))

	// in-arrears lines wont get synced with this deadline so we'll only have the in advance line on the draft invoice
	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Customers: []string{s.Customer.ID},
		Expand:    billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.Len(invoices.Items, 1)

	instantInvoice := invoices.Items[0]
	s.DebugDumpInvoice("instant invoice", instantInvoice)

	// Instant invoice should have the in advance fee
	s.expectLines(instantInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceInstantBillingOnSubscriptionCreationWithSubscriptionStartInFuture() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z")) // This will be the future

	// Given
	//  we have a subscription with a single phase with an in advance fee
	// When
	//  we start the subscription in the future
	// Then
	//  we'll have the lines on the gathering invoice
	//
	// Note that the UBP line is not synced because the subscription is not active yet

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	clock.FreezeTime(s.mustParseTime("2024-01-20T00:00:00Z")) // This will be the present

	s.NoError(s.Handler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, subsView, clock.Now()))

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Customers: []string{s.Customer.ID},
		Expand:    billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.Len(invoices.Items, 1)

	gatheringInvoice := invoices.Items[0]

	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	// Gathering invoice should have the UBP line
	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) expectValidationIssueForLine(line *billing.Line, issue billing.ValidationIssue) {
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
	s.Equal(SubscriptionSyncComponentName, issue.Component)
	s.Equal(fmt.Sprintf("lines/%s", line.ID), issue.Path)
}

func (s *SubscriptionHandlerTestSuite) TestDiscountSynchronization() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
						Discounts: productcatalog.Discounts{
							Percentage: &productcatalog.PercentageDiscount{
								Percentage: models.NewPercentage(100),
							},
						},
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
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
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, subsView, clock.Now().Add(time.Minute))) // time is frozen to start time (syncing in arrears upto which would sync nothing, and we want both the instant invoice for in advance as well as the gathering for UBP)

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Customers: []string{s.Customer.ID},
		Expand:    billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.Len(invoices.Items, 2)

	var gatheringInvoice *billing.Invoice
	var instantInvoice *billing.Invoice

	for _, invoice := range invoices.Items {
		if invoice.Status == billing.InvoiceStatusGathering {
			gatheringInvoice = &invoice
			continue
		}

		instantInvoice = &invoice
	}

	s.NotNil(gatheringInvoice, "gathering invoice should be present")
	s.NotNil(instantInvoice, "instant invoice should be present")

	s.DebugDumpInvoice("gathering invoice", *gatheringInvoice)
	s.DebugDumpInvoice("instant invoice", *instantInvoice)

	s.expectLines(*gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		// Gathering invoice should have the UBP line
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
		// And next Billing Period's in advance line
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				PeriodMin: 1,
				PeriodMax: 1,
				Version:   0,
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})

	// Instant invoice should have the in advance fee
	s.expectLines(*instantInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
	})

	// The advance fee should have 100% discount
	line := instantInvoice.Lines.OrEmpty()[0]
	s.Equal(float64(6), line.Discounts.Amount[0].Amount.InexactFloat64())
}

func (s *SubscriptionHandlerTestSuite) TestUseUsageBasedFlatFeeLinesCompatibility() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: isodate.MustParse(s.T(), "P1M"),
				},
			},
		},
	})

	// Given:
	// - a flat fee has been created as part of the synchronization
	// When:
	// - we enable the new feature flag
	// Then
	// - the resynchronization should not replace existing usage based lines
	// - new lines syncronized should receive the usage based line approach

	featureFlagSwitchoverAt := s.mustParseTime("2024-02-01T00:00:00Z")
	clock.FreezeTime(featureFlagSwitchoverAt)

	// let provision the lines in the old way
	s.Handler.featureFlags.UseUsageBasedFlatFeeLines = false
	defer func() {
		s.Handler.featureFlags.UseUsageBasedFlatFeeLines = false
	}()

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

	invoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", invoice)

	lines := invoice.Lines.OrEmpty()
	s.Len(lines, 2)
	s.Equal(billing.InvoiceLineTypeFee, lines[0].Type)
	s.Equal(float64(6), lines[0].FlatFee.PerUnitAmount.InexactFloat64())

	// When we enable the new feature flag
	s.Handler.featureFlags.UseUsageBasedFlatFeeLines = true

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

	invoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", invoice)

	// assert no change
	lines = invoice.Lines.OrEmpty()
	s.Len(lines, 2)
	s.Equal(billing.InvoiceLineTypeFee, lines[0].Type)
	s.Equal(float64(6), lines[0].FlatFee.PerUnitAmount.InexactFloat64())
	firstSyncLineIDs := lo.Map(lines, func(line *billing.Line, _ int) string {
		return line.ID
	})

	// The new line should usage based
	clock.FreezeTime(s.mustParseTime("2024-03-01T00:00:00Z")) // so this matches up with the end of the last prev generated line (the first one) thats why the switchover happens as now we get to generate more lines

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))
	invoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - mixed line setup", invoice)

	linesByType := lo.GroupBy(invoice.Lines.OrEmpty(), func(line *billing.Line) billing.InvoiceLineType {
		return line.Type
	})

	s.Len(linesByType[billing.InvoiceLineTypeFee], 2)
	s.ElementsMatch(firstSyncLineIDs, lo.Map(linesByType[billing.InvoiceLineTypeFee], func(line *billing.Line, _ int) string {
		return line.ID
	}))
	s.Len(linesByType[billing.InvoiceLineTypeUsageBased], 1)
}

func (s *SubscriptionHandlerTestSuite) TestAlignedSubscriptionProratingBehavior() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// Given
	//	a subscription with two phases started, with prorating enabled
	//   the first phase is 2 weeks long, the second phase is unlimited
	//   the phases have in advance, in arrears and usage based lines
	// When
	//  we cancel the subscription asof 2025-03-01
	//  we syncronize the subscription data up to 2025-03-01
	// Then
	//  The in-advance and in arrears lines should be prorated for the first phase
	//  The usage based line's price is intact, only the period length is changed
	//  The second phase's lines are aligned to the phase's start (as we don't have custom anchor set)
	//  The second phase's in-advance and in arreas lines are not prorated (for the 2nd half period), as we only support prorating due to alignment for now

	// NOTE[implicit behavior]: Handler's prorating logic is disabled before the test execution.

	secondPhase := productcatalog.Phase{
		PhaseMeta: s.phaseMeta("second-phase", ""),
		RateCards: productcatalog.RateCards{
			&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "in-advance",
					Name: "in-advance",
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(5),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
				},
				BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1M")),
			},
			&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "in-arrears",
					Name: "in-arrears",
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(5),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
				BillingCadence: lo.ToPtr(testutils.GetISODuration(s.T(), "P1M")),
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
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
			},
		},
	}

	firstPhase := secondPhase // Note: we are not copying the phase's rate cards, but that's fine
	firstPhase.PhaseMeta = s.phaseMeta("first-phase", "P2W")

	// Let's create the initial subscription
	subView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				firstPhase,
				secondPhase,
			},
		},
	})

	// Let's cancel the subscription asof 2025-03-01
	clock.FreezeTime(s.mustParseTime("2024-03-01T00:00:00Z"))
	_, err := s.SubscriptionService.Cancel(ctx, subView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)

	// Let's refetch the subscription view
	subView, err = s.SubscriptionService.GetView(ctx, subView.Subscription.NamespacedID)
	s.NoError(err)

	// Let's syncrhonize subscription data for 1 month
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subView, s.mustParseTime("2024-03-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subView.Subscription.ID, []expectedLine{
		// January is 31 days, wechange phase after 2 weeks (14 days)
		// 5 * 14/31 = 2.258... which we round to 2.26
		// First phase lines
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(2.26),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears",
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(2.26),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "api-requests-total",
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(10)})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
		// We align billing to the 1st of month, so we'll prorate the first iteration
		// January is 31 days, 31 - 14 = 17 days, 5 * 17/31 = 2.741... which we round to 2.74
		// Second phase lines
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-advance",
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(2.74),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-01-15T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-advance",
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(5.0),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-arrears",
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(2.74),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-arrears",
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Qty:       mo.Some(1.0),
			UnitPrice: mo.Some(5.0),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-03-01T00:00:00Z")}),
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "api-requests-total",
				PeriodMin: 0,
				PeriodMax: 1,
			},
			// UBP does not need prorating on price due to period being shorter
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(10.0)})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-15T00:00:00Z"),
					End:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-02-01T00:00:00Z"),
					End:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z"), s.mustParseTime("2024-03-01T00:00:00Z")}),
		},
	})
}

type expectedLine struct {
	Matcher   lineMatcher
	Qty       mo.Option[float64]
	UnitPrice mo.Option[float64]
	Price     mo.Option[*productcatalog.Price]
	Periods   []billing.Period
	InvoiceAt mo.Option[[]time.Time]
}

func (s *SubscriptionHandlerTestSuite) expectLines(invoice billing.Invoice, subscriptionID string, expectedLines []expectedLine) {
	s.T().Helper()

	lines := invoice.Lines.OrEmpty()

	existingLineChildIDs := lo.Map(lines, func(line *billing.Line, _ int) string {
		return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID)
	})

	expectedLineIds := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subscriptionID)
	}))

	s.ElementsMatch(expectedLineIds, existingLineChildIDs)

	for _, expectedLine := range expectedLines {
		childIDs := expectedLine.Matcher.ChildIDs(subscriptionID)
		for idx, childID := range childIDs {
			line, found := lo.Find(lines, func(line *billing.Line) bool {
				return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID) == childID
			})
			s.Truef(found, "line not found with child id %s", childID)
			s.NotNil(line)

			if expectedLine.Qty.IsPresent() {
				if line.Type == billing.InvoiceLineTypeFee {
					if line.FlatFee == nil {
						s.Failf("flat fee line not found", "line not found with child id %s", childID)
					} else {
						s.Equal(expectedLine.Qty.OrEmpty(), line.FlatFee.Quantity.InexactFloat64(), "%s: quantity", childID)
					}
				} else {
					if line.UsageBased == nil {
						s.Failf("usage based line not found", "line not found with child id %s", childID)
					} else if line.UsageBased.Quantity == nil {
						s.Failf("usage based line quantity not found", "line not found with child id %s", childID)
					} else {
						s.Equal(expectedLine.Qty.OrEmpty(), line.UsageBased.Quantity.InexactFloat64(), "%s: quantity", childID)
					}
				}
			}

			if expectedLine.UnitPrice.IsPresent() {
				s.Equal(billing.InvoiceLineTypeFee, line.Type, "%s: line type", childID)
				s.Equal(expectedLine.UnitPrice.OrEmpty(), line.FlatFee.PerUnitAmount.InexactFloat64(), "%s: unit price \n out: %+v", childID, line)
			}

			if expectedLine.Price.IsPresent() {
				s.Equal(billing.InvoiceLineTypeUsageBased, line.Type, "%s: line type", childID)
				s.Equal(*expectedLine.Price.OrEmpty(), *line.UsageBased.Price, "%s: price", childID)
			}

			s.Equal(expectedLine.Periods[idx].Start, line.Period.Start, "%s: period start", childID)
			s.Equal(expectedLine.Periods[idx].End, line.Period.End, "%s: period end", childID)

			if expectedLine.InvoiceAt.IsPresent() {
				s.Equal(expectedLine.InvoiceAt.OrEmpty()[idx], line.InvoiceAt, "%s: invoice at", childID)
			}
		}
	}
}

type lineMatcher interface {
	ChildIDs(subsID string) []string
}

type recurringLineMatcher struct {
	PhaseKey  string
	ItemKey   string
	Version   int
	PeriodMin int
	PeriodMax int
}

func (m recurringLineMatcher) ChildIDs(subsID string) []string {
	out := []string{}
	for periodID := m.PeriodMin; periodID <= m.PeriodMax; periodID++ {
		out = append(out, fmt.Sprintf("%s/%s/%s/v[%d]/period[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version, periodID))
	}

	return out
}

type oneTimeLineMatcher struct {
	PhaseKey string
	ItemKey  string
	Version  int
}

func (m oneTimeLineMatcher) ChildIDs(subsID string) []string {
	return []string{fmt.Sprintf("%s/%s/%s/v[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version)}
}

// helpers

//nolint:unparam
func (s *SubscriptionHandlerTestSuite) phaseMeta(key string, duration string) productcatalog.PhaseMeta {
	out := productcatalog.PhaseMeta{
		Key:  key,
		Name: key,
	}

	if duration != "" {
		out.Duration = lo.ToPtr(isodate.MustParse(s.T(), duration))
	}

	return out
}

func (s *SubscriptionHandlerTestSuite) enableProgressiveBilling() {
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	})
}

func (s *SubscriptionHandlerTestSuite) updateProfile(modify func(profile *billing.Profile)) {
	defaultProfile, err := s.BillingService.GetDefaultProfile(s.Context, billing.GetDefaultProfileInput{
		Namespace: s.Namespace,
	})
	s.NoError(err)

	modify(defaultProfile)

	defaultProfile.AppReferences = nil

	_, err = s.BillingService.UpdateProfile(s.Context, billing.UpdateProfileInput(defaultProfile.BaseProfile))
	s.NoError(err)
}

type subscriptionAddItem struct {
	PhaseKey       string
	ItemKey        string
	Price          *productcatalog.Price
	BillingCadence *isodate.Period
	FeatureKey     string
	TaxConfig      *productcatalog.TaxConfig
}

func (i subscriptionAddItem) AsPatch() subscription.Patch {
	var rc productcatalog.RateCard

	meta := productcatalog.RateCardMeta{
		Name:       i.ItemKey,
		Key:        i.ItemKey,
		Price:      i.Price,
		FeatureKey: lo.EmptyableToPtr(i.FeatureKey),
		TaxConfig:  i.TaxConfig,
	}

	switch {
	case i.Price == nil:
		rc = &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: i.BillingCadence,
		}
	case i.Price.Type() == productcatalog.FlatPriceType:
		rc = &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: i.BillingCadence,
		}
	default:
		rc = &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: *i.BillingCadence,
		}
	}

	return patch.PatchAddItem{
		PhaseKey: i.PhaseKey,
		ItemKey:  i.ItemKey,
		CreateInput: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: i.PhaseKey,
					ItemKey:  i.ItemKey,
					RateCard: rc,
				},
			},
		},
	}
}

func (s *SubscriptionHandlerTestSuite) generatePeriods(startStr, endStr string, cadenceStr string, n int) []billing.Period { //nolint: unparam
	start := s.mustParseTime(startStr)
	end := s.mustParseTime(endStr)
	cadence := isodate.MustParse(s.T(), cadenceStr)

	out := []billing.Period{}

	for n != 0 {
		out = append(out, billing.Period{
			Start: start,
			End:   end,
		})

		start, _ = cadence.AddTo(start)
		end, _ = cadence.AddTo(end)

		n--
	}
	return out
}

// populateChildIDsFromParents copies over the child ID from the parent line, if it's not already set
// as line splitting doesn't set the child ID on child lines to prevent conflicts if multiple split lines
// end up on a single invoice.
func (s *SubscriptionHandlerTestSuite) populateChildIDsFromParents(invoice *billing.Invoice) {
	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID == nil && line.SplitLineGroupID != nil {
			line.ChildUniqueReferenceID = line.SplitLineHierarchy.Group.UniqueReferenceID
		}
	}
}

// helpers

func (s *SubscriptionHandlerTestSuite) createSubscriptionFromPlanPhases(phases []productcatalog.Phase) subscription.SubscriptionView {
	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: isodate.MustParse(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: phases,
		},
	}

	return s.createSubscriptionFromPlan(planInput)
}

func (s *SubscriptionHandlerTestSuite) createSubscriptionFromPlan(planInput plan.CreatePlanInput) subscription.SubscriptionView {
	ctx := s.Context

	plan, err := s.PlanService.CreatePlan(ctx, planInput)
	s.NoError(err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(clock.Now()),
			},
			Name: "subs-1",
		},
		Namespace:  s.Namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)
	return subsView
}

func (s *SubscriptionHandlerTestSuite) timingImmediate() subscription.Timing {
	return subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	}
}

func getPhaseByKey(t *testing.T, subsView subscription.SubscriptionView, key string) subscription.SubscriptionPhaseView {
	for _, phase := range subsView.Phases {
		if phase.SubscriptionPhase.Key == key {
			return phase
		}
	}

	t.Fatalf("phase with key %s not found", key)
	return subscription.SubscriptionPhaseView{}
}

func (s *SubscriptionHandlerTestSuite) gatheringInvoice(ctx context.Context, namespace string, customerID string) billing.Invoice {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 1)
	return invoices.Items[0]
}

func (s *SubscriptionHandlerTestSuite) expectNoGatheringInvoice(ctx context.Context, namespace string, customerID string) {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 0)
}

func (s *SubscriptionHandlerTestSuite) enableProrating() {
	s.Handler.featureFlags.EnableFlatFeeInAdvanceProrating = true
	s.Handler.featureFlags.EnableFlatFeeInArrearsProrating = true
}

func (s *SubscriptionHandlerTestSuite) getLineByChildID(invoice billing.Invoice, childID string) *billing.Line {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			return line
		}
	}

	s.Failf("line not found", "line with child id %s not found", childID)

	return nil
}

func (s *SubscriptionHandlerTestSuite) expectNoLineWithChildID(invoice billing.Invoice, childID string) {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			s.Failf("line found", "line with child id %s found", childID)
		}
	}
}
