package customerbalance

import (
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestListCreditTransactionsCorrections(t *testing.T) {
	for _, tt := range []struct {
		name     string
		funded   int64
		backfill int64
	}{
		{name: "purchased credit", funded: 100},
		{name: "uncovered advance"},
		{name: "partially backfilled advance", backfill: 4},
		{name: "fully backfilled advance", backfill: 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			start := clock.Now()
			clock.FreezeTime(start)
			defer clock.UnFreeze()

			// given: a collected fee, optionally funded before or after collection.
			if tt.funded > 0 {
				env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(tt.funded), env.Currency, nil)
			}
			bookedAt := start.Add(time.Hour)
			period := timeutil.ClosedPeriod{From: bookedAt, To: bookedAt.Add(24 * time.Hour)}
			created, err := env.flatFeeService.Create(t.Context(), flatfee.CreateInput{
				Namespace: env.Namespace,
				Intents: []flatfee.Intent{{
					Intent: meta.Intent{
						ManagedBy:  billing.SystemManagedLine,
						CustomerID: env.CustomerID.ID,
						Currency:   currenciestestutils.NewFiatCurrency(t, env.Currency),
						TaxConfig:  productcatalog.TaxCodeConfig{TaxCodeID: env.taxCodeID},
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name: "Platform Fee", ServicePeriod: period, FullServicePeriod: period, BillingPeriod: period,
						},
						InvoiceAt:             bookedAt,
						PaymentTerm:           productcatalog.InAdvancePaymentTerm,
						AmountBeforeProration: alpacadecimal.NewFromInt(9),
						ProRating:             productcatalog.ProRatingConfig{Enabled: true, Mode: productcatalog.ProRatingModeProratePrices},
					},
					SettlementMode: productcatalog.CreditOnlySettlementMode,
				}},
			})
			require.NoError(t, err)
			require.Len(t, created, 1)
			charge := created[0].Charge
			clock.FreezeTime(bookedAt.Add(time.Second))
			charge = env.advanceFlatFeeCharge(t, charge)
			if tt.backfill > 0 {
				clock.FreezeTime(bookedAt.Add(time.Hour))
				env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(tt.backfill), env.Currency, nil)
			}

			// when: a period edit refunds half the fee, then deletion refunds the rest.
			firstCorrectionAt := bookedAt.Add(2 * time.Hour)
			clock.FreezeTime(firstCorrectionAt)
			shrink, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
				ChangeSource:           billing.ChangeSourceSystem,
				NewServicePeriodTo:     bookedAt.Add(12 * time.Hour),
				NewFullServicePeriodTo: period.To,
				NewBillingPeriodTo:     period.To,
				NewInvoiceAt:           charge.Intent.GetEffectiveInvoiceAt(),
			})
			require.NoError(t, err)
			_, err = env.flatFeeService.TriggerPatch(t.Context(), charge.GetChargeID(), shrink)
			require.NoError(t, err)
			clock.FreezeTime(bookedAt.Add(3 * time.Hour))
			deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem, Policy: meta.RefundAsCreditsDeletePolicy,
			})
			require.NoError(t, err)
			_, err = env.flatFeeService.TriggerPatch(t.Context(), charge.GetChargeID(), deletePatch)
			require.NoError(t, err)

			// then: two correction activities offset the original usage exactly,
			// retaining booking time, recording time, charge metadata, and page balances.
			input := ListCreditTransactionsInput{CustomerID: env.CustomerID, Limit: 20, Currency: &env.Currency}
			all, err := env.Service.ListCreditTransactions(t.Context(), input)
			require.NoError(t, err)
			var corrections []CreditTransaction
			var consumed []CreditTransaction
			total := alpacadecimal.Zero
			for _, item := range all.Items {
				total = total.Add(item.Amount)
				switch item.Type {
				case CreditTransactionTypeCorrection:
					corrections = append(corrections, item)
					require.Equal(t, float64(4.5), item.Amount.InexactFloat64())
					require.Equal(t, bookedAt, item.BookedAt.UTC())
					require.False(t, item.CreatedAt.Before(firstCorrectionAt))
					require.Equal(t, charge.ID, item.Annotations[ledger.AnnotationChargeID])
					require.Equal(t, "Platform Fee", item.Name)
					require.Equal(t, float64(4.5), item.Balance.After.Sub(item.Balance.Before).InexactFloat64())
				case CreditTransactionTypeConsumed:
					consumed = append(consumed, item)
				}
			}
			require.Len(t, corrections, 2)
			require.Len(t, consumed, 1, "internal advance reversals must not be displayed as new usage")
			require.Equal(t, float64(-9), consumed[0].Amount.InexactFloat64())
			require.Equal(t, float64(tt.funded+tt.backfill), total.InexactFloat64())
			require.Equal(t, total.InexactFloat64(), all.Items[0].Balance.After.InexactFloat64())

			for _, txType := range []*CreditTransactionType{nil, lo.ToPtr(CreditTransactionTypeCorrection), lo.ToPtr(CreditTransactionTypeConsumed)} {
				input.Type = txType
				expected := lo.Filter(all.Items, func(item CreditTransaction, _ int) bool { return txType == nil || item.Type == *txType })
				input.Limit = 1
				forward := collectCreditTransactionsForward(t, env.Service, input, len(expected))
				require.Len(t, forward.items, len(expected))
				backward := collectCreditTransactionsBackward(t, env.Service, input, forward.lastPage, len(expected))
				slices.Reverse(backward)
				require.Len(t, backward, len(expected))
				for idx, want := range expected {
					requireCreditTransactionBalanceMatches(t, want, forward.items[idx])
					requireCreditTransactionBalanceMatches(t, want, backward[idx])
				}
			}
			input.Type = lo.ToPtr(CreditTransactionTypeCorrection)
			input.AsOf = lo.ToPtr(bookedAt.Add(-time.Second))
			before, err := env.Service.ListCreditTransactions(t.Context(), input)
			require.NoError(t, err)
			require.Empty(t, before.Items)
		})
	}
}

