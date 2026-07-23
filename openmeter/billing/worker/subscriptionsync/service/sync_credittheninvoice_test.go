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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
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
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/filter"
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
	s.Ledger = ledgerDeps.HistoricalLedger

	transactionManager := enttx.NewCreator(s.DBClient)

	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerDeps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: ledgerDeps.ResolversService,
			AccountCatalog: ledgerDeps.AccountService,
			BalanceQuerier: ledgerDeps.HistoricalLedger,
		},
		AccountLocker:      ledgerDeps.AccountService,
		TransactionManager: transactionManager,
	})
	s.NoError(err)

	creditPurchaseHandler, err := ledgerchargeadapter.NewCreditPurchaseHandler(
		ledgerDeps.HistoricalLedger,
		ledgerDeps.HistoricalLedger,
		ledgerDeps.ResolversService,
		ledgerDeps.AccountService,
		ledgerbreakage.NewNoopService(),
		transactionManager,
	)
	s.NoError(err)

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:             s.DBClient,
		Logger:             logger,
		BillingService:     s.BillingService,
		FeatureService:     s.FeatureService,
		StreamingConnector: s.MockStreamingConnector,
		TaxCodeService:     s.TaxCodeService,
		FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(
			ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			},
			collectorService,
		),
		CreditPurchaseHandler: creditPurchaseHandler,
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
		LegacyBillingLineEngine: s.LegacyBillingLineEngine,
		ChargesService:          s.Charges,
		Logger:                  s.Service.logger,
		Tracer:                  s.Service.tracer,
		SubscriptionSyncAdapter: s.Adapter,
		SubscriptionService:     s.SubscriptionService,
		FeatureFlags: FeatureFlags{
			EnableCreditThenInvoice: true,
		},
		FeatureGate: featuregate.NewFeatureGateChecker(featuregate.NewNoop(), featuregate.Flags{
			featuregate.CtxKeyCredits: string(featuregate.CtxKeyCredits),
		}, map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true}),
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.SetTime(start.Add(time.Minute))

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
		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

		// then:
		// - billing has a gathering invoice for the charge-backed line
		// - one credit-then-invoice usage-based charge is created
		// - no ledger balances changed during provisioning
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{namespace},
			CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
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
		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now().AddDate(0, 1, 0)))

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
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
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
		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

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
		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.SetTime(start.Add(time.Minute))

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

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

		s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

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
			return line.ServicePeriod.Duration() != time.Hour*24 // all other lines will be 1 day
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
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

	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusFinal),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 1, // as its in-advance, we'll generate the item for the next month too
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
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
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 0,
						PeriodMax: 0,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
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

	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
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
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 0,
						PeriodMax: 0,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
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

	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			// We'll wait till the end of the billing cadence of the item
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:40Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			// We'll wait till the end of the billing cadence of the item
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
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

	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(0.32), // 10 * 1 / 31
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(20),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-02T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(19.35), // 20 * 30 / 31
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(20),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
			// Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 5),
			// InvoiceAt: s.generateDailyTimestamps("2024-01-01T12:00:00Z", 5),
		},
	})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceGatheringSyncDraftInvoiceProrated() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding draft invoice and we edit the subscription later
	// Then
	//  then the draft invoice gets updated with the new pro-rated fee and the new fee
	//  item will be available as a gathering invoice

	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}

	var subsView subscription.SubscriptionView
	var updatedSubsView subscription.SubscriptionView
	var draftInvoice billing.StandardInvoice

	s.Run("create subscription", func() {
		s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
			Namespace: s.Namespace,
			Customer:  s.Customer.GetID(),
			Currency:  currencyx.Code(currency.USD),
			Amount:    alpacadecimal.NewFromInt(2),
			At:        start,
		})
		s.assertCreditThenInvoiceBalances(startBalances)

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
										Amount:      alpacadecimal.NewFromFloat(6),
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

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(start.Add(time.Minute))
	})

	s.Run("create gathering invoice", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
		s.assertCreditThenInvoiceBalances(startBalances)
	})

	s.Run("create draft invoice", func() {
		clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))

		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		s.DebugDumpInvoice("draft invoice", draftInvoices[0])
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         2,
			AccruedPromotional: 2,
			WashAll:            -2,
			WashPromotional:    -2,
		})

		draftInvoice = draftInvoices[0]
		s.assertCharges(ctx, subsView, []expectedCharge{
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusActiveRealizationProcessing),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				Realizations: []expectedChargeRealization{
					{
						Status:   draftInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
						Totals: totals.Totals{
							Amount:       alpacadecimal.NewFromFloat(6),
							CreditsTotal: alpacadecimal.NewFromFloat(2),
							Total:        alpacadecimal.NewFromFloat(4),
						},
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
		})
	})

	s.Run("edit subscription", func() {
		var err error
		updatedSubsView, err = s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
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

		s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	})

	s.Run("invoices after edit", func() {
		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

		s.assertCharges(ctx, updatedSubsView, []expectedCharge{
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusActiveRealizationProcessing),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-02T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				Realizations: []expectedChargeRealization{
					{
						Status:   draftInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(0.19), // 6 * 1 / 31
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusDeleted),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-02T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(9.68), // 10 * 30 / 31
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
		})

		var err error
		draftInvoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: draftInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             1.81,
			FBOPromotional:     1.81,
			AccruedAll:         0.19,
			AccruedPromotional: 0.19,
			WashAll:            -2,
			WashPromotional:    -2,
		})

		s.assertCharges(ctx, updatedSubsView, []expectedCharge{
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusActiveRealizationProcessing),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-02T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				Realizations: []expectedChargeRealization{
					{
						Status:   draftInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(0.19), // 6 * 1 / 31
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
						Totals: totals.Totals{
							Amount:       alpacadecimal.NewFromFloat(0.19),
							CreditsTotal: alpacadecimal.NewFromFloat(0.19),
						},
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusDeleted),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-02T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(9.68), // 10 * 30 / 31
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
		})
	})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceGatheringSyncIssuedInvoiceProrated() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding invoice that has been already finalized and we edit the subscription later
	// Then
	//  the finalized invoice doesn't get updated with the new pro-rated fee, but we
	//  add a warning to the invoice

	var subsView subscription.SubscriptionView
	var updatedSubsView subscription.SubscriptionView
	var draftInvoice billing.StandardInvoice
	var approvedInvoice billing.StandardInvoice

	s.Run("create subscription", func() {
		// given:
		// - the customer has promotional credits that can partially cover the invoice
		s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
			Namespace: s.Namespace,
			Customer:  s.Customer.GetID(),
			Currency:  currencyx.Code(currency.USD),
			Amount:    alpacadecimal.NewFromInt(2),
			At:        start,
		})
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:          2,
			FBOPromotional:  2,
			WashAll:         -2,
			WashPromotional: -2,
		})

		// when:
		// - a credit-then-invoice plan with one in-advance flat fee is created
		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
										Amount:      alpacadecimal.NewFromFloat(6),
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

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(start.Add(time.Minute))
	})

	s.Run("create gathering invoice", func() {
		// when:
		// - the subscription is synchronized into billing
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

		// then:
		// - provisioning the gathering line does not allocate credits yet
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:          2,
			FBOPromotional:  2,
			WashAll:         -2,
			WashPromotional: -2,
		})
	})

	s.Run("create draft invoice", func() {
		// when:
		// - billing creates a draft invoice from the pending gathering line
		clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		draftInvoice = draftInvoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)

		// then:
		// - draft line creation allocates the available promotional credits
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         2,
			AccruedPromotional: 2,
			WashAll:            -2,
			WashPromotional:    -2,
		})
	})

	s.Run("approve invoice", func() {
		// when:
		// - the draft invoice is approved and paid
		var err error
		approvedInvoice, err = s.BillingService.ApproveInvoice(ctx, draftInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, approvedInvoice.Status)

		// then:
		// - the paid invoice keeps the original full-period flat fee
		// - the fiat remainder is represented with invoice-cost-basis ledger rows
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         6,
			AccruedPromotional: 2,
			AccruedInvoice:     4,
			WashAll:            -6,
			WashPromotional:    -2,
			WashInvoice:        -4,
		})
		s.assertCharges(ctx, subsView, []expectedCharge{
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusFinal),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				Realizations: []expectedChargeRealization{
					{
						Status:   approvedInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
						Totals: totals.Totals{
							Amount:       alpacadecimal.NewFromFloat(6),
							CreditsTotal: alpacadecimal.NewFromFloat(2),
							Total:        alpacadecimal.NewFromFloat(4),
						},
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
		})
	})

	s.Run("edit subscription", func() {
		// when:
		// - the subscription is edited after the invoice was paid
		var err error
		updatedSubsView, err = s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
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

		s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	})

	s.Run("invoices after edit", func() {
		// then:
		// - the gathering invoice carries the replacement current and next period lines
		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

		s.assertCharges(ctx, updatedSubsView, []expectedCharge{
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusFinal),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-02T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				Realizations: []expectedChargeRealization{
					{
						Period: timeutil.ClosedPeriod{
							From: s.mustParseTime("2024-01-01T00:00:00Z"),
							To:   s.mustParseTime("2024-02-01T00:00:00Z"),
						},
						Status:   billing.StandardInvoiceStatusPaid,
						IsVoided: true,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
						Totals: totals.Totals{
							Amount:       alpacadecimal.NewFromFloat(6),
							CreditsTotal: alpacadecimal.NewFromFloat(2),
							Total:        alpacadecimal.NewFromFloat(4),
						},
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   0,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusDeleted),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 0,
					PeriodMax: 0,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-02T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(9.68), // 10 * 30 / 31
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
					},
				},
			},
			{
				Matcher: recurringLineMatcher{
					PhaseKey:  "first-phase",
					ItemKey:   "in-advance",
					Version:   1,
					PeriodMin: 1,
					PeriodMax: 1,
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
		})

		// then:
		// - the already-paid invoice is immutable and retains the original line
		// - sync records a warning on that invoice instead of rewriting ledger history
		var err error
		approvedInvoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: draftInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         6,
			AccruedPromotional: 2,
			AccruedInvoice:     4,
			WashAll:            -6,
			WashPromotional:    -2,
			WashInvoice:        -4,
		})

		s.Len(approvedInvoice.ValidationIssues, 1)

		issue := approvedInvoice.ValidationIssues[0]
		s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
		s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
		s.Equal(billing.ComponentName("charges.invoiceupdater"), issue.Component)
		s.Equal(fmt.Sprintf("lines/%s", approvedInvoice.Lines.OrEmpty()[0].ID), issue.Path)
	})
}

func (s *CreditThenInvoiceTestSuite) TestDefactoZeroPrices() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single FlatFee price that is zero
	// When
	//  we provision the lines
	// Then
	//  No lines should be invoiced

	var subView subscription.SubscriptionView

	s.Run("create subscription", func() {
		// given:
		// - the customer has no credit allocations or invoice bookings
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		// when:
		// - a credit-then-invoice subscription has a zero in-advance flat fee
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
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "in-advance",
									Name: "in-advance",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromInt(0),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
							},
						},
					},
				},
			},
		})

		// then:
		// - creating the subscription does not affect ledger balances
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(clock.Now().Add(time.Minute))
	})

	// Now let's synchronize the subscription
	s.Run("synchronize subscription", func() {
		// when:
		// - subscription sync reaches the zero-priced service periods
		asOf := s.mustParseTime("2024-01-03T12:00:00Z")
		s.NoError(s.Service.SyncByView(ctx, subView, asOf))

		// then:
		// - no gathering invoices are materialized
		// - no credits or invoice amounts are booked to the ledger
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{s.Namespace},
			CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
			Statuses: []string{
				string(billing.StandardInvoiceStatusGathering),
			},
		})
		require.NoError(s.T(), err)

		require.Len(s.T(), invoices.Items, 0)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
	})
}

func (s *CreditThenInvoiceTestSuite) TestAlignedSubscriptionInvoicing() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

	// Given
	//	a subscription with a single phase with a single item with multiple versions of it
	// When
	//  we provision the lines
	// Then
	//  in-arrears lines should be invoiced aligned
	//  in-advance lines should be invoiced immediately aligned

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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P4W"),
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
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1W")),
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
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1W")),
						},
					},
				},
			},
		},
	})

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

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
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1W")),
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
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1W")),
						},
					},
				},
			},
		},
	}, s.timingImmediate())
	s.NoError(err)

	// Now let's synchronize the subscription

	asOf := s.mustParseTime("2024-01-03T12:00:00Z")
	s.NoError(s.Service.SyncByView(ctx, subView, asOf))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 7,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(8),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-02T00:00:00Z"),
					To:   s.mustParseTime("2024-01-08T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-08T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-15T00:00:00Z"),
					To:   s.mustParseTime("2024-01-22T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-22T00:00:00Z"),
					To:   s.mustParseTime("2024-01-29T00:00:00Z"),
				},
				// As these are in advance items, we also generate them for the next Billing Period (from 2024-01-29 to 2024-02-26)
				{
					From: s.mustParseTime("2024-01-29T00:00:00Z"),
					To:   s.mustParseTime("2024-02-05T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-05T00:00:00Z"),
					To:   s.mustParseTime("2024-02-12T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-12T00:00:00Z"),
					To:   s.mustParseTime("2024-02-19T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-19T00:00:00Z"),
					To:   s.mustParseTime("2024-02-26T00:00:00Z"),
				},
			},
			// in-advance items are invoiced immediately when change happens
			InvoiceAt: []*time.Time{
				// In Advance Items are invoicable at the start of the Billing Period (even if thats before the start of their creation / service period)
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 0,
						PeriodMax: 3,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						Version:   1,
						PeriodMin: 4,
						PeriodMax: 7,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(7),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-02T00:00:00Z"),
					To:   s.mustParseTime("2024-01-08T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-08T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-15T00:00:00Z"),
					To:   s.mustParseTime("2024-01-22T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-01-22T00:00:00Z"),
					To:   s.mustParseTime("2024-01-29T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-arrears",
						Version:   1,
						PeriodMin: 0,
						PeriodMax: 3,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-29T00:00:00Z")),
				},
			},
		},
	}

	s.assertCharges(ctx, subView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestAlignedSubscriptionCancellation() {
	ctx := s.T().Context()
	startTime := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(startTime)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription with a trial phase and a future paid phase
	// when:
	// - the subscription is synchronized into the future and then canceled during the trial
	// then:
	// - future paid phase lines and charges are removed without ledger movement
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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
						Name:     "trial",
						Key:      "trial",
						Duration: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
					},
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
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	// Let's synchronize the subscription until well into the second phase
	syncUntil := startTime.AddDate(0, 3, 0) // 3 months should suffice
	s.NoError(s.Service.SyncByView(ctx, subView, syncUntil))

	// Let's check the invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Trial isn't synchronized as its a free trial...
	// Let's check the default phase
	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "default",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price:  productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(5)}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: startTime.AddDate(0, 1, 0),
					To:   startTime.AddDate(0, 2, 0),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(startTime.AddDate(0, 2, 0))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(startTime.AddDate(0, 2, 0)),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "default",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price:  productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(5)}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: startTime.AddDate(0, 2, 0),
					To:   startTime.AddDate(0, 3, 0),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(startTime.AddDate(0, 3, 0))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(startTime.AddDate(0, 3, 0)),
				},
			},
		},
	}
	s.assertCharges(ctx, subView, expectedCharges)

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
	s.NoError(s.Service.SyncByView(ctx, subView, syncUntil))

	// Let's validate that every line was canceled
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	expectedCharges[0].Status = string(usagebased.StatusDeleted)
	expectedCharges[0].GatheringLines = nil
	expectedCharges[1].Status = string(usagebased.StatusDeleted)
	expectedCharges[1].GatheringLines = nil
	s.assertCharges(ctx, subView, expectedCharges)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
}

