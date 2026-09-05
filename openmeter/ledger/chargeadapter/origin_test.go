package chargeadapter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type originTestEnv struct {
	*creditPurchaseHandlerTestEnv
	collector  collector.Service
	recognizer recognizer.Service
	legacy     lineage.Service
}

func newOriginTestEnv(t *testing.T, custom bool) *originTestEnv {
	t.Helper()
	base := newCreditPurchaseHandlerTestEnv(t)
	if custom {
		base.currency = currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
		base.currency.Namespace = base.Namespace
	}
	deps := transactions.ResolverDependencies{AccountService: base.Deps.ResolversService, AccountCatalog: base.Deps.AccountService, BalanceQuerier: base.Deps.HistoricalLedger}
	collect, err := collector.NewService(collector.Config{
		Ledger: base.Deps.HistoricalLedger, Dependencies: deps,
		Breakage: base.breakage, AccountLocker: base.Deps.AccountService, TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)
	adapter, err := lineageadapter.New(lineageadapter.Config{Client: base.DB})
	require.NoError(t, err)
	legacy, err := lineageservice.New(lineageservice.Config{Adapter: adapter})
	require.NoError(t, err)
	rec, err := recognizer.NewService(recognizer.Config{
		Ledger: base.Deps.HistoricalLedger, Dependencies: deps,
		Lineage: legacy, TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)
	return &originTestEnv{creditPurchaseHandlerTestEnv: base, collector: collect, recognizer: rec, legacy: legacy}
}

func (e *originTestEnv) purchase(t *testing.T, amount int64, basis float64, expires bool) string {
	t.Helper()
	charge := e.newExternalCharge(alpacadecimal.NewFromInt(amount), alpacadecimal.NewFromFloat(basis))
	if e.currency.IsCustom() {
		charge = e.newExternalChargeCustomCurrency(t, e.currency, alpacadecimal.NewFromInt(amount), alpacadecimal.NewFromFloat(basis), "USD")
	}
	charge.ID = ulid.Make().String()
	period := timeutil.ClosedPeriod{From: e.Now().Add(-time.Hour), To: e.Now()}
	charge.Intent.ServicePeriod = period
	charge.Intent.FullServicePeriod = period
	charge.Intent.BillingPeriod = period
	charge.Intent.Priority = lo.ToPtr(7)
	if expires {
		charge.Intent.ExpiresAt = lo.ToPtr(e.Now().Add(24 * time.Hour))
	}
	ref, err := e.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	err = e.legacy.BackfillAdvanceLineageSegments(t.Context(), lineage.BackfillAdvanceLineageSegmentsInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, Currency: e.currency, Amount: alpacadecimal.NewFromInt(amount), BackingTransactionGroupID: ref.TransactionGroupID,
	})
	require.NoError(t, err)
	return charge.ID
}

func (e *originTestEnv) collect(t *testing.T, spend string, amount int64) creditrealization.Realizations {
	t.Helper()
	allocations, err := e.collector.CollectToAccrued(t.Context(), collector.CollectToAccruedInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, BookedAt: e.Now(), SourceBalanceAsOf: e.Now(),
		Currency: e.currency.Reference(), SettlementMode: productcatalog.CreditOnlySettlementMode,
		ServicePeriod: timeutil.ClosedPeriod{From: e.Now().Add(-time.Hour), To: e.Now()}, Amount: alpacadecimal.NewFromInt(amount),
	})
	require.NoError(t, err)
	var out creditrealization.Realizations
	for i, allocation := range allocations.AsCreateInputs() {
		allocation.ID = ulid.Make().String()
		require.Equal(t, true, allocation.Annotations[ledger.AnnotationOriginTracked])
		out = append(out, creditrealization.Realization{
			NamespacedModel: models.NamespacedModel{Namespace: e.Namespace},
			ManagedModel:    models.ManagedModel{CreatedAt: e.Now(), UpdatedAt: e.Now()}, CreateInput: allocation, SortHint: i,
		})
	}
	return out
}

