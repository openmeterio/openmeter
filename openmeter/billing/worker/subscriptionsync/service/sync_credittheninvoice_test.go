package service

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreditThenInvoiceTestSuite struct {
	SuiteBase

	BalanceQuerier ledger.BalanceQuerier
	LedgerResolver *ledgerresolvers.AccountResolver
}

func TestCreditThenInvoiceScenarios(t *testing.T) {
	suite.Run(t, new(CreditThenInvoiceTestSuite))
}

func (s *CreditThenInvoiceTestSuite) SetupSuite() {
	s.SuiteBase.SetupSuite()

	logger := slog.Default()

	ledgerDeps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.NoError(err)

	s.BalanceQuerier = ledgerDeps.HistoricalLedger
	s.LedgerResolver = ledgerDeps.ResolversService

	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerDeps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: ledgerDeps.ResolversService,
			AccountCatalog: ledgerDeps.AccountService,
			BalanceQuerier: ledgerDeps.HistoricalLedger,
		},
	})

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:             s.DBClient,
		Logger:             logger,
		BillingService:     s.BillingService,
		FeatureService:     s.FeatureService,
		StreamingConnector: s.MockStreamingConnector,
		FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(
			ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			collectorService,
		),
		CreditPurchaseHandler: ledgerchargeadapter.NewCreditPurchaseHandler(
			ledgerDeps.HistoricalLedger,
			ledgerDeps.HistoricalLedger,
			ledgerDeps.ResolversService,
			ledgerDeps.AccountService,
		),
		UsageBasedHandler: ledgerchargeadapter.NewUsageBasedHandler(
			ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			collectorService,
		),
	})
	s.NoError(err)

	s.Charges = stack.ChargesService

	service, err := New(Config{
		BillingService:          s.BillingService,
		ChargesService:          s.Charges,
		Logger:                  s.Service.logger,
		Tracer:                  s.Service.tracer,
		SubscriptionSyncAdapter: s.Adapter,
		SubscriptionService:     s.SubscriptionService,
		FeatureFlags: FeatureFlags{
			EnableCreditThenInvoice: true,
		},
	})
	s.NoError(err)

	s.Service = service
}

func (s *CreditThenInvoiceTestSuite) BeforeTest(suiteName, testName string) {
	s.SuiteBase.BeforeTest(suiteName, testName)

	_, err := s.LedgerResolver.EnsureBusinessAccounts(s.T().Context(), s.Namespace)
	s.NoError(err)

	_, err = s.LedgerResolver.CreateCustomerAccounts(s.T().Context(), s.Customer.GetID())
	s.NoError(err)
}

