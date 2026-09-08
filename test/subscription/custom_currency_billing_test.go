package subscription_test

import (
	"encoding/json"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestSubscriptionSyncCustomCurrencyBilling(t *testing.T) {
	const namespace = "test-namespace"

	setupAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	startsAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	invoiceAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	usageAt := startsAt.Add(14 * 24 * time.Hour)

	tests := []struct {
		name                  string
		settlementMode        productcatalog.SettlementMode
		costBasisMode         subscription.CostBasisMode
		planCurrency          currencyx.Code
		customItem            bool
		mixedFiatItem         bool
		createCostBasis       bool
		createReplacementRate bool
		expectInvoice         bool
		expectInvoiceTotal    float64
		expectCustomLineTotal float64
		expectFiatLineTotal   float64
		expectCostBasisKind   costbasis.Mode
	}{
		{
			name:           "credit only custom plan needs no cost basis and never invoices overage",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			costBasisMode:  subscription.CostBasisModeDynamic,
			planCurrency:   "CREDITS",
		},
		{
			name:                  "credit then invoice custom plan converts uncovered usage dynamically",
			settlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			costBasisMode:         subscription.CostBasisModeDynamic,
			planCurrency:          "CREDITS",
			createCostBasis:       true,
			expectInvoice:         true,
			expectInvoiceTotal:    4,
			expectCustomLineTotal: 4,
			expectCostBasisKind:   costbasis.ModeDynamic,
		},
		{
			name:                  "dynamic custom pricing uses the replacement rate at realization",
			settlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			costBasisMode:         subscription.CostBasisModeDynamic,
			planCurrency:          "CREDITS",
			createCostBasis:       true,
			createReplacementRate: true,
			expectInvoice:         true,
			expectInvoiceTotal:    6,
			expectCustomLineTotal: 6,
			expectCostBasisKind:   costbasis.ModeDynamic,
		},
		{
			name:                  "credit then invoice custom plan keeps its pinned cost basis after a replacement",
			settlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			costBasisMode:         subscription.CostBasisModePinned,
			planCurrency:          "CREDITS",
			createCostBasis:       true,
			createReplacementRate: true,
			expectInvoice:         true,
			expectInvoiceTotal:    4,
			expectCustomLineTotal: 4,
			expectCostBasisKind:   costbasis.ModePinned,
		},
		{
			name:                  "fiat plan routes a custom currency item through dynamic conversion",
			settlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			costBasisMode:         subscription.CostBasisModeDynamic,
			planCurrency:          "USD",
			customItem:            true,
			createCostBasis:       true,
			expectInvoice:         true,
			expectInvoiceTotal:    4,
			expectCustomLineTotal: 4,
			expectCostBasisKind:   costbasis.ModeDynamic,
		},
		{
			name:                  "mixed fiat and custom items share one fiat invoice while only custom usage is converted",
			settlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
			costBasisMode:         subscription.CostBasisModePinned,
			planCurrency:          "USD",
			customItem:            true,
			mixedFiatItem:         true,
			createCostBasis:       true,
			createReplacementRate: true,
			expectInvoice:         true,
			expectInvoiceTotal:    7,
			expectCustomLineTotal: 4,
			expectFiatLineTotal:   3,
			expectCostBasisKind:   costbasis.ModePinned,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a subscription whose usage is 5 units at 2 CREDITS per unit
			// - only 2 CREDITS are available to cover its 10 CREDITS charge
			// - the billing profile and settlement mode select whether an overage can become a USD invoice
			clock.FreezeTime(setupAt)
			defer clock.UnFreeze()

			deps := setup(t, setupConfig{enableCharges: true, enableCreditThenInvoice: true})
			defer deps.cleanup(t)
			provisionSubscriptionDefaultTaxCodes(t, deps, namespace)

			profileInput := minimalCreateProfileInputTemplate(deps.sandboxApp.GetID())
			profileInput.WorkflowConfig.Collection.Interval = datetime.MustParseDuration(t, "P1D")
			profileInput.WorkflowConfig.Invoicing.AutoAdvance = false
			_, err := deps.billingService.CreateProfile(t.Context(), profileInput)
			require.NoError(t, err)

			customCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
				namespace, "CREDITS", "Credits", "CR",
			))
			require.NoError(t, err)

			var pinnedCostBasisID string
			if tc.createCostBasis {
				createdCostBasis, err := deps.CurrencyService.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
					Namespace:  namespace,
					CurrencyID: customCurrency.ID,
					FiatCode:   "USD",
					Rate:       decimal.NewFromFloat(0.5),
				})
				require.NoError(t, err)
				pinnedCostBasisID = createdCostBasis.ID
			}

			features := deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
			createdPlan := createUsageSubscriptionPlan(t, deps, usageSubscriptionPlanInput{
				Namespace:      namespace,
				Key:            "custom-currency-billing",
				PlanCurrency:   tc.planCurrency,
				CustomCurrency: customCurrency,
				CustomItem:     tc.customItem,
				MixedFiatItem:  tc.mixedFiatItem,
				Features:       features,
				EffectiveFrom:  setupAt.Add(-time.Second),
			})

			customer := createUSDSubscriptionCustomer(t, deps, namespace, "custom-currency-billing")
			_, err = deps.ledgerDeps.ResolversService.EnsureBusinessAccounts(t.Context(), namespace)
			require.NoError(t, err)
			accounts, err := deps.ledgerDeps.ResolversService.CreateCustomerAccounts(t.Context(), customer.GetID())
			require.NoError(t, err)
			period := timeutil.ClosedPeriod{From: setupAt, To: setupAt}
			grants, err := deps.chargesService.Create(t.Context(), charges.CreateInput{
				Namespace: namespace,
				Intents: charges.ChargeIntents{charges.NewChargeIntent(creditpurchase.Intent{
					Intent: chargesmeta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: customer.ID, Currency: customCurrency},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						IntentMutableFields: chargesmeta.IntentMutableFields{Name: "Initial custom credits", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
						CreditAmount:        decimal.NewFromInt(2), EffectiveAt: &setupAt,
						Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
					},
				})},
			})
			require.NoError(t, err)
			require.Len(t, grants, 1)
			createdSubscription, err := createCustomCurrencySubscription(
				t,
				deps,
				createdPlan,
				customer.ID,
				startsAt,
				tc.settlementMode,
				tc.costBasisMode,
			)
			require.NoError(t, err)

			if tc.createReplacementRate {
				replacementEffectiveFrom := setupAt.Add(15 * 24 * time.Hour)
				_, err = deps.CurrencyService.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
					Namespace:     namespace,
					CurrencyID:    customCurrency.ID,
					FiatCode:      "USD",
					Rate:          decimal.NewFromFloat(0.75),
					EffectiveFrom: &replacementEffectiveFrom,
				})
				require.NoError(t, err)
			}

			view, err := deps.subscriptionService.GetView(t.Context(), createdSubscription.NamespacedID)
			require.NoError(t, err)
			require.Equal(t, currencyx.Code("USD"), view.Subscription.InvoiceCurrency)
			assertSubscriptionItemCurrencies(t, view, customCurrency.ID, tc.mixedFiatItem)

			serializedView, err := json.Marshal(view)
			require.NoError(t, err)
			var eventView subscription.SubscriptionView
			require.NoError(t, json.Unmarshal(serializedView, &eventView))

			// when:
			// - the serialized subscription event is synchronized through the charge backend twice
			// - usage is realized at the end of the billing period
			require.NoError(t, deps.subscriptionSyncService.SyncByView(t.Context(), eventView, invoiceAt))
			chargeIDsBeforeRetry := listSubscriptionChargeIDs(t, deps, createdSubscription.ID)
			require.NotEmpty(t, chargeIDsBeforeRetry)
			require.NoError(t, deps.subscriptionSyncService.SyncByView(t.Context(), eventView, invoiceAt))
			require.Equal(t, chargeIDsBeforeRetry, listSubscriptionChargeIDs(t, deps, createdSubscription.ID))

			assertSubscriptionChargeCurrenciesAndCostBasis(
				t,
				deps,
				createdSubscription.ID,
				customCurrency.ID,
				tc.expectCostBasisKind,
				pinnedCostBasisID,
			)

			deps.MockStreamingConnector.AddSimpleEvent(
				subscriptiontestutils.ExampleFeatureMeterSlug,
				5,
				usageAt,
			)
			clock.FreezeTime(invoiceAt)

			if tc.settlementMode == productcatalog.CreditOnlySettlementMode {
				_, err = deps.chargesService.AdvanceCharges(t.Context(), charges.AdvanceChargesInput{
					Customer: customer.GetID(),
				})
				require.NoError(t, err)
			} else {
				invoices, err := deps.billingService.InvoicePendingLines(t.Context(), billing.InvoicePendingLinesInput{
					Customer: customer.GetID(),
					AsOf:     &invoiceAt,
				})
				require.NoError(t, err)
				require.Len(t, invoices, 1)
				assertCustomCurrencyInvoice(t, invoices[0], tc.expectInvoiceTotal, tc.expectCustomLineTotal, tc.expectFiatLineTotal)
			}

			// then:
			// - credit-only usage is fully realized without creating billing artifacts
			// - credit-then-invoice usage consumes 2 CREDITS and invoices only the converted remainder
			expectedAllocated := float64(2)
			if !tc.expectInvoice {
				expectedAllocated = 10
				assertNoSubscriptionInvoices(t, deps, customer.ID)
			}
			balance, err := deps.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), accounts.FBOAccount, ledger.RouteFilter{Currency: customCurrency.Reference()}, ledger.BalanceQuery{})
			require.NoError(t, err)
			require.Equal(t, float64(0), balance.InexactFloat64())
			if !tc.expectInvoice {
				advance, err := deps.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), accounts.ReceivableAccount, ledger.RouteFilter{Currency: customCurrency.Reference()}, ledger.BalanceQuery{})
				require.NoError(t, err)
				require.Equal(t, float64(-8), advance.InexactFloat64())
			}

			lineages, err := deps.lineageService.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{Namespace: namespace, CustomerID: customer.ID, Currency: customCurrency.Reference()})
			require.NoError(t, err)
			require.NotEmpty(t, lineages)
			allocated := decimal.Zero
			for _, entry := range lineages {
				for _, segment := range entry.Segments {
					allocated = allocated.Add(segment.Amount)
				}
			}
			require.Equal(t, expectedAllocated, allocated.InexactFloat64())
			// Retrying synchronization and advancement must preserve the journal,
			// not just stable charge identifiers.
			beforeEntries, err := deps.DBDeps.DBClient.LedgerEntry.Query().Count(t.Context())
			require.NoError(t, err)
			require.NoError(t, deps.subscriptionSyncService.SyncByView(t.Context(), eventView, invoiceAt))
			_, err = deps.chargesService.AdvanceCharges(t.Context(), charges.AdvanceChargesInput{Customer: customer.GetID()})
			require.NoError(t, err)
			afterEntries, err := deps.DBDeps.DBClient.LedgerEntry.Query().Count(t.Context())
			require.NoError(t, err)
			require.Equal(t, beforeEntries, afterEntries)
			if !tc.expectInvoice {
				// A paid purchase backfills the subscription's uncovered advance;
				// payment settlement must recognize the paid native spend exactly once.
				backfillAt := invoiceAt.Add(time.Minute)
				clock.FreezeTime(backfillAt)
				period := timeutil.ClosedPeriod{From: backfillAt, To: backfillAt}
				fiat, err := currencyx.NewFiatCurrency("USD")
				require.NoError(t, err)
				purchases, err := deps.chargesService.Create(t.Context(), charges.CreateInput{
					Namespace: namespace,
					Intents: charges.ChargeIntents{charges.NewChargeIntent(creditpurchase.Intent{
						Intent: chargesmeta.Intent{ManagedBy: billing.ManuallyManagedLine, CustomerID: customer.ID, Currency: customCurrency},
						IntentMutableFields: creditpurchase.IntentMutableFields{
							IntentMutableFields: chargesmeta.IntentMutableFields{Name: "Advance backfill", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period},
							CreditAmount:        decimal.NewFromInt(8), EffectiveAt: &backfillAt,
							Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus}),
						},
						CostBasis: creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: decimal.NewFromFloat(0.5)})),
					})},
				})
				require.NoError(t, err)
				require.Len(t, purchases, 1)
				purchaseID, err := purchases[0].GetChargeID()
				require.NoError(t, err)
				for _, status := range []payment.Status{payment.StatusAuthorized, payment.StatusSettled} {
					_, err = deps.chargesService.HandleCreditPurchaseExternalPaymentStateTransition(t.Context(), charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{ChargeID: purchaseID, TargetPaymentState: status})
					require.NoError(t, err)
				}
				advance, err := deps.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), accounts.ReceivableAccount, ledger.RouteFilter{Currency: customCurrency.Reference()}, ledger.BalanceQuery{})
				require.NoError(t, err)
				require.Equal(t, float64(0), advance.InexactFloat64())
				backfilled, err := deps.lineageService.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{Namespace: namespace, CustomerID: customer.ID, Currency: customCurrency.Reference()})
				require.NoError(t, err)
				business, err := deps.ledgerDeps.ResolversService.GetBusinessAccounts(t.Context(), namespace)
				require.NoError(t, err)
				earnings, err := deps.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), business.EarningsAccount, ledger.RouteFilter{Currency: customCurrency.Reference()}, ledger.BalanceQuery{})
				require.NoError(t, err)
				require.Equal(t, float64(8), earnings.InexactFloat64())
				accrued, err := deps.ledgerDeps.HistoricalLedger.GetAccountBalance(t.Context(), accounts.AccruedAccount, ledger.RouteFilter{Currency: customCurrency.Reference()}, ledger.BalanceQuery{})
				require.NoError(t, err)
				require.Equal(t, float64(2), accrued.InexactFloat64())

				// Nil-cost-basis promotional credits remain deferred. Recognition
				// must belong entirely to the paid backfill, matching the journal.
				require.Len(t, backfilled, 2)
				for _, entry := range backfilled {
					require.Len(t, entry.Segments, 1)
					segment := entry.Segments[0]
					switch entry.OriginKind {
					case creditrealization.LineageOriginKindRealCredit:
						require.Equal(t, float64(2), segment.Amount.InexactFloat64())
						require.Equal(t, creditrealization.LineageSegmentStateRealCredit, segment.State)
					case creditrealization.LineageOriginKindAdvance:
						require.Equal(t, float64(8), segment.Amount.InexactFloat64())
						require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, segment.State)
						require.Equal(t, creditrealization.LineageSegmentStateAdvanceBackfilled, lo.FromPtr(segment.SourceState))
					default:
						t.Fatalf("unexpected lineage origin %q", entry.OriginKind)
					}
				}
				beforeEntries, err = deps.DBDeps.DBClient.LedgerEntry.Query().Count(t.Context())
				require.NoError(t, err)
				_, err = deps.chargesService.AdvanceCharges(t.Context(), charges.AdvanceChargesInput{Customer: customer.GetID()})
				require.NoError(t, err)
				afterEntries, err = deps.DBDeps.DBClient.LedgerEntry.Query().Count(t.Context())
				require.NoError(t, err)
				require.Equal(t, beforeEntries, afterEntries)
				assertNoSubscriptionInvoices(t, deps, customer.ID)
			}
		})
	}
}

