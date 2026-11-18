package subscriptionsync_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	synchooks "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/hooks"
	periodicreconciler "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
	syncservice "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionsyncstatev2"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	advancetestutils "github.com/openmeterio/openmeter/openmeter/ledger/advance/testutils"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestReconciliation(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		suite.Run(t, &ReconciliationTestSuite{})
	})
	t.Run("charges", func(t *testing.T) {
		suite.Run(t, &ReconciliationTestSuite{UseCharges: true})
	})
}

type ReconciliationTestSuite struct {
	billingtest.BaseSuite
	billingtest.SubscriptionMixin

	UseCharges              bool
	Service                 subscriptionsync.Service
	Charges                 charges.Service
	LedgerDeps              ledgertestutils.Deps
	Namespace               string
	Customer                *customer.Customer
	APIRequestsTotalFeature feature.Feature
}

func (s *ReconciliationTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.SubscriptionMixin.SetupSuite(s.T(), s.GetSubscriptionMixInDependencies())
	logger := testutils.NewDiscardLogger(s.T())

	ledgerDeps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.Require().NoError(err)
	s.LedgerDeps = ledgerDeps
	transactionManager := enttx.NewCreator(s.DBClient)
	breakageService := ledgerbreakage.NewNoopService()
	advanceService := advancetestutils.NewService(s.T(), ledgerDeps, breakageService)
	collectorService, err := ledgercollector.NewService(ledgercollector.Config{
		Meter:   metricnoop.NewMeterProvider().Meter("test"),
		Logger:  logger,
		Advance: advanceService,
		Ledger:  ledgerDeps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService: ledgerDeps.ResolversService,
			AccountCatalog: ledgerDeps.AccountService,
			BalanceQuerier: ledgerDeps.HistoricalLedger,
		},
		Breakage:           breakageService,
		AccountLocker:      ledgerDeps.AccountService,
		TransactionManager: transactionManager,
	})
	s.Require().NoError(err)
	creditPurchaseHandler, err := ledgerchargeadapter.NewCreditPurchaseHandler(ledgerchargeadapter.CreditPurchaseHandlerConfig{
		Ledger:             ledgerDeps.HistoricalLedger,
		BalanceQuerier:     ledgerDeps.HistoricalLedger,
		AccountResolver:    ledgerDeps.ResolversService,
		AccountCatalog:     ledgerDeps.AccountService,
		AdvanceService:     advanceService,
		BreakageService:    breakageService,
		TransactionManager: transactionManager,
	})
	s.Require().NoError(err)
	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:               s.DBClient,
		Logger:               logger,
		BillingService:       s.BillingService,
		FeatureMeterResolver: s.FeatureMeterResolver,
		StreamingConnector:   s.MockStreamingConnector,
		TaxCodeService:       s.TaxCodeEnv.Service,
		CustomerService:      s.CustomerService,
		SubscriptionService:  s.SubscriptionService,
		FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			}, collectorService),
		CreditPurchaseHandler: creditPurchaseHandler,
		UsageBasedHandler: ledgerchargeadapter.NewUsageBasedHandler(ledgerDeps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService: ledgerDeps.ResolversService,
				AccountCatalog: ledgerDeps.AccountService,
				BalanceQuerier: ledgerDeps.HistoricalLedger,
			}, collectorService),
	})
	s.Require().NoError(err)
	s.Charges = stack.ChargesService
	adapter, err := syncadapter.New(syncadapter.Config{Client: s.DBClient})
	s.Require().NoError(err)
	syncHook, err := synchooks.New(synchooks.Config{Adapter: adapter, Logger: logger})
	s.Require().NoError(err)
	s.Require().NoError(s.SubscriptionService.RegisterHook(syncHook))
	s.Service, err = syncservice.New(syncservice.Config{
		BillingService:          s.BillingService,
		LegacyBillingLineEngine: s.LegacyBillingLineEngine,
		ChargesService:          s.Charges,
		SubscriptionService:     s.SubscriptionService,
		SubscriptionSyncAdapter: adapter,
		FeatureFlags:            syncservice.FeatureFlags{EnableCreditThenInvoice: s.UseCharges},
		Logger:                  logger,
		Tracer:                  noop.NewTracerProvider().Tracer("test"),
		FeatureGate: featuregate.NewFeatureGateChecker(featuregate.NewNoop(), featuregate.Flags{
			featuregate.CtxKeyCredits: string(featuregate.CtxKeyCredits),
		}, map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true}),
	})
	s.Require().NoError(err)
}