func (s *CreditThenInvoiceTestSuite) TestSubscriptionHappyPath() {
	ctx := s.testContext()
	namespace := s.Namespace
	start := s.mustParseTime("2024-01-01T00:00:00.123456Z")
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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
						Duration: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "discounted phase",
						Key:      "discounted-phase",
						Duration: lo.ToPtr(datetime.MustParseDuration(s.T(), "P2M")),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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

	freeTierPhase := s.getPhaseByKey(s.T(), subsView, "free-trial")
	s.Equal(lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")), freeTierPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].Spec.RateCard.GetBillingCadence())

	discountedPhase := s.getPhaseByKey(s.T(), subsView, "discounted-phase")
	var gatheringInvoiceID billing.InvoiceID
	var usageBasedChargeID string
	promotionalCreditAmount := alpacadecimal.NewFromInt(100)
	var startBalances expectedCreditThenInvoiceBalances
	var partialInvoiceBalances expectedCreditThenInvoiceBalances

	s.Run("buy promotional credits", func() {
		// given:
		// - a ledger-backed customer has no credit allocations or invoice bookings
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		// when:
		// - the customer receives promotional credits before subscription invoicing
		s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
			Namespace: namespace,
			Customer:  s.Customer.GetID(),
			Currency:  currencyx.Code(currency.USD),
			Amount:    promotionalCreditAmount,
			At:        start,
		})

		// then:
		// - promotional credits are available in customer FBO
		// - the promotional grant is backed by wash at zero cost basis
		startBalances = expectedCreditThenInvoiceBalances{
			FBOAll:          100,
			FBOPromotional:  100,
			WashAll:         -100,
			WashPromotional: -100,
		}
		s.assertCreditThenInvoiceBalances(startBalances)
	})

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		// given:
		// - a ledger-backed customer has promotional credits but no invoice bookings
		// - the subscription is synchronized through the free trial into the first billable period
		s.assertCreditThenInvoiceBalances(startBalances)

		// when:
		// - the first set of billable items is provisioned
		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

		// then:
		// - billing has a gathering invoice for the charge-backed line
		// - one credit-then-invoice usage-based charge is created
		// - no ledger balances changed during provisioning
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

		expectedPeriod := timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-02-01T00:00:00Z"),
			To:   s.mustParseTime("2024-03-01T00:00:00Z"),
		}
		expectedInvoiceAt := s.mustParseTime("2024-03-01T00:00:00Z")

		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
		s.Equal(expectedPeriod, line.Subscription.BillingPeriod)

		// 1 month free tier + in arrears billing with 1 month cadence
		s.Equal(line.InvoiceAt, expectedInvoiceAt)

		charge := s.mustGetOnlyUsageBasedCharge(ctx, subsView.Subscription.ID)
		usageBasedChargeID = charge.ID
		chargeUpdatedAt := charge.UpdatedAt

		s.assertCreditThenInvoiceUsageBasedCharge(charge, expectedUsageBasedChargeInput{
			Status:         usagebased.StatusCreated,
			ServicePeriod:  expectedPeriod,
			InvoiceAt:      expectedInvoiceAt,
			CustomerID:     s.Customer.ID,
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			Price:          *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(5)}),
			SubscriptionID: subsView.Subscription.ID,
			PhaseID:        discountedPhase.SubscriptionPhase.ID,
			ItemID:         discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID,
		})
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Nil(charge.State.AdvanceAfter)

		s.Equal(billing.LineEngineTypeChargeUsageBased, line.Engine)
		s.Require().NotNil(line.ChargeID)
		s.Equal(charge.ID, *line.ChargeID)

		s.assertCreditThenInvoiceBalances(startBalances)

		// When we advance the clock the invoice doesn't get changed
		clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

		reconciledCharge := s.mustGetOnlyUsageBasedCharge(ctx, subsView.Subscription.ID)

		gatheringInvoice := s.gatheringInvoice(ctx, namespace, s.Customer.ID)
		s.NoError(err)
		gatheringInvoiceID = gatheringInvoice.GetInvoiceID()

		s.DebugDumpInvoice("gathering invoice - 2nd update", gatheringInvoice)

		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(invoiceUpdatedAt, gatheringInvoice.GetUpdatedAt())
		s.Equal(line.GetUpdatedAt(), gatheringLine.GetUpdatedAt())
		s.Equal(charge.ID, reconciledCharge.ID)
		s.Equal(usagebased.StatusCreated, reconciledCharge.Status)
		s.Nil(reconciledCharge.State.CurrentRealizationRunID)
		s.Nil(reconciledCharge.State.AdvanceAfter)
		s.Equal(chargeUpdatedAt, reconciledCharge.UpdatedAt)
		s.assertCreditThenInvoiceBalances(startBalances)
	})

	s.NoError(gatheringInvoiceID.Validate())

	// Progressive billing updates
	s.Run("progressive billing updates", func() {
		// given:
		// - usage arrives for the first half of the billable period
		// - credit-then-invoice charge-backed invoicing uses an internal collection period before auto-approval
		s.MockStreamingConnector.AddSimpleEvent(
			*s.APIRequestsTotalFeature.MeterSlug,
			100,
			s.mustParseTime("2024-02-02T00:00:00Z"))
		clock.FreezeTime(s.mustParseTime("2024-02-15T00:00:01Z"))

		// when:
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

		// then:
		// - the invoice waits for charge collection before it can auto-advance
		// - the invoice line remains linked to the same usage-based charge
		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
		s.assertTotals(invoice.Totals, expectedTotalsInput{
			Amount:       5 * 100,
			CreditsTotal: 100,
			Total:        5*100 - 100,
		})

		s.Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.Equal(billing.LineEngineTypeChargeUsageBased, line.Engine)
		s.Require().NotNil(line.ChargeID)
		s.Equal(usageBasedChargeID, *line.ChargeID)
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-02-15T00:00:00Z"))
		s.Equal(line.Period, timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-02-01T00:00:00Z"),
			To:   s.mustParseTime("2024-02-15T00:00:00Z"),
		})
		s.assertTotals(line.Totals, expectedTotalsInput{
			Amount:       5 * 100,
			CreditsTotal: 100,
			Total:        5*100 - 100,
		})
		s.Require().NotNil(line.OverrideCollectionPeriodEnd)
		s.True(line.OverrideCollectionPeriodEnd.Equal(s.mustParseTime("2024-02-15T00:01:00Z")))
		s.Require().NotNil(invoice.CollectionAt)
		s.True(line.OverrideCollectionPeriodEnd.Equal(*invoice.CollectionAt))

		charge := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
			Namespace: namespace,
			ID:        usageBasedChargeID,
		}, chargesmeta.Expands{chargesmeta.ExpandRealizations})
		s.Equal(usageBasedChargeID, charge.ID)
		s.Equal(usagebased.StatusActivePartialInvoiceWaitingForCollection, charge.Status)
		s.Require().NotNil(charge.State.CurrentRealizationRunID)
		s.Require().NotNil(charge.State.AdvanceAfter)
		s.True(line.OverrideCollectionPeriodEnd.Equal(*charge.State.AdvanceAfter))
		s.Len(charge.Realizations, 1)

		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, currentRun.Type)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, currentRun.InitialType)
		s.Equal(s.mustParseTime("2024-02-15T00:00:00Z"), currentRun.StoredAtLT)
		s.Equal(s.mustParseTime("2024-02-15T00:00:00Z"), currentRun.ServicePeriodTo)
		s.Equal(line.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(100), currentRun.MeteredQuantity)
		s.Equal(promotionalCreditAmount, currentRun.CreditsAllocated.Sum())
		s.Nil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)
		s.assertTotals(currentRun.Totals, expectedTotalsInput{
			Amount:       5 * 100,
			CreditsTotal: 100,
			Total:        5*100 - 100,
		})

		creditAllocationAmount := currentRun.CreditsAllocated.Sum().InexactFloat64()
		partialInvoiceBalances = expectedCreditThenInvoiceBalances{
			FBOAll:             100 - creditAllocationAmount,
			FBOPromotional:     100 - creditAllocationAmount,
			AccruedAll:         creditAllocationAmount,
			AccruedPromotional: creditAllocationAmount,
			WashAll:            -100,
			WashPromotional:    -100,
		}
		s.assertCreditThenInvoiceBalances(partialInvoiceBalances)

		// let's fetch the gathering invoice
		gatheringInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]
		s.Equal(billing.LineEngineTypeChargeUsageBased, gatheringLine.Engine)
		s.Require().NotNil(gatheringLine.ChargeID)
		s.Equal(usageBasedChargeID, *gatheringLine.ChargeID)
		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))
		s.Equal(gatheringLine.ServicePeriod, timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-02-15T00:00:00Z"),
			To:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})

		// TODO[OM-1037]: let's add/change some items of the subscription then expect that the new item appears on the gathering
		// invoice, but the draft invoice is untouched.
	})

	s.Run("subscription cancellation", func() {
		// given:
		// - the subscription is canceled at the end of the current billing period
		clock.FreezeTime(s.mustParseTime("2024-02-20T00:00:00Z"))

		cancelAt := s.mustParseTime("2024-03-01T00:00:00.123456Z")
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

		// when:
		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now()))

		// then:
		// - the remaining gathering line stays charge-managed
		// - charge-managed direct lines do not carry split hierarchies
		gatheringInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(billing.LineEngineTypeChargeUsageBased, gatheringLine.Engine)
		s.Require().NotNil(gatheringLine.ChargeID)
		s.Equal(usageBasedChargeID, *gatheringLine.ChargeID)
		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.ServicePeriod, timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-02-15T00:00:00Z"),
			To:   cancelAt.Truncate(streaming.MinimumWindowSizeDuration),
		})
		s.Equal(gatheringLine.InvoiceAt, cancelAt.Truncate(streaming.MinimumWindowSizeDuration))

		// split group
		s.Nil(gatheringLine.SplitLineHierarchy)
		s.assertCreditThenInvoiceBalances(partialInvoiceBalances)
	})

	s.Run("continue subscription", func() {
		// given:
		// - the canceled subscription is continued before the cancellation takes effect
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

		// when:
		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now()))

		// then:
		// - the gathering line keeps the charge-backed line engine
		// - charge-managed direct lines do not carry split hierarchies
		gatheringInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(billing.LineEngineTypeChargeUsageBased, gatheringLine.Engine)
		s.Require().NotNil(gatheringLine.ChargeID)
		s.Equal(usageBasedChargeID, *gatheringLine.ChargeID)
		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.ServicePeriod, timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-02-15T00:00:00Z"),
			To:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// split group
		s.Nil(gatheringLine.SplitLineHierarchy)
		s.assertCreditThenInvoiceBalances(partialInvoiceBalances)
	})
}

