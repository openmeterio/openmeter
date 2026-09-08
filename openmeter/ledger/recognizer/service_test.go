package recognizer_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	legacylineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/adapter"
	legacylineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type recognizerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	recognizer  recognizer.Service
	lineage     legacylineage.Service
	lastGroupID string
}

func newRecognizerTestEnv(t *testing.T) *recognizerTestEnv {
	t.Helper()

	base := ledgertestutils.NewIntegrationEnv(t, "recognizer")
	deps := transactions.ResolverDependencies{
		AccountService: base.Deps.ResolversService,
		AccountCatalog: base.Deps.AccountService,
		BalanceQuerier: base.Deps.HistoricalLedger,
	}

	lngeAdapter, err := legacylineageadapter.New(legacylineageadapter.Config{
		Client: base.DB,
	})
	require.NoError(t, err)

	dbLineage, err := lineageservice.New(lineageservice.Config{
		Adapter: lngeAdapter,
	})
	require.NoError(t, err)

	lngeSvc := &ledgertestutils.LineageWithAllocations{Service: dbLineage}

	recSvc, err := recognizer.NewService(recognizer.Config{
		Ledger:             base.Deps.HistoricalLedger,
		Dependencies:       deps,
		Lineage:            lngeSvc,
		TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)

	return &recognizerTestEnv{
		IntegrationEnv: base,
		recognizer:     recSvc,
		lineage:        lngeSvc,
	}
}

func testID() string {
	return ulid.Make().String()
}

func (e *recognizerTestEnv) resolverDeps() transactions.ResolverDependencies {
	return transactions.ResolverDependencies{
		AccountService: e.Deps.ResolversService,
		AccountCatalog: e.Deps.AccountService,
		BalanceQuerier: e.Deps.HistoricalLedger,
	}
}

func (e *recognizerTestEnv) resolveAndCommit(t *testing.T, templates ...transactions.TransactionTemplate) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		e.resolverDeps(),
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		templates...,
	)
	require.NoError(t, err)

	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
	e.lastGroupID = group.ID().ID
}

// ensureCharge creates a minimal charge record in the DB if it doesn't exist.
func (e *recognizerTestEnv) ensureCharge(t *testing.T, chargeID string) {
	t.Helper()

	exists, err := e.DB.Charge.Get(t.Context(), chargeID)
	if err == nil && exists != nil {
		return
	}

	_, err = e.DB.Charge.Create().
		SetID(chargeID).
		SetNamespace(e.Namespace).
		SetType(meta.ChargeTypeFlatFee).
		Save(t.Context())
	require.NoError(t, err)
}

// createLineageForRealization creates a lineage record for a realization, mimicking
// what the charges system does after credit allocation.
func (e *recognizerTestEnv) createLineageForRealization(t *testing.T, chargeID, realizationID string, currency currencies.Currency, amount alpacadecimal.Decimal, originKind creditrealization.LineageOriginKind) {
	t.Helper()

	e.ensureCharge(t, chargeID)

	err := e.legacylineage.CreateInitialLineages(t.Context(), legacylineage.CreateInitialLineagesInput{
		Namespace:  e.Namespace,
		ChargeID:   chargeID,
		CustomerID: e.CustomerID.ID,
		Currency:   currency,
		Realizations: creditrealization.Realizations{
			{
				CreateInput: creditrealization.CreateInput{
					ID:     realizationID,
					Amount: amount,
					Type:   creditrealization.TypeAllocation,
					ServicePeriod: timeutil.ClosedPeriod{
						From: clock.Now().Add(-24 * time.Hour),
						To:   clock.Now(),
					},
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: e.lastGroupID,
					},
					Annotations: creditrealization.LineageAnnotations(originKind),
				},
			},
		},
	})
	require.NoError(t, err)
}

func TestRecognizeEarnings_IdempotencyOnUnchangedState(t *testing.T) {
	env := newRecognizerTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	currency := currenciestestutils.NewFiatCurrency(t, env.Currency)

	chargeID := testID()
	sourceChargeID := testID()
	realID := testID()

	// Set up accrued balance and legacylineage.
	env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.CurrencyReference(), CostBasis: &costBasis,
		SourceChargeID: &sourceChargeID, SpendChargeID: &chargeID,
	})
	env.createLineageForRealization(t, chargeID, realID, currency, alpacadecimal.NewFromInt(50), creditrealization.LineageOriginKindRealCredit)

	// First recognition.
	result1, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   currency,
	})
	require.NoError(t, err)
	require.True(t, result1.RecognizedAmount.Equal(alpacadecimal.NewFromInt(50)))
	require.NotEmpty(t, result1.LedgerGroupID)

	// Second recognition with unchanged state should be a no-op.
	result2, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   currency,
	})
	require.NoError(t, err)
	require.True(t, result2.RecognizedAmount.IsZero())
	require.Empty(t, result2.LedgerGroupID)

	// Balances should be stable.
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(50)))
}