func (s *CreditThenInvoiceTestSuite) TestAlignedSubscriptionProgressiveBillingCancellation() {
	ctx := s.T().Context()
	startTime := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(startTime)
	defer clock.UnFreeze()

	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing = billing.InvoicingConfig{
			AutoAdvance:                  true,
			DraftPeriod:                  datetime.MustParseDuration(s.T(), "P0D"),
			ProgressiveBilling:           true,
			SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
		}

		s.True(profile.Default)
	})
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2023-01-01T00:00:00Z"))

	// given:
	// - a credit-then-invoice subscription with a usage-based rate card
	// - the usage-based line has already been progressively billed for a day
	// when:
	// - the subscription is canceled during the first billing period
	// then:
	// - the remaining gathering line is removed without additional ledger movement
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(startTime.Add(time.Minute))

	// Let's synchronize the subscription
	s.NoError(s.Service.SyncByView(ctx, subView, clock.Now()))

	// Let's check the invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Let's check the default phase
	initialExpectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "default",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price:  testPrice,
			Periods: []timeutil.ClosedPeriod{
				{
					From: startTime,
					To:   startTime.AddDate(0, 1, 0),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(startTime.AddDate(0, 1, 0))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(startTime.AddDate(0, 1, 0)),
				},
			},
		},
	}
	s.assertCharges(ctx, subView, initialExpectedCharges)

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
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	partialInvoiceLines := createdInvoice.Lines.OrEmpty()
	s.Require().Len(partialInvoiceLines, 1)
	s.True(testPrice.Equal(partialInvoiceLines[0].GetPrice()), "partial invoice price")
	s.Equal(timeutil.ClosedPeriod{
		From: startTime,
		To:   startTime.AddDate(0, 0, 1),
	}, partialInvoiceLines[0].GetServicePeriod())
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), partialInvoiceLines[0].Totals.Amount, "partial invoice amount")
	s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), partialInvoiceLines[0].Totals.Total, "partial invoice total")

	// Let's fetch the gathering invoice again
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice - after progressive billing", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	gatheringLinesAfterProgressiveBilling := gatheringInvoice.Lines.OrEmpty()
	s.Require().Len(gatheringLinesAfterProgressiveBilling, 1)
	s.True(testPrice.Equal(gatheringLinesAfterProgressiveBilling[0].GetPrice()), "gathering invoice after progressive billing price")
	s.Equal(timeutil.ClosedPeriod{
		From: startTime.AddDate(0, 0, 1),
		To:   startTime.AddDate(0, 1, 0),
	}, gatheringLinesAfterProgressiveBilling[0].GetServicePeriod())
	s.Equal(startTime.AddDate(0, 1, 0), gatheringLinesAfterProgressiveBilling[0].GetInvoiceAt())

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
	s.NoError(s.Service.SyncByView(ctx, subView, clock.Now()))

	// Let's validate that the gathering invoice is gone too
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceOneTimeFeeSyncing() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

	// Given
	//  we have a subscription with a single phase with a single one-time fee in advance
	// When
	//  we we provision the lines
	// Then
	//  the gathering invoice will contain the generated item

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
			},
		},
	})
	s.assertCreditThenInvoiceBalances(startBalances)

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	expectedCharges := []expectedCharge{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
				Version:  0,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
	}

	s.assertCharges(ctx, subsView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestGatheringManualEditSync() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee charge
	// - the initial sync creates one charge-backed gathering line
	// when:
	// - the gathering line is edited through the invoice API
	// then:
	// - subscription sync keeps reconciling the base charge intent
	// - the API edit owns the customer-facing effective charge intent

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	baseTaxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(defaultTaxCodes.InvoicingTaxCodeID),
	}

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
								TaxConfig: baseTaxConfig,
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	originalLine, err := gatheringInvoice.Lines.OrEmpty()[0].Clone()
	s.NoError(err)

	var updatedLine billing.GatheringLine
	updatedIntent := expectedFlatFeeIntent{
		ServicePeriod: timeutil.ClosedPeriod{
			From: originalLine.ServicePeriod.From.Add(time.Hour),
			To:   originalLine.ServicePeriod.To.Add(time.Hour),
		},
		Amount:      7,
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
		TaxConfig:   productcatalog.TaxCodeConfigFrom(baseTaxConfig),
	}
	updatedIntent.InvoiceAt = updatedIntent.ServicePeriod.To

	s.Run("manual API edit creates an override intent", func() {
		// given:
		// - subscription sync owns the flat-fee charge base intent
		// when:
		// - the user edits the charge-backed gathering line amount, payment term, period, and invoice date through the API
		// then:
		// - the base intent keeps the subscription target values
		// - the override intent and gathering line expose the API-edited values
		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				lines := invoice.Lines.OrEmpty()
				s.Require().Len(lines, 1)
				line := &lines[0]

				price, err := line.Price.AsFlat()
				s.NoError(err)

				price.Amount = alpacadecimal.NewFromFloat(updatedIntent.Amount)
				price.PaymentTerm = updatedIntent.PaymentTerm
				line.Price = *productcatalog.NewPriceFrom(price)

				line.ServicePeriod = updatedIntent.ServicePeriod
				line.InvoiceAt = updatedIntent.InvoiceAt

				updatedLine, err = line.Clone()
				s.NoError(err)
				return nil
			},
		})
		s.NoError(err)

		editedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
				billing.GatheringInvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)
		s.DebugDumpInvoice("edited invoice", editedInvoice)

		invoiceLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == updatedLine.ID
		})
		s.True(found, "line should be found")
		s.Equal(billing.SubscriptionManagedLine, updatedLine.ManagedBy, "edit request should not stamp managed by")
		expectedLine := updatedLine
		expectedLine.ManagedBy = billing.ManuallyManagedLine
		s.True(invoiceLine.GatheringLineBase.Equal(expectedLine.GatheringLineBase), "line should expose API-edited values")

		s.assertFlatFeeChargeIntentsForInvoiceLine(ctx, "after manual edit", updatedLine, expectedFlatFeeIntent{
			ServicePeriod: originalLine.ServicePeriod,
			InvoiceAt:     originalLine.InvoiceAt,
			Amount:        5,
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			TaxConfig:     productcatalog.TaxCodeConfigFrom(baseTaxConfig),
		}, updatedIntent)
	})

	s.Run("resync preserves the manual override", func() {
		// given:
		// - a flat-fee charge has a subscription-owned base intent and an API-owned override intent
		// when:
		// - subscription sync runs again with the same subscription target
		// then:
		// - the edited gathering line remains customer-facing
		// - the override intent still matches the API edit
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

		invoiceLine, found := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == updatedLine.ID
		})
		s.True(found, "line should be found")
		expectedLine := updatedLine
		expectedLine.ManagedBy = billing.ManuallyManagedLine
		s.True(invoiceLine.GatheringLineBase.Equal(expectedLine.GatheringLineBase), "line should not be updated")

		s.assertFlatFeeChargeIntentsForInvoiceLine(ctx, "after resync", updatedLine, expectedFlatFeeIntent{
			ServicePeriod: originalLine.ServicePeriod,
			InvoiceAt:     originalLine.InvoiceAt,
			Amount:        5,
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			TaxConfig:     productcatalog.TaxCodeConfigFrom(baseTaxConfig),
		}, updatedIntent)
	})

	s.Run("subscription cancellation changes only the base intent", func() {
		// given:
		// - the customer-facing charge intent is manually overridden
		// when:
		// - time advances to the cancellation point and the subscription is canceled immediately
		// - subscription sync reconciles the canceled subscription
		// then:
		// - subscription sync shrinks the base intent to the cancellation boundary
		// - the override intent and gathering line keep the API-edited values
		cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")
		clock.FreezeTime(cancelAt)

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(err)

		canceledSubsView, err := s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SyncByView(ctx, canceledSubsView, cancelAt))
		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - after cancel sync", gatheringInvoice)

		invoiceLine, found := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == updatedLine.ID
		})
		s.True(found, "line should be found")
		expectedLine := updatedLine
		expectedLine.ManagedBy = billing.ManuallyManagedLine
		s.True(invoiceLine.GatheringLineBase.Equal(expectedLine.GatheringLineBase), "line should keep the API-edited override values")

		s.assertFlatFeeChargeIntentsForInvoiceLine(ctx, "after cancellation sync", updatedLine, expectedFlatFeeIntent{
			ServicePeriod: timeutil.ClosedPeriod{
				From: originalLine.ServicePeriod.From,
				To:   cancelAt,
			},
			InvoiceAt:   cancelAt,
			Amount:      5,
			PaymentTerm: productcatalog.InArrearsPaymentTerm,
			TaxConfig:   productcatalog.TaxCodeConfigFrom(baseTaxConfig),
		}, updatedIntent)
	})
}

func (s *CreditThenInvoiceTestSuite) TestGatheringManualCreateSync() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee item and one recurring usage-based item
	// - subscription sync owns the initial charge-backed gathering lines
	// when:
	// - the user appends a new flat-fee line through the gathering invoice API
	// then:
	// - billing preallocates and persists the gathering line identity before charges create the manual charge
	// - the new line references the newly-created manually managed charge
	// - subscription sync continues to own the subscription-backed charges

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	subscriptionTaxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(defaultTaxCodes.InvoicingTaxCodeID),
	}
	manualTaxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
	}

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears-flat-fee",
								Name: "in-arrears-flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
								TaxConfig: subscriptionTaxConfig,
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
								TaxConfig: subscriptionTaxConfig,
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 2)

	subscriptionFlatFeeLine, found := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.Engine == billing.LineEngineTypeChargeFlatFee
	})
	s.True(found, "subscription flat-fee line should be found")
	s.Equal(billing.SubscriptionManagedLine, subscriptionFlatFeeLine.ManagedBy)

	subscriptionUsageBasedLine, found := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.Engine == billing.LineEngineTypeChargeUsageBased
	})
	s.True(found, "subscription usage-based line should be found")
	s.Equal(billing.SubscriptionManagedLine, subscriptionUsageBasedLine.ManagedBy)

	subscriptionFlatFeeCharge := s.mustGetFlatFeeChargeForInvoiceLine(ctx, subscriptionFlatFeeLine.AsGenericLine())
	s.Equal(billing.SubscriptionManagedLine, subscriptionFlatFeeCharge.Intent.GetBaseIntent().ManagedBy)
	s.False(subscriptionFlatFeeCharge.Intent.HasOverrideLayer(), "subscription flat-fee charge override layer")

	subscriptionUsageBasedCharge := s.mustGetUsageBasedChargeForInvoiceLine(ctx, subscriptionUsageBasedLine.AsGenericLine())
	s.Equal(billing.SubscriptionManagedLine, subscriptionUsageBasedCharge.Intent.GetBaseIntent().ManagedBy)
	s.False(subscriptionUsageBasedCharge.Intent.HasOverrideLayer(), "subscription usage-based charge override layer")

	manualLinePeriod := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-10T00:00:00Z"),
		To:   s.mustParseTime("2024-01-20T00:00:00Z"),
	}
	updatedInvoice, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
		Invoice:      gatheringInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.GatheringInvoice) error {
			lines := invoice.Lines.OrEmpty()
			lines = append(lines, billing.GatheringLine{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: invoice.Namespace,
						Name:      "Manual setup fee",
					}),
					ManagedBy:     billing.SystemManagedLine,
					Currency:      invoice.Currency,
					ServicePeriod: manualLinePeriod,
					InvoiceAt:     manualLinePeriod.To,
					Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(3),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
			})
			invoice.Lines = billing.NewGatheringInvoiceLines(lines)

			return nil
		},
	})
	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", updatedInvoice)

	createdLine, found := lo.Find(updatedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.Name == "Manual setup fee"
	})
	s.True(found, "manual line should be found")
	s.NotEmpty(createdLine.ID, "manual line id")
	s.Require().NotNil(createdLine.ChargeID, "manual line charge id")
	s.NotEmpty(*createdLine.ChargeID, "manual line charge id")
	s.Equal(billing.LineEngineTypeChargeFlatFee, createdLine.Engine)
	s.Equal(billing.ManuallyManagedLine, createdLine.ManagedBy)
	s.Nil(createdLine.Subscription)
	s.Nil(createdLine.ChildUniqueReferenceID)
	s.Equal(manualLinePeriod, createdLine.ServicePeriod)
	s.assertTaxCodeConfigEqual(manualTaxConfig, productcatalog.TaxCodeConfigFrom(createdLine.TaxConfig), "manual line tax config")

	manualCharge := s.mustGetFlatFeeChargeForInvoiceLine(ctx, createdLine.AsGenericLine())
	s.Equal(*createdLine.ChargeID, manualCharge.ID)
	s.Equal(billing.ManuallyManagedLine, manualCharge.Intent.GetBaseIntent().ManagedBy)
	s.False(manualCharge.Intent.HasOverrideLayer(), "manual charge override layer")
	s.Nil(manualCharge.Intent.GetSubscription())
	s.Nil(manualCharge.Intent.GetUniqueReferenceID())
	s.Equal(manualLinePeriod, manualCharge.Intent.GetBaseIntent().ServicePeriod)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, manualCharge.Intent.GetSettlementMode())
	s.assertTaxCodeConfigEqual(manualTaxConfig, manualCharge.Intent.GetTaxConfig(), "manual charge tax config")

	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears-flat-fee",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

	resyncedManualLine, found := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ID == createdLine.ID
	})
	s.True(found, "manual line should remain after resync")
	s.Require().NotNil(resyncedManualLine.ChargeID, "resynced manual line charge id")
	s.Equal(*createdLine.ChargeID, *resyncedManualLine.ChargeID)
	s.Equal(billing.ManuallyManagedLine, resyncedManualLine.ManagedBy)

	manualCharge = s.mustGetFlatFeeChargeForInvoiceLine(ctx, resyncedManualLine.AsGenericLine())
	s.Equal(billing.ManuallyManagedLine, manualCharge.Intent.GetBaseIntent().ManagedBy)
	s.False(manualCharge.Intent.HasOverrideLayer(), "manual charge override layer after resync")

	subscriptionFlatFeeLine, found = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ID == subscriptionFlatFeeLine.ID
	})
	s.True(found, "subscription flat-fee line should remain after resync")
	s.Equal(billing.SubscriptionManagedLine, subscriptionFlatFeeLine.ManagedBy)

	subscriptionUsageBasedLine, found = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ID == subscriptionUsageBasedLine.ID
	})
	s.True(found, "subscription usage-based line should remain after resync")
	s.Equal(billing.SubscriptionManagedLine, subscriptionUsageBasedLine.ManagedBy)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedGatheringManualCreateSync() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - subscription sync owns an initial usage-based gathering line
	// when:
	// - the user appends a new usage-based line through the gathering invoice API
	// then:
	// - billing routes the created line to the usage-based charge engine
	// - charges creates a manually managed usage-based charge for the new line
	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	manualTaxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
	}

	var subsView subscription.SubscriptionView
	var gatheringInvoice billing.GatheringInvoice
	var createdLine billing.GatheringLine
	manualLinePeriod := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-10T00:00:00Z"),
		To:   s.mustParseTime("2024-01-20T00:00:00Z"),
	}

	s.Run("create subscription gathering invoice", func() {
		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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

		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)
		s.Equal(billing.LineEngineTypeChargeUsageBased, gatheringInvoice.Lines.OrEmpty()[0].Engine)
		s.Equal(billing.SubscriptionManagedLine, gatheringInvoice.Lines.OrEmpty()[0].ManagedBy)
	})

	s.Run("append manual usage-based gathering line", func() {
		updatedInvoice, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				lines := invoice.Lines.OrEmpty()
				lines = append(lines, billing.GatheringLine{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: invoice.Namespace,
							Name:      "Manual API usage",
						}),
						ManagedBy:     billing.SystemManagedLine,
						Currency:      invoice.Currency,
						ServicePeriod: manualLinePeriod,
						InvoiceAt:     manualLinePeriod.To,
						Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(3),
						}),
						FeatureKey: s.APIRequestsTotalFeature.Key,
					},
				})
				invoice.Lines = billing.NewGatheringInvoiceLines(lines)

				return nil
			},
		})
		s.NoError(err)
		s.DebugDumpInvoice("edited gathering invoice", updatedInvoice)

		var found bool
		createdLine, found = lo.Find(updatedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.Name == "Manual API usage"
		})
		s.True(found, "manual usage-based line should be found")
		s.NotEmpty(createdLine.ID, "manual line id")
		s.Require().NotNil(createdLine.ChargeID, "manual line charge id")
		s.NotEmpty(*createdLine.ChargeID, "manual line charge id")
		s.Equal(billing.LineEngineTypeChargeUsageBased, createdLine.Engine)
		s.Equal(billing.ManuallyManagedLine, createdLine.ManagedBy)
		s.Nil(createdLine.Subscription)
		s.Nil(createdLine.ChildUniqueReferenceID)
		s.Equal(manualLinePeriod, createdLine.ServicePeriod)
		s.Equal(manualLinePeriod.To, createdLine.InvoiceAt)
		s.Equal(s.APIRequestsTotalFeature.Key, createdLine.FeatureKey)
		s.assertTaxCodeConfigEqual(manualTaxConfig, productcatalog.TaxCodeConfigFrom(createdLine.TaxConfig), "manual line tax config")
	})

	s.Run("manual usage-based charge is created", func() {
		manualCharge := s.mustGetUsageBasedChargeForInvoiceLine(ctx, createdLine.AsGenericLine())
		s.Equal(*createdLine.ChargeID, manualCharge.ID)
		s.Equal(usagebased.StatusCreated, manualCharge.Status)
		s.Equal(billing.ManuallyManagedLine, manualCharge.Intent.GetBaseIntent().ManagedBy)
		s.False(manualCharge.Intent.HasOverrideLayer(), "manual charge override layer")
		s.Nil(manualCharge.Intent.GetSubscription())
		s.Nil(manualCharge.Intent.GetUniqueReferenceID())
		s.Equal(manualLinePeriod, manualCharge.Intent.GetBaseIntent().ServicePeriod)
		s.Equal(manualLinePeriod.To, manualCharge.Intent.GetBaseIntent().InvoiceAt)
		s.Equal(s.APIRequestsTotalFeature.Key, manualCharge.Intent.GetBaseIntent().FeatureKey)
		s.Equal(productcatalog.CreditThenInvoiceSettlementMode, manualCharge.Intent.GetSettlementMode())
		s.assertTaxCodeConfigEqual(manualTaxConfig, manualCharge.Intent.GetTaxConfig(), "manual charge tax config")
	})
}

func (s *CreditThenInvoiceTestSuite) TestGatheringManualDeleteSync() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	var subsView subscription.SubscriptionView
	var gatheringInvoice billing.GatheringInvoice
	var deletedLine billing.GatheringLine
	var chargeID chargesmeta.ChargeID

	s.Run("create gathering line", func() {
		// given:
		// - subscription sync owns the flat-fee charge base intent
		// when:
		// - the active subscription is synced
		// then:
		// - sync creates one customer-facing charge-backed gathering line
		defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
		s.NoError(err)
		baseTaxConfig := &productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr(defaultTaxCodes.InvoicingTaxCodeID),
		}

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "in-arrears",
									Name: "in-arrears",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromFloat(5),
										PaymentTerm: productcatalog.InArrearsPaymentTerm,
									}),
									TaxConfig: baseTaxConfig,
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
							},
						},
					},
				},
			},
		})

		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)
	})

	s.Run("delete gathering line through API", func() {
		// when:
		// - the user deletes the gathering line through the invoice API
		// then:
		// - the API delete is persisted as a deleted override intent
		// - subscription sync keeps owning the undeleted base intent
		var err error
		_, err = s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				lines := invoice.Lines.OrEmpty()
				s.Require().Len(lines, 1)
				line := &lines[0]

				line.DeletedAt = lo.ToPtr(clock.Now())

				deletedLine, err = line.Clone()
				s.NoError(err)
				return nil
			},
			IncludeDeletedLines: true,
		})
		s.NoError(err)

		editedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
				billing.GatheringInvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)
		s.DebugDumpInvoice("deleted invoice", editedInvoice)

		invoiceLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == deletedLine.ID
		})
		s.True(found, "deleted line should be found")
		s.NotNil(invoiceLine.DeletedAt)
		s.Equal(billing.ManuallyManagedLine, invoiceLine.ManagedBy)

		flatFeeCharge := s.mustGetFlatFeeChargeForInvoiceLine(ctx, deletedLine.AsGenericLine())
		chargeID = flatFeeCharge.GetChargeID()
		s.True(flatFeeCharge.Intent.HasOverrideLayer(), "override layer")
		s.Nil(flatFeeCharge.Intent.GetBaseIntent().IntentDeletedAt)
		overrideIntent, err := flatFeeCharge.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)
	})

	s.Run("subscription sync does not recreate deleted gathering line", func() {
		// when:
		// - subscription sync runs again for the active subscription
		// then:
		// - it does not recreate the customer-facing gathering line
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	})

	s.Run("subscription cancellation reconciles deleted charge base intent", func() {
		// when:
		// - the active subscription is canceled after the customer-facing override delete
		// - subscription sync reconciles the canceled subscription
		// then:
		// - sync shrinks the hidden base/source intent without entering charge lifecycle
		// - the deleted override remains customer-facing
		cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")
		clock.FreezeTime(cancelAt)

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(err)

		canceledSubsView, err := s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SyncByView(ctx, canceledSubsView, cancelAt))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

		chargeAfterCancelGeneric, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands:  chargesmeta.Expands{chargesmeta.ExpandRealizations},
		})
		s.NoError(err)

		chargeAfterCancel, err := chargeAfterCancelGeneric.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatfee.StatusDeleted, chargeAfterCancel.Status)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().ServicePeriod.To)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().BillingPeriod.To)
		s.True(chargeAfterCancel.Intent.HasOverrideLayer(), "override layer")
		overrideIntent, err := chargeAfterCancel.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedGatheringManualDeleteWithoutRealizations() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	var subsView subscription.SubscriptionView
	var gatheringInvoice billing.GatheringInvoice
	var deletedLine billing.GatheringLine
	var chargeID chargesmeta.ChargeID

	s.Run("create gathering line without realizations", func() {
		// given:
		// - subscription sync owns a usage-based charge base intent
		// - the initial sync creates one charge-backed gathering line
		// - no realization run has been created for the charge
		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
				},
			},
		})

		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)
	})

	s.Run("delete gathering line through API", func() {
		// when:
		// - the user deletes the gathering line through the invoice API
		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				lines := invoice.Lines.OrEmpty()
				s.Require().Len(lines, 1)
				line := &lines[0]

				line.DeletedAt = lo.ToPtr(clock.Now())

				clonedLine, err := line.Clone()
				s.NoError(err)
				deletedLine = clonedLine
				return nil
			},
			IncludeDeletedLines: true,
		})
		s.NoError(err)
	})

	s.Run("assert charge is manually deleted", func() {
		// then:
		// - the API delete is persisted as a deleted override intent
		// - the gathering line is deleted without creating realization history
		editedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
				billing.GatheringInvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)
		s.DebugDumpInvoice("deleted invoice", editedInvoice)

		invoiceLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == deletedLine.ID
		})
		s.True(found, "deleted line should be found")
		s.NotNil(invoiceLine.DeletedAt)
		s.Equal(billing.ManuallyManagedLine, invoiceLine.ManagedBy)

		chargeAfterDelete := s.mustGetUsageBasedChargeForInvoiceLine(ctx, deletedLine.AsGenericLine())
		chargeID = chargeAfterDelete.GetChargeID()
		s.Equal(usagebased.StatusDeleted, chargeAfterDelete.Status)
		s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
		s.Nil(chargeAfterDelete.Intent.GetBaseIntent().IntentDeletedAt)
		overrideIntent, err := chargeAfterDelete.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)

		chargeWithRealizations := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeAfterDelete.GetChargeID(), chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Empty(chargeWithRealizations.Realizations)
	})

	s.Run("subscription sync does not recreate deleted line", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	})

	s.Run("subscription cancellation reconciles the deleted charge base intent", func() {
		// given:
		// - the active subscription still owns the usage-based charge base intent
		// - the customer-facing charge was manually deleted through an override intent
		// when:
		// - the subscription is canceled immediately
		// - subscription sync reconciles the canceled subscription
		// then:
		// - sync can shrink the base intent without entering the deleted effective charge lifecycle
		// - the deleted override remains customer-facing and the gathering line is not recreated
		cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")
		clock.FreezeTime(cancelAt)

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(err)

		canceledSubsView, err := s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SyncByView(ctx, canceledSubsView, cancelAt))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

		chargeAfterCancel := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeID, chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Equal(usagebased.StatusDeleted, chargeAfterCancel.Status)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().ServicePeriod.To)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().BillingPeriod.To)
		s.True(chargeAfterCancel.Intent.HasOverrideLayer(), "override layer")
		overrideIntent, err := chargeAfterCancel.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)
		s.Empty(chargeAfterCancel.Realizations)
	})
}