func (s *CreditThenInvoiceTestSuite) TestInArrearsProratingGathering() {
	ctx := s.T().Context()
	namespace := s.Namespace
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()
	s.enableProrating()

	customerEntity := s.Customer
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: namespace,
		Customer:  customerEntity.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P3M"),
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
		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
			Expand:     billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		oneDayLines := lo.Filter(lines, func(line billing.GatheringLine, _ int) bool {
			return line.ServicePeriod.Duration() == time.Hour*24
		})
		s.Len(oneDayLines, 31) // january is 31 days long, and we generate lines for each daily for in arrears price

		for _, line := range oneDayLines {
			s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID, "failed for line %v", line.ID)
			s.Equal(line.Subscription.PhaseID, subsView.Phases[0].SubscriptionPhase.ID, "failed for line %v", line.ID)
			s.Equal(line.Subscription.ItemID, subsView.Phases[0].ItemsByKey["in-arrears"][0].SubscriptionItem.ID, "failed for line %v", line.ID)
			s.Equal(line.InvoiceAt, s.mustParseTime("2024-02-01T00:00:00Z"), "failed for line %v", line.ID)
			s.Equal(line.ServicePeriod, timeutil.ClosedPeriod{
				From: s.mustParseTime("2024-01-01T00:00:00Z").AddDate(0, 0, line.ServicePeriod.From.Day()-1),
				To:   s.mustParseTime("2024-01-01T00:00:00Z").AddDate(0, 0, line.ServicePeriod.From.Day()),
			}, "failed for line %v", line.ID)
			price, err := line.Price.AsFlat()
			s.NoError(err)
			s.Equal(price.Amount.InexactFloat64(), 5.0, "failed for line %v", line.ID)
			s.Equal(price.PaymentTerm, productcatalog.InArrearsPaymentTerm, "failed for line %v", line.ID)
		}

		s.assertCreditThenInvoiceBalances(startBalances)
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

		s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
			Expand:     billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		threeMonthLines := lo.Filter(lines, func(line billing.GatheringLine, _ int) bool {
			return line.ServicePeriod.Duration() != time.Hour*24 // all other lines will be 1 dqy
		})
		s.Len(threeMonthLines, 1)

		flatFeeLine := threeMonthLines[0]
		s.Equal(flatFeeLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(flatFeeLine.InvoiceAt, cancelAt)
		s.Equal(flatFeeLine.ServicePeriod, timeutil.ClosedPeriod{
			From: s.mustParseTime("2024-01-01T00:00:00Z"),
			To:   cancelAt,
		})
		price, err := flatFeeLine.Price.AsFlat()
		s.NoError(err)
		s.Equal(price.Amount.InexactFloat64(), 3.07, "failed for line %v", flatFeeLine.ID)
		s.Equal(price.PaymentTerm, productcatalog.InArrearsPaymentTerm, "failed for line %v", flatFeeLine.ID)
		s.assertCreditThenInvoiceBalances(startBalances)
	})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceGatheringSyncNonBillableAmountProrated() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will only contain the new version of the fee, as the old one was
	//  pro-rated and the total amount is 0

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

	subsView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.assertCreditThenInvoiceBalances(startBalances)

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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Service.SynchronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 1, // as its in-advance, we'll generate the item for the next month too
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:40Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
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

