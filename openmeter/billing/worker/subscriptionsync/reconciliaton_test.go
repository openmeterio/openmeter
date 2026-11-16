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
	syncservice "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
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

type customerDeletionSubscriptionInput struct {
	NoBillables     bool
	SkipInitialSync bool
}

func (s *ReconciliationTestSuite) createSubscriptionForCustomerDeletion(in customerDeletionSubscriptionInput) subscription.SubscriptionView {
	s.T().Helper()
	// Each boundary case owns its customer and plan while reusing the suite's billing stack.
	ctx := s.T().Context()
	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.Namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:            lo.ToPtr(ulid.Make().String()),
			Name:           "Customer deletion cleanup",
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
				Name:           "Customer deletion cleanup",
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
	view, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: lo.ToPtr(clock.Now().Add(24 * time.Hour))},
			Name:   "Subscription for customer deletion",
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
			view := s.createSubscriptionForCustomerDeletion(customerDeletionSubscriptionInput{
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
				s.Empty(states)
				invoice := s.gatheringInvoice()
				s.NotEmpty(invoice.Lines.OrEmpty())

				return
			}

			s.expectNoGatheringInvoice()
			s.Require().Len(states, 1)
			s.Equal(subscriptionsync.SyncStateV2{
				SubscriptionID:   subscriptionID,
				LastSuccessState: subscriptionsync.SyncStateV2StateCustomerDeletedBefore,
				LastSuccessAt:    now,
			}, states[0].SyncStateV2)
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
					Namespace:       s.Namespace,
					SubscriptionIDs: []string{subscriptionID.ID},
					IncludeDeleted:  true,
					Status:          &filter.FilterString{Eq: lo.ToPtr(string(chargesmeta.ChargeStatusDeleted))},
				})
				s.Require().NoError(err)
				s.Require().Len(chargeList.Items, initialLineCount)
			}
		})
	}
}

func (s *ReconciliationTestSuite) TestCustomerDeletedBeforeCleanupDryRun() {
	ctx := s.T().Context()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given future billing artifacts whose customer was deleted before subscription start.
	view := s.createSubscriptionForCustomerDeletion(customerDeletionSubscriptionInput{})
	subscriptionID := view.Subscription.NamespacedID
	invoice := s.gatheringInvoice()
	legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
	s.Require().NoError(err)
	s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).SetDeletedAt(now).Exec(ctx))

	// When cleanup is requested as a dry run, with invoicing also requested.
	s.Require().NoError(s.Service.SyncByID(ctx, subscriptionID, now, subscriptionsync.EnableDryRun(), subscriptionsync.WithInvoicePendingLines()))

	// Then neither artifacts nor scheduling change and no successful cleanup is recorded.
	s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
	legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
	s.Require().NoError(err)
	s.Equal(legacyBefore, legacyAfter)
	states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
	s.Require().NoError(err)
	s.Empty(states)
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
				return fmt.Sprintf("subscription_id <> %s", pq.QuoteLiteral(subscriptionID.ID))
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
			view := s.createSubscriptionForCustomerDeletion(customerDeletionSubscriptionInput{})
			subscriptionID := view.Subscription.NamespacedID
			invoice := s.gatheringInvoice()
			legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
			s.Require().NoError(err)
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

			// Then the billing transaction restores artifacts and scheduling without recording completion.
			s.Equal(invoice.Lines.OrEmpty(), s.gatheringInvoice().Lines.OrEmpty())
			legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{subscriptionID})
			s.Require().NoError(err)
			s.Equal(legacyBefore, legacyAfter)
			states, err := s.Service.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
			s.Require().NoError(err)
			s.Empty(states)
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
	view := s.createSubscriptionForCustomerDeletion(customerDeletionSubscriptionInput{})
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
	s.Equal(later, states[0].LastSuccessAt)

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