func (e *originTestEnv) correct(t *testing.T, spend string, allocation creditrealization.Realization, amount int64) {
	t.Helper()
	_, err := e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(),
		Corrections: creditrealization.CorrectionRequest{{Allocation: allocation, Amount: alpacadecimal.NewFromInt(-amount)}},
	})
	require.NoError(t, err)
}

func (e *originTestEnv) recognize(t *testing.T) float64 {
	t.Helper()
	result, err := e.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{CustomerID: e.CustomerID, Currency: e.currency, At: e.Now()})
	require.NoError(t, err)
	return result.RecognizedAmount.InexactFloat64()
}

func (e *originTestEnv) provenanceBalance(t *testing.T, account ledger.Account, source, spend *string) float64 {
	t.Helper()
	buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: e.Namespace, Filters: ledger.Filters{
			AccountID: lo.ToPtr(account.ID().ID), SourceChargeID: mo.Some(source),
			SpendChargeID: mo.Some(spend), AsOf: lo.ToPtr(e.Now()), Route: ledger.RouteFilter{Currency: e.currency.Reference()},
		},
	})
	require.NoError(t, err)
	amount := alpacadecimal.Zero
	for _, bucket := range buckets {
		amount = amount.Add(bucket.SettledAmount)
	}
	return amount.InexactFloat64()
}

func TestOriginRecognitionCorrectionIsolatesRunsAndCostBases(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := "fiat"
		if custom {
			name = "custom"
		}
		t.Run(name, func(t *testing.T) {
			// given two runs of one charge and another spend, recognized together.
			e := newOriginTestEnv(t, custom)
			sourceA := e.purchase(t, 60, .5, false)
			spendA, spendB := ulid.Make().String(), ulid.Make().String()
			first := e.collect(t, spendA, 30)
			second := e.collect(t, spendA, 30)
			sourceB := e.purchase(t, 30, .8, false)
			third := e.collect(t, spendB, 30)
			require.Len(t, first, 1)
			require.Len(t, second, 1)
			require.Len(t, third, 1)
			require.Equal(t, float64(90), e.recognize(t))
			// when the first run is partially and repeatedly corrected.
			e.correct(t, spendA, first[0], 10)
			e.correct(t, spendA, first[0], 20)
			// then the second run and the other cost-basis/spend remain recognized.
			require.Equal(t, float64(30), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spendA))
			require.Equal(t, float64(30), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceB, &spendB))
			require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.AccruedAccount, &sourceA, &spendA))
			require.Zero(t, e.recognize(t))
			e.correct(t, spendA, second[0], 30)
			require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spendA))
			count, err := e.DB.CreditRealizationLineage.Query().Count(t.Context())
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestOriginAdvanceBackfillCorrectionReopensExactPurchasedSources(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := "fiat"
		if custom {
			name = "custom"
		}
		t.Run(name, func(t *testing.T) {
			// given one advance, two expiring partial backfills, and recognition.
			e := newOriginTestEnv(t, custom)
			spend := ulid.Make().String()
			allocations := e.collect(t, spend, 10)
			require.Len(t, allocations, 1)
			sourceA := e.purchase(t, 4, .5, true)
			sourceB := e.purchase(t, 3, .8, true)
			require.Equal(t, float64(7), e.recognize(t))
			// when corrections cross purchase boundaries and then exhaust advance.
			e.correct(t, spend, allocations[0], 5)
			require.Equal(t, float64(2), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spend)+e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceB, &spend))
			e.correct(t, spend, allocations[0], 5)
			// then all consumption/advance is gone; both purchases are reusable.
			require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spend))
			require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceB, &spend))
			require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.AccruedAccount, nil, &spend))
			require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.ReceivableAccount, nil, &spend))
			require.Equal(t, float64(4), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &sourceA, nil))
			require.Equal(t, float64(3), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &sourceB, nil))
			reused := e.collect(t, ulid.Make().String(), 7)
			require.NotEmpty(t, reused)
			require.Equal(t, float64(7), e.recognize(t))
		})
	}
}