func (s *CreditThenInvoiceTestSuite) TestStandardInvoiceManualEditSync() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee charge
	// - the billing profile requires manual invoice approval
	// - the initial gathering line is collected into a mutable draft standard invoice
	// when:
	// - the standard invoice line is edited through the invoice API
	// then:
	// - the charge override intent and current realization run reflect the edited standard line
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	})

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	baseTaxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(defaultTaxCodes.InvoicingTaxCodeID),
	}

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
								TaxConfig: baseTaxConfig,
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(startBalances)
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	})
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)

	originalLine, err := draftInvoice.Lines.OrEmpty()[0].Clone()
	s.NoError(err)

	chargeBeforeEdit := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, originalLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusActiveRealizationProcessing, chargeBeforeEdit.Status)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun.LineID)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(originalLine.ID, *chargeBeforeEdit.Realizations.CurrentRun.LineID)
	s.Equal(draftInvoice.ID, *chargeBeforeEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(originalLine.Period, chargeBeforeEdit.Realizations.CurrentRun.ServicePeriod)
	s.Equal(float64(5), chargeBeforeEdit.Realizations.CurrentRun.AmountAfterProration.InexactFloat64())
	s.assertTotals(chargeBeforeEdit.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount:       5,
		CreditsTotal: 2,
		Total:        3,
	})

	updatedIntent := expectedFlatFeeIntent{
		ServicePeriod: timeutil.ClosedPeriod{
			From: originalLine.Period.From.Add(time.Hour),
			To:   originalLine.Period.To.Add(time.Hour),
		},
		// TODO: add standard-invoice manual edit coverage for editing the charge-backed flat-fee line to zero.
		InvoiceAt:   chargeBeforeEdit.Intent.GetEffectiveInvoiceAt(),
		Amount:      7,
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
		TaxConfig:   productcatalog.TaxCodeConfigFrom(baseTaxConfig),
	}

	var updatedLine *billing.StandardLine
	editedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draftInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Require().Len(lines, 1)
			line := lines[0]

			linePrice := line.GetPrice()
			s.Require().NotNil(linePrice)

			price, err := linePrice.AsFlat()
			s.NoError(err)

			price.Amount = alpacadecimal.NewFromFloat(updatedIntent.Amount)
			price.PaymentTerm = updatedIntent.PaymentTerm
			line.SetPrice(*productcatalog.NewPriceFrom(price))

			line.Period = updatedIntent.ServicePeriod

			updatedLine, err = line.Clone()
			s.NoError(err)
			return nil
		},
	})
	s.Require().NoError(err)
	s.DebugDumpInvoice("edited draft invoice", editedInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	})

	editedInvoiceLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.ID == updatedLine.ID
	})
	s.Require().True(found, "edited standard line should be found")
	s.Equal(billing.SubscriptionManagedLine, updatedLine.ManagedBy, "edit request should not stamp managed by")
	s.Equal(billing.ManuallyManagedLine, editedInvoiceLine.ManagedBy)
	s.Equal(updatedIntent.ServicePeriod, editedInvoiceLine.Period)

	editedLinePrice := editedInvoiceLine.GetPrice()
	s.Require().NotNil(editedLinePrice)

	editedFlatPrice, err := editedLinePrice.AsFlat()
	s.NoError(err)
	s.Equal(updatedIntent.Amount, editedFlatPrice.Amount.InexactFloat64())
	s.Equal(updatedIntent.PaymentTerm, editedFlatPrice.PaymentTerm)

	s.Require().NotNil(editedInvoiceLine.TaxConfig)
	s.True(baseTaxConfig.Equal(editedInvoiceLine.TaxConfig.ToProductCatalog()), "edited standard line should keep base tax config")

	s.assertFlatFeeChargeIntentsForInvoiceLine(ctx, "after standard line manual edit", updatedLine, expectedFlatFeeIntent{
		ServicePeriod: originalLine.Period,
		InvoiceAt:     chargeBeforeEdit.Intent.GetBaseIntent().InvoiceAt,
		Amount:        5,
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		TaxConfig:     productcatalog.TaxCodeConfigFrom(baseTaxConfig),
	}, updatedIntent)

	chargeAfterEdit := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, updatedLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusActiveRealizationProcessing, chargeAfterEdit.Status)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun.LineID)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(updatedLine.ID, *chargeAfterEdit.Realizations.CurrentRun.LineID)
	s.Equal(editedInvoice.ID, *chargeAfterEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(updatedIntent.ServicePeriod, chargeAfterEdit.Realizations.CurrentRun.ServicePeriod)
	s.Equal(updatedIntent.Amount, chargeAfterEdit.Realizations.CurrentRun.AmountAfterProration.InexactFloat64())
	s.assertTotals(chargeAfterEdit.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount:       updatedIntent.Amount,
		CreditsTotal: 2,
		Total:        5,
	})
}

func (s *CreditThenInvoiceTestSuite) TestStandardInvoiceManualDiscountEditSync() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee charge
	// - the billing profile requires manual invoice approval
	// - the mutable draft standard invoice line is still managed by the flat-fee charge
	// when:
	// - the standard invoice line's percentage discount is edited through the invoice API
	// then:
	// - the edit routes through the flat-fee line engine and updates the charge override/current run
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	defaultTaxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
	}

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(15000),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	originalLine, err := draftInvoice.Lines.OrEmpty()[0].Clone()
	s.NoError(err)
	s.Equal(billing.SubscriptionManagedLine, originalLine.ManagedBy)
	s.Equal(billing.LineEngineTypeChargeFlatFee, originalLine.Engine)
	s.Require().NotNil(originalLine.ChargeID)
	s.Nil(originalLine.RateCardDiscounts.Percentage)
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(15000), originalLine.Totals.Amount, "original amount")
	s.AssertDecimalEqual(alpacadecimal.Zero, originalLine.Totals.DiscountsTotal, "original discount total")
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(15000), originalLine.Totals.Total, "original total")

	chargeBeforeEdit := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, originalLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusActiveRealizationProcessing, chargeBeforeEdit.Status)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun)
	s.False(chargeBeforeEdit.Realizations.CurrentRun.Immutable)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun.LineID)
	s.Require().NotNil(chargeBeforeEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(originalLine.ID, *chargeBeforeEdit.Realizations.CurrentRun.LineID)
	s.Equal(draftInvoice.ID, *chargeBeforeEdit.Realizations.CurrentRun.InvoiceID)
	s.Nil(chargeBeforeEdit.Intent.GetBaseIntent().PercentageDiscounts)
	s.assertTotals(chargeBeforeEdit.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount: 15000,
		Total:  15000,
	})

	discount := productcatalog.PercentageDiscount{
		Percentage: models.NewPercentage(50),
	}
	var updatedLine *billing.StandardLine
	editedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draftInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Require().Len(lines, 1)

			line := lines[0]
			line.RateCardDiscounts = billing.Discounts{
				Percentage: &billing.PercentageDiscount{
					PercentageDiscount: discount,
				},
			}

			updatedLine, err = line.Clone()
			s.NoError(err)
			return nil
		},
	})
	s.Require().NoError(err)
	s.DebugDumpInvoice("edited draft invoice", editedInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	editedInvoiceLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.ID == updatedLine.ID
	})
	s.Require().True(found, "edited standard line should be found")
	s.Equal(billing.SubscriptionManagedLine, updatedLine.ManagedBy, "edit request should not stamp managed by")
	s.Equal(billing.ManuallyManagedLine, editedInvoiceLine.ManagedBy)
	s.Equal(billing.LineEngineTypeChargeFlatFee, editedInvoiceLine.Engine)
	s.Require().NotNil(editedInvoiceLine.ChargeID)
	s.Equal(*originalLine.ChargeID, *editedInvoiceLine.ChargeID)
	s.Require().NotNil(editedInvoiceLine.RateCardDiscounts.Percentage)
	s.Equal(discount.Percentage, editedInvoiceLine.RateCardDiscounts.Percentage.Percentage)
	s.NotEmpty(editedInvoiceLine.RateCardDiscounts.Percentage.CorrelationID)
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(15000), editedInvoiceLine.Totals.Amount, "edited amount")
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(7500), editedInvoiceLine.Totals.DiscountsTotal, "edited discount total")
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(7500), editedInvoiceLine.Totals.Total, "edited total")

	s.assertFlatFeeChargeIntentsForInvoiceLine(ctx, "after standard line discount edit", updatedLine, expectedFlatFeeIntent{
		ServicePeriod: originalLine.Period,
		InvoiceAt:     chargeBeforeEdit.Intent.GetBaseIntent().InvoiceAt,
		Amount:        15000,
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		TaxConfig:     defaultTaxConfig,
	}, expectedFlatFeeIntent{
		ServicePeriod: originalLine.Period,
		InvoiceAt:     chargeBeforeEdit.Intent.GetEffectiveInvoiceAt(),
		Amount:        15000,
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		PercentageDiscounts: &billing.PercentageDiscount{
			PercentageDiscount: discount,
			CorrelationID:      editedInvoiceLine.RateCardDiscounts.Percentage.CorrelationID,
		},
		TaxConfig: defaultTaxConfig,
	})

	chargeAfterEdit := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, updatedLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusActiveRealizationProcessing, chargeAfterEdit.Status)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun)
	s.False(chargeAfterEdit.Realizations.CurrentRun.Immutable)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun.LineID)
	s.Require().NotNil(chargeAfterEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(updatedLine.ID, *chargeAfterEdit.Realizations.CurrentRun.LineID)
	s.Equal(editedInvoice.ID, *chargeAfterEdit.Realizations.CurrentRun.InvoiceID)
	s.Equal(originalLine.Period, chargeAfterEdit.Realizations.CurrentRun.ServicePeriod)
	s.Equal(float64(15000), chargeAfterEdit.Realizations.CurrentRun.AmountAfterProration.InexactFloat64())
	s.assertTotals(chargeAfterEdit.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount:         15000,
		DiscountsTotal: 7500,
		Total:          7500,
	})
}