type usageSubscriptionPlanInput struct {
	Namespace      string
	Key            string
	PlanCurrency   currencyx.Code
	CustomCurrency currencies.Currency
	CustomItem     bool
	MixedFiatItem  bool
	Features       []feature.Feature
	EffectiveFrom  time.Time
}

func createUsageSubscriptionPlan(t *testing.T, deps testDeps, input usageSubscriptionPlanInput) plan.Plan {
	t.Helper()
	require.NotEmpty(t, input.Features)

	month := datetime.MustParseDuration(t, "P1M")
	customItem := input.PlanCurrency == input.CustomCurrency.GetCode() || input.CustomItem
	customRateCard := &productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:     input.Features[0].Key,
			Name:    "Custom usage",
			Feature: productcatalog.NewFeatureReference(&input.Features[0].ID, &input.Features[0].Key),
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: decimal.NewFromInt(2),
			}),
		},
		BillingCadence: month,
	}
	if input.CustomItem && input.PlanCurrency != input.CustomCurrency.GetCode() {
		customRateCard.RateCardMeta.Currency = lo.ToPtr(input.CustomCurrency.Reference())
	}
	require.True(t, customItem)

	rateCards := productcatalog.RateCards{customRateCard}
	if input.MixedFiatItem {
		require.GreaterOrEqual(t, len(input.Features), 2)
		rateCards = append(rateCards, &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:     input.Features[1].Key,
				Name:    "Fiat usage",
				Feature: productcatalog.NewFeatureReference(&input.Features[1].ID, &input.Features[1].Key),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: decimal.NewFromFloat(0.6),
				}),
			},
			BillingCadence: month,
		})
	}

	createdPlan, err := deps.PlanService.CreatePlan(t.Context(), plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: input.Namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:            input.Key,
				Name:           "Custom currency billing",
				Currency:       currencies.NewCurrencyReference(input.PlanCurrency),
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				BillingCadence: month,
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
				RateCards: rateCards,
			}},
		},
	})
	require.NoError(t, err)

	createdPlan, err = deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: createdPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: &input.EffectiveFrom,
		},
	})
	require.NoError(t, err)

	return *createdPlan
}

