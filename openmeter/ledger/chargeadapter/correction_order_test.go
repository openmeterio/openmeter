package chargeadapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (e *originTestEnv) collectLegacyAdvance(t *testing.T, spend string, amount int64) creditrealization.Realization {
	t.Helper()
	deps := transactions.ResolverDependencies{AccountService: e.Deps.ResolversService, AccountCatalog: e.Deps.AccountService, BalanceQuerier: e.Deps.HistoricalLedger}
	value := alpacadecimal.NewFromInt(amount)
	inputs, err := transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace},
		transactions.IssueCustomerReceivableTemplate{At: e.Now(), Amount: value, Currency: e.currency.Reference(), SpendChargeID: &spend},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{At: e.Now(), Amount: value, Currency: e.currency.Reference(), SpendChargeID: &spend})
	require.NoError(t, err)
	return e.recordLegacyCollection(t, spend, amount, creditrealization.LineageOriginKindAdvance, inputs)
}

func (e *originTestEnv) recordLegacyCollection(t *testing.T, spend string, amount int64, kind creditrealization.LineageOriginKind, inputs []ledger.TransactionInput) creditrealization.Realization {
	t.Helper()
	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
	allocation := creditrealization.Realization{
		NamespacedModel: models.NamespacedModel{Namespace: e.Namespace}, ManagedModel: models.ManagedModel{CreatedAt: e.Now(), UpdatedAt: e.Now()},
		CreateInput: creditrealization.CreateInput{
			ID: ulid.Make().String(), Annotations: creditrealization.LineageAnnotations(kind),
			ServicePeriod: timeutil.ClosedPeriod{From: e.Now().Add(-time.Hour), To: e.Now()}, LedgerTransaction: ledgertransaction.GroupReference{TransactionGroupID: group.ID().ID}, Amount: alpacadecimal.NewFromInt(amount), Type: creditrealization.TypeAllocation,
		},
	}
	_, err = e.DB.Charge.Create().SetNamespace(e.Namespace).SetID(spend).SetType(meta.ChargeTypeUsageBased).Save(t.Context())
	require.NoError(t, err)
	require.NoError(t, e.legacy.CreateInitialLineages(t.Context(), legacylineage.CreateInitialLineagesInput{Namespace: e.Namespace, ChargeID: spend, CustomerID: e.CustomerID.ID, Currency: e.currency, Realizations: creditrealization.Realizations{allocation}}))
	// This handler fixture has no billing allocation row; retain its real ledger reference.
	e.originalAdvanceGroups[allocation.ID] = group.ID().ID
	return allocation
}

func (e *originTestEnv) correctWithLegacyPersistence(t *testing.T, spend string, allocation creditrealization.Realization, amount int64) {
	t.Helper()
	require.NoError(t, transaction.RunWithNoValue(t.Context(), enttx.NewCreator(e.DB), func(ctx context.Context) error {
		segments, err := e.legacy.LoadActiveSegmentsByRealizationID(ctx, e.Namespace, []string{allocation.ID})
		if err != nil {
			return err
		}
		corrections, err := e.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
			Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(),
			Corrections: creditrealization.CorrectionRequest{{Allocation: allocation, Amount: alpacadecimal.NewFromInt(-amount)}}, LineageSegmentsByRealization: segments,
		})
		if err != nil {
			return err
		}
		inputs, err := corrections.AsCreateInputs(creditrealization.Realizations{allocation})
		if err != nil {
			return err
		}
		var realized creditrealization.Realizations
		for _, input := range inputs {
			realized = append(realized, creditrealization.Realization{CreateInput: input})
		}
		return e.legacy.PersistCorrectionLineageSegments(ctx, legacylineage.PersistCorrectionLineageSegmentsInput{Namespace: e.Namespace, Realizations: realized})
	}))
}