func (s *ReconciliationTestSuite) BeforeTest(_, _ string) {
	ctx := s.T().Context()
	s.Namespace = s.GetUniqueNamespace("subscription-reconciliation")
	sandboxApp := s.InstallSandboxApp(s.T(), s.Namespace)
	s.ProvisionBillingProfile(ctx, s.Namespace, sandboxApp.GetID())
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), s.Namespace)
	testFeature := s.SetupApiRequestsTotalFeature(ctx, s.Namespace)
	s.APIRequestsTotalFeature = testFeature.Feature
	s.T().Cleanup(testFeature.Cleanup)
	_, err := s.LedgerDeps.ResolversService.EnsureBusinessAccounts(ctx, s.Namespace)
	s.Require().NoError(err)
}

func (s *ReconciliationTestSuite) gatheringInvoice() billing.GatheringInvoice {
	s.T().Helper()
	invoices, err := s.BillingService.ListGatheringInvoices(s.T().Context(), billing.ListGatheringInvoicesInput{
		Namespace: s.Namespace,
		Customers: []string{s.Customer.ID},
		Expand:    billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines},
	})
	s.Require().NoError(err)
	s.Require().Len(invoices.Items, 1)

	return invoices.Items[0]
}

func (s *ReconciliationTestSuite) expectNoGatheringInvoice() {
	s.T().Helper()
	invoices, err := s.BillingService.ListGatheringInvoices(s.T().Context(), billing.ListGatheringInvoicesInput{
		Namespace: s.Namespace,
		Customers: []string{s.Customer.ID},
	})
	s.Require().NoError(err)
	s.Empty(invoices.Items)
}

type reconciliationSubscriptionInput struct {
	NoBillables     bool
	SkipInitialSync bool
	ActiveFrom      time.Time
}

func (s *ReconciliationTestSuite) createSubscriptionForReconciliation(in reconciliationSubscriptionInput) subscription.SubscriptionView {
	s.T().Helper()
	// Each boundary case owns its customer and plan while reusing the suite's billing stack.
	ctx := s.T().Context()
	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.Namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:            lo.ToPtr(ulid.Make().String()),
			Name:           "Subscription reconciliation",
			PrimaryEmail:   lo.ToPtr("test@example.com"),
			BillingAddress: &models.Address{Country: lo.ToPtr(models.CountryCode("US"))},
		},
	})
	s.Require().NoError(err)
	s.Customer = customerEntity
	_, err = s.LedgerDeps.ResolversService.CreateCustomerAccounts(ctx, s.Customer.GetID())
	s.Require().NoError(err)
	rateCards := productcatalog.RateCards{
		&productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  "non-billable",
				Name: "Non-billable item",
			},
			BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
		},
	}
	if !in.NoBillables {
		rateCards = productcatalog.RateCards{
			&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "flat-fee",
					Name: "Flat fee",
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(1),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
				},
				BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
			},
			&productcatalog.UsageBasedRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:     s.APIRequestsTotalFeature.Key,
					Name:    "Usage",
					Feature: productcatalog.NewFeatureReference(nil, lo.ToPtr(s.APIRequestsTotalFeature.Key)),
					Price:   productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				},
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
			},
		}
	}

	planInput := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Subscription reconciliation",
				Key:            ulid.Make().String(),
				Version:        1,
				Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default phase"},
				RateCards: rateCards,
			}},
		},
	}
	createdPlan, err := s.PlanService.CreatePlan(ctx, planInput)
	s.Require().NoError(err)
	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     createdPlan.Key,
		Version: lo.ToPtr(1),
	})
	s.Require().NoError(err)
	activeFrom := in.ActiveFrom
	if activeFrom.IsZero() {
		activeFrom = clock.Now().Add(24 * time.Hour)
	}

	view, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: lo.ToPtr(activeFrom)},
			Name:   "Subscription for reconciliation",
		},
		Namespace:  s.Namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)
	s.Require().NoError(err)

	if !in.SkipInitialSync {
		s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, view.Subscription.ActiveFrom.Add(time.Minute)))
		if !in.NoBillables {
			invoice := s.gatheringInvoice()
			s.Require().GreaterOrEqual(len(invoice.Lines.OrEmpty()), 2)
			for _, line := range invoice.Lines.OrEmpty() {
				if !s.UseCharges {
					s.Equal(billing.LineEngineTypeInvoice, line.Engine)
				} else {
					s.NotEqual(billing.LineEngineTypeInvoice, line.Engine)
					s.NotNil(line.ChargeID)
				}
			}
		}
	}

	return view
}