func (s *CreditThenInvoiceTestSuite) TestStandardInvoiceManualCreateSync() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee charge
	// - the customer has promotional credits that partially cover the existing and API-created lines
	// - the billing profile requires manual invoice approval so the standard invoice remains mutable
	// when:
	// - the user appends a new flat-fee line through the standard invoice API
	// then:
	// - creating a zero-amount flat-fee line is rejected with delete/create guidance
	// - billing preallocates the standard line identity
	// - charges creates a manually managed flat-fee charge and attaches a current run to that line
	// - promotional credits are allocated to the new run and line
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	})

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	defaultTaxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
	}

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(7),
		At:        start,
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:          7,
		FBOPromotional:  7,
		WashAll:         -7,
		WashPromotional: -7,
	})

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             2,
		FBOPromotional:     2,
		AccruedAll:         5,
		AccruedPromotional: 5,
		WashAll:            -7,
		WashPromotional:    -7,
	})

	manualLinePeriod := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-02-01T00:00:00Z"),
		To:   s.mustParseTime("2024-02-10T00:00:00Z"),
	}

	_, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draftInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines := invoice.Lines.OrEmpty()
			zeroLine := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				Namespace:     invoice.Namespace,
				InvoiceID:     invoice.ID,
				Name:          "Zero manual standard setup fee",
				Currency:      invoice.Currency,
				Period:        manualLinePeriod,
				InvoiceAt:     manualLinePeriod.To,
				PerUnitAmount: alpacadecimal.Zero,
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			})
			zeroLine.Engine = ""
			lines = append(lines, zeroLine)
			invoice.Lines = billing.NewStandardInvoiceLines(lines)

			return nil
		},
	})
	s.ErrorIs(err, billing.ErrInvoiceLineZeroAmountCreate)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             2,
		FBOPromotional:     2,
		AccruedAll:         5,
		AccruedPromotional: 5,
		WashAll:            -7,
		WashPromotional:    -7,
	})

	var createdLineID string
	editedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draftInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines := invoice.Lines.OrEmpty()
			manualLine := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				Namespace:     invoice.Namespace,
				InvoiceID:     invoice.ID,
				Name:          "Manual standard setup fee",
				Currency:      invoice.Currency,
				Period:        manualLinePeriod,
				InvoiceAt:     manualLinePeriod.To,
				PerUnitAmount: alpacadecimal.NewFromFloat(3),
				PaymentTerm:   productcatalog.InArrearsPaymentTerm,
			})
			manualLine.Engine = ""
			manualLine.TaxConfig = nil
			lines = append(lines, manualLine)
			invoice.Lines = billing.NewStandardInvoiceLines(lines)

			return nil
		},
	})
	s.Require().NoError(err)
	s.DebugDumpInvoice("edited draft invoice", editedInvoice)
	s.Require().Len(editedInvoice.Lines.OrEmpty(), 2)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         7,
		AccruedPromotional: 7,
		WashAll:            -7,
		WashPromotional:    -7,
	})

	createdLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.Name == "Manual standard setup fee"
	})
	s.Require().True(found, "manual standard line should be found")
	createdLineID = createdLine.ID
	s.NotEmpty(createdLineID, "manual standard line id")
	s.Require().NotNil(createdLine.ChargeID, "manual standard line charge id")
	s.NotEmpty(*createdLine.ChargeID, "manual standard line charge id")
	s.Equal(billing.LineEngineTypeChargeFlatFee, createdLine.Engine)
	s.Equal(billing.ManuallyManagedLine, createdLine.ManagedBy)
	s.Nil(createdLine.Subscription)
	s.Nil(createdLine.ChildUniqueReferenceID)
	s.Equal(manualLinePeriod, createdLine.Period)
	s.assertTaxCodeConfigEqual(defaultTaxConfig, productcatalog.TaxCodeConfigFrom(createdLine.TaxConfig.ToProductCatalog()), "manual standard line tax config")
	s.Require().Len(createdLine.CreditsApplied, 1)
	s.Equal(float64(2), createdLine.CreditsApplied[0].Amount.InexactFloat64())
	s.assertTotals(createdLine.Totals, expectedTotalsInput{
		Amount:       3,
		CreditsTotal: 2,
		Total:        1,
	})

	manualCharge := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, createdLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(*createdLine.ChargeID, manualCharge.ID)
	s.Equal(flatfee.StatusActiveRealizationProcessing, manualCharge.Status)
	s.Equal(billing.ManuallyManagedLine, manualCharge.Intent.GetBaseIntent().ManagedBy)
	s.False(manualCharge.Intent.HasOverrideLayer(), "manual charge override layer")
	s.Nil(manualCharge.Intent.GetSubscription())
	s.Nil(manualCharge.Intent.GetUniqueReferenceID())
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, manualCharge.Intent.GetSettlementMode())
	s.assertFlatFeeIntent("manual charge base intent", manualCharge.Intent.GetBaseIntent(), expectedFlatFeeIntent{
		ServicePeriod: manualLinePeriod,
		InvoiceAt:     manualLinePeriod.To,
		Amount:        3,
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		TaxConfig:     defaultTaxConfig,
	})

	s.Require().NotNil(manualCharge.Realizations.CurrentRun)
	s.Require().NotNil(manualCharge.Realizations.CurrentRun.LineID)
	s.Require().NotNil(manualCharge.Realizations.CurrentRun.InvoiceID)
	s.Equal(createdLine.ID, *manualCharge.Realizations.CurrentRun.LineID)
	s.Equal(editedInvoice.ID, *manualCharge.Realizations.CurrentRun.InvoiceID)
	s.Equal(manualLinePeriod, manualCharge.Realizations.CurrentRun.ServicePeriod)
	s.Equal(float64(3), manualCharge.Realizations.CurrentRun.AmountAfterProration.InexactFloat64())
	s.assertTotals(manualCharge.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount:       3,
		CreditsTotal: 2,
		Total:        1,
	})
	s.Require().Len(manualCharge.Realizations.CurrentRun.CreditRealizations, 1)
	s.Equal(float64(2), manualCharge.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())

	refetchedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: editedInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)
	s.Require().Len(refetchedInvoice.Lines.OrEmpty(), 2)
	refetchedCreatedLine, found := lo.Find(refetchedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.ID == createdLineID
	})
	s.Require().True(found, "manual standard line should persist")
	s.Require().NotNil(refetchedCreatedLine.ChargeID)
	s.Equal(*createdLine.ChargeID, *refetchedCreatedLine.ChargeID)
	s.Equal(billing.ManuallyManagedLine, refetchedCreatedLine.ManagedBy)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedStandardInvoiceManualCreateSync() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - the customer has promotional credits that partially cover the existing and API-created lines
	// - a draft standard invoice remains mutable because manual approval is required
	// when:
	// - the user appends a new usage-based line through the standard invoice API
	// then:
	// - billing preallocates the standard line identity
	// - charges creates a manually managed usage-based charge
	// - the created standard line becomes the charge's current ongoing realization
	// - promotional credits are allocated to the new run and line
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	})

	defaultTaxCodes, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.NoError(err)
	defaultTaxConfig := productcatalog.TaxCodeConfig{
		TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
	}

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(7),
		At:        start,
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:          7,
		FBOPromotional:  7,
		WashAll:         -7,
		WashPromotional: -7,
	})

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2000, s.mustParseTime("2024-02-02T00:00:00Z"))

	var draftInvoice billing.StandardInvoice
	var editedInvoice billing.StandardInvoice
	var createdLine *billing.StandardLine
	manualLinePeriod := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-02-01T00:00:00Z"),
		To:   s.mustParseTime("2024-02-10T00:00:00Z"),
	}
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	s.Run("create mutable draft invoice", func() {
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
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "in-arrears",
									Name: "in-arrears",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromFloat(5),
										PaymentTerm: productcatalog.InArrearsPaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
							},
						},
					},
				},
			},
		})

		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

		clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		draftInvoice = draftInvoices[0]
		s.DebugDumpInvoice("draft invoice", draftInvoice)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
		s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             2,
			FBOPromotional:     2,
			AccruedAll:         5,
			AccruedPromotional: 5,
			WashAll:            -7,
			WashPromotional:    -7,
		})
	})

	s.Run("append manual usage-based standard line", func() {
		var err error
		editedInvoice, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
			Invoice:      draftInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.StandardInvoice) error {
				lines := invoice.Lines.OrEmpty()
				lines = append(lines, &billing.StandardLine{
					StandardLineBase: billing.StandardLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: invoice.Namespace,
							Name:      "Manual standard API usage",
						}),
						ManagedBy: billing.SystemManagedLine,
						InvoiceID: invoice.ID,
						Currency:  invoice.Currency,
						Period:    manualLinePeriod,
						InvoiceAt: manualLinePeriod.To,
					},
					UsageBased: &billing.UsageBasedLine{
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(3),
						}),
						FeatureKey: s.APIRequestsTotalFeature.Key,
						UnitConfig: lo.ToPtr(unitConfig.Clone()),
					},
				})
				invoice.Lines = billing.NewStandardInvoiceLines(lines)

				return nil
			},
		})
		s.Require().NoError(err)
		s.DebugDumpInvoice("edited draft invoice", editedInvoice)
		s.Require().Len(editedInvoice.Lines.OrEmpty(), 2)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         7,
			AccruedPromotional: 7,
			WashAll:            -7,
			WashPromotional:    -7,
		})

		var found bool
		createdLine, found = lo.Find(editedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
			return line != nil && line.Name == "Manual standard API usage"
		})
		s.Require().True(found, "manual usage-based standard line should be found")
		s.NotEmpty(createdLine.ID, "manual standard line id")
		s.Require().NotNil(createdLine.ChargeID, "manual standard line charge id")
		s.NotEmpty(*createdLine.ChargeID, "manual standard line charge id")
		s.Equal(billing.LineEngineTypeChargeUsageBased, createdLine.Engine)
		s.Equal(billing.ManuallyManagedLine, createdLine.ManagedBy)
		s.Nil(createdLine.Subscription)
		s.Nil(createdLine.ChildUniqueReferenceID)
		s.Equal(manualLinePeriod, createdLine.Period)
		s.Equal(s.APIRequestsTotalFeature.Key, createdLine.UsageBased.FeatureKey)
		s.Require().NotNil(createdLine.UsageBased.UnitConfig)
		s.True(unitConfig.Equal(createdLine.UsageBased.UnitConfig))
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2000), *createdLine.UsageBased.MeteredQuantity, "manual usage-based standard line metered quantity")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2), *createdLine.UsageBased.Quantity, "manual usage-based standard line billable quantity")
		s.assertTaxCodeConfigEqual(defaultTaxConfig, productcatalog.TaxCodeConfigFrom(createdLine.TaxConfig.ToProductCatalog()), "manual standard line tax config")
		s.Require().Len(createdLine.CreditsApplied, 1)
		s.Equal(float64(2), createdLine.CreditsApplied[0].Amount.InexactFloat64())
		s.assertTotals(createdLine.Totals, expectedTotalsInput{
			Amount:       6,
			CreditsTotal: 2,
			Total:        4,
		})
	})

	s.Run("manual charge has ongoing realization for created standard line", func() {
		manualCharge := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
			Namespace: createdLine.Namespace,
			ID:        *createdLine.ChargeID,
		}, chargesmeta.Expands{chargesmeta.ExpandRealizations, chargesmeta.ExpandDetailedLines})

		s.Equal(*createdLine.ChargeID, manualCharge.ID)
		s.Equal(usagebased.StatusActiveRealizationStarted, manualCharge.Status)
		s.Equal(billing.ManuallyManagedLine, manualCharge.Intent.GetBaseIntent().ManagedBy)
		s.False(manualCharge.Intent.HasOverrideLayer(), "manual charge override layer")
		s.Nil(manualCharge.Intent.GetSubscription())
		s.Nil(manualCharge.Intent.GetUniqueReferenceID())
		s.Equal(productcatalog.CreditThenInvoiceSettlementMode, manualCharge.Intent.GetSettlementMode())
		s.Equal(manualLinePeriod, manualCharge.Intent.GetBaseIntent().ServicePeriod)
		s.Equal(manualLinePeriod.To, manualCharge.Intent.GetBaseIntent().InvoiceAt)
		s.Equal(s.APIRequestsTotalFeature.Key, manualCharge.Intent.GetBaseIntent().FeatureKey)
		s.Require().NotNil(manualCharge.Intent.GetBaseIntent().UnitConfig)
		s.True(unitConfig.Equal(manualCharge.Intent.GetBaseIntent().UnitConfig))
		s.assertTaxCodeConfigEqual(defaultTaxConfig, manualCharge.Intent.GetTaxConfig(), "manual charge tax config")

		s.Require().NotNil(manualCharge.State.CurrentRealizationRunID)
		currentRun, err := manualCharge.GetCurrentRealizationRun()
		s.Require().NoError(err)
		s.Require().NotNil(currentRun.LineID)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(createdLine.ID, *currentRun.LineID)
		s.Equal(editedInvoice.ID, *currentRun.InvoiceID)
		s.Equal(manualLinePeriod.To, currentRun.ServicePeriodTo)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2000), currentRun.MeteredQuantity, "manual usage-based current run raw metered quantity")
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.False(currentRun.IsVoidedBillingHistory())
		s.assertTotals(currentRun.Totals, expectedTotalsInput{
			Amount:       6,
			CreditsTotal: 2,
			Total:        4,
		})
		s.Require().Len(currentRun.CreditsAllocated, 1)
		s.Equal(float64(2), currentRun.CreditsAllocated[0].Amount.InexactFloat64())
	})

	s.Run("created standard line persists", func() {
		refetchedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: editedInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Require().Len(refetchedInvoice.Lines.OrEmpty(), 2)
		refetchedCreatedLine, found := lo.Find(refetchedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
			return line != nil && line.ID == createdLine.ID
		})
		s.Require().True(found, "manual usage-based standard line should persist")
		s.Require().NotNil(refetchedCreatedLine.ChargeID)
		s.Equal(*createdLine.ChargeID, *refetchedCreatedLine.ChargeID)
		s.Equal(billing.ManuallyManagedLine, refetchedCreatedLine.ManagedBy)
		s.Equal(billing.LineEngineTypeChargeUsageBased, refetchedCreatedLine.Engine)
		s.Require().NotNil(refetchedCreatedLine.UsageBased.UnitConfig)
		s.True(unitConfig.Equal(refetchedCreatedLine.UsageBased.UnitConfig))
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2000), *refetchedCreatedLine.UsageBased.MeteredQuantity, "persisted manual usage-based standard line metered quantity")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2), *refetchedCreatedLine.UsageBased.Quantity, "persisted manual usage-based standard line billable quantity")
	})
}

func (s *CreditThenInvoiceTestSuite) TestStandardInvoiceManualDeleteSync() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one recurring flat-fee charge
	// - the customer has promotional credits that partially cover the draft standard invoice line
	// - the billing profile requires manual invoice approval so the standard line remains mutable
	// when:
	// - the standard invoice line is deleted through the invoice API
	// then:
	// - the charge override intent records the customer-facing deletion
	// - the charge detaches from the mutable standard line
	// - the promotional credit allocation is corrected back to customer FBO
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	})

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(startBalances)
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	})

	originalLine, err := draftInvoice.Lines.OrEmpty()[0].Clone()
	s.NoError(err)

	chargeBeforeDelete := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, originalLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusActiveRealizationProcessing, chargeBeforeDelete.Status)
	s.Require().NotNil(chargeBeforeDelete.Realizations.CurrentRun)
	s.Require().NotNil(chargeBeforeDelete.Realizations.CurrentRun.LineID)
	s.Equal(originalLine.ID, *chargeBeforeDelete.Realizations.CurrentRun.LineID)
	s.assertTotals(chargeBeforeDelete.Realizations.CurrentRun.Totals, expectedTotalsInput{
		Amount:       5,
		CreditsTotal: 2,
		Total:        3,
	})

	var deletedLine *billing.StandardLine
	deletedInvoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draftInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Require().Len(lines, 1)
			line := lines[0]

			line.DeletedAt = lo.ToPtr(clock.Now())

			deletedLine, err = line.Clone()
			s.NoError(err)
			return nil
		},
		IncludeDeletedLines: true,
	})
	s.Require().NoError(err)
	s.DebugDumpInvoice("deleted draft invoice", deletedInvoice)

	deletedInvoiceLine, found := lo.Find(deletedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.ID == deletedLine.ID
	})
	s.Require().True(found, "deleted standard line should be found")
	s.NotNil(deletedInvoiceLine.DeletedAt)
	s.Equal(billing.ManuallyManagedLine, deletedInvoiceLine.ManagedBy)

	chargeAfterDelete := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, deletedLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Equal(flatfee.StatusDeleted, chargeAfterDelete.Status)
	s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
	s.Nil(chargeAfterDelete.Intent.GetBaseIntent().IntentDeletedAt)
	overrideIntent, err := chargeAfterDelete.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
	s.NoError(err)
	s.NotNil(overrideIntent.IntentDeletedAt)
	s.Nil(chargeAfterDelete.Realizations.CurrentRun)
	s.assertCreditThenInvoiceBalances(startBalances)

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	resyncedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: deletedInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll.With(billing.StandardInvoiceExpandDeletedLines),
	})
	s.NoError(err)
	s.DebugDumpInvoice("resynced deleted draft invoice", resyncedInvoice)

	resyncedDeletedLine, found := lo.Find(resyncedInvoice.Lines.OrEmpty(), func(line *billing.StandardLine) bool {
		return line != nil && line.ID == deletedLine.ID
	})
	s.Require().True(found, "deleted standard line should remain on the invoice")
	s.NotNil(resyncedDeletedLine.DeletedAt)
	s.Equal(billing.ManuallyManagedLine, resyncedDeletedLine.ManagedBy)
	s.assertCreditThenInvoiceBalances(startBalances)
}

func (s *CreditThenInvoiceTestSuite) TestDeleteStandardInvoiceWithSingleUsageBasedRunDeletesUsageBasedLine() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice subscription has one usage-based item and one flat-fee item
	// - progressive billing collects the usage-based line before the full period invoice
	// when:
	// - the progressive standard invoice is deleted through the invoice API
	// then:
	// - usage-based line cleanup deletes the invoice
	// - the usage-based charge records the customer-facing line deletion
	s.updateProfile(func(profile *billing.Profile) {
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
		profile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	})

	s.MockStreamingConnector.AddSimpleEvent(
		*s.APIRequestsTotalFeature.MeterSlug,
		10,
		s.mustParseTime("2024-01-02T00:00:00Z"))

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
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee",
								Name: "flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(7),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	})

	clock.FreezeTime(start.Add(time.Minute))
	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 2)

	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:01Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("progressive draft invoice", draftInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)

	usageBasedLine := draftInvoice.Lines.OrEmpty()[0]
	s.Equal(billing.LineEngineTypeChargeUsageBased, usageBasedLine.Engine)
	s.Require().NotNil(usageBasedLine.ChargeID)

	deletedInvoice, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
		Invoice:        draftInvoice.GetInvoiceID(),
		DeletionSource: billing.ChangeSourceAPIRequest,
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleted, deletedInvoice.Status)
	s.NotNil(deletedInvoice.DeletedAt)

	refetchedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: draftInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll.With(billing.StandardInvoiceExpandDeletedLines),
	})
	s.NoError(err)
	s.NotNil(refetchedInvoice.DeletedAt)
	s.Equal(billing.StandardInvoiceStatusDeleted, refetchedInvoice.Status)
	s.Require().Len(refetchedInvoice.Lines.OrEmpty(), 1)

	chargeAfterDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
		Namespace: usageBasedLine.Namespace,
		ID:        *usageBasedLine.ChargeID,
	}, chargesmeta.Expands{
		chargesmeta.ExpandRealizations,
		chargesmeta.ExpandDeletedRealizations,
	})
	s.Equal(usagebased.StatusDeleted, chargeAfterDelete.Status)
	s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
	s.Nil(chargeAfterDelete.Intent.GetBaseIntent().IntentDeletedAt)
	overrideIntent, err := chargeAfterDelete.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
	s.NoError(err)
	s.NotNil(overrideIntent.IntentDeletedAt)
	s.Require().Len(chargeAfterDelete.Realizations, 1)
	s.NotNil(chargeAfterDelete.Realizations[0].DeletedAt)
}

func (s *CreditThenInvoiceTestSuite) TestDeleteStandardInvoiceWithMultipleUsageBasedRunsReturnsProgressiveBillingValidationIssue() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given:
	// - progressive billing has already realized one usage-based standard line
	// - a second progressive standard invoice is waiting for collection
	// when:
	// - the second standard invoice is deleted through the invoice API
	// then:
	// - the delete is rejected because the usage-based charge has multiple non-voided runs
	// - the second invoice remains undeleted
	s.enableProgressiveBilling()

	s.MockStreamingConnector.AddSimpleEvent(
		*s.APIRequestsTotalFeature.MeterSlug,
		10,
		s.mustParseTime("2024-01-02T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(
		*s.APIRequestsTotalFeature.MeterSlug,
		5,
		s.mustParseTime("2024-01-16T00:00:00Z"))

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
			},
		},
	})

	clock.FreezeTime(start.Add(time.Minute))
	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:01Z"))
	firstDraftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
	})
	s.NoError(err)
	s.Require().Len(firstDraftInvoices, 1)

	firstInvoice := firstDraftInvoices[0]
	s.DebugDumpInvoice("first progressive draft invoice", firstInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, firstInvoice.Status)
	s.Require().NotNil(firstInvoice.CollectionAt)

	clock.FreezeTime(firstInvoice.CollectionAt.Add(time.Minute))
	firstInvoice, err = s.BillingService.AdvanceInvoice(ctx, firstInvoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, firstInvoice.Status)

	firstInvoice, err = s.BillingService.ApproveInvoice(ctx, firstInvoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, firstInvoice.Status)

	clock.FreezeTime(s.mustParseTime("2024-01-20T00:00:01Z"))
	secondDraftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-20T00:00:00Z")),
	})
	s.NoError(err)
	s.Require().Len(secondDraftInvoices, 1)

	secondInvoice := secondDraftInvoices[0]
	s.DebugDumpInvoice("second progressive draft invoice", secondInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, secondInvoice.Status)
	s.Require().Len(secondInvoice.Lines.OrEmpty(), 1)

	usageBasedLine := secondInvoice.Lines.OrEmpty()[0]
	s.Equal(billing.LineEngineTypeChargeUsageBased, usageBasedLine.Engine)
	s.Require().NotNil(usageBasedLine.ChargeID)

	chargeBeforeDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
		Namespace: usageBasedLine.Namespace,
		ID:        *usageBasedLine.ChargeID,
	}, chargesmeta.Expands{chargesmeta.ExpandRealizations})
	s.Len(chargeBeforeDelete.Realizations.WithoutVoidedBillingHistory(), 2)

	deletedInvoice, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
		Invoice:        secondInvoice.GetInvoiceID(),
		DeletionSource: billing.ChangeSourceAPIRequest,
	})
	s.Error(err)
	s.ErrorContains(err, billing.ErrCannotEditProgressivelyBilledUsageBasedLine.Error())
	s.Empty(deletedInvoice.ID)

	refetchedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: secondInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)
	s.Nil(refetchedInvoice.DeletedAt)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, refetchedInvoice.Status)
	s.Empty(refetchedInvoice.ValidationIssues)
}