func (s *CreditThenInvoiceTestSuite) TestInAdvanceGatheringSyncNonBillableAmount() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will contain both versions of the fee as we are not
	//  doing any pro-rating logic

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}

	subsView := s.createSubscriptionFromPlan(planInput)

	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.assertCreditThenInvoiceBalances(startBalances)

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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Service.SynchronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:40Z"),
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

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:40Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: mo.Some([]time.Time{
				s.mustParseTime("2024-01-01T00:00:00Z"),
				s.mustParseTime("2024-02-01T00:00:00Z"),
			}),
		},
	})
}

func (s *CreditThenInvoiceTestSuite) TestInArrearsGatheringSyncNonBillableAmount() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

	// Given
	//  we have a subscription with a single phase with a single static fee in arrears
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will contain both versions of the fee as we are not
	//  doing any pro-rating logic

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}

	subsView := s.createSubscriptionFromPlan(planInput)

	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.assertCreditThenInvoiceBalances(startBalances)

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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Service.SynchronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:40Z"),
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

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:40Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			// We'll wait till the end of the billing cadence of the item
			InvoiceAt: mo.Some([]time.Time{s.mustParseTime("2024-02-01T00:00:00Z")}),
		},
	})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceGatheringSyncBillableAmountProrated() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription later
	// Then
	//  the gathering invoice will contain the pro-rated previous fee and the new fee

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

	subsView := s.createSubscriptionFromPlan(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test-plan",
				Version:        1,
				Currency:       currency.USD,
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SynchronizeSubscription(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.assertCreditThenInvoiceBalances(startBalances)

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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Service.SynchronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(0.32), // 10 * 1 / 31
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-02T00:00:00Z"),
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

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(19.35), // 20 * 30 / 31
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-02T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
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

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(20),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
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

func (s *CreditThenInvoiceTestSuite) expectValidationIssueForLine(line *billing.StandardLine, issue billing.ValidationIssue) {
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
	s.Equal(SubscriptionSyncComponentName, issue.Component)
	s.Equal(fmt.Sprintf("lines/%s", line.ID), issue.Path)
}

type expectedCreditThenInvoiceBalances struct {
	FBOAll         float64
	FBOPromotional float64
	FBOInvoice     float64

	OpenReceivableAll         float64
	OpenReceivablePromotional float64
	OpenReceivableInvoice     float64

	AuthorizedReceivableAll         float64
	AuthorizedReceivablePromotional float64
	AuthorizedReceivableInvoice     float64

	AccruedAll         float64
	AccruedPromotional float64
	AccruedInvoice     float64

	WashAll         float64
	WashPromotional float64
	WashInvoice     float64

	EarningsAll float64
}

func (s *CreditThenInvoiceTestSuite) assertCreditThenInvoiceBalances(expected expectedCreditThenInvoiceBalances) {
	s.T().Helper()

	customerID := s.Customer.GetID()
	currencyCode := currencyx.Code(currency.USD)
	promotionalCostBasis := alpacadecimal.Zero
	invoiceCostBasis := alpacadecimal.NewFromInt(1)
	allCostBasis := mo.None[*alpacadecimal.Decimal]()
	promotional := mo.Some(&promotionalCostBasis)
	invoice := mo.Some(&invoiceCostBasis)

	s.Equal(expected.FBOAll, s.mustCustomerFBOBalance(customerID, currencyCode, allCostBasis).InexactFloat64(), "FBO all-cost-basis balance")
	s.Equal(expected.FBOPromotional, s.mustCustomerFBOBalance(customerID, currencyCode, promotional).InexactFloat64(), "FBO promotional balance")
	s.Equal(expected.FBOInvoice, s.mustCustomerFBOBalance(customerID, currencyCode, invoice).InexactFloat64(), "FBO invoice-cost-basis balance")

	s.Equal(expected.OpenReceivableAll, s.mustCustomerReceivableBalance(customerID, currencyCode, allCostBasis, ledger.TransactionAuthorizationStatusOpen).InexactFloat64(), "open receivable all-cost-basis balance")
	s.Equal(expected.OpenReceivablePromotional, s.mustCustomerReceivableBalance(customerID, currencyCode, promotional, ledger.TransactionAuthorizationStatusOpen).InexactFloat64(), "open receivable promotional balance")
	s.Equal(expected.OpenReceivableInvoice, s.mustCustomerReceivableBalance(customerID, currencyCode, invoice, ledger.TransactionAuthorizationStatusOpen).InexactFloat64(), "open receivable invoice-cost-basis balance")

	s.Equal(expected.AuthorizedReceivableAll, s.mustCustomerReceivableBalance(customerID, currencyCode, allCostBasis, ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64(), "authorized receivable all-cost-basis balance")
	s.Equal(expected.AuthorizedReceivablePromotional, s.mustCustomerReceivableBalance(customerID, currencyCode, promotional, ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64(), "authorized receivable promotional balance")
	s.Equal(expected.AuthorizedReceivableInvoice, s.mustCustomerReceivableBalance(customerID, currencyCode, invoice, ledger.TransactionAuthorizationStatusAuthorized).InexactFloat64(), "authorized receivable invoice-cost-basis balance")

	s.Equal(expected.AccruedAll, s.mustCustomerAccruedBalance(customerID, currencyCode, allCostBasis).InexactFloat64(), "accrued all-cost-basis balance")
	s.Equal(expected.AccruedPromotional, s.mustCustomerAccruedBalance(customerID, currencyCode, promotional).InexactFloat64(), "accrued promotional balance")
	s.Equal(expected.AccruedInvoice, s.mustCustomerAccruedBalance(customerID, currencyCode, invoice).InexactFloat64(), "accrued invoice-cost-basis balance")

	s.Equal(expected.WashAll, s.mustWashBalance(s.Namespace, currencyCode, allCostBasis).InexactFloat64(), "wash all-cost-basis balance")
	s.Equal(expected.WashPromotional, s.mustWashBalance(s.Namespace, currencyCode, promotional).InexactFloat64(), "wash promotional balance")
	s.Equal(expected.WashInvoice, s.mustWashBalance(s.Namespace, currencyCode, invoice).InexactFloat64(), "wash invoice-cost-basis balance")

	s.Equal(expected.EarningsAll, s.mustEarningsBalance(s.Namespace, currencyCode).InexactFloat64(), "earnings all-cost-basis balance")
}

func (s *CreditThenInvoiceTestSuite) mustCustomerFBOBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.FBOAccount, ledger.RouteFilter{
		Currency:       code,
		CostBasis:      costBasis,
		CreditPriority: lo.ToPtr(ledger.DefaultCustomerFBOPriority),
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditThenInvoiceTestSuite) mustCustomerReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], status ledger.TransactionAuthorizationStatus) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: lo.ToPtr(status),
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditThenInvoiceTestSuite) mustCustomerAccruedBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.AccruedAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditThenInvoiceTestSuite) mustWashBalance(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.WashAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditThenInvoiceTestSuite) mustEarningsBalance(namespace string, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.EarningsAccount, ledger.RouteFilter{
		Currency: code,
	}, nil)
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditThenInvoiceTestSuite) mustGetOnlyUsageBasedCharge(ctx context.Context, subscriptionID string) usagebased.Charge {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeUsageBased},
	})
	s.NoError(err)
	s.Require().Len(res.Items, 1)

	usageBasedCharge, err := res.Items[0].AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}