func (s *ReconciliationTestSuite) TestCustomerDeletedBeforeCleanup() {
	cases := []struct {
		name            string
		deletedAtOffset time.Duration
		noBillables     bool
		skipInitialSync bool
		wantCleanup     bool
	}{
		{
			name:            "customer deleted in the past before subscription start",
			deletedAtOffset: -time.Hour,
			wantCleanup:     true,
		},
		{
			name:            "customer deletion ahead of worker clock before subscription start",
			deletedAtOffset: time.Hour,
			wantCleanup:     true,
		},
		{
			name:            "customer deleted exactly at subscription start",
			deletedAtOffset: 24 * time.Hour,
			wantCleanup:     true,
		},
		{
			name:            "customer deleted after subscription start retains billable service",
			deletedAtOffset: 25 * time.Hour,
		},
		{
			name:        "no billables still records successful cleanup",
			noBillables: true,
			wantCleanup: true,
		},
		{
			name:            "no existing artifacts still records successful cleanup",
			skipInitialSync: true,
			wantCleanup:     true,
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			ctx := s.T().Context()
			now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(now)
			defer clock.UnFreeze()

			// Given an existing non-deleted subscription, including inconsistent historical customer lifecycles.
			view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{
				NoBillables:     tc.noBillables,
				SkipInitialSync: tc.skipInitialSync,
			})
			subscriptionID := view.Subscription.NamespacedID
			initialLineCount := 0
			if !tc.noBillables && !tc.skipInitialSync {
				initialLineCount = len(s.gatheringInvoice().Lines.OrEmpty())
			}

			deletedAt := now.Add(tc.deletedAtOffset)
			// The customer API rejects deletion with a scheduled subscription; seed the existing database inconsistency.
			s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(deletedAt).Exec(ctx))

			// When sync observes customer deletion, even with a historical billing horizon.
			s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now.Add(-time.Hour)))

			// Then only subscriptions starting at or after deletion have an empty billing target.
			states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
			s.Require().NoError(err)
			if !tc.wantCleanup {
				s.Require().Len(states, 1)
				s.Equal(subscriptionsync.SyncStateV2StateActive, states[0].LastSuccessState)
				s.Equal(lo.ToPtr(now), states[0].LastSuccessAt)
				s.NotNil(states[0].NextReconciliationAt)
				invoice := s.gatheringInvoice()
				s.NotEmpty(invoice.Lines.OrEmpty())

				return
			}

			s.expectNoGatheringInvoice()
			s.Require().Len(states, 1)
			s.Equal(subscriptionID, states[0].SubscriptionID)
			s.Equal(subscriptionsync.SyncStateV2StateCustomerDeletedBefore, states[0].LastSuccessState)
			s.Equal(lo.ToPtr(now), states[0].LastSuccessAt)
			s.NotEmpty(states[0].CurrentSubsRevision)
			s.Equal(states[0].CurrentSubsRevision, states[0].LastSyncedSubsRevision)
			s.NoError(states[0].Validate())
			root, err := s.DBClient.Subscription.Get(ctx, subscriptionID.ID)
			s.Require().NoError(err)
			s.Nil(root.DeletedAt)
			legacy, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
			s.Require().NoError(err)
			s.Require().Len(legacy, 1)
			s.False(legacy[0].HasBillables)
			s.Nil(legacy[0].NextSyncAfter)

			if s.UseCharges && !tc.noBillables && !tc.skipInitialSync {
				chargeList, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
					Namespace:      s.Namespace,
					SubscriptionID: filter.NewULIDIn(subscriptionID.ID),
					IncludeDeleted: true,
					Status:         &filter.FilterString{Eq: lo.ToPtr(string(chargesmeta.ChargeStatusDeleted))},
				})
				s.Require().NoError(err)
				s.Require().Len(chargeList.Items, initialLineCount)
			}
		})
	}
}

func (s *ReconciliationTestSuite) TestOrdinarySyncAcknowledgesCurrentRevision() {
	ctx := s.T().Context()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a live subscription and v2 state whose current revision has advanced.
	view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{})
	subscriptionID := view.Subscription.NamespacedID
	s.Require().NoError(s.DBClient.SubscriptionSyncStateV2.Update().
		Where(subscriptionsyncstatev2.SubscriptionID(subscriptionID.ID)).
		SetLastSuccessState(subscriptionsync.SyncStateV2StateCustomerDeletedBefore).
		ClearNextReconciliationAt().
		SetLastSuccessAt(now.Add(-time.Hour)).
		SetLastSyncedSubsRevision(ulid.Make().String()).
		SetCurrentSubsRevision(ulid.Make().String()).Exec(ctx))
	before, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(before, 1)
	s.Require().NoError(before[0].Validate())

	// When ordinary reconciliation succeeds, it acknowledges the observed revision.
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, view.Subscription.ActiveFrom.Add(time.Minute)))
	after, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(after, 1)
	s.Equal(before[0].ID, after[0].ID)
	s.Equal(before[0].CurrentSubsRevision, after[0].CurrentSubsRevision)
	s.Equal(after[0].CurrentSubsRevision, after[0].LastSyncedSubsRevision)
	s.Equal(subscriptionsync.SyncStateV2StateActive, after[0].LastSuccessState)
	s.Equal(lo.ToPtr(now), after[0].LastSuccessAt)
	s.NotNil(after[0].NextReconciliationAt)
}