func TestCorrectionCreditTransactionsProjection(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	customA := currenciestestutils.NewManagedCurrency(t, "ns", "currency-a", "TOKENS").Reference()
	customB := currenciestestutils.NewManagedCurrency(t, "ns", "currency-b", "TOKENS").Reference()
	var txs []*ledgerhistorical.Transaction
	// given: one batch contains different currencies, effective times, feature
	// scopes, and internal movements that must not leak into customer activity.
	for _, movement := range []struct {
		id          string
		currency    currencies.CurrencyReference
		amount      int64
		features    []string
		at          time.Time
		annotations models.Annotations
	}{
		{id: "a", currency: customA, amount: 9, at: now},
		{id: "b", currency: customB, amount: 7, at: now},
		{id: "c", currency: customA, amount: -3, at: now},
		{id: "restricted", currency: customA, amount: 4, at: now, features: []string{"feature-a"}},
		{id: "past", currency: customA, amount: 2, at: now.Add(-time.Hour)},
		{id: "zero-positive", currency: currencies.NewCurrencyReference("EUR"), amount: 5, at: now},
		{id: "zero-negative", currency: currencies.NewCurrencyReference("EUR"), amount: -5, at: now},
		{id: "future", currency: customA, amount: 100, at: now.Add(time.Hour)},
		{id: "internal", currency: customA, amount: 100, at: now, annotations: models.Annotations{ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal}},
		{id: "breakage", currency: customA, amount: -100, at: now, annotations: models.Annotations{ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage}},
		{id: "void", currency: customA, amount: -100, at: now, annotations: models.Annotations{creditvoid.AnnotationCreditVoidRecordID: "void-id"}},
		{id: "forward", currency: customA, amount: 100, at: now, annotations: models.Annotations{ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionForward)}},
	} {
		annotations := models.Annotations{ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection), ledger.AnnotationChargeID: "charge-id"}
		for key, value := range movement.annotations {
			annotations[key] = value
		}
		route := ledger.Route{Currency: movement.currency, Features: movement.features}
		key, err := ledger.BuildRoutingKey(route)
		require.NoError(t, err)
		tx, err := ledgerhistorical.NewTransactionFromData(ledgerhistorical.TransactionData{
			Namespace: "ns", GroupID: "group-id", ID: movement.id,
			CreatedAt: now.Add(time.Minute), BookedAt: movement.at, Annotations: annotations,
		}, []ledgerhistorical.EntryData{{
			Namespace: "ns", ID: movement.id + "-entry", TransactionID: movement.id,
			SubAccountID: "subaccount", AccountType: ledger.AccountTypeCustomerFBO,
			Route: route, RouteID: movement.id + "-route", RouteKey: key.Value(), RouteKeyVer: key.Version(),
			Amount: alpacadecimal.NewFromInt(movement.amount),
		}})
		require.NoError(t, err)
		txs = append(txs, tx)
	}
	group := ledgerhistorical.NewTransactionGroupFromData(ledgerhistorical.TransactionGroupData{Namespace: "ns", ID: "group-id"}, txs)

	for _, tt := range []struct {
		name    string
		input   creditTransactionLoaderInput
		amounts map[string]float64
	}{
		{name: "all", input: creditTransactionLoaderInput{AsOf: now}, amounts: map[string]float64{"restricted": 10, "b": 7, "past": 2}},
		{name: "unrestricted", input: creditTransactionLoaderInput{AsOf: now, FeatureFilter: NewUnrestrictedFeatureFilter()}, amounts: map[string]float64{"c": 6, "b": 7, "past": 2}},
		{name: "matching feature", input: creditTransactionLoaderInput{AsOf: now, FeatureFilter: NewFeatureFilter([]string{"feature-a"})}, amounts: map[string]float64{"restricted": 10, "b": 7, "past": 2}},
		{name: "other feature", input: creditTransactionLoaderInput{AsOf: now, FeatureFilter: NewFeatureFilter([]string{"feature-b"})}, amounts: map[string]float64{"c": 6, "b": 7, "past": 2}},
		{name: "other currency", input: creditTransactionLoaderInput{AsOf: now, Currency: lo.ToPtr(currencies.NewCurrencyReference("USD").Code)}, amounts: map[string]float64{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// when: the correction is projected into the requested balance scope.
			rows := correctionCreditTransactions(group, tt.input)
			// then: currency identities and booked times stay separate, while each
			// same-scope batch has its net amount and last contributing cursor.
			require.Len(t, rows, len(tt.amounts))
			for _, row := range rows {
				amount, found := tt.amounts[row.ID.ID]
				require.True(t, found)
				require.Equal(t, amount, row.Amount.InexactFloat64())
				require.Equal(t, CreditTransactionTypeCorrection, row.Type)
				require.Equal(t, now.Add(time.Minute), row.CreatedAt)
				require.Equal(t, "charge-id", row.Annotations[ledger.AnnotationChargeID])
				require.NotNil(t, row.CustomCurrencyID)
				if row.ID.ID == "b" {
					require.Equal(t, "currency-b", *row.CustomCurrencyID)
				} else {
					require.Equal(t, "currency-a", *row.CustomCurrencyID)
				}
			}
		})
	}
}