func assertSubscriptionItemCurrencies(t *testing.T, view subscription.SubscriptionView, customCurrencyID string, expectFiatItem bool) {
	t.Helper()

	phase, ok := view.GetPhaseByKey("default")
	require.True(t, ok)
	customItems := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
	require.Len(t, customItems, 1)
	customCurrency := customItems[0].SubscriptionItem.RateCard.AsMeta().Currency
	require.NotNil(t, customCurrency)
	require.Equal(t, customCurrencyID, lo.FromPtr(customCurrency.CustomCurrencyID))

	if expectFiatItem {
		fiatItems := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey2]
		require.Len(t, fiatItems, 1)
		fiatCurrency := fiatItems[0].SubscriptionItem.RateCard.AsMeta().Currency
		require.NotNil(t, fiatCurrency)
		require.True(t, fiatCurrency.IsFiat())
		require.Equal(t, currencyx.Code("USD"), fiatCurrency.GetCode())
	}
}

func listSubscriptionChargeIDs(t *testing.T, deps testDeps, subscriptionID string) []string {
	t.Helper()

	result, err := deps.chargesService.ListCharges(t.Context(), charges.ListChargesInput{
		Page:            pagination.Page{PageNumber: 1, PageSize: 100},
		Namespace:       "test-namespace",
		SubscriptionIDs: []string{subscriptionID},
		OrderBy:         "id",
	})
	require.NoError(t, err)

	chargeIDs := make([]string, 0, len(result.Items))
	for _, charge := range result.Items {
		chargeID, err := charge.GetChargeID()
		require.NoError(t, err)
		chargeIDs = append(chargeIDs, chargeID.ID)
	}

	return chargeIDs
}