func (s *ReconciliationTestSuite) TestCleanupAcknowledgesCurrentRevision() {
	ctx := s.T().Context()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a customer deletion whose current revision differs from the last acknowledged revision.
	view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{})
	subscriptionID := view.Subscription.NamespacedID
	currentRevision := ulid.Make().String()
	s.Require().NoError(s.DBClient.SubscriptionSyncStateV2.Update().
		Where(subscriptionsyncstatev2.SubscriptionID(subscriptionID.ID)).
		SetLastSuccessState(subscriptionsync.SyncStateV2StateCustomerDeletedBefore).
		ClearNextReconciliationAt().
		SetLastSuccessAt(now.Add(-time.Hour)).
		SetLastSyncedSubsRevision(ulid.Make().String()).
		SetCurrentSubsRevision(currentRevision).Exec(ctx))
	seededStates, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(seededStates, 1)
	s.Require().NoError(seededStates[0].Validate())
	s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(now).Exec(ctx))

	// When cleanup succeeds, the observed current revision becomes acknowledged.
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now))
	states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(states, 1)
	s.Equal(currentRevision, states[0].CurrentSubsRevision)
	s.Equal(states[0].CurrentSubsRevision, states[0].LastSyncedSubsRevision)
	s.Equal(lo.ToPtr(now), states[0].LastSuccessAt)
	s.expectNoGatheringInvoice()
}

func (s *ReconciliationTestSuite) TestCustomerDeletedBeforeCleanupDryRun() {
	ctx := s.T().Context()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given future billing artifacts whose customer was deleted before subscription start.
	view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{})
	subscriptionID := view.Subscription.NamespacedID
	invoice := s.gatheringInvoice()
	legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
	s.Require().NoError(err)
	s.Require().NoError(s.DBClient.SubscriptionSyncStateV2.Update().
		Where(subscriptionsyncstatev2.SubscriptionID(subscriptionID.ID)).
		SetLastSuccessState(subscriptionsync.SyncStateV2StateCustomerDeletedBefore).
		ClearNextReconciliationAt().
		SetLastSuccessAt(now.Add(-time.Hour)).
		SetLastSyncedSubsRevision(ulid.Make().String()).
		SetCurrentSubsRevision(ulid.Make().String()).Exec(ctx))
	v2Before, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(v2Before, 1)
	s.Require().NoError(v2Before[0].Validate())
	s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(now).Exec(ctx))

	// When cleanup is requested as a dry run, with invoicing also requested.
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now, subscriptionsync.EnableDryRun(), subscriptionsync.WithInvoicePendingLines()))

	// Then artifacts, scheduling, and the previous v2 revisions remain unchanged.
	s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
	legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
	s.Require().NoError(err)
	s.Equal(legacyBefore, legacyAfter)
	states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Equal(v2Before, states)
}