func TestRecognizeEarnings_ReceivableCoverageDoesNotRecognizeUnrelatedAccrued(t *testing.T) {
	env := newRecognizerTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	currency := currenciestestutils.NewFiatCurrency(t, env.Currency)
	accruedChargeID := testID()
	coverageChargeID := testID()
	sourceChargeID := testID()
	realizationID := testID()

	// given:
	// - accrued value from one charge
	// - an unrelated receivable-coverage lineage for the same customer and currency
	env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(5), Currency: env.CurrencyReference(), CostBasis: &costBasis,
		SourceChargeID: &sourceChargeID, SpendChargeID: &accruedChargeID,
	})
	env.createLineageForRealization(t, coverageChargeID, realizationID, currency, alpacadecimal.NewFromInt(3), creditrealization.LineageOriginKindReceivableCoverage)

	// when:
	// - recognition runs for the shared customer and currency
	result, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   currency,
	})
	require.NoError(t, err)

	// then:
	// - receivable coverage cannot consume the unrelated accrued balance
	require.True(t, result.RecognizedAmount.IsZero())
	require.Empty(t, result.LedgerGroupID)
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(5)))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).IsZero())

	lineages, err := env.legacylineage.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   env.CurrencyReference(),
	})
	require.NoError(t, err)
	require.Len(t, lineages, 1)
	require.Len(t, lineages[0].Segments, 1)
	require.Equal(t, creditrealization.LineageSegmentStateReceivableCoverage, lineages[0].Segments[0].State)
}

func TestRecognizeEarnings_CustomCurrencyCreditBackedAccrued(t *testing.T) {
	env := newRecognizerTestEnv(t)
	customCurrency := currenciestestutils.NewCustomCurrency(t, currencyx.Code("ACME"), 2)
	customCurrencyReference := customCurrency.Reference()
	fiatCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)
	amount := alpacadecimal.NewFromInt(40)
	chargeID := testID()
	sourceChargeID := testID()
	realizationID := testID()

	// given:
	// - custom-currency accrued value backed by a distinct credit purchase
	// - matching real-credit lineage in the native custom currency
	env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
		At:                env.Now(),
		Amount:            amount,
		Currency:          customCurrencyReference,
		CostBasisCurrency: &fiatCurrency,
		CostBasis:         &costBasis,
		SourceChargeID:    &sourceChargeID,
		SpendChargeID:     &chargeID,
	})
	env.createLineageForRealization(t, chargeID, realizationID, customCurrency, amount, creditrealization.LineageOriginKindRealCredit)

	// when:
	// - earnings are recognized in the native custom currency
	result, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   customCurrency,
	})
	require.NoError(t, err)
	require.True(t, result.RecognizedAmount.Equal(amount))
	require.NotEmpty(t, result.LedgerGroupID)

	// then:
	// - the custom accrued route is cleared and the equivalent earnings route is credited
	accrued := env.AccruedSubAccountForCurrency(t, customCurrencyReference, &fiatCurrency, &costBasis, nil)
	earnings, err := env.BusinessAccounts.EarningsAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:          customCurrencyReference,
		CostBasisCurrency: &fiatCurrency,
		CostBasis:         &costBasis,
	})
	require.NoError(t, err)
	require.True(t, env.SumBalance(t, accrued).IsZero())
	require.True(t, env.SumBalance(t, earnings).Equal(amount))

	lineages, err := env.legacylineage.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   customCurrencyReference,
	})
	require.NoError(t, err)
	require.Len(t, lineages, 1)
	require.Len(t, lineages[0].Segments, 1)
	require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, lineages[0].Segments[0].State)
	require.NotNil(t, lineages[0].Segments[0].BackingTransactionGroupID)
}