func assertSubscriptionChargeCurrenciesAndCostBasis(
	t *testing.T,
	deps testDeps,
	subscriptionID string,
	customCurrencyID string,
	expectKind costbasis.Mode,
	expectPinnedCostBasisID string,
) {
	t.Helper()

	result, err := deps.chargesService.ListCharges(t.Context(), charges.ListChargesInput{
		Page:            pagination.Page{PageNumber: 1, PageSize: 100},
		Namespace:       "test-namespace",
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeUsageBased},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Items)

	customCharges := 0
	fiatCharges := 0
	for _, item := range result.Items {
		charge, err := item.AsUsageBasedCharge()
		require.NoError(t, err)
		if charge.Intent.GetCurrency().IsFiat() {
			fiatCharges++
			require.Nil(t, charge.Intent.GetCostBasisIntent())
			continue
		}

		customCharges++
		require.Equal(t, customCurrencyID, charge.Intent.GetCurrency().ID)
		costBasisIntent := charge.Intent.GetCostBasisIntent()
		if expectKind == "" {
			require.Nil(t, costBasisIntent)
			continue
		}

		require.NotNil(t, costBasisIntent)
		require.Equal(t, expectKind, costBasisIntent.Kind())
		if expectKind == costbasis.ModePinned {
			pinned, err := costBasisIntent.AsPinned()
			require.NoError(t, err)
			require.Equal(t, expectPinnedCostBasisID, pinned.CurrencyCostBasisID)
		}
	}
	require.Positive(t, customCharges)
	if fiatCharges > 0 {
		require.Equal(t, costbasis.ModePinned, expectKind)
	}
}