func (s *CreditThenInvoiceTestSuite) TestDeleteStandardInvoiceWithFlatFeeOnlyDeletesFlatFeeLine() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	var subsView subscription.SubscriptionView
	var draftInvoice billing.StandardInvoice
	var flatFeeLine *billing.StandardLine
	var chargeID chargesmeta.ChargeID

	s.Run("create mutable standard invoice", func() {
		// given:
		// - a credit-then-invoice subscription has only one recurring flat-fee charge
		// - auto-advance is disabled so the collected standard invoice stays mutable
		// when:
		// - subscription sync creates the gathering line and billing collects it
		// then:
		// - the flat-fee line is attached to a draft standard invoice
		s.updateProfile(func(profile *billing.Profile) {
			profile.WorkflowConfig.Invoicing.AutoAdvance = false
		})

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "flat-fee",
									Name: "flat-fee",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromFloat(7),
										PaymentTerm: productcatalog.InArrearsPaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
							},
						},
					},
				},
			},
		})

		clock.FreezeTime(start.Add(time.Minute))
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))

		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

		clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		draftInvoice = draftInvoices[0]
		s.DebugDumpInvoice("flat-fee draft invoice", draftInvoice)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
		s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)

		flatFeeLine, err = draftInvoice.Lines.OrEmpty()[0].Clone()
		s.NoError(err)
		s.Equal(billing.LineEngineTypeChargeFlatFee, flatFeeLine.Engine)
		s.Require().NotNil(flatFeeLine.ChargeID)

		chargeBeforeDelete := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, flatFeeLine, chargesmeta.Expands{chargesmeta.ExpandRealizations})
		chargeID = chargeBeforeDelete.GetChargeID()
		s.Equal(flatfee.StatusActiveRealizationProcessing, chargeBeforeDelete.Status)
		s.Require().NotNil(chargeBeforeDelete.Realizations.CurrentRun)
		s.Require().NotNil(chargeBeforeDelete.Realizations.CurrentRun.LineID)
		s.Equal(flatFeeLine.ID, *chargeBeforeDelete.Realizations.CurrentRun.LineID)
	})

	s.Run("delete standard invoice through API", func() {
		// when:
		// - the standard invoice is deleted through the invoice API
		// then:
		// - the invoice deletion succeeds
		// - the flat-fee charge records the customer-facing line deletion as an override
		deletedInvoice, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
			Invoice:        draftInvoice.GetInvoiceID(),
			DeletionSource: billing.ChangeSourceAPIRequest,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDeleted, deletedInvoice.Status)
		s.NotNil(deletedInvoice.DeletedAt)

		chargeAfterDelete := s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, flatFeeLine, chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Equal(flatfee.StatusDeleted, chargeAfterDelete.Status)
		s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
		s.Nil(chargeAfterDelete.Intent.GetBaseIntent().IntentDeletedAt)
		overrideIntent, err := chargeAfterDelete.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)
		s.Nil(chargeAfterDelete.Realizations.CurrentRun)
	})

	s.Run("subscription cancellation reconciles deleted charge base intent", func() {
		// when:
		// - the active subscription is canceled after the customer-facing override delete
		// - subscription sync reconciles the canceled subscription
		// then:
		// - sync shrinks the hidden base/source intent without entering charge lifecycle
		// - the deleted override remains customer-facing
		cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")
		clock.FreezeTime(cancelAt)

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(err)

		canceledSubsView, err := s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SyncByView(ctx, canceledSubsView, cancelAt))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

		chargeAfterCancelGeneric, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands: chargesmeta.Expands{
				chargesmeta.ExpandRealizations,
				chargesmeta.ExpandDeletedRealizations,
			},
		})
		s.NoError(err)

		chargeAfterCancel, err := chargeAfterCancelGeneric.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatfee.StatusDeleted, chargeAfterCancel.Status)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().ServicePeriod.To)
		s.Equal(cancelAt, chargeAfterCancel.Intent.GetBaseIntent().BillingPeriod.To)
		s.True(chargeAfterCancel.Intent.HasOverrideLayer(), "override layer")
		overrideIntent, err := chargeAfterCancel.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
		s.NoError(err)
		s.NotNil(overrideIntent.IntentDeletedAt)
		s.Nil(chargeAfterCancel.Realizations.CurrentRun)
	})
}

func (s *CreditThenInvoiceTestSuite) TestInArrearsOneTimeFeeSyncing() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

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
	s.assertCreditThenInvoiceBalances(startBalances)

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.assertCreditThenInvoiceBalances(startBalances)

	// let's cancel the subscription
	cancelAt := s.mustParseTime("2024-01-15T00:00:00Z")

	subs, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Custom: &cancelAt,
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subs.NamespacedID)
	s.NoError(err)

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(startBalances)

	expectedCharges := []expectedCharge{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears",
				Version:  0,
			},

			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
				},
			},
		},
	}

	s.assertCharges(ctx, subsView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedGatheringUpdate() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - we have a credit-then-invoice subscription with a single phase with an usage based price
	// - the gathering invoice contains the items
	// when:
	// - we add a new phase, that disrupts the period of previous items with a new usage based price for the same feature
	// then:
	// - the gathering invoice is updated, the period of the previous items are updated accordingly
	// - charges are reconciled without ledger movement
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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
			},
			Phases: []productcatalog.Phase{
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
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	initialExpectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, initialExpectedCharges)

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datetime.MustParseDuration(s.T(), "P2D"),
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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.Run("dry run does not repair the persisted charge item reference", func() {
		chargeBeforeDryRun := s.mustGetOnlyUsageBasedCharge(ctx, subsView.Subscription.ID)
		s.Require().NotNil(chargeBeforeDryRun.Intent.GetSubscription())
		itemIDBeforeDryRun := chargeBeforeDryRun.Intent.GetSubscription().ItemID

		phase := s.getPhaseByKey(s.T(), updatedSubsView, "first-phase")
		targetItemID := phase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID
		s.NotEqual(itemIDBeforeDryRun, targetItemID)

		s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z"), subscriptionsync.EnableDryRun()))

		chargeAfterDryRun := s.mustGetOnlyUsageBasedCharge(ctx, subsView.Subscription.ID)
		s.Require().NotNil(chargeAfterDryRun.Intent.GetSubscription())
		s.Equal(itemIDBeforeDryRun, chargeAfterDryRun.Intent.GetSubscription().ItemID)
	})

	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-02-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	updatedExpectedCharges := []expectedCharge{
		// we'll have the single line in the first phase truncated to its 2 day length
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-03T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-03T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-03T00:00:00Z")),
				},
			},
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
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-03T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, updatedSubsView, updatedExpectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedGatheringUpdateDraftInvoice() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - we have a credit-then-invoice subscription with a single phase with an usage based price
	// - a draft invoice has been created
	// when:
	// - we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for the same feature
	// then:
	// - the gathering invoice is updated, the period of the previous items are updated accordingly in the draft invoice
	// - charges are reconciled without ledger movement before invoice issuance
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario

	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))

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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	// we sync two months so we have lines on gathering
	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	initialExpectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   s.APIRequestsTotalFeature.Key,
						Version:   0,
						PeriodMin: 0,
						PeriodMax: 0,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   s.APIRequestsTotalFeature.Key,
						Version:   0,
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, initialExpectedCharges)

	// Some time has passed, we're syncing the draft invoice
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, draftInvoice.Status)
	s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
	finalRealizationLine := draftInvoice.Lines.OrEmpty()[0]
	s.Require().NotNil(finalRealizationLine.ChargeID)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActiveRealizationWaitingForCollection),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			Realizations: []expectedChargeRealization{
				{
					Status:   draftInvoice.Status,
					BookedAt: s.mustParseTime("2024-02-01T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(110),
						Total:  alpacadecimal.NewFromFloat(110),
					},
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	})

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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
					StartAfter: datetime.MustParseDuration(s.T(), "P30D"),
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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Now the time-travel is over, let's reset back to the "present"
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	updatedExpectedCharges := []expectedCharge{
		// The mutable draft line is deleted and recreated as a gathering line for the shortened first phase.
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActive),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-31T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-31T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-31T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-31T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
		},
	}
	s.assertCharges(ctx, updatedSubsView, updatedExpectedCharges)

	updatedDraftInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: draftInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - 2nd sync", updatedDraftInvoice)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
	s.Equal(billing.StandardInvoiceStatusDeleted, updatedDraftInvoice.Status)
	s.expectLines(updatedDraftInvoice, subsView.Subscription.ID, nil)

	chargeAfterDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
		Namespace: finalRealizationLine.Namespace,
		ID:        *finalRealizationLine.ChargeID,
	}, chargesmeta.Expands{
		chargesmeta.ExpandRealizations,
		chargesmeta.ExpandDeletedRealizations,
	})
	s.Equal(usagebased.StatusActive, chargeAfterDelete.Status)
	s.Nil(chargeAfterDelete.State.CurrentRealizationRunID)
	deletedRun, err := chargeAfterDelete.Realizations.GetByLineID(finalRealizationLine.ID)
	s.NoError(err)
	s.NotNil(deletedRun.DeletedAt)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedGatheringUpdateIssuedInvoice() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - we have a credit-then-invoice subscription with a single phase with an usage based price
	// - an issued invoice has been created
	// when:
	// - we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for the same feature
	// then:
	// - the gathering invoice is updated
	// - the paid invoice and its ledger bookings are not updated
	// - a validation issue is added for the immutable invoice change
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario
	//
	// NOTE: This is variant of the TestUsageBasedGatheringUpdateDraftInvoice so we are keeping the checks at a minimum here

	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        clock.Now(),
	})

	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	afterDraftInvoiceBalances := expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	}
	afterIssuedInvoiceBalances := expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         120,
		AccruedPromotional: 2,
		AccruedInvoice:     118,
		WashAll:            -120,
		WashPromotional:    -2,
		WashInvoice:        -118,
	}
	s.assertCreditThenInvoiceBalances(startBalances)

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))
	// We need usage at the period change to trigger the validation issue
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-31T12:00:00Z"))

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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(startBalances)

	initialExpectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   s.APIRequestsTotalFeature.Key,
						Version:   0,
						PeriodMin: 0,
						PeriodMax: 0,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   s.APIRequestsTotalFeature.Key,
						Version:   0,
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, initialExpectedCharges)

	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)
	s.assertCreditThenInvoiceBalances(afterDraftInvoiceBalances)

	draftInvoice := draftInvoices[0]
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, draftInvoice.Status)

	s.Require().NotNil(draftInvoice.CollectionAt)
	clock.FreezeTime(draftInvoice.CollectionAt.Add(time.Minute))
	draftInvoice, err = s.BillingService.AdvanceInvoice(ctx, draftInvoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)
	s.assertCreditThenInvoiceBalances(afterDraftInvoiceBalances)

	issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, issuedInvoice.Status)
	s.Len(issuedInvoice.ValidationIssues, 0)
	s.DebugDumpInvoice("issued invoice", issuedInvoice)
	s.assertCreditThenInvoiceBalances(afterIssuedInvoiceBalances)
	issuedInvoiceLines := issuedInvoice.Lines.OrEmpty()
	s.Require().Len(issuedInvoiceLines, 1)
	issuedInvoiceLine := issuedInvoiceLines[0]
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(12), *issuedInvoiceLine.GetQuantity(), "issued invoice quantity")
	s.True(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(10),
	}).Equal(issuedInvoiceLine.GetPrice()), "issued invoice price")
	s.Equal(timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-01T00:00:00Z"),
		To:   s.mustParseTime("2024-02-01T00:00:00Z"),
	}, issuedInvoiceLine.GetServicePeriod())

	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.assertCreditThenInvoiceBalances(afterIssuedInvoiceBalances)

	// Now lets travel back in time
	clock.FreezeTime(s.mustParseTime("2024-01-30T00:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datetime.MustParseDuration(s.T(), "P30D"),
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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(afterIssuedInvoiceBalances)

	// gathering invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActive),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-31T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-31T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-31T00:00:00Z")),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
					Status:   billing.StandardInvoiceStatusPaid,
					IsVoided: true,
					BookedAt: s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-31T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	})

	updatedIssuedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: issuedInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("issued invoice - 2nd sync", updatedIssuedInvoice)
	s.assertCreditThenInvoiceBalances(afterIssuedInvoiceBalances)

	updatedIssuedInvoiceLines := updatedIssuedInvoice.Lines.OrEmpty()
	s.Require().Len(updatedIssuedInvoiceLines, 1)
	updatedIssuedInvoiceLine := updatedIssuedInvoiceLines[0]
	s.AssertDecimalEqual(alpacadecimal.NewFromFloat(12), *updatedIssuedInvoiceLine.GetQuantity(), "updated issued invoice quantity")
	s.True(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(10),
	}).Equal(updatedIssuedInvoiceLine.GetPrice()), "updated issued invoice price")
	s.Equal(timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-01T00:00:00Z"),
		To:   s.mustParseTime("2024-02-01T00:00:00Z"), // This is not updated, which is what we want
	}, updatedIssuedInvoiceLine.GetServicePeriod())

	s.Len(updatedIssuedInvoice.ValidationIssues, 1)
	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedUpdateWithLineSplits() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

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

	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	})

	// invoice 1: issued invoice creation
	clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
	draftInvoices1, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
	})
	s.NoError(err)
	s.Len(draftInvoices1, 1)

	s.Require().NotNil(draftInvoices1[0].CollectionAt)
	clock.FreezeTime(draftInvoices1[0].CollectionAt.Add(time.Minute))
	invoice1, err := s.BillingService.AdvanceInvoice(ctx, draftInvoices1[0].GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice1.Status)

	invoice1, err = s.BillingService.ApproveInvoice(ctx, invoice1.GetInvoiceID())
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice1.Status)

	s.populateChildIDsFromParents(&invoice1)
	s.DebugDumpInvoice("issued invoice1", invoice1)

	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActive),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-15T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-15T00:00:00Z"),
					},
					Status:   billing.StandardInvoiceStatusPaid,
					BookedAt: s.mustParseTime("2024-01-15T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(20),
						Total:  alpacadecimal.NewFromFloat(20),
					},
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
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
	s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, draftInvoice2.Status)
	s.Require().Len(draftInvoice2.Lines.OrEmpty(), 1)
	draftInvoice2Line := draftInvoice2.Lines.OrEmpty()[0]
	s.Require().NotNil(draftInvoice2Line.ChargeID)
	progressiveRemainingPeriodBeforeDelete := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-18T00:00:00Z"),
		To:   s.mustParseTime("2024-02-01T00:00:00Z"),
	}

	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActiveRealizationWaitingForCollection),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-18T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-15T00:00:00Z"),
					},
					Status:   billing.StandardInvoiceStatusPaid,
					BookedAt: s.mustParseTime("2024-01-15T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(20),
						Total:  alpacadecimal.NewFromFloat(20),
					},
				},
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-15T00:00:00Z"),
						To:   s.mustParseTime("2024-01-18T00:00:00Z"),
					},
					Status:   billing.StandardInvoiceStatusDraftWaitingForCollection,
					BookedAt: s.mustParseTime("2024-01-18T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(30),
						Total:  alpacadecimal.NewFromFloat(30),
					},
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	})

	// gathering invoice checks
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)
	_, foundProgressiveRemainingLine := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ChargeID != nil &&
			*line.ChargeID == *draftInvoice2Line.ChargeID &&
			line.ServicePeriod == progressiveRemainingPeriodBeforeDelete
	})
	s.True(foundProgressiveRemainingLine, "progressive remaining gathering line should exist before draft standard-line delete")

	clock.FreezeTime(s.mustParseTime("2024-01-09T12:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datetime.MustParseDuration(s.T(), "P10D"),
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
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}.AsPatch(),
	}, s.timingImmediate())

	s.NoError(err)
	s.NotNil(updatedSubsView)

	// THEN
	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-01-18T00:00:00Z"))
	s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-03-01T00:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)
	_, foundProgressiveRemainingLine = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ChargeID != nil &&
			*line.ChargeID == *draftInvoice2Line.ChargeID &&
			line.ServicePeriod == progressiveRemainingPeriodBeforeDelete
	})
	s.False(foundProgressiveRemainingLine, "progressive standard-line delete should delete the remaining gathering line for the same charge")

	s.assertCharges(ctx, updatedSubsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActive),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-11T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-11T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-11T00:00:00Z")),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-01-15T00:00:00Z"),
					},
					Status:   billing.StandardInvoiceStatusPaid,
					IsVoided: true,
					BookedAt: s.mustParseTime("2024-01-15T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(20),
						Total:  alpacadecimal.NewFromFloat(20),
					},
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusDeleted),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-11T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	})

	// invoice 1 (issued) checks
	updatedIssuedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice1.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedIssuedInvoice)
	s.DebugDumpInvoice("invoice1 (issued) - 2nd sync", updatedIssuedInvoice)

	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])

	// invoice 2 (draft) checks
	updatedDraftInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: draftInvoice2.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedDraftInvoice)
	s.DebugDumpInvoice("draft invoice2 - 2nd sync", updatedDraftInvoice)
	s.Len(updatedDraftInvoice.Lines.OrEmpty(), 0)
	s.Equal(billing.StandardInvoiceStatusDeleted, updatedDraftInvoice.Status)

	chargeAfterDraftLineDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
		Namespace: draftInvoice2Line.Namespace,
		ID:        *draftInvoice2Line.ChargeID,
	}, chargesmeta.Expands{
		chargesmeta.ExpandRealizations,
		chargesmeta.ExpandDeletedRealizations,
	})
	s.Equal(usagebased.StatusActive, chargeAfterDraftLineDelete.Status)
	s.Nil(chargeAfterDraftLineDelete.State.CurrentRealizationRunID)
	deletedRun, err := chargeAfterDraftLineDelete.Realizations.GetByLineID(draftInvoice2Line.ID)
	s.NoError(err)
	s.NotNil(deletedRun.DeletedAt)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedProgressiveStandardInvoiceDeletionDeletesGatheringLine() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	// given:
	// - a credit-then-invoice usage-based charge has one paid progressive run
	// - a second mutable progressive standard invoice is waiting for collection
	// - that second run has a remaining gathering line on the same charge
	// when:
	// - system code deletes the progressive standard invoice
	// then:
	// - the standard invoice is deleted
	// - the current realization run is deleted and detached from the charge
	// - the remaining gathering line is deleted with the standard invoice line
	var draftInvoice billing.StandardInvoice
	var draftLine *billing.StandardLine
	var chargeID chargesmeta.ChargeID
	remainingGatheringPeriod := timeutil.ClosedPeriod{
		From: s.mustParseTime("2024-01-18T00:00:00Z"),
		To:   s.mustParseTime("2024-02-01T00:00:00Z"),
	}

	s.Run("create progressive draft invoice", func() {
		// given:
		// - a credit-then-invoice usage-based subscription with progressive billing enabled
		// when:
		// - sync creates the subscription charges and two progressive standard invoices
		// then:
		// - the second draft invoice and its backing charge are ready for deletion assertions
		s.enableProgressiveBilling()

		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-12T09:30:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-15T11:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 7, s.mustParseTime("2024-01-18T12:30:00Z"))

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

		clock.FreezeTime(clock.Now().Add(time.Minute))
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))

		clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
		firstDraftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
		})
		s.NoError(err)
		s.Require().Len(firstDraftInvoices, 1)

		firstInvoice := firstDraftInvoices[0]
		s.Require().NotNil(firstInvoice.CollectionAt)
		clock.FreezeTime(firstInvoice.CollectionAt.Add(time.Minute))
		firstInvoice, err = s.BillingService.AdvanceInvoice(ctx, firstInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, firstInvoice.Status)

		firstInvoice, err = s.BillingService.ApproveInvoice(ctx, firstInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, firstInvoice.Status)

		clock.FreezeTime(s.mustParseTime("2024-01-18T00:00:00Z"))
		secondDraftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(s.mustParseTime("2024-01-18T00:00:00Z")),
		})
		s.NoError(err)
		s.Require().Len(secondDraftInvoices, 1)

		draftInvoice = secondDraftInvoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, draftInvoice.Status)
		s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
		draftLine = draftInvoice.Lines.OrEmpty()[0]
		s.Equal(billing.LineEngineTypeChargeUsageBased, draftLine.Engine)
		s.Require().NotNil(draftLine.ChargeID)

		chargeID = chargesmeta.ChargeID{
			Namespace: draftLine.Namespace,
			ID:        *draftLine.ChargeID,
		}
	})

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice before standard invoice delete", gatheringInvoice)
	_, foundRemainingLine := lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ChargeID != nil &&
			*line.ChargeID == chargeID.ID &&
			line.ServicePeriod == remainingGatheringPeriod
	})
	s.True(foundRemainingLine, "progressive remaining gathering line should exist before standard invoice delete")

	deletedInvoice, err := s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
		Invoice:        draftInvoice.GetInvoiceID(),
		DeletionSource: billing.ChangeSourceSystem,
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleted, deletedInvoice.Status)
	s.NotNil(deletedInvoice.DeletedAt)

	refetchedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: draftInvoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll.With(billing.StandardInvoiceExpandDeletedLines),
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleted, refetchedInvoice.Status)
	s.NotNil(refetchedInvoice.DeletedAt)
	s.Require().Len(refetchedInvoice.Lines.OrEmpty(), 1)

	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice after standard invoice delete", gatheringInvoice)
	_, foundRemainingLine = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
		return line.ChargeID != nil &&
			*line.ChargeID == chargeID.ID &&
			line.ServicePeriod == remainingGatheringPeriod
	})
	s.False(foundRemainingLine, "progressive standard invoice delete should delete the remaining gathering line for the same charge")

	chargeAfterInvoiceDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeID, chargesmeta.Expands{
		chargesmeta.ExpandRealizations,
		chargesmeta.ExpandDeletedRealizations,
	})
	s.Equal(usagebased.StatusActive, chargeAfterInvoiceDelete.Status)
	s.Nil(chargeAfterInvoiceDelete.State.CurrentRealizationRunID)
	deletedRun, err := chargeAfterInvoiceDelete.Realizations.GetByLineID(draftLine.ID)
	s.NoError(err)
	s.NotNil(deletedRun.DeletedAt)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedProgressiveGatheringLineManualDeleteShrinksCharge() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	var subsView subscription.SubscriptionView
	var progressiveInvoice billing.StandardInvoice
	var progressiveLine *billing.StandardLine
	var chargeID chargesmeta.ChargeID
	var gatheringInvoice billing.GatheringInvoice
	var remainingLine billing.GatheringLine
	var deletedLine billing.GatheringLine

	s.Run("create progressive usage-based subscription", func() {
		// given:
		// - progressive billing is enabled
		// - visible usage exists before the progressive invoice cutoff
		// - subscription sync owns a credit-then-invoice usage-based charge
		s.enableProgressiveBilling()

		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-12T09:30:00Z"))

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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

		clock.FreezeTime(clock.Now().Add(time.Minute))
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-03-01T00:00:00Z")))
	})

	s.Run("create paid progressive invoice and remaining gathering tail", func() {
		// given:
		// - a progressive standard invoice is created for the first part of the period
		// - the invoice is advanced and paid
		// then:
		// - a remaining gathering invoice line exists for the unbilled tail
		clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		progressiveInvoice = draftInvoices[0]
		s.Require().Len(progressiveInvoice.Lines.OrEmpty(), 1)
		progressiveLine = progressiveInvoice.Lines.OrEmpty()[0]
		s.Require().NotNil(progressiveLine.ChargeID)
		chargeID = chargesmeta.ChargeID{
			Namespace: progressiveLine.Namespace,
			ID:        *progressiveLine.ChargeID,
		}
		s.Require().NotNil(progressiveInvoice.CollectionAt)

		clock.FreezeTime(progressiveInvoice.CollectionAt.Add(time.Minute))
		progressiveInvoice, err = s.BillingService.AdvanceInvoice(ctx, progressiveInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, progressiveInvoice.Status)

		progressiveInvoice, err = s.BillingService.ApproveInvoice(ctx, progressiveInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, progressiveInvoice.Status)

		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.populateChildIDsFromParents(&gatheringInvoice)
		s.DebugDumpInvoice("gathering invoice before manual delete", gatheringInvoice)

		var found bool
		remainingLine, found = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ChargeID != nil && *line.ChargeID == chargeID.ID
		})
		s.Require().True(found, "remaining gathering line should exist before manual delete")
		s.Equal(progressiveLine.GetServicePeriod().To, remainingLine.ServicePeriod.From)
	})

	s.Run("delete remaining gathering line through API", func() {
		// when:
		// - the user deletes the remaining gathering line through the invoice API
		_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				lines := invoice.Lines.OrEmpty()
				for idx := range lines {
					if lines[idx].ID != remainingLine.ID {
						continue
					}

					lines[idx].DeletedAt = lo.ToPtr(clock.Now())

					clonedLine, err := lines[idx].Clone()
					s.NoError(err)
					deletedLine = clonedLine

					return nil
				}

				return fmt.Errorf("remaining gathering line not found")
			},
			IncludeDeletedLines: true,
		})
		s.NoError(err)
	})

	s.Run("assert charge was shrunk to realized period", func() {
		// then:
		// - the standard invoice history remains intact
		// - the charge effective period is manually shortened to the deleted line's start
		// - the kept partial realization becomes the final realization
		editedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
			Invoice: gatheringInvoice.GetInvoiceID(),
			Expand: billing.GatheringInvoiceExpands{
				billing.GatheringInvoiceExpandLines,
				billing.GatheringInvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)
		s.DebugDumpInvoice("gathering invoice after manual delete", editedInvoice)
		editedLine, found := lo.Find(editedInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ID == deletedLine.ID
		})
		s.True(found, "deleted gathering line should be present when deleted lines are expanded")
		s.NotNil(editedLine.DeletedAt)
		s.Equal(billing.ManuallyManagedLine, editedLine.ManagedBy)

		refetchedStandardInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: progressiveInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, refetchedStandardInvoice.Status)
		s.Require().Len(refetchedStandardInvoice.Lines.OrEmpty(), 1)
		s.Equal(progressiveLine.ID, refetchedStandardInvoice.Lines.OrEmpty()[0].ID)

		chargeAfterDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeID, chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Equal(usagebased.StatusActive, chargeAfterDelete.Status)
		s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
		s.Equal(s.mustParseTime("2024-02-01T00:00:00Z"), chargeAfterDelete.Intent.GetBaseIntent().ServicePeriod.To)
		s.Equal(deletedLine.ServicePeriod.From, chargeAfterDelete.Intent.GetEffectiveServicePeriod().To)
		s.Require().Len(chargeAfterDelete.Realizations.WithoutVoidedBillingHistory(), 1)
		keptRun := chargeAfterDelete.Realizations.WithoutVoidedBillingHistory()[0]
		s.Equal(usagebased.RealizationRunTypeFinalRealization, keptRun.Type)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, keptRun.InitialType)
		s.Nil(keptRun.DeletedAt)
	})

	s.Run("subscription sync does not recreate deleted tail", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedProgressiveGatheringInvoiceManualDeleteFinalizesCharge() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	var subsView subscription.SubscriptionView
	var progressiveInvoice billing.StandardInvoice
	var progressiveLine *billing.StandardLine
	var chargeID chargesmeta.ChargeID
	var gatheringInvoice billing.GatheringInvoice
	var remainingLine billing.GatheringLine

	s.Run("create progressive usage-based subscription", func() {
		// given:
		// - progressive billing is enabled
		// - visible usage exists before the progressive invoice cutoff
		// - subscription sync owns a credit-then-invoice usage-based charge
		s.enableProgressiveBilling()

		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T00:00:00Z"))
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-12T09:30:00Z"))

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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

		clock.FreezeTime(clock.Now().Add(time.Minute))
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
	})

	s.Run("create paid progressive invoice and remaining gathering tail", func() {
		// given:
		// - a progressive standard invoice is created for the first part of the period
		// - the invoice is advanced and paid
		// then:
		// - one remaining gathering invoice line exists for the unbilled tail
		clock.FreezeTime(s.mustParseTime("2024-01-15T00:00:00Z"))
		draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
		})
		s.NoError(err)
		s.Require().Len(draftInvoices, 1)

		progressiveInvoice = draftInvoices[0]
		s.Require().Len(progressiveInvoice.Lines.OrEmpty(), 1)
		progressiveLine = progressiveInvoice.Lines.OrEmpty()[0]
		s.Require().NotNil(progressiveLine.ChargeID)
		chargeID = chargesmeta.ChargeID{
			Namespace: progressiveLine.Namespace,
			ID:        *progressiveLine.ChargeID,
		}
		s.Require().NotNil(progressiveInvoice.CollectionAt)

		clock.FreezeTime(progressiveInvoice.CollectionAt.Add(time.Minute))
		progressiveInvoice, err = s.BillingService.AdvanceInvoice(ctx, progressiveInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, progressiveInvoice.Status)

		progressiveInvoice, err = s.BillingService.ApproveInvoice(ctx, progressiveInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, progressiveInvoice.Status)

		gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.populateChildIDsFromParents(&gatheringInvoice)
		s.DebugDumpInvoice("gathering invoice before manual delete", gatheringInvoice)
		s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

		var found bool
		remainingLine, found = lo.Find(gatheringInvoice.Lines.OrEmpty(), func(line billing.GatheringLine) bool {
			return line.ChargeID != nil && *line.ChargeID == chargeID.ID
		})
		s.Require().True(found, "remaining gathering line should exist before manual delete")
		s.Equal(progressiveLine.GetServicePeriod().To, remainingLine.ServicePeriod.From)
	})

	s.Run("delete gathering invoice and schedule charge advancement", func() {
		// when:
		// - the user deletes the remaining gathering invoice through the invoice API
		// then:
		// - the paid standard invoice history remains intact
		// - the charge effective period is manually shortened to the deleted tail start
		// - the kept partial realization becomes the final realization
		// - the charge remains active but is scheduled for immediate advancement from the shrunk boundary
		deletedInvoice, err := s.BillingService.DeleteGatheringInvoice(ctx, billing.DeleteInvoiceInput{
			Invoice:        gatheringInvoice.GetInvoiceID(),
			DeletionSource: billing.ChangeSourceAPIRequest,
		})
		s.NoError(err)
		s.NotNil(deletedInvoice.DeletedAt)

		refetchedStandardInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: progressiveInvoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, refetchedStandardInvoice.Status)
		s.Require().Len(refetchedStandardInvoice.Lines.OrEmpty(), 1)
		s.Equal(progressiveLine.ID, refetchedStandardInvoice.Lines.OrEmpty()[0].ID)

		chargeAfterDelete := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeID, chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Equal(usagebased.StatusActive, chargeAfterDelete.Status)
		s.Require().NotNil(chargeAfterDelete.State.AdvanceAfter)
		s.True(chargeAfterDelete.Intent.HasOverrideLayer(), "override layer")
		s.Equal(s.mustParseTime("2024-02-01T00:00:00Z"), chargeAfterDelete.Intent.GetBaseIntent().ServicePeriod.To)
		s.Equal(remainingLine.ServicePeriod.From, chargeAfterDelete.Intent.GetEffectiveServicePeriod().To)
		s.True(chargeAfterDelete.Intent.GetEffectiveServicePeriod().To.Equal(*chargeAfterDelete.State.AdvanceAfter))
		s.Require().Len(chargeAfterDelete.Realizations.WithoutVoidedBillingHistory(), 1)
		keptRun := chargeAfterDelete.Realizations.WithoutVoidedBillingHistory()[0]
		s.Equal(usagebased.RealizationRunTypeFinalRealization, keptRun.Type)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, keptRun.InitialType)
		s.Nil(keptRun.DeletedAt)
	})

	s.Run("advance charge to final", func() {
		// when:
		// - the charge worker advances the active charge after the shrunk boundary
		// then:
		// - the already-paid final realization lets the charge reach final
		_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: s.Customer.GetID(),
		})
		s.NoError(err)

		chargeAfterAdvance := s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargeID, chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDeletedRealizations,
		})
		s.Equal(usagebased.StatusFinal, chargeAfterAdvance.Status)
	})

	s.Run("subscription sync does not recreate deleted tail", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-02-01T00:00:00Z")))
		s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	})
}