func TestOriginCorrectionSplitsOneAllocationAcrossSameRouteSources(t *testing.T) {
	// given two purchases sharing a route, collapsed into one billing allocation.
	e := newOriginTestEnv(t, true)
	sourceA := e.purchase(t, 20, .5, false)
	sourceB := e.purchase(t, 20, .5, false)
	spend := ulid.Make().String()
	allocations := e.collect(t, spend, 40)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(40), e.recognize(t))
	// when repeated corrections cross the original source-entry boundary.
	e.correct(t, spend, allocations[0], 15)
	e.correct(t, spend, allocations[0], 15)
	// then only ten units of the original first source remain recognized.
	require.Equal(t, float64(10), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spend))
	require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceB, &spend))
	e.correct(t, spend, allocations[0], 10)
	require.Zero(t, e.recognize(t))
}

func TestOriginRecognitionAndCorrectionSerializeBalanceReads(t *testing.T) {
	// given unrecognized credit-backed usage.
	e := newOriginTestEnv(t, true)
	source := e.purchase(t, 30, .5, false)
	spend := ulid.Make().String()
	allocations := e.collect(t, spend, 30)
	// when recognition and correction start concurrently.
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := e.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{CustomerID: e.CustomerID, Currency: e.currency, At: e.Now()})
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
			Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(),
			Corrections: creditrealization.CorrectionRequest{{Allocation: allocations[0], Amount: alpacadecimal.NewFromInt(-10)}},
		})
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	// then either valid ordering produces the same booked state.
	require.Equal(t, float64(20), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &source, &spend))
	require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.AccruedAccount, &source, &spend))
	require.Zero(t, e.recognize(t))
}

func TestOriginAndLegacyHistoriesSharePurchasesWithoutSharingCorrectionState(t *testing.T) {
	// given an actual pre-cutover advance with lineage and a new advance for the same customer.
	e := newOriginTestEnv(t, true)
	legacySpend, spend := ulid.Make().String(), ulid.Make().String()
	amount := alpacadecimal.NewFromInt(20)
	deps := transactions.ResolverDependencies{AccountService: e.Deps.ResolversService, AccountCatalog: e.Deps.AccountService, BalanceQuerier: e.Deps.HistoricalLedger}
	inputs, err := transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace},
		transactions.IssueCustomerReceivableTemplate{At: e.Now(), Amount: amount, Currency: e.currency.Reference(), SpendChargeID: &legacySpend},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{At: e.Now(), Amount: amount, Currency: e.currency.Reference(), SpendChargeID: &legacySpend})
	require.NoError(t, err)
	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
	legacyAllocation := creditrealization.Realization{
		NamespacedModel: models.NamespacedModel{Namespace: e.Namespace}, ManagedModel: models.ManagedModel{CreatedAt: e.Now(), UpdatedAt: e.Now()},
		CreateInput: creditrealization.CreateInput{
			ID: ulid.Make().String(), Annotations: creditrealization.LineageAnnotations(creditrealization.LineageOriginKindAdvance),
			ServicePeriod: timeutil.ClosedPeriod{From: e.Now().Add(-time.Hour), To: e.Now()}, LedgerTransaction: ledgertransaction.GroupReference{TransactionGroupID: group.ID().ID}, Amount: amount, Type: creditrealization.TypeAllocation,
		},
	}
	_, err = e.DB.Charge.Create().SetNamespace(e.Namespace).SetID(legacySpend).SetType(meta.ChargeTypeUsageBased).Save(t.Context())
	require.NoError(t, err)
	require.NoError(t, e.legacy.CreateInitialLineages(t.Context(), lineage.CreateInitialLineagesInput{Namespace: e.Namespace, ChargeID: legacySpend, CustomerID: e.CustomerID.ID, Currency: e.currency, Realizations: creditrealization.Realizations{legacyAllocation}}))
	allocated := e.collect(t, spend, 20)
	// when one purchase and one recognition process both pools.
	source := e.purchase(t, 40, .5, false)
	require.Equal(t, float64(40), e.recognize(t))
	segments, err := e.legacy.LoadActiveSegmentsByRealizationID(t.Context(), e.Namespace, []string{legacyAllocation.ID})
	require.NoError(t, err)
	require.Len(t, segments[legacyAllocation.ID], 1)
	require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, segments[legacyAllocation.ID][0].State)
	require.Equal(t, float64(20), segments[legacyAllocation.ID][0].Amount.InexactFloat64())
	// then new correction leaves exactly the legacy recognized amount and its segment untouched.
	e.correct(t, spend, allocated[0], 20)
	require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &source, &spend))
	require.Equal(t, float64(20), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &source, &legacySpend))
	// The compatibility reader can still unwind its pre-cutover allocation.
	_, err = e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: legacySpend, AllocateAt: e.Now(),
		Corrections: creditrealization.CorrectionRequest{{Allocation: legacyAllocation, Amount: amount.Neg()}}, LineageSegmentsByRealization: segments,
	})
	require.NoError(t, err)
	require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &source, &legacySpend))
	require.Equal(t, float64(40), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &source, nil))
}