func assertCustomCurrencyInvoice(t *testing.T, invoice billing.StandardInvoice, expectedTotal, expectedCustomTotal, expectedFiatTotal float64) {
	t.Helper()

	require.Equal(t, currencyx.FiatCode("USD"), invoice.Currency)
	require.Equal(t, expectedTotal, invoice.Totals.Total.InexactFloat64())

	lines := invoice.Lines.OrEmpty()
	expectedLineCount := 1
	if expectedFiatTotal > 0 {
		expectedLineCount++
	}
	require.Len(t, lines, expectedLineCount)

	customLine, found := lo.Find(lines, func(line *billing.StandardLine) bool {
		return line.Name == "Custom usage"
	})
	require.True(t, found)
	require.Equal(t, currencyx.FiatCode("USD"), customLine.Currency)
	require.Equal(t, expectedCustomTotal, customLine.Totals.Total.InexactFloat64())
	require.Len(t, customLine.DetailedLines, 1)
	require.Equal(t, float64(8), customLine.DetailedLines[0].Quantity.InexactFloat64())
	require.Equal(t, expectedCustomTotal/8, customLine.DetailedLines[0].PerUnitAmount.InexactFloat64())

	if expectedFiatTotal > 0 {
		fiatLine, found := lo.Find(lines, func(line *billing.StandardLine) bool {
			return line.Name == "Fiat usage"
		})
		require.True(t, found)
		require.Equal(t, expectedFiatTotal, fiatLine.Totals.Total.InexactFloat64())
	}
}