func TestRecognizeEarnings_DeterministicAllocationAndSegmentTransition(t *testing.T) {
	env := newRecognizerTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	currency := currenciestestutils.NewFiatCurrency(t, env.Currency)
	chargeID := testID()
	sourceChargeID := testID()
	realA := testID()
	realB := testID()

	// Set up accrued balance and two lineages.
	env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(70), Currency: env.CurrencyReference(), CostBasis: &costBasis,
		SourceChargeID: &sourceChargeID, SpendChargeID: &chargeID,
	})
	env.createLineageForRealization(t, chargeID, realA, currency, alpacadecimal.NewFromInt(30), creditrealization.LineageOriginKindRealCredit)
	env.createLineageForRealization(t, chargeID, realB, currency, alpacadecimal.NewFromInt(40), creditrealization.LineageOriginKindRealCredit)

	result, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: env.CustomerID,
		At:         clock.Now(),
		Currency:   currency,
	})
	require.NoError(t, err)
	require.True(t, result.RecognizedAmount.Equal(alpacadecimal.NewFromInt(70)))

	// Verify segments transitioned to earnings_recognized.
	lineages, err := env.legacylineage.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   currencies.NewCurrencyReference(env.Currency),
	})
	require.NoError(t, err)

	for _, l := range lineages {
		for _, seg := range l.Segments {
			require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, seg.State,
				"segment %s should be earnings_recognized", seg.ID)
			require.NotNil(t, seg.BackingTransactionGroupID)
			require.NotNil(t, seg.SourceState)
			require.Equal(t, creditrealization.LineageSegmentStateRealCredit, *seg.SourceState)
		}
	}
}

func TestRecognizeEarnings_AccruedSourceIsolation(t *testing.T) {
	for _, tc := range []struct {
		name              string
		alreadyRecognized int64
	}{
		{name: "matching source only"},
		{name: "partially available source", alreadyRecognized: 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newRecognizerTestEnv(t)
			currency := currenciestestutils.NewFiatCurrency(t, env.Currency)
			costBasis := alpacadecimal.NewFromInt(1)
			spend, source := testID(), testID()

			// given: one allocation has lineage, and an unrelated source/spend
			// shares its accrued subaccount. Existing journal recognition may
			// leave less live accrued than the lineage's face amount.
			env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
				At: env.Now(), Amount: alpacadecimal.NewFromInt(40), Currency: env.CurrencyReference(), CostBasis: &costBasis,
				SourceChargeID: &source, SpendChargeID: &spend,
			})
			env.createLineageForRealization(t, spend, testID(), currency, alpacadecimal.NewFromInt(40), creditrealization.LineageOriginKindRealCredit)
			if tc.alreadyRecognized > 0 {
				env.resolveAndCommit(t, transactions.RecognizeEarningsFromAttributableAccruedTemplate{
					At: env.Now(), Amount: alpacadecimal.NewFromInt(tc.alreadyRecognized), Currency: env.CurrencyReference(),
				})
			}
			// This source sorts before the tracked one, so scalar recognition
			// would consume it even when the tracked source has enough balance.
			unrelatedSource, unrelatedSpend := "00000000000000000000000001", testID()
			env.resolveAndCommit(t, transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
				At: env.Now(), Amount: alpacadecimal.NewFromInt(10), Currency: env.CurrencyReference(), CostBasis: &costBasis,
				SourceChargeID: &unrelatedSource, SpendChargeID: &unrelatedSpend,
			})

			// when: recognition selects only the allocation's actual remaining source.
			result, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
				CustomerID: env.CustomerID, At: env.Now(), Currency: currency,
			})
			require.NoError(t, err)
			require.Equal(t, float64(40-tc.alreadyRecognized), result.RecognizedAmount.InexactFloat64())

			// then: all postings have the allocation's source/spend, unrelated
			// accrued remains deferred, and any partial segment retains its state.
			for _, entry := range env.TransactionGroupEntries(t, result.LedgerGroupID) {
				require.Equal(t, &source, entry.SourceChargeID)
				require.Equal(t, &spend, entry.SpendChargeID)
			}
			require.Equal(t, float64(10), env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())
			roots, err := env.legacylineage.LoadLineagesByCustomer(t.Context(), legacylineage.LoadLineagesByCustomerInput{
				Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.CurrencyReference(),
			})
			require.NoError(t, err)
			require.Len(t, roots, 1)
			amounts := make(map[creditrealization.LineageSegmentState]float64)
			for _, segment := range roots[0].Segments {
				amounts[segment.State] += segment.Amount.InexactFloat64()
				if segment.State == creditrealization.LineageSegmentStateEarningsRecognized {
					require.Equal(t, &result.LedgerGroupID, segment.BackingTransactionGroupID)
					require.NotNil(t, segment.SourceState)
					require.Equal(t, creditrealization.LineageSegmentStateRealCredit, *segment.SourceState)
				}
			}
			require.Equal(t, float64(40-tc.alreadyRecognized), amounts[creditrealization.LineageSegmentStateEarningsRecognized])
			require.Equal(t, float64(tc.alreadyRecognized), amounts[creditrealization.LineageSegmentStateRealCredit])
			retry, err := env.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{CustomerID: env.CustomerID, At: env.Now(), Currency: currency})
			require.NoError(t, err)
			require.Zero(t, retry.RecognizedAmount.InexactFloat64())
			require.Empty(t, retry.LedgerGroupID)
		})
	}
}