func TestCorrectionSelectsNewestBackingBeforeRecognitionState(t *testing.T) {
	for _, storage := range []string{"provenance", "legacy"} {
		t.Run(storage, func(t *testing.T) {
			for _, recognition := range []string{"together", "between purchases", "only older backing"} {
				t.Run(recognition, func(t *testing.T) {
					// Given an advance of 10 backed by A5 then B5, with different recognition batching.
					e := newOriginTestEnv(t, false)
					spend := ulid.Make().String()
					var allocation creditrealization.Realization
					if storage == "legacy" {
						allocation = e.collectLegacyAdvance(t, spend, 10)
					} else {
						allocation = e.collect(t, spend, 10)[0]
					}
					a := e.purchase(t, 5, 1, false)
					if recognition != "together" {
						require.Equal(t, float64(5), e.recognize(t))
					}
					b := e.purchase(t, 5, 2, false)
					if recognition == "together" {
						require.Equal(t, float64(10), e.recognize(t))
					}
					if recognition == "between purchases" {
						require.Equal(t, float64(5), e.recognize(t))
					}
					// When correcting 2, then another 4, B must be exhausted before touching A.
					e.correctWithLegacyPersistence(t, spend, allocation, 2)
					require.Zero(t, e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &a, nil))
					require.Equal(t, float64(2), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &b, nil))
					e.correctWithLegacyPersistence(t, spend, allocation, 4)
					require.Equal(t, float64(1), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &a, nil))
					require.Equal(t, float64(5), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &b, nil))
					require.Equal(t, float64(4), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &a, &spend))
					require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &b, &spend))
					// Then the legacy writer must retain precisely A4 too, so another correction works.
					e.correctWithLegacyPersistence(t, spend, allocation, 4)
					require.Equal(t, float64(5), e.provenanceBalance(t, e.CustomerAccounts.FBOAccount, &a, nil))
					require.Zero(t, e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &a, &spend))
				})
			}
		})
	}
}

func TestRecollectionMintsNewOriginAndCorrectionKeepsOlderRemainder(t *testing.T) {
	// Given a collection of 10, corrected by 4, then a new collection of those 4.
	e := newOriginTestEnv(t, false)
	start := e.Now()
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	source := e.purchase(t, 10, 1, false)
	spend := ulid.Make().String()
	first := e.collect(t, spend, 10)
	corrections, err := e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(),
		Corrections: creditrealization.CorrectionRequest{{Allocation: first[0], Amount: alpacadecimal.NewFromInt(-4)}},
	})
	require.NoError(t, err)
	realized, err := corrections.AsCreateInputs(first)
	require.NoError(t, err)
	all := append(creditrealization.Realizations{}, first...)
	for _, r := range realized {
		r.ID = ulid.Make().String()
		all = append(all, creditrealization.Realization{CreateInput: r})
	}
	clock.FreezeTime(start.Add(time.Hour))
	second := e.collect(t, spend, 4)
	all = append(all, second...)
	firstGroup, err := e.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: e.Namespace, ID: first[0].LedgerTransaction.TransactionGroupID})
	require.NoError(t, err)
	secondGroup, err := e.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: e.Namespace, ID: second[0].LedgerTransaction.TransactionGroupID})
	require.NoError(t, err)
	oldOrigin := firstGroup.Transactions()[0].Entries()[0].CollectionOriginID()
	newOrigin := secondGroup.Transactions()[0].Entries()[0].CollectionOriginID()
	require.NotNil(t, oldOrigin)
	require.NotNil(t, newOrigin)
	require.NotEqual(t, *oldOrigin, *newOrigin)
	// When correcting 5, billing selects the new 4 before 1 from the old remainder.
	request, err := all.CreateCorrectionRequest(alpacadecimal.NewFromInt(-5), e.currency)
	require.NoError(t, err)
	require.Len(t, request, 2)
	require.Equal(t, second[0].ID, request[0].Allocation.ID)
	require.Equal(t, float64(-4), request[0].Amount.InexactFloat64())
	_, err = e.collector.CorrectCollectedAccrued(t.Context(), collector.CorrectCollectedAccruedInput{
		Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(), Corrections: request,
	})
	require.NoError(t, err)
	// Then O1 retains 5 and O2 is empty; no generation counting or history replay is needed.
	for id, expected := range map[string]float64{*oldOrigin: 5, *newOrigin: 0} {
		buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
			Namespace: e.Namespace,
			Filters:   ledger.Filters{AccountID: lo.ToPtr(e.CustomerAccounts.AccruedAccount.ID().ID), CollectionOriginID: mo.Some(&id)},
		})
		require.NoError(t, err)
		total := alpacadecimal.Zero
		for _, bucket := range buckets {
			total = total.Add(bucket.SettledAmount)
		}
		require.Equal(t, expected, total.InexactFloat64())
	}
	require.Equal(t, float64(5), e.availableSourceCredit(t, source))
}