func (s *ReconciliationTestSuite) TestCustomerDeletedBeforeCleanupRollback() {
	cleanupCtx := s.T().Context()
	cases := []struct {
		name     string
		table    string
		checkSQL func(models.NamespacedID, billing.GatheringInvoice) string
	}{
		{
			name:  "billing artifact write fails",
			table: "billing_invoice_lines",
			checkSQL: func(_ models.NamespacedID, invoice billing.GatheringInvoice) string {
				return fmt.Sprintf("invoice_id <> %s OR deleted_at IS NULL", pq.QuoteLiteral(invoice.ID))
			},
		},
		{
			name:  "v2 completion write fails",
			table: "subscription_sync_states_v2",
			checkSQL: func(subscriptionID models.NamespacedID, _ billing.GatheringInvoice) string {
				return fmt.Sprintf("subscription_id <> %s OR last_synced_subs_revision IS DISTINCT FROM current_subs_revision", pq.QuoteLiteral(subscriptionID.ID))
			},
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			ctx := s.T().Context()
			now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(now)
			defer clock.UnFreeze()

			// Given existing artifacts and a cleanup attempt that will fail inside the billing transaction.
			view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{})
			subscriptionID := view.Subscription.NamespacedID
			invoice := s.gatheringInvoice()
			legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
			s.Require().NoError(err)
			s.Require().NoError(s.DBClient.SubscriptionSyncStateV2.Update().
				Where(subscriptionsyncstatev2.SubscriptionID(subscriptionID.ID)).
				SetLastSuccessState(subscriptionsync.SyncStateV2StateCustomerDeletedBefore).
				ClearNextReconciliationAt().
				SetLastSuccessAt(now.Add(-time.Hour)).
				SetLastSyncedSubsRevision(ulid.Make().String()).
				SetCurrentSubsRevision(ulid.Make().String()).Exec(ctx))
			v2Before, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
			s.Require().NoError(err)
			s.Require().Len(v2Before, 1)
			s.Require().NoError(v2Before[0].Validate())
			s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(now).Exec(ctx))
			// Reject the real cleanup write in PostgreSQL without replacing any service or adapter.
			// Scope the constraint to this fixture so the suite's other records remain valid.
			const constraintName = "test_customer_deletion_cleanup_failure"
			addConstraint := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
				pq.QuoteIdentifier(tc.table), pq.QuoteIdentifier(constraintName), tc.checkSQL(subscriptionID, invoice))
			dropConstraint := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s",
				pq.QuoteIdentifier(tc.table), pq.QuoteIdentifier(constraintName))
			_, err = s.TestDB.PGDriver.DB().ExecContext(ctx, addConstraint)
			s.Require().NoError(err)
			s.T().Cleanup(func() {
				_, cleanupErr := s.TestDB.PGDriver.DB().ExecContext(cleanupCtx, dropConstraint)
				s.NoError(cleanupErr)
			})

			// When artifact cleanup or recording completion fails.
			s.Require().ErrorContains(s.Service.SyncByID(ctx, subscriptionID, now), constraintName)

			// Then artifact changes and revision acknowledgments roll back with the billing transaction.
			s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
			legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
			s.Require().NoError(err)
			s.Equal(legacyBefore, legacyAfter)
			states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
			s.Require().NoError(err)
			s.Equal(v2Before, states)
			_, err = s.TestDB.PGDriver.DB().ExecContext(ctx, dropConstraint)
			s.Require().NoError(err)
			s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now))
			s.expectNoGatheringInvoice()
		})
	}
}

func (s *ReconciliationTestSuite) TestCustomerDeletedBeforeCleanupRepeated() {
	ctx := s.T().Context()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given subscription artifacts and an unrelated line that is already eligible for collection.
	view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{})
	subscriptionID := view.Subscription.NamespacedID
	pending, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: s.Customer.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines(billing.GatheringLines{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Period:        timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now.Add(-time.Hour)},
				InvoiceAt:     now.Add(-time.Hour),
				Name:          "Unrelated historical service",
				ManagedBy:     billing.ManuallyManagedLine,
				PerUnitAmount: alpacadecimal.NewFromInt(1),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	s.Require().NoError(err)
	s.Require().Len(pending.Lines, 1)
	unrelatedLineID := pending.Lines[0].ID
	s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(now).Exec(ctx))

	// When cleanup removes subscription artifacts, including on a repeated request with nothing left to delete.
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now, subscriptionsync.WithInvoicePendingLines()))
	later := now.Add(time.Minute)
	clock.FreezeTime(later)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, later, subscriptionsync.WithInvoicePendingLines()))

	// Then the unrelated line stays pending: cleanup must not collect the customer's other billing work.
	invoice := s.gatheringInvoice()
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(unrelatedLineID, invoice.Lines.OrEmpty()[0].ID)
	states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(states, 1)
	s.Equal(subscriptionsync.SyncStateV2StateCustomerDeletedBefore, states[0].LastSuccessState)
	s.Equal(lo.ToPtr(later), states[0].LastSuccessAt)

	// When the subscription itself is subsequently deleted, that result takes precedence.
	s.Require().NoError(s.SubscriptionService.Delete(ctx, subscriptionID))
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, later))
	states, err = s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(states, 1)
	s.Equal(subscriptionsync.SyncStateV2StateSubscriptionDeleted, states[0].LastSuccessState)

	// Explicit collection can invoice that line, proving cleanup left eligible work pending.
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	}, billing.WithPartialInvoiceLinesDisabled())
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
	s.Equal(unrelatedLineID, invoices[0].Lines.OrEmpty()[0].ID)
}