func (s *CreditThenInvoiceTestSuite) TestRateCardTaxSyncFlatFee() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have tax information set in the rate card
	// When
	//  we synchronize the subscription phases
	// Then
	//  the gathering invoice will contain the tax details

	// The namespace's default invoicing tax code is auto-stamped on charges and
	// propagated lines when the rate card leaves TaxCodeID nil. Set it explicitly
	// here so the rate card carries the tax code we then assert is propagated.
	defaults, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.Require().NoError(err)

	taxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(defaults.InvoicingTaxCodeID),
	}

	var subsView subscription.SubscriptionView
	var updatedSubsView subscription.SubscriptionView
	var draftInvoices []billing.StandardInvoice

	s.Run("create subscription", func() {
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "in-arrears",
									Name: "in-arrears",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromFloat(5),
										PaymentTerm: productcatalog.InArrearsPaymentTerm,
									}),
									TaxConfig: taxConfig,
								},
								BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
							},
						},
					},
				},
			},
		})

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(clock.Now().Add(time.Minute))
	})

	s.Run("gathering invoice", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: s.Customer.GetID(),
		})
		s.NoError(err)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
		s.assertCreditThenInvoiceChargeTaxConfigs(ctx, subsView.Subscription.ID, chargesmeta.ChargeTypeFlatFee, taxConfig)

		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

		s.assertGatheringLineTaxConfigs(gatheringInvoice.Lines.OrEmpty(), taxConfig)
	})

	s.Run("gathering invoice after edit", func() {
		// Given we edit the subscription the tax config is carried over to the lines

		clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))
		var err error
		updatedSubsView, err = s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
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
				BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
			}.AsPatch(),
		}, s.timingImmediate())
		s.NoError(err)
		s.NotNil(updatedSubsView)

		s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: s.Customer.GetID(),
		})
		s.NoError(err)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
		s.assertCreditThenInvoiceChargeTaxConfigs(ctx, updatedSubsView.Subscription.ID, chargesmeta.ChargeTypeFlatFee, taxConfig)

		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - after edit", gatheringInvoice)

		s.assertGatheringLineTaxConfigs(gatheringInvoice.Lines.OrEmpty(), taxConfig)
	})

	s.Run("draft invoices", func() {
		var err error
		draftAt := s.mustParseTime("2024-02-01T00:00:00Z")
		for at := s.mustParseTime("2024-01-03T00:00:00Z"); at.Before(draftAt); at = at.AddDate(0, 0, 1) {
			clock.FreezeTime(at)
			_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
				Customer: s.Customer.GetID(),
			})
			s.NoError(err)
		}
		clock.FreezeTime(draftAt)

		draftInvoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().NotEmpty(draftInvoices)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		for idx, draftInvoice := range draftInvoices {
			s.DebugDumpInvoice(fmt.Sprintf("draft invoice %d", idx), draftInvoice)
			s.assertStandardLineTaxConfigs(draftInvoice.Lines.OrEmpty(), taxConfig)
		}
	})

	s.Run("issued invoices", func() {
		s.Require().NotEmpty(draftInvoices)

		for idx, draftInvoice := range draftInvoices {
			issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.GetInvoiceID())
			s.NoError(err)
			s.DebugDumpInvoice(fmt.Sprintf("issued invoice %d", idx), issuedInvoice)
			s.assertStandardLineTaxConfigs(issuedInvoice.Lines.OrEmpty(), taxConfig)
		}
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			AccruedAll:     24.68,
			AccruedInvoice: 24.68,
			WashAll:        -24.68,
			WashInvoice:    -24.68,
		})
	})
}