func assertNoSubscriptionInvoices(t *testing.T, deps testDeps, customerID string) {
	t.Helper()

	standardInvoices, err := deps.billingService.ListInvoices(t.Context(), billing.ListInvoicesInput{
		Page:         pagination.Page{PageNumber: 1, PageSize: 100},
		Namespace:    "test-namespace",
		OnlyStandard: true,
		Expand:       billing.InvoiceExpandAll,
	})
	require.NoError(t, err)
	require.Empty(t, standardInvoices.Items)

	gatheringInvoices, err := deps.billingService.ListGatheringInvoices(t.Context(), billing.ListGatheringInvoicesInput{
		Page:      pagination.Page{PageNumber: 1, PageSize: 100},
		Namespace: "test-namespace",
		Customers: []string{customerID},
		Expand:    billing.GatheringInvoiceExpandAll,
	})
	require.NoError(t, err)
	require.Empty(t, gatheringInvoices.Items)
}

func provisionSubscriptionDefaultTaxCodes(t *testing.T, deps testDeps, namespace string) {
	t.Helper()

	invoicing, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: namespace,
		Key:       taxcode.ProviderDefaultTaxCodeKey,
		Name:      "Provider Default",
	})
	require.NoError(t, err)
	creditGrant, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: namespace,
		Key:       "default-credit-grant",
		Name:      "Default Credit Grant",
	})
	require.NoError(t, err)

	_, err = deps.TaxCodeService.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
		Namespace:            namespace,
		InvoicingTaxCodeID:   invoicing.ID,
		CreditGrantTaxCodeID: creditGrant.ID,
	})
	require.NoError(t, err)
}