func TestListCreditTransactionsLegacyCorrection(t *testing.T) {
	env := newTestEnv(t)
	start := clock.Now()
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given: an originless collection made through the legacy transaction template.
	funding := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(10), env.Currency, nil)
	fundedGroup, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{
		Namespace: env.Namespace, ID: funding.Realizations.CreditGrantRealization.TransactionGroupID,
	})
	require.NoError(t, err)
	var source ledger.Entry
	for _, tx := range fundedGroup.Transactions() {
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsPositive() {
				source = entry
			}
		}
	}
	require.NotNil(t, source)
	clock.FreezeTime(start.Add(time.Hour))
	chargeID := "legacy-spend"
	deps := transactions.ResolverDependencies{
		AccountService: env.Deps.ResolversService, AccountCatalog: env.Deps.AccountService, BalanceQuerier: env.Deps.HistoricalLedger,
	}
	inputs, err := transactions.ResolveTransactions(t.Context(), deps, transactions.ResolutionScope{
		Namespace: env.Namespace, CustomerID: env.CustomerID,
	}, transactions.TransferCustomerFBOToAccruedTemplate{
		At: clock.Now(), Currency: env.CurrencyReference(),
		Sources: []transactions.PostingAmount{{
			Address: source.PostingAddress(), Amount: alpacadecimal.NewFromInt(9),
			Identity: ledger.EntryIdentityParts{Provenance: ledger.Provenance{SourceChargeID: &funding.ID, SpendChargeID: &chargeID}},
		}},
	})
	require.NoError(t, err)
	annotations := ledger.ChargeAnnotations(models.NamespacedID{Namespace: env.Namespace, ID: chargeID})
	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, annotations, inputs...))
	require.NoError(t, err)
	require.Len(t, group.Transactions(), 1)

	// when: the original collection is reversed without provenance or billing lineage.
	clock.FreezeTime(start.Add(2 * time.Hour))
	corrected, err := transactions.CorrectTransaction(t.Context(), deps, transactions.CorrectionInput{
		At: clock.Now(), Amount: alpacadecimal.NewFromInt(9), OriginalTransaction: group.Transactions()[0], OriginalGroup: group,
	})
	require.NoError(t, err)
	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, annotations, corrected...))
	require.NoError(t, err)

	// then: old corrections remain visible without resolving their original activity.
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID: env.CustomerID, Limit: 10, Type: lo.ToPtr(CreditTransactionTypeCorrection),
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, float64(9), result.Items[0].Amount.InexactFloat64())
	require.Equal(t, float64(1), result.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(10), result.Items[0].Balance.After.InexactFloat64())
}