func (s *CreditThenInvoiceTestSuite) TestRateCardTaxSyncUsageBased() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have tax information set in the rate card
	// When
	//  we synchronize the subscription phases
	// Then
	//  the gathering invoice will contain the tax details

	// The namespace's default invoicing tax code is auto-stamped on charges and
	// propagated lines when the rate card leaves TaxCodeID nil. Set it explicitly
	// here so the rate card carries the tax code we then assert is propagated.
	defaults, err := s.TaxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: s.Namespace})
	s.Require().NoError(err)

	taxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr(defaults.InvoicingTaxCodeID),
	}

	var subsView subscription.SubscriptionView
	var updatedSubsView subscription.SubscriptionView
	var draftInvoices []billing.StandardInvoice

	s.Run("create subscription", func() {
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
		s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2024-01-01T10:00:00Z"))

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
									Key:        s.APIRequestsTotalFeature.Key,
									Name:       s.APIRequestsTotalFeature.Key,
									FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
									FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
									Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
										Amount: alpacadecimal.NewFromFloat(5),
									}),
									TaxConfig: taxConfig,
								},
								BillingCadence: datetime.MustParseDuration(s.T(), "P1D"),
							},
						},
					},
				},
			},
		})

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(clock.Now().Add(time.Minute))
	})

	s.Run("gathering invoice", func() {
		s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
		s.assertCreditThenInvoiceChargeTaxConfigs(ctx, subsView.Subscription.ID, chargesmeta.ChargeTypeUsageBased, taxConfig)

		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

		s.assertGatheringLineTaxConfigs(gatheringInvoice.Lines.OrEmpty(), taxConfig)
	})

	s.Run("gathering invoice after edit", func() {
		// Given we edit the subscription the tax config is carried over to the lines

		clock.FreezeTime(s.mustParseTime("2024-01-02T00:00:00Z"))
		var err error
		updatedSubsView, err = s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
			patch.PatchRemoveItem{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			subscriptionAddItem{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				}),
				FeatureKey:     s.APIRequestsTotalFeature.Key,
				TaxConfig:      taxConfig,
				BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1D")),
			}.AsPatch(),
		}, s.timingImmediate())
		s.NoError(err)
		s.NotNil(updatedSubsView)

		s.NoError(s.Service.SyncByView(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
		s.assertCreditThenInvoiceChargeTaxConfigs(ctx, updatedSubsView.Subscription.ID, chargesmeta.ChargeTypeUsageBased, taxConfig)

		gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
		s.DebugDumpInvoice("gathering invoice - after edit", gatheringInvoice)

		s.assertGatheringLineTaxConfigs(gatheringInvoice.Lines.OrEmpty(), taxConfig)
	})

	s.Run("draft invoices", func() {
		clock.FreezeTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: s.Customer.GetID(),
		})
		s.NoError(err)

		draftInvoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(clock.Now()),
		})
		s.NoError(err)
		s.Require().NotEmpty(draftInvoices)
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		for idx, draftInvoice := range draftInvoices {
			s.DebugDumpInvoice(fmt.Sprintf("draft invoice %d", idx), draftInvoice)
			s.assertStandardLineTaxConfigs(draftInvoice.Lines.OrEmpty(), taxConfig)
		}
	})

	s.Run("issued invoices", func() {
		s.Require().NotEmpty(draftInvoices)

		for idx, draftInvoice := range draftInvoices {
			s.Require().NotNil(draftInvoice.CollectionAt)
			clock.FreezeTime(draftInvoice.CollectionAt.Add(time.Minute))
			readyInvoice, err := s.BillingService.AdvanceInvoice(ctx, draftInvoice.GetInvoiceID())
			s.NoError(err)
			s.assertStandardLineTaxConfigs(readyInvoice.Lines.OrEmpty(), taxConfig)
			s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

			issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.GetInvoiceID())
			s.NoError(err)
			s.DebugDumpInvoice(fmt.Sprintf("issued invoice %d", idx), issuedInvoice)
			s.assertStandardLineTaxConfigs(issuedInvoice.Lines.OrEmpty(), taxConfig)
		}
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			AccruedAll:     60,
			AccruedInvoice: 60,
			WashAll:        -60,
			WashInvoice:    -60,
		})
	})
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceInstantBillingOnSubscriptionCreation() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)

	// Given
	//  we have a subscription with a single phase with an in advance fee
	// When
	//  we start the subscription
	// Then
	//  the gathering invoice will automatically be invoiced so that the in advance fee is billed (those are always flat fees)
	//
	// Note that the UBP line is not synced because the subscription is not active yet

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        start,
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	})

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
									Amount:      alpacadecimal.NewFromFloat(6),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(start.Add(time.Minute))

	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, start))
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	})

	// in-arrears lines wont get synced with this deadline so we'll only have the in advance line on the draft invoice
	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
		Expand:     billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.Len(invoices.Items, 1)

	instantInvoice, err := invoices.Items[0].AsStandardInvoice()
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, instantInvoice.Status)

	s.DebugDumpInvoice("instant invoice", instantInvoice)

	// Instant invoice should have the in advance fee
	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusActiveRealizationProcessing),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(6),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Status:   instantInvoice.Status,
					BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestInAdvanceInstantBillingOnSubscriptionCreationWithSubscriptionStartInFuture() {
	ctx := s.T().Context()
	futureStart := s.mustParseTime("2024-02-01T00:00:00Z")
	present := s.mustParseTime("2024-01-20T00:00:00Z")
	clock.FreezeTime(futureStart) // This will be the future

	// Given
	//  we have a subscription with a single phase with an in advance fee
	// When
	//  we start the subscription in the future
	// Then
	//  we'll have the lines on the gathering invoice
	//
	// Note that the UBP line is not synced because the subscription is not active yet

	s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
		Namespace: s.Namespace,
		Customer:  s.Customer.GetID(),
		Currency:  currencyx.Code(currency.USD),
		Amount:    alpacadecimal.NewFromInt(2),
		At:        present,
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
									Amount:      alpacadecimal.NewFromFloat(6),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(present.Add(time.Minute)) // This will be the present

	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
	s.assertCreditThenInvoiceBalances(startBalances)

	invoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{s.Namespace},
		Customers:  []string{s.Customer.ID},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
			billing.GatheringInvoiceExpandDeletedLines,
		},
	})
	s.NoError(err)
	s.Len(invoices.Items, 1)

	gatheringInvoice := invoices.Items[0]

	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	// Gathering invoice should have the UBP line
	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(6),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(futureStart)},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(futureStart),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestDiscountSynchronization() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}

	var gatheringInvoice *billing.GatheringInvoice
	var instantInvoice *billing.StandardInvoice
	var subsView subscription.SubscriptionView

	s.Run("given promotional credits and a fully discounted subscription", func() {
		// given:
		// - the customer has promotional credits available
		// - the plan uses credit-then-invoice settlement mode
		// - the in-advance flat fee is fully discounted
		// when:
		// - the subscription is created
		// then:
		// - the promotional credits are funded and no credit allocation has happened yet
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
			Namespace: s.Namespace,
			Customer:  s.Customer.GetID(),
			Currency:  currencyx.Code(currency.USD),
			Amount:    alpacadecimal.NewFromInt(2),
			At:        start,
		})
		s.assertCreditThenInvoiceBalances(startBalances)

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
										Amount:      alpacadecimal.NewFromFloat(6),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
									Discounts: productcatalog.Discounts{
										Percentage: &productcatalog.PercentageDiscount{
											Percentage: models.NewPercentage(100),
										},
									},
								},
								BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
		s.assertCreditThenInvoiceBalances(startBalances)

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(start.Add(time.Minute))
	})

	s.Run("when syncing at subscription start", func() {
		// given:
		// - the subscription exists at the start of its first billing period
		// when:
		// - subscription sync runs just after the start timestamp
		// then:
		// - the in-advance fee is instant-invoiced and future charge lines are gathered
		s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
		s.assertCreditThenInvoiceBalances(startBalances)

		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
			Expand:     billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 2)

		for _, invoice := range invoices.Items {
			if invoice.Type() == billing.InvoiceTypeGathering {
				invoiceAsGathering, err := invoice.AsGatheringInvoice()
				s.NoError(err)
				gatheringInvoice = &invoiceAsGathering
				continue
			}

			invoiceAsStandard, err := invoice.AsStandardInvoice()
			s.NoError(err)
			instantInvoice = &invoiceAsStandard
		}

		s.Require().NotNil(gatheringInvoice, "gathering invoice should be present")
		s.Require().NotNil(instantInvoice, "instant invoice should be present")

		s.DebugDumpInvoice("gathering invoice", *gatheringInvoice)
		s.DebugDumpInvoice("instant invoice", *instantInvoice)
		s.assertCreditThenInvoiceBalances(startBalances)
	})

	s.Run("then charges invoices and ledger balances reflect the full discount", func() {
		// given:
		// - subscription sync created the gathering and instant invoices
		// when:
		// - invoice lines and charges are inspected
		// then:
		// - the instant line has a full discount and no credit allocation
		// - promotional credits remain available on the ledger
		expectedCharges := []expectedCharge{
			// Gathering invoice should have the UBP line
			{
				Matcher: recurringLineMatcher{
					PhaseKey: "first-phase",
					ItemKey:  s.APIRequestsTotalFeature.Key,
				},
				Type:   chargesmeta.ChargeTypeUsageBased,
				Status: string(usagebased.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
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
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
			// Instant invoice should have the in advance fee
			{
				Matcher: recurringLineMatcher{
					PhaseKey: "first-phase",
					ItemKey:  "in-advance",
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusActiveRealizationProcessing),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				Realizations: []expectedChargeRealization{
					{
						Status:   instantInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
					},
				},
			},
		}
		s.assertCharges(ctx, subsView, expectedCharges)
		s.assertCreditThenInvoiceBalances(startBalances)

		// The advance fee should have 100% discount
		line := instantInvoice.Lines.OrEmpty()[0]
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(6), line.Totals.DiscountsTotal, "discount total")
		s.AssertDecimalEqual(alpacadecimal.Zero, line.Totals.Total, "total")
		s.AssertDecimalEqual(alpacadecimal.Zero, line.Totals.CreditsTotal, "credits total")
		s.assertCreditThenInvoiceBalances(startBalances)
	})
}

func (s *CreditThenInvoiceTestSuite) TestDiscountSynchronizationWithPartialDiscount() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	startBalances := expectedCreditThenInvoiceBalances{
		FBOAll:          2,
		FBOPromotional:  2,
		WashAll:         -2,
		WashPromotional: -2,
	}
	afterInstantInvoiceBalances := expectedCreditThenInvoiceBalances{
		FBOAll:             0,
		FBOPromotional:     0,
		AccruedAll:         2,
		AccruedPromotional: 2,
		WashAll:            -2,
		WashPromotional:    -2,
	}

	var gatheringInvoice *billing.GatheringInvoice
	var instantInvoice *billing.StandardInvoice
	var subsView subscription.SubscriptionView

	s.Run("given promotional credits and a partially discounted subscription", func() {
		// given:
		// - the customer has promotional credits available
		// - the plan uses credit-then-invoice settlement mode
		// - the in-advance flat fee has a 50% discount
		// when:
		// - the subscription is created
		// then:
		// - the promotional credits are funded and no credit allocation has happened yet
		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

		s.createPromotionalCreditFunding(ctx, createPromotionalCreditFundingInput{
			Namespace: s.Namespace,
			Customer:  s.Customer.GetID(),
			Currency:  currencyx.Code(currency.USD),
			Amount:    alpacadecimal.NewFromInt(2),
			At:        start,
		})
		s.assertCreditThenInvoiceBalances(startBalances)

		subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
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
										Amount:      alpacadecimal.NewFromFloat(6),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
									Discounts: productcatalog.Discounts{
										Percentage: &productcatalog.PercentageDiscount{
											Percentage: models.NewPercentage(50),
										},
									},
								},
								BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
		s.assertCreditThenInvoiceBalances(startBalances)

		// Simulate async subscription sync running shortly after subscription creation.
		clock.FreezeTime(start.Add(time.Minute))
	})

	s.Run("when syncing at subscription start", func() {
		// given:
		// - the subscription exists at the start of its first billing period
		// when:
		// - subscription sync runs just after the start timestamp
		// then:
		// - the payable part of the in-advance fee consumes promotional credits
		// - future charge lines are gathered
		s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
		s.assertCreditThenInvoiceBalances(afterInstantInvoiceBalances)

		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
			Expand:     billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 2)
		s.assertCreditThenInvoiceBalances(afterInstantInvoiceBalances)

		for _, invoice := range invoices.Items {
			if invoice.Type() == billing.InvoiceTypeGathering {
				invoiceAsGathering, err := invoice.AsGatheringInvoice()
				s.NoError(err)
				gatheringInvoice = &invoiceAsGathering
				continue
			}

			invoiceAsStandard, err := invoice.AsStandardInvoice()
			s.NoError(err)
			instantInvoice = &invoiceAsStandard
		}

		s.Require().NotNil(gatheringInvoice, "gathering invoice should be present")
		s.Require().NotNil(instantInvoice, "instant invoice should be present")

		s.DebugDumpInvoice("gathering invoice", *gatheringInvoice)
		s.DebugDumpInvoice("instant invoice", *instantInvoice)
	})

	s.Run("then charges invoices and ledger balances reflect the partial discount", func() {
		// given:
		// - subscription sync created the gathering and instant invoices
		// when:
		// - invoice lines, charges, and ledger balances are inspected
		// then:
		// - the instant line has a 50% discount and consumes available promotional credits
		// - the remaining invoice total is not covered by credits
		expectedCharges := []expectedCharge{
			// Gathering invoice should have the UBP line
			{
				Matcher: recurringLineMatcher{
					PhaseKey: "first-phase",
					ItemKey:  s.APIRequestsTotalFeature.Key,
				},
				Type:   chargesmeta.ChargeTypeUsageBased,
				Status: string(usagebased.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
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
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusCreated),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-02-01T00:00:00Z"),
						To:   s.mustParseTime("2024-03-01T00:00:00Z"),
					},
				},
				InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
				GatheringLines: []expectedChargeGatheringLine{
					{
						InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
					},
				},
			},
			// Instant invoice should have the in advance fee
			{
				Matcher: recurringLineMatcher{
					PhaseKey: "first-phase",
					ItemKey:  "in-advance",
				},
				Type:   chargesmeta.ChargeTypeFlatFee,
				Status: string(flatfee.StatusActiveRealizationProcessing),
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(6),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Periods: []timeutil.ClosedPeriod{
					{
						From: s.mustParseTime("2024-01-01T00:00:00Z"),
						To:   s.mustParseTime("2024-02-01T00:00:00Z"),
					},
				},
				Realizations: []expectedChargeRealization{
					{
						Status:   instantInvoice.Status,
						BookedAt: s.mustParseTime("2024-01-01T00:00:00Z"),
					},
				},
			},
		}
		s.assertCharges(ctx, subsView, expectedCharges)
		s.assertCreditThenInvoiceBalances(afterInstantInvoiceBalances)

		// The advance fee should have 50% discount and consume the available credits
		line := instantInvoice.Lines.OrEmpty()[0]
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(6), line.Totals.Amount, "amount")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(3), line.Totals.DiscountsTotal, "discount total")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(2), line.Totals.CreditsTotal, "credits total")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(1), line.Totals.Total, "total")
		s.assertCreditThenInvoiceBalances(afterInstantInvoiceBalances)
	})

	s.Run("then issued invoice books the fiat remainder", func() {
		// given:
		// - the instant invoice has consumed all available promotional credits
		// when:
		// - the invoice is approved and paid by the sandbox app
		// then:
		// - the remaining invoice amount is booked with invoice cost basis
		issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, instantInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, issuedInvoice.Status)
		s.DebugDumpInvoice("issued invoice", issuedInvoice)

		s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{
			FBOAll:             0,
			FBOPromotional:     0,
			AccruedAll:         3,
			AccruedPromotional: 2,
			AccruedInvoice:     1,
			WashAll:            -3,
			WashPromotional:    -2,
			WashInvoice:        -1,
		})
	})
}

func (s *CreditThenInvoiceTestSuite) TestAlignedSubscriptionProratingBehavior() {
	ctx := s.T().Context()
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
				BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
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
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

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
	s.NoError(s.Service.SyncByView(ctx, subView, s.mustParseTime("2024-03-01T00:00:00Z")))
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	expectedCharges := []expectedCharge{
		// January is 31 days, wechange phase after 2 weeks (14 days)
		// 5 * 14/31 = 2.258... which we round to 2.26
		// First phase lines
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(2.26),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(2.26),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "api-requests-total",
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price:  productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(10)}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-01T00:00:00Z"),
					To:   s.mustParseTime("2024-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
				},
			},
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
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-15T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(2.74),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-01-15T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-advance",
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-arrears",
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-15T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(2.74),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "in-arrears",
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   "api-requests-total",
				PeriodMin: 0,
				PeriodMax: 1,
			},
			// UBP does not need prorating on price due to period being shorter
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price:  productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(10.0)}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2024-01-15T00:00:00Z"),
					To:   s.mustParseTime("2024-02-01T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2024-02-01T00:00:00Z"),
					To:   s.mustParseTime("2024-03-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "second-phase",
						ItemKey:   "api-requests-total",
						PeriodMin: 0,
						PeriodMax: 0,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-02-01T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "second-phase",
						ItemKey:   "api-requests-total",
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2024-03-01T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subView, expectedCharges)
}

func (s *CreditThenInvoiceTestSuite) TestSynchronizeSubscriptionPeriodAlgorithmChange() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2025-01-31T00:00:00Z"))
	defer clock.UnFreeze()

	// Given
	//	a subscription started with a monthly in advance flat fee
	//  the first month is already synced
	// When we change the algorithm we use to calculate the period (emulated by an invoice change)
	// Then
	//  The next line will be automatically adjusted to start at the end of the previous period's end

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
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				},
			},
		},
	})

	// Simulate async subscription sync running shortly after subscription creation.
	clock.FreezeTime(clock.Now().Add(time.Minute))

	s.NoError(s.Service.SyncByView(ctx, subsView, s.mustParseTime("2025-01-31T00:00:00Z")))

	invoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", invoice)
	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(6),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-01-31T00:00:00Z"),
					To:   s.mustParseTime("2025-02-28T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-01-31T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-01-31T00:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, expectedCharges)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
		Invoice:      invoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceSystem,
		EditFn: func(invoice *billing.GatheringInvoice) error {
			line := invoice.Lines.OrEmpty()[0]
			// simulate some faulty behavior (the old algo would have set the end to 03-03, but this way we can test this with both the old and new alog)
			line.ServicePeriod.From = s.mustParseTime("2025-01-31T00:00:00Z")
			line.ServicePeriod.To = s.mustParseTime("2025-03-02T00:00:00Z")
			line.Annotations = models.Annotations{
				billing.AnnotationSubscriptionSyncIgnore:               true,
				billing.AnnotationSubscriptionSyncForceContinuousLines: true,
			}

			invoice.Lines = billing.NewGatheringInvoiceLines([]billing.GatheringLine{
				line,
			})
			return nil
		},
	})
	s.NoError(err)

	invoice, err = s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
			billing.GatheringInvoiceExpandDeletedLines,
		},
	})
	s.NoError(err)

	s.DebugDumpInvoice("gathering invoice - updated", invoice)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(timeutil.ClosedPeriod{
		From: s.mustParseTime("2025-01-31T00:00:00Z"),
		To:   s.mustParseTime("2025-03-02T00:00:00Z"),
	}, invoice.Lines.OrEmpty()[0].ServicePeriod)
	expectedCharges[0].GatheringLines[0].Period = timeutil.ClosedPeriod{
		From: s.mustParseTime("2025-01-31T00:00:00Z"),
		To:   s.mustParseTime("2025-03-02T00:00:00Z"),
	}
	s.assertCharges(ctx, subsView, expectedCharges)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Let's generate the next set of items
	clock.FreezeTime(s.mustParseTime("2025-02-28T00:00:00Z"))

	s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

	invoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - updated", invoice)

	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				PeriodMin: 0,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(6),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-01-31T00:00:00Z"),
					To:   s.mustParseTime("2025-02-28T00:00:00Z"),
				},
				{
					From: s.mustParseTime("2025-02-28T00:00:00Z"),
					To:   s.mustParseTime("2025-03-31T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{
				lo.ToPtr(s.mustParseTime("2025-01-31T00:00:00Z")),
				lo.ToPtr(s.mustParseTime("2025-02-28T00:00:00Z")),
			},
			GatheringLines: []expectedChargeGatheringLine{
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						PeriodMin: 0,
						PeriodMax: 0,
					},
					Period: timeutil.ClosedPeriod{
						From: s.mustParseTime("2025-01-31T00:00:00Z"),
						To:   s.mustParseTime("2025-03-02T00:00:00Z"),
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-01-31T00:00:00Z")),
				},
				{
					LineMatcher: recurringLineMatcher{
						PhaseKey:  "first-phase",
						ItemKey:   "in-advance",
						PeriodMin: 1,
						PeriodMax: 1,
					},
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-02-28T00:00:00Z")),
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
}

func (s *CreditThenInvoiceTestSuite) TestDeletedCustomerHandling() {
	// Given
	//  a customer with a subscription
	//  the subscription has UBP prices
	// When
	//  the subscription is canceled
	//  the customer is deleted
	// Then
	//  we can still sync the subscription
	//  and the deleted customer is billed for the outstanding amount

	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2025-01-01T00:00:00Z"))
	defer clock.UnFreeze()

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2025-01-01T00:30:00Z"))

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
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				},
			},
		},
	})

	// We advance the clock and cancel the subscription
	clock.FreezeTime(s.mustParseTime("2025-01-01T01:00:00Z"))
	subs, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	s.NoError(err)
	s.NotEmpty(subs)

	// We advance the clock and delete the customer
	clock.FreezeTime(s.mustParseTime("2025-01-01T02:00:00Z"))
	err = s.CustomerService.DeleteCustomer(ctx, s.Customer.GetID())
	s.NoError(err)

	// We advance the clock and simulate a late sync on the subscription
	clock.FreezeTime(s.mustParseTime("2025-01-01T03:00:00Z"))

	// Let's get the subscription
	subsView, err = s.SubscriptionService.GetView(ctx, subs.NamespacedID)
	s.NoError(err)

	s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))

	// Then the gathering invoice should be available
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	// 2025-01-01T00:00:00Z -> 2025-01-01T01:00:00Z
	expectedCharges := []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-01-01T00:00:00Z"),
					To:   s.mustParseTime("2025-01-01T01:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-01-01T01:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-01-01T01:00:00Z")),
				},
			},
		},
	}
	s.assertCharges(ctx, subsView, expectedCharges)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Then we can invoice the customer
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]

	s.DebugDumpInvoice("invoice", invoice)
	// We expect that the line is only covering the subscription's duration
	// 2025-01-01T00:00:00Z -> 2025-01-01T01:00:00Z
	// We expect that the invoice reaches a paid/non-error status
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Type:   chargesmeta.ChargeTypeUsageBased,
			Status: string(usagebased.StatusActiveRealizationProcessing),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-01-01T00:00:00Z"),
					To:   s.mustParseTime("2025-01-01T01:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-01-01T01:00:00Z"))},
			Realizations: []expectedChargeRealization{
				{
					Status:   invoice.Status,
					BookedAt: s.mustParseTime("2025-01-01T01:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(60),
						Total:  alpacadecimal.NewFromFloat(60),
					},
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	// Invoice expectations:
	s.Equal(billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	s.Equal(float64(5*12), invoice.Totals.Total.InexactFloat64())
}

func (s *CreditThenInvoiceTestSuite) TestFirstDayOfMonthBillingForSubPeriodLength() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2025-10-15T00:00:00Z"))
	defer clock.UnFreeze()

	// Given
	//	a monthly, first-day-of-month anchored subscription started mid-month
	// When
	// 	we end the subscription before the end of month
	// Then
	//  lines should only be billable with the start of the month

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
		Namespace:     s.Namespace,
		CustomerID:    s.Customer.ID,
		BillingAnchor: lo.ToPtr(s.mustParseTime("2025-10-01T00:00:00Z")),
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	clock.FreezeTime(s.mustParseTime("2024-01-20T00:00:00Z")) // This will be the present

	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-10-15T00:00:00Z"),
					To:   s.mustParseTime("2025-11-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-10-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-10-15T00:00:00Z")),
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	s.NoError(s.Service.SyncByView(ctx, subsView, clock.Now()))
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-10-15T00:00:00Z"),
					To:   s.mustParseTime("2025-11-01T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-10-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-10-15T00:00:00Z")),
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})
}