func TestOriginCorrectionRollbackAndOvercorrectionLeaveJournalUnchanged(t *testing.T) {
	// given recognized consumption backed by an expiring purchase.
	e := newOriginTestEnv(t, true)
	spend := ulid.Make().String()
	allocated := e.collect(t, spend, 20)
	source := e.purchase(t, 20, .5, true)
	require.Equal(t, float64(20), e.recognize(t))
	count, err := e.DB.LedgerEntry.Query().Count(t.Context())
	require.NoError(t, err)
	input := collector.CorrectCollectedAccruedInput{Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(), Corrections: creditrealization.CorrectionRequest{{Allocation: allocated[0], Amount: alpacadecimal.NewFromInt(-10)}}}
	// when the owning billing transaction fails after ledger and breakage writes.
	failed := errors.New("billing persistence failed")
	err = transaction.RunWithNoValue(t.Context(), enttx.NewCreator(e.DB), func(ctx context.Context) error {
		if _, err := e.collector.CorrectCollectedAccrued(ctx, input); err != nil {
			return err
		}
		return failed
	})
	require.ErrorIs(t, err, failed)
	// then the same correction can be retried; no journal or release state escaped rollback.
	after, err := e.DB.LedgerEntry.Query().Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, count, after)
	require.Equal(t, float64(20), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &source, &spend))
	// Different realization IDs still cannot claim the same original source twice.
	duplicate := input.Corrections[0]
	duplicate.Allocation.ID = ulid.Make().String()
	input.Corrections = append(input.Corrections, duplicate)
	_, err = e.collector.CorrectCollectedAccrued(t.Context(), input)
	require.ErrorContains(t, err, "repeat an original collection source")
	input.Corrections = input.Corrections[:1]
	e.correct(t, spend, allocated[0], 10)
	input.Corrections[0].Amount = alpacadecimal.NewFromInt(-11)
	count, err = e.DB.LedgerEntry.Query().Count(t.Context())
	require.NoError(t, err)
	_, err = e.collector.CorrectCollectedAccrued(t.Context(), input)
	require.ErrorContains(t, err, "exceeds remaining origin balance")
	after, err = e.DB.LedgerEntry.Query().Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, count, after)
	e.correct(t, spend, allocated[0], 10)
	require.Equal(t, float64(20), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &source, nil))
}