func TestLegacyCollapsedSourcesSelectNewerUnrecognizedCredit(t *testing.T) {
	// Given one legacy collection containing A5 then B5 on the same FBO route.
	e := newOriginTestEnv(t, false)
	a := e.purchase(t, 5, 1, false)
	b := e.purchase(t, 5, 1, false)
	spend := ulid.Make().String()
	var sources []transactions.PostingAmount
	for order, id := range []string{a, b} {
		buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
			Namespace: e.Namespace,
			Filters:   ledger.Filters{AccountID: lo.ToPtr(e.CustomerAccounts.FBOAccount.ID().ID), SourceChargeID: mo.Some(&id)},
		})
		require.NoError(t, err)
		require.Len(t, buckets, 1)
		sources = append(sources, transactions.PostingAmount{
			Address: buckets[0].Address, Amount: alpacadecimal.NewFromInt(5),
			Identity: ledger.EntryIdentityParts{SourceChargeID: lo.ToPtr(id), SpendChargeID: &spend}, Annotations: models.Annotations{ledger.AnnotationCollectionSourceOrder: order},
		})
	}
	deps := transactions.ResolverDependencies{AccountService: e.Deps.ResolversService, AccountCatalog: e.Deps.AccountService, BalanceQuerier: e.Deps.HistoricalLedger}
	inputs, err := transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace}, transactions.TransferCustomerFBOToAccruedTemplate{At: e.Now(), Currency: e.currency.Reference(), Sources: sources})
	require.NoError(t, err)
	allocation := e.recordLegacyCollection(t, spend, 10, creditrealization.LineageOriginKindRealCredit, inputs)
	// Only A is recognized; record the same transition through the legacy service.
	buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{Namespace: e.Namespace, Filters: ledger.Filters{AccountID: lo.ToPtr(e.CustomerAccounts.AccruedAccount.ID().ID), SourceChargeID: mo.Some(&a)}})
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	inputs, err = transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace}, transactions.RecognizeEarningsFromAttributableAccruedTemplate{At: e.Now(), Currency: e.currency.Reference(), Amount: alpacadecimal.NewFromInt(5), Sources: []transactions.PostingAmount{{Address: buckets[0].Address, Amount: alpacadecimal.NewFromInt(5), Identity: ledger.EntryIdentityParts{SourceChargeID: &a, SpendChargeID: &spend}}}})
	require.NoError(t, err)
	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
	segments, err := e.legacy.LoadActiveSegmentsByRealizationID(t.Context(), e.Namespace, []string{allocation.ID})
	require.NoError(t, err)
	require.Len(t, segments[allocation.ID], 1)
	original := segments[allocation.ID][0]
	require.NoError(t, e.legacy.CloseSegment(t.Context(), original.ID, e.Now()))
	require.NoError(t, e.legacy.CreateSegment(t.Context(), legacylineage.CreateSegmentInput{LineageID: original.LineageID, State: creditrealization.LineageSegmentStateRealCredit, Amount: alpacadecimal.NewFromInt(5)}))
	require.NoError(t, e.legacy.CreateSegment(t.Context(), legacylineage.CreateSegmentInput{LineageID: original.LineageID, State: creditrealization.LineageSegmentStateEarningsRecognized, SourceState: lo.ToPtr(creditrealization.LineageSegmentStateRealCredit), BackingTransactionGroupID: lo.ToPtr(group.ID().ID), Amount: alpacadecimal.NewFromInt(5)}))
	// When correcting 2 and then 4, B is returned first and only 1 of A's recognition is undone.
	e.correctWithLegacyPersistence(t, spend, allocation, 2)
	require.Equal(t, float64(2), e.availableSourceCredit(t, b))
	require.Equal(t, float64(5), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &a, &spend))
	e.correctWithLegacyPersistence(t, spend, allocation, 4)
	require.Equal(t, float64(5), e.availableSourceCredit(t, b))
	require.Equal(t, float64(1), e.availableSourceCredit(t, a))
	require.Equal(t, float64(4), e.provenanceBalance(t, e.BusinessAccounts.EarningsAccount, &a, &spend))
}