func (s *CreditThenInvoiceTestSuite) TestSyncStateUpdateNoBillables() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2025-10-15T00:00:00Z"))
	defer clock.UnFreeze()

	// Given
	//	a subscription with no billables
	// When
	// 	we synchronize the subscription
	// Then
	//  the sync state should be updated to reflect that the subscription has no billables

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
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
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
		Namespace:  s.Namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	clock.FreezeTime(s.mustParseTime("2024-01-20T00:00:00Z")) // This will be the present

	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
	s.assertCharges(ctx, subsView, nil)
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	syncStates, err := s.Adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{
		{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
	})

	require.NoError(s.T(), err)
	require.Len(s.T(), syncStates, 1)

	s.Equal(subscriptionsync.SyncState{
		SubscriptionID: models.NamespacedID{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
		HasBillables:  false,
		SyncedAt:      clock.Now().UTC(),
		NextSyncAfter: nil,
	}, syncStates[0])
}

func (s *CreditThenInvoiceTestSuite) TestSyncStateUpdateWithFreePhaseActiveInTheFuture() {
	ctx := s.T().Context()
	clock.FreezeTime(s.mustParseTime("2025-10-15T00:00:00Z"))
	defer clock.UnFreeze()

	// Given
	//	a subscription with a free phase, then a paid subscription
	//  the subscription only active from the future
	// When
	// 	we synchronize the subscription
	// Then
	//  the sync state should be updated to reflect that the subscription has billables and the next sync after is the future

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
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: false,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("free-phase", "P2M"), // two months
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-advance",
								Name: "in-advance",
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: s.phaseMeta("paid-phase", ""),
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

	clock.FreezeTime(s.mustParseTime("2024-01-20T00:00:00Z")) // This will be the present

	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "paid-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-12-15T00:00:00Z"),
					To:   s.mustParseTime("2026-01-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2025-12-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2025-12-15T00:00:00Z")),
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	syncStates, err := s.Adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{
		{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
	})

	require.NoError(s.T(), err)
	require.Len(s.T(), syncStates, 1)

	s.Equal(subscriptionsync.SyncState{
		SubscriptionID: models.NamespacedID{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
		HasBillables:  true,
		SyncedAt:      clock.Now().UTC(),
		NextSyncAfter: lo.ToPtr(s.mustParseTime("2025-12-15T00:00:00Z")),
	}, syncStates[0])

	// Let's advance the clock to simulate the next sync happening
	clock.FreezeTime(s.mustParseTime("2025-12-15T01:00:00Z"))
	s.NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, subsView, clock.Now()))
	s.assertCharges(ctx, subsView, []expectedCharge{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "paid-phase",
				ItemKey:  "in-advance",
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusActiveRealizationProcessing),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2025-12-15T00:00:00Z"),
					To:   s.mustParseTime("2026-01-15T00:00:00Z"),
				},
			},
			Realizations: []expectedChargeRealization{
				{
					Status:   billing.StandardInvoiceStatusDraftWaitingAutoApproval,
					BookedAt: s.mustParseTime("2025-12-15T00:00:00Z"),
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(5),
						Total:  alpacadecimal.NewFromFloat(5),
					},
				},
			},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "paid-phase",
				ItemKey:   "in-advance",
				PeriodMin: 1,
				PeriodMax: 1,
			},
			Type:   chargesmeta.ChargeTypeFlatFee,
			Status: string(flatfee.StatusCreated),
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(5),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			Periods: []timeutil.ClosedPeriod{
				{
					From: s.mustParseTime("2026-01-15T00:00:00Z"),
					To:   s.mustParseTime("2026-02-15T00:00:00Z"),
				},
			},
			InvoiceAt: []*time.Time{lo.ToPtr(s.mustParseTime("2026-01-15T00:00:00Z"))},
			GatheringLines: []expectedChargeGatheringLine{
				{
					InvoiceAt: lo.ToPtr(s.mustParseTime("2026-01-15T00:00:00Z")),
				},
			},
		},
	})
	s.assertCreditThenInvoiceBalances(expectedCreditThenInvoiceBalances{})

	syncStates, err = s.Adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{
		{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
	})

	require.NoError(s.T(), err)
	require.Len(s.T(), syncStates, 1)

	s.Equal(subscriptionsync.SyncState{
		SubscriptionID: models.NamespacedID{
			Namespace: subsView.Subscription.Namespace,
			ID:        subsView.Subscription.ID,
		},
		HasBillables:  true,
		SyncedAt:      clock.Now().UTC(),
		NextSyncAfter: lo.ToPtr(s.mustParseTime("2026-01-15T00:00:00Z")),
	}, syncStates[0])
}

func (s *CreditThenInvoiceTestSuite) expectValidationIssueForLine(line *billing.StandardLine, issue billing.ValidationIssue) {
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
	s.Equal(billing.ComponentName("charges.invoiceupdater"), issue.Component)
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
	}, ledger.BalanceQuery{})
	s.NoError(err)

	return balance
}

func (s *CreditThenInvoiceTestSuite) mustCustomerReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal], status ledger.TransactionAuthorizationStatus) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.ReceivableAccount, ledger.RouteFilter{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: lo.ToPtr(status),
	}, ledger.BalanceQuery{})
	s.NoError(err)

	return balance
}

func (s *CreditThenInvoiceTestSuite) mustCustomerAccruedBalance(customerID customer.CustomerID, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), customerAccounts.AccruedAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, ledger.BalanceQuery{})
	s.NoError(err)

	return balance
}

func (s *CreditThenInvoiceTestSuite) mustWashBalance(namespace string, code currencyx.Code, costBasis mo.Option[*alpacadecimal.Decimal]) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.WashAccount, ledger.RouteFilter{
		Currency:  code,
		CostBasis: costBasis,
	}, ledger.BalanceQuery{})
	s.NoError(err)

	return balance
}

func (s *CreditThenInvoiceTestSuite) mustEarningsBalance(namespace string, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), businessAccounts.EarningsAccount, ledger.RouteFilter{
		Currency: code,
	}, ledger.BalanceQuery{})
	s.NoError(err)

	return balance
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

func (s *CreditThenInvoiceTestSuite) mustGetUsageBasedChargeForInvoiceLine(ctx context.Context, line billing.GenericInvoiceLineReader) usagebased.Charge {
	s.T().Helper()

	s.Require().NotNil(line, "line")
	chargeID := line.GetChargeID()
	s.Require().NotNil(chargeID, "line charge id")

	return s.mustGetUsageBasedChargeByIDWithExpands(ctx, chargesmeta.ChargeID{
		Namespace: line.GetLineID().Namespace,
		ID:        *chargeID,
	}, nil)
}

type expectedFlatFeeIntent struct {
	ServicePeriod       timeutil.ClosedPeriod
	InvoiceAt           time.Time
	Amount              float64
	PaymentTerm         productcatalog.PaymentTermType
	PercentageDiscounts *billing.PercentageDiscount
	TaxConfig           productcatalog.TaxCodeConfig
}

func (s *CreditThenInvoiceTestSuite) mustGetFlatFeeChargeForInvoiceLine(ctx context.Context, line billing.GenericInvoiceLineReader) flatfee.Charge {
	return s.mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx, line, nil)
}

func (s *CreditThenInvoiceTestSuite) mustGetFlatFeeChargeForInvoiceLineWithExpands(ctx context.Context, line billing.GenericInvoiceLineReader, expands chargesmeta.Expands) flatfee.Charge {
	s.T().Helper()

	s.Require().NotNil(line, "line")
	chargeID := line.GetChargeID()
	s.Require().NotNil(chargeID, "line charge id")

	charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
		ChargeID: chargesmeta.ChargeID{
			Namespace: line.GetLineID().Namespace,
			ID:        *chargeID,
		},
		Expands: expands,
	})
	s.NoError(err)

	flatFeeCharge, err := charge.AsFlatFeeCharge()
	s.NoError(err)

	return flatFeeCharge
}

func (s *CreditThenInvoiceTestSuite) assertFlatFeeIntent(label string, actual flatfee.Intent, expected expectedFlatFeeIntent) {
	s.T().Helper()

	s.Equal(expected.ServicePeriod, actual.ServicePeriod, "%s: service period", label)
	s.Equal(expected.InvoiceAt, actual.InvoiceAt, "%s: invoice at", label)
	s.Equal(expected.PaymentTerm, actual.PaymentTerm, "%s: payment term", label)
	s.Equal(expected.Amount, actual.AmountBeforeProration.InexactFloat64(), "%s: amount before proration", label)
	if expected.PercentageDiscounts == nil {
		s.Nil(actual.PercentageDiscounts, "%s: percentage discounts", label)
	} else {
		s.Require().NotNil(actual.PercentageDiscounts, "%s: percentage discounts", label)
		s.Equal(expected.PercentageDiscounts.Percentage, actual.PercentageDiscounts.Percentage, "%s: percentage discount", label)
		s.Equal(expected.PercentageDiscounts.CorrelationID, actual.PercentageDiscounts.CorrelationID, "%s: percentage discount correlation id", label)
	}
	s.assertTaxCodeConfigEqual(expected.TaxConfig, actual.TaxConfig, label)
}

func (s *CreditThenInvoiceTestSuite) assertFlatFeeChargeIntentsForInvoiceLine(ctx context.Context, label string, line billing.GenericInvoiceLineReader, expectedBase, expectedOverride expectedFlatFeeIntent) {
	s.T().Helper()

	flatFeeCharge := s.mustGetFlatFeeChargeForInvoiceLine(ctx, line)
	s.True(flatFeeCharge.Intent.HasOverrideLayer(), "%s: override layer", label)

	baseIntent, err := flatFeeCharge.Intent.GetIntentForTarget(chargesmeta.ChangeTargetBase)
	s.NoError(err)
	overrideIntent, err := flatFeeCharge.Intent.GetIntentForTarget(chargesmeta.ChangeTargetOverride)
	s.NoError(err)

	s.assertFlatFeeIntent(label+": base intent", baseIntent, expectedBase)
	s.assertFlatFeeIntent(label+": override intent", overrideIntent, expectedOverride)
}

func (s *CreditThenInvoiceTestSuite) assertGatheringLineTaxConfigs(lines []billing.GatheringLine, expected *productcatalog.TaxConfig) {
	s.T().Helper()

	s.Require().NotEmpty(lines)
	for _, line := range lines {
		s.Require().NotNil(line.TaxConfig)
		s.True(expected.Equal(line.TaxConfig), "line %s tax config: expected %+v, got %+v", line.ID, expected, line.TaxConfig)
	}
}

func (s *CreditThenInvoiceTestSuite) assertStandardLineTaxConfigs(lines []*billing.StandardLine, expected *productcatalog.TaxConfig) {
	s.T().Helper()

	s.Require().NotEmpty(lines)
	for _, line := range lines {
		s.Require().NotNil(line.TaxConfig)
		s.True(expected.Equal(line.TaxConfig.ToProductCatalog()), "line %s tax config: expected %+v, got %+v", line.ID, expected, line.TaxConfig)
	}
}

func (s *CreditThenInvoiceTestSuite) assertCreditThenInvoiceChargeTaxConfigs(ctx context.Context, subscriptionID string, chargeType chargesmeta.ChargeType, expected *productcatalog.TaxConfig) {
	s.T().Helper()

	expectedChargeTaxConfig := productcatalog.TaxCodeConfigFrom(expected)
	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargeType},
	})
	s.NoError(err)
	s.Require().NotEmpty(res.Items)

	for _, charge := range res.Items {
		switch chargeType {
		case chargesmeta.ChargeTypeFlatFee:
			flatFeeCharge, err := charge.AsFlatFeeCharge()
			s.NoError(err)
			s.assertTaxCodeConfigEqual(expectedChargeTaxConfig, flatFeeCharge.Intent.GetTaxConfig(), flatFeeCharge.ID)
		case chargesmeta.ChargeTypeUsageBased:
			usageBasedCharge, err := charge.AsUsageBasedCharge()
			s.NoError(err)
			s.assertTaxCodeConfigEqual(expectedChargeTaxConfig, usageBasedCharge.Intent.GetTaxConfig(), usageBasedCharge.ID)
		default:
			s.Failf("unsupported charge type", "unsupported charge type %s", chargeType)
		}
	}
}

func (s *CreditThenInvoiceTestSuite) assertTaxCodeConfigEqual(expected, actual productcatalog.TaxCodeConfig, label string) {
	s.T().Helper()

	if lo.IsEmpty(expected) {
		s.True(lo.IsEmpty(actual), "%s: tax config", label)
		return
	}

	if expected.Behavior == nil {
		s.Nil(actual.Behavior, "%s: tax behavior", label)
	} else {
		s.Require().NotNil(actual.Behavior, "%s: tax behavior", label)
		s.Equal(*expected.Behavior, *actual.Behavior, "%s: tax behavior", label)
	}

	if expected.TaxCodeID == "" {
		s.Empty(actual.TaxCodeID, "%s: tax code id", label)
	} else {
		s.Require().NotEmpty(actual.TaxCodeID, "%s: tax code id", label)
		s.Equal(expected.TaxCodeID, actual.TaxCodeID, "%s: tax code id", label)
	}
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
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, charge.Intent.GetSettlementMode())
	s.Equal(input.ServicePeriod, charge.Intent.GetBaseIntent().ServicePeriod)
	s.Equal(input.ServicePeriod, charge.Intent.GetBaseIntent().FullServicePeriod)
	s.Equal(input.ServicePeriod, charge.Intent.GetBaseIntent().BillingPeriod)
	s.Equal(input.InvoiceAt, charge.Intent.GetBaseIntent().InvoiceAt)
	s.Equal(input.CustomerID, charge.Intent.GetCustomerID())
	s.Equal(input.FeatureKey, charge.Intent.GetBaseIntent().FeatureKey)
	price := charge.Intent.GetBaseIntent().Price
	s.Truef(input.Price.Equal(&price), "price expected %v, got %v", input.Price, price)
	s.Require().NotNil(charge.Intent.GetSubscription())
	s.Equal(input.SubscriptionID, charge.Intent.GetSubscription().SubscriptionID)
	s.Equal(input.PhaseID, charge.Intent.GetSubscription().PhaseID)
	s.Equal(input.ItemID, charge.Intent.GetSubscription().ItemID)
}

type expectedTotalsInput struct {
	Amount         float64
	DiscountsTotal float64
	CreditsTotal   float64
	Total          float64
}

func (s *CreditThenInvoiceTestSuite) assertTotals(actual totals.Totals, input expectedTotalsInput) {
	s.T().Helper()

	require.Equal(s.T(), input.Amount, actual.Amount.InexactFloat64(), "amount")
	require.Equal(s.T(), input.DiscountsTotal, actual.DiscountsTotal.InexactFloat64(), "discounts total")
	require.Equal(s.T(), input.CreditsTotal, actual.CreditsTotal.InexactFloat64(), "credits total")
	require.Equal(s.T(), input.Total, actual.Total.InexactFloat64(), "total")
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
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: input.Customer.ID,
					Currency:   currenciestestutils.NewFiatCurrency(s.T(), input.Currency),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: chargesmeta.IntentMutableFields{
						Name:              "Promotional Credit Purchase",
						ServicePeriod:     timeutil.ClosedPeriod{From: input.At, To: input.At},
						FullServicePeriod: timeutil.ClosedPeriod{From: input.At, To: input.At},
						BillingPeriod:     timeutil.ClosedPeriod{From: input.At, To: input.At},
					},
					CreditAmount: input.Amount,
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
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