// Generate a second billing period before scheduling cancellation at the first
// period's end, so the next sync must remove lines extending past ActiveTo.
func (s *ReconciliationTestSuite) createCanceledSubscriptionForReconciliation() subscription.Subscription {
	s.T().Helper()
	view := s.createSubscriptionForReconciliation(reconciliationSubscriptionInput{ActiveFrom: clock.Now()})
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, view.Subscription.ActiveFrom.AddDate(0, 1, 0).Add(time.Minute)))
	subs, err := s.SubscriptionService.Cancel(s.T().Context(), view.Subscription.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
	})
	s.Require().NoError(err)
	s.Require().NotNil(subs.ActiveTo)

	return subs
}

func (s *ReconciliationTestSuite) getSyncStateV2(subscriptionID models.NamespacedID) subscriptionsync.SyncStateV2Managed {
	s.T().Helper()
	states, err := s.Service.GetSyncStatesV2(s.T().Context(), subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Require().Len(states, 1)
	s.Equal(subscriptionID, states[0].SubscriptionID)
	s.Require().NoError(states[0].Validate())

	return states[0]
}

func (s *ReconciliationTestSuite) requireReconciledSubscriptionEnd(subs subscription.Subscription) {
	s.T().Helper()
	invoice := s.gatheringInvoice()
	s.Require().NotEmpty(invoice.Lines.OrEmpty())
	for _, line := range invoice.Lines.OrEmpty() {
		s.False(line.ServicePeriod.To.After(*subs.ActiveTo), "billing lines must not extend past the subscription end")
	}
}

func (s *ReconciliationTestSuite) TestEndedSubscriptionSyncResult() {
	cases := []struct {
		name          string
		syncOffset    time.Duration
		horizonOffset time.Duration
		wantState     subscriptionsync.SyncStateV2State
	}{
		{
			name:          "recently ended subscription",
			syncOffset:    24 * time.Hour,
			horizonOffset: 24 * time.Hour,
			wantState:     subscriptionsync.SyncStateV2StateEnded,
		},
		{
			name:      "worker clock and horizon equal subscription end",
			wantState: subscriptionsync.SyncStateV2StateEnded,
		},
		{
			name:          "exactly thirty days old still reconciles",
			syncOffset:    30 * 24 * time.Hour,
			horizonOffset: 30 * 24 * time.Hour,
			wantState:     subscriptionsync.SyncStateV2StateEnded,
		},
		{
			name:          "future horizon does not mark future cancellation ended",
			syncOffset:    -time.Hour,
			horizonOffset: time.Hour,
			wantState:     subscriptionsync.SyncStateV2StateActive,
		},
		{
			// No billing period exists this far past the end; the final sync still waits for ActiveTo.
			name:          "horizon several periods past a future cancellation waits for the end",
			syncOffset:    -time.Hour,
			horizonOffset: 90 * 24 * time.Hour,
			wantState:     subscriptionsync.SyncStateV2StateActive,
		},
		{
			name:          "historical horizon does not record final reconciliation",
			syncOffset:    24 * time.Hour,
			horizonOffset: -time.Hour,
			wantState:     subscriptionsync.SyncStateV2StateActive,
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Given billing lines generated before a future cancellation was scheduled.
			clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			subs := s.createCanceledSubscriptionForReconciliation()
			syncAt := subs.ActiveTo.Add(tc.syncOffset)
			clock.FreezeTime(syncAt)
			defer clock.UnFreeze()

			// When sync runs with the worker clock and requested horizon around ActiveTo.
			s.Require().NoError(s.Service.SyncByID(s.T().Context(), subs.NamespacedID, subs.ActiveTo.Add(tc.horizonOffset)))

			// Then only reconciliation covering an already-ended subscription records completion.

			state := s.getSyncStateV2(subs.NamespacedID)
			s.Equal(tc.wantState, state.LastSuccessState)
			s.Equal(lo.ToPtr(syncAt), state.LastSuccessAt)
			s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
			if tc.wantState == subscriptionsync.SyncStateV2StateActive {
				s.Equal(subs.ActiveTo, state.NextReconciliationAt)
			} else {
				s.Nil(state.NextReconciliationAt)
				s.requireReconciledSubscriptionEnd(subs)
			}
		})
	}
}