func TestOriginBackfillAndCorrectionSerializeBalanceReads(t *testing.T) {
	// given uncovered advance and a pending purchase.
	e := newOriginTestEnv(t, true)
	spend := ulid.Make().String()
	allocated := e.collect(t, spend, 30)
	charge := e.newExternalChargeCustomCurrency(t, e.currency, alpacadecimal.NewFromInt(30), alpacadecimal.NewFromFloat(.5), "USD")
	charge.ID = ulid.Make().String()
	charge.Intent.Priority = lo.ToPtr(7)
	charge.Intent.ServicePeriod = timeutil.ClosedPeriod{From: e.Now().Add(-time.Hour), To: e.Now()}
	charge.Intent.FullServicePeriod = charge.Intent.ServicePeriod
	charge.Intent.BillingPeriod = charge.Intent.ServicePeriod
	// when purchase backfill and correction race.
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := e.handler.OnCreditPurchaseInitiated(t.Context(), charge)
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
			Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(), Corrections: creditrealization.CorrectionRequest{{Allocation: allocated[0], Amount: alpacadecimal.NewFromInt(-10)}},
		})
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	// then both valid orderings leave 20 consumed and 10 reusable, with no uncovered advance.
	require.Equal(t, float64(20), e.recognize(t))
	require.Equal(t, float64(10), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &charge.ID, nil))
	require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.ReceivableAccount, nil, &spend))
	require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.AccruedAccount, nil, &spend))
}

func TestOriginRecognitionRespectsBookingTimeAndManagedCurrencyIdentity(t *testing.T) {
	// given two managed currencies reusing a code, each with a separate collection.
	e := newOriginTestEnv(t, true)
	firstCurrency := e.currency
	sourceA := e.purchase(t, 20, .5, false)
	spend := ulid.Make().String()
	first := e.collect(t, spend, 20)
	e.currency = currenciestestutils.NewCustomCurrency(t, firstCurrency.GetCode(), 2)
	e.currency.Namespace = e.Namespace
	sourceB := e.purchase(t, 30, .8, false)
	second := e.collect(t, spend, 30)
	require.NotEqual(t, firstCurrency.ID, e.currency.ID)
	// when recognition is queried before booking, neither future balance is eligible.
	result, err := e.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{CustomerID: e.CustomerID, Currency: e.currency, At: e.Now().Add(-time.Minute)})
	require.NoError(t, err)
	require.Zero(t, result.RecognizedAmount.InexactFloat64())
	// then recognition and correction stay scoped by managed identity, even for the same spend.
	require.Equal(t, float64(30), e.recognize(t))
	e.correct(t, spend, second[0], 30)
	require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceB, &spend))
	e.currency = firstCurrency
	require.Equal(t, float64(20), e.recognize(t))
	require.Equal(t, float64(20), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &sourceA, &spend))
	group, err := e.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: e.Namespace, ID: first[0].LedgerTransaction.TransactionGroupID})
	require.NoError(t, err)
	origin := group.Transactions()[0].Entries()[0].OriginID()
	require.NotNil(t, origin)
	// Indexed traversal returns complete origin pairs across pages without sibling origins.
	query := ledger.ListTransactionsInput{Namespace: e.Namespace, OriginID: origin, Limit: 1}
	var ids []string
	for {
		page, err := e.Deps.HistoricalLedger.ListTransactions(t.Context(), query)
		require.NoError(t, err)
		for _, tx := range page.Items {
			ids = append(ids, tx.ID().ID)
			require.NotEmpty(t, tx.GroupID().ID)
			for _, entry := range tx.Entries() {
				require.Equal(t, origin, entry.OriginID())
			}
		}
		if page.NextCursor == nil {
			break
		}
		query.Cursor = page.NextCursor
	}
	require.Len(t, ids, 2)
	buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{Namespace: e.Namespace, Filters: ledger.Filters{AccountID: lo.ToPtr(e.BusinessAccounts.EarningsAccount.ID().ID), OriginID: mo.Some(origin)}, GroupBy: []string{ledger.BalanceBucketGroupByOriginID}})
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	require.Equal(t, origin, buckets[0].GroupByValues[ledger.BalanceBucketGroupByOriginID])
	require.Equal(t, float64(20), buckets[0].SettledAmount.InexactFloat64())
}