func (s *CreditThenInvoiceTestSuite) mustGetUsageBasedChargeByIDWithExpands(ctx context.Context, chargeID chargesmeta.ChargeID, expands chargesmeta.Expands) usagebased.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  expands,
	})
	s.NoError(err)

	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}

type expectedUsageBasedChargeInput struct {
	Status         usagebased.Status
	ServicePeriod  timeutil.ClosedPeriod
	InvoiceAt      time.Time
	CustomerID     string
	FeatureKey     string
	Price          productcatalog.Price
	SubscriptionID string
	PhaseID        string
	ItemID         string
}

func (s *CreditThenInvoiceTestSuite) assertCreditThenInvoiceUsageBasedCharge(charge usagebased.Charge, input expectedUsageBasedChargeInput) {
	s.T().Helper()

	s.Equal(input.Status, charge.Status)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, charge.Intent.SettlementMode)
	s.Equal(input.ServicePeriod, charge.Intent.ServicePeriod)
	s.Equal(input.ServicePeriod, charge.Intent.FullServicePeriod)
	s.Equal(input.ServicePeriod, charge.Intent.BillingPeriod)
	s.Equal(input.InvoiceAt, charge.Intent.InvoiceAt)
	s.Equal(input.CustomerID, charge.Intent.CustomerID)
	s.Equal(input.FeatureKey, charge.Intent.FeatureKey)
	s.Equal(input.Price, charge.Intent.Price)
	s.Require().NotNil(charge.Intent.Subscription)
	s.Equal(input.SubscriptionID, charge.Intent.Subscription.SubscriptionID)
	s.Equal(input.PhaseID, charge.Intent.Subscription.PhaseID)
	s.Equal(input.ItemID, charge.Intent.Subscription.ItemID)
}