func (s *ReconciliationTestSuite) TestOldSubscriptionSyncSkipped() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// Given billing lines that still need shortening, just beyond the thirty-day cutoff.
	subs := s.createCanceledSubscriptionForReconciliation()
	invoice := s.gatheringInvoice()
	legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
	s.Require().NoError(err)
	now := subs.ActiveTo.Add(30*24*time.Hour + time.Microsecond)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// When ordinary sync is requested, even with pending-line invoicing enabled.
	s.Require().NoError(s.Service.SyncByID(ctx, subs.NamespacedID, now, subscriptionsync.WithInvoicePendingLines()))

	// Then only the intentional skip is recorded; artifacts and legacy scheduling remain unchanged.
	state := s.getSyncStateV2(subs.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateTooOld, state.LastSuccessState)
	s.Equal(lo.ToPtr(now), state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
	legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
	s.Require().NoError(err)
	s.Equal(legacyBefore, legacyAfter)
}

func (s *ReconciliationTestSuite) TestForceResyncOldSubscription() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// Given an old subscription whose default sync has recorded a skip.
	subs := s.createCanceledSubscriptionForReconciliation()
	now := subs.ActiveTo.Add(40 * 24 * time.Hour)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(ctx, subs.NamespacedID, now))
	before := s.getSyncStateV2(subs.NamespacedID)
	s.Require().Equal(subscriptionsync.SyncStateV2StateTooOld, before.LastSuccessState)

	// When the caller explicitly overrides the thirty-day cutoff.
	s.Require().NoError(s.Service.SyncByID(ctx, subs.NamespacedID, now, subscriptionsync.WithForceResyncOldSubscriptions()))

	// Then normal billing reconciliation replaces the skip with an ended result.
	after := s.getSyncStateV2(subs.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateEnded, after.LastSuccessState)
	s.Equal(before.CurrentSubsRevision, after.CurrentSubsRevision)
	s.Equal(after.CurrentSubsRevision, after.LastSyncedSubsRevision)
	s.requireReconciledSubscriptionEnd(subs)
}

func (s *ReconciliationTestSuite) TestForcedPeriodicResyncOldSubscription() {
	ctx := s.T().Context()
	clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// Given a skipped subscription outside the CLI's default lookback.
	subs := s.createCanceledSubscriptionForReconciliation()
	now := subs.ActiveTo.Add(40 * 24 * time.Hour)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(ctx, subs.NamespacedID, now))
	before := s.getSyncStateV2(subs.NamespacedID)
	periodic, err := periodicreconciler.NewReconciler(periodicreconciler.ReconcilerConfig{
		SubscriptionSync:    s.Service,
		SubscriptionService: s.SubscriptionService,
		CustomerService:     s.CustomerService,
		Logger:              testutils.NewDiscardLogger(s.T()),
	})
	s.Require().NoError(err)
	input := periodicreconciler.ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: periodicreconciler.ReconcilerListSubscriptionsInput{
			Namespaces: []string{s.Namespace},
			Customers:  []string{s.Customer.ID},
			Lookback:   24 * time.Hour,
		},
		Force: true,
	}

	// When force is used without widening selection, the old subscription stays outside the list.
	s.Require().NoError(periodic.All(ctx, input))
	s.Equal(before, s.getSyncStateV2(subs.NamespacedID))

	// Widening selection and forcing billing deadlines does not override the age cutoff.
	input.Lookback = 90 * 24 * time.Hour
	s.Require().NoError(periodic.All(ctx, input))
	s.Equal(subscriptionsync.SyncStateV2StateTooOld, s.getSyncStateV2(subs.NamespacedID).LastSuccessState)

	// The separate old-subscription override, also enabled by CLI --force-all, reaches sync.
	input.ForceResyncOldSubscriptions = true
	s.Require().NoError(periodic.All(ctx, input))
	s.Equal(subscriptionsync.SyncStateV2StateEnded, s.getSyncStateV2(subs.NamespacedID).LastSuccessState)
	s.requireReconciledSubscriptionEnd(subs)
}

func (s *ReconciliationTestSuite) TestTerminalSubscriptionSyncDryRun() {
	for _, age := range []time.Duration{24 * time.Hour, 40 * 24 * time.Hour} {
		s.Run(age.String(), func() {
			ctx := s.T().Context()
			clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()

			// Given an ended subscription with generated billing artifacts.
			subs := s.createCanceledSubscriptionForReconciliation()
			v2Before := s.getSyncStateV2(subs.NamespacedID)
			invoice := s.gatheringInvoice()
			legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
			s.Require().NoError(err)
			now := subs.ActiveTo.Add(age)
			clock.FreezeTime(now)
			defer clock.UnFreeze()

			// When either the reconciliation or the age skip is a dry run.
			s.Require().NoError(s.Service.SyncByID(ctx, subs.NamespacedID, now, subscriptionsync.EnableDryRun(), subscriptionsync.WithInvoicePendingLines()))

			// Then no billing changes, collection, scheduling update, or v2 result is committed.
			s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
			legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
			s.Require().NoError(err)
			s.Equal(legacyBefore, legacyAfter)
			states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subs.NamespacedID})
			s.Require().NoError(err)
			s.Require().Len(states, 1)
			s.Equal(v2Before, states[0])
		})
	}
}