// FBO consumption keeps spend provenance; availability sums all spends for the funding source.
func (e *originTestEnv) availableSourceCredit(t *testing.T, source string) float64 {
	t.Helper()
	buckets, err := e.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: e.Namespace,
		Filters:   ledger.Filters{AccountID: lo.ToPtr(e.CustomerAccounts.FBOAccount.ID().ID), SourceChargeID: mo.Some(&source), AsOf: lo.ToPtr(e.Now()), Route: ledger.RouteFilter{Currency: e.currency.Reference()}},
	})
	require.NoError(t, err)
	total := alpacadecimal.Zero
	for _, bucket := range buckets {
		total = total.Add(bucket.SettledAmount)
	}
	return total.InexactFloat64()
}

func TestStaleLegacyCorrectionSelectionRollsBackLedger(t *testing.T) {
	// Given legacy backing partially corrected after a caller loaded its segments.
	e := newOriginTestEnv(t, false)
	spend := ulid.Make().String()
	allocation := e.collectLegacyAdvance(t, spend, 10)
	source := e.purchase(t, 10, 1, false)
	stale, err := e.legacy.LoadActiveSegmentsByRealizationID(t.Context(), e.Namespace, []string{allocation.ID})
	require.NoError(t, err)
	e.correctWithLegacyPersistence(t, spend, allocation, 2)
	before, err := e.DB.LedgerTransactionGroup.Query().Count(t.Context())
	require.NoError(t, err)
	// When that stale selection is posted and handed to compatibility persistence in one transaction.
	err = transaction.RunWithNoValue(t.Context(), enttx.NewCreator(e.DB), func(ctx context.Context) error {
		corrections, err := e.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
			Namespace: e.Namespace, CustomerID: e.CustomerID.ID, ChargeID: spend, AllocateAt: e.Now(),
			Corrections: creditrealization.CorrectionRequest{{Allocation: allocation, Amount: alpacadecimal.NewFromInt(-2)}}, LineageSegmentsByRealization: stale,
		})
		if err != nil {
			return err
		}
		inputs, err := corrections.AsCreateInputs(creditrealization.Realizations{allocation})
		if err != nil {
			return err
		}
		var realized creditrealization.Realizations
		for _, input := range inputs {
			realized = append(realized, creditrealization.Realization{CreateInput: input})
		}
		return e.legacy.PersistCorrectionLineageSegments(ctx, legacylineage.PersistCorrectionLineageSegmentsInput{Namespace: e.Namespace, Realizations: realized})
	})
	// Then rejecting the stale segment IDs also rolls back the ledger postings.
	require.ErrorContains(t, err, "stale correction segment selection")
	after, err := e.DB.LedgerTransactionGroup.Query().Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, float64(2), e.availableSourceCredit(t, source))
}