type expectedTotalsInput struct {
	Amount       float64
	CreditsTotal float64
	Total        float64
}

func (s *CreditThenInvoiceTestSuite) assertTotals(actual totals.Totals, input expectedTotalsInput) {
	s.T().Helper()

	s.Equal(input.Amount, actual.Amount.InexactFloat64(), "amount")
	s.Equal(input.CreditsTotal, actual.CreditsTotal.InexactFloat64(), "credits total")
	s.Equal(input.Total, actual.Total.InexactFloat64(), "total")
}

type createPromotionalCreditFundingInput struct {
	Namespace string
	Customer  customer.CustomerID
	Currency  currencyx.Code
	Amount    alpacadecimal.Decimal
	At        time.Time
}

func (s *CreditThenInvoiceTestSuite) createPromotionalCreditFunding(ctx context.Context, input createPromotionalCreditFundingInput) creditpurchase.Charge {
	s.T().Helper()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(creditpurchase.Intent{
				Intent: chargesmeta.Intent{
					Name:              "Promotional Credit Purchase",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        input.Customer.ID,
					Currency:          input.Currency,
					ServicePeriod:     timeutil.ClosedPeriod{From: input.At, To: input.At},
					FullServicePeriod: timeutil.ClosedPeriod{From: input.At, To: input.At},
					BillingPeriod:     timeutil.ClosedPeriod{From: input.At, To: input.At},
				},
				CreditAmount: input.Amount,
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(res, 1)
	s.Equal(chargesmeta.ChargeTypeCreditPurchase, res[0].Type())

	charge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.Require().NotNil(charge.Realizations.CreditGrantRealization)
	s.NotEmpty(charge.Realizations.CreditGrantRealization.TransactionGroupID)

	return charge
}