func (s *ReconciliationTestSuite) TestOldSubscriptionCleanupPrecedence() {
	cases := []struct {
		name      string
		setup     func(subscription.Subscription)
		wantState subscriptionsync.SyncStateV2State
	}{
		{
			name: "subscription deletion",
			setup: func(subs subscription.Subscription) {
				// Existing database records can retain a deletion marker and an old service end.
				s.Require().NoError(s.DBClient.Subscription.UpdateOneID(subs.ID).SetDeletedAt(clock.Now()).Exec(s.T().Context()))
			},
			wantState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		},
		{
			name: "customer deletion before subscription start",
			setup: func(subs subscription.Subscription) {
				s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(subs.ActiveFrom).Exec(s.T().Context()))
			},
			wantState: subscriptionsync.SyncStateV2StateCustomerDeletedBefore,
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Given deletion cleanup for a subscription whose ActiveTo is also old.
			clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()
			subs := s.createCanceledSubscriptionForReconciliation()
			tc.setup(subs)
			now := subs.ActiveTo.Add(40 * 24 * time.Hour)
			clock.FreezeTime(now)
			defer clock.UnFreeze()

			// When ordinary sync runs, deletion cleanup takes precedence over skipping old billing work.
			s.Require().NoError(s.Service.SyncByID(s.T().Context(), subs.NamespacedID, now, subscriptionsync.WithInvoicePendingLines()))

			// Then artifacts are removed and the deletion result is recorded.
			s.Equal(tc.wantState, s.getSyncStateV2(subs.NamespacedID).LastSuccessState)
			s.expectNoGatheringInvoice()
		})
	}
}

func (s *ReconciliationTestSuite) TestTerminalSyncResultRollback() {
	cleanupCtx := s.T().Context()
	for _, age := range []time.Duration{24 * time.Hour, 40 * 24 * time.Hour} {
		s.Run(age.String(), func() {
			ctx := s.T().Context()
			clock.FreezeTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
			defer clock.UnFreeze()

			// Given a result write that will fail after revision acknowledgment.
			subs := s.createCanceledSubscriptionForReconciliation()
			invoice := s.gatheringInvoice()
			legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
			s.Require().NoError(err)
			now := subs.ActiveTo.Add(age)
			clock.FreezeTime(now)
			defer clock.UnFreeze()
			s.Require().NoError(s.DBClient.SubscriptionSyncStateV2.Update().
				Where(subscriptionsyncstatev2.SubscriptionID(subs.ID)).
				SetLastSuccessState(subscriptionsync.SyncStateV2StateTooOld).
				ClearNextReconciliationAt().
				SetLastSuccessAt(now.Add(-time.Hour)).
				SetLastSyncedSubsRevision(ulid.Make().String()).
				SetCurrentSubsRevision(ulid.Make().String()).Exec(ctx))
			before := s.getSyncStateV2(subs.NamespacedID)
			const constraintName = "test_terminal_sync_result_failure"
			_, err = s.TestDB.PGDriver.DB().ExecContext(ctx, fmt.Sprintf(
				`ALTER TABLE subscription_sync_states_v2 ADD CONSTRAINT %s CHECK (subscription_id <> %s OR last_synced_subs_revision IS DISTINCT FROM current_subs_revision)`,
				pq.QuoteIdentifier(constraintName), pq.QuoteLiteral(subs.ID)))
			s.Require().NoError(err)
			dropConstraint := fmt.Sprintf("ALTER TABLE subscription_sync_states_v2 DROP CONSTRAINT IF EXISTS %s", pq.QuoteIdentifier(constraintName))
			s.T().Cleanup(func() {
				_, cleanupErr := s.TestDB.PGDriver.DB().ExecContext(cleanupCtx, dropConstraint)
				s.NoError(cleanupErr)
			})

			// When recording an ended or skipped result fails inside the billing transaction.
			s.Require().ErrorContains(s.Service.SyncByID(ctx, subs.NamespacedID, now), constraintName)

			// Then existing results and any billing or scheduling changes are rolled back.
			s.Equal(before, s.getSyncStateV2(subs.NamespacedID))
			s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
			legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subs.NamespacedID})
			s.Require().NoError(err)
			s.Equal(legacyBefore, legacyAfter)
		})
	}
}
