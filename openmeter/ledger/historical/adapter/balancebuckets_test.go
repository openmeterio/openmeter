package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	transactionstestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRepo_GetBalanceBuckets_ProvenanceGroupingAndSelectors(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	ctx := t.Context()
	namespace := testNamespace()

	fbo := env.createSubAccountOfType(
		t,
		namespace,
		ledger.AccountTypeCustomerFBO,
		ledger.Route{
			Currency:       currencies.NewCurrencyReference(currencyx.Code("USD")),
			CostBasis:      lo.ToPtr(mustDecimal(t, "0.70")),
			CreditPriority: lo.ToPtr(1),
		},
	)
	counterpart := env.createSubAccountOfType(
		t,
		namespace,
		ledger.AccountTypeWash,
		ledger.Route{
			Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
		},
	)

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	sourceCharge1 := "01JABCDEF0123456789ABCDEFG"
	sourceCharge2 := "01JBCDEFG0123456789ABCDEFG"
	spendCharge1 := "01JCDEFGH0123456789ABCDEFG"
	spendCharge2 := "01JDEFGHI0123456789ABCDEFG"

	bookedAtEarly := time.Now().UTC().Add(-2 * time.Hour)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, mustSetUpHistoricalTransactionInput(
		t,
		bookedAtEarly,
		[]*transactionstestutils.AnyEntryInput{
			provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(100), &sourceCharge1, &spendCharge1),
			provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-100), &sourceCharge1, &spendCharge1),
			provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(50), &sourceCharge1, &spendCharge2),
			provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-50), &sourceCharge1, &spendCharge2),
		},
	))
	require.NoError(t, err)

	asOf := bookedAtEarly.Add(time.Hour)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, mustSetUpHistoricalTransactionInput(
		t,
		asOf.Add(time.Hour),
		[]*transactionstestutils.AnyEntryInput{
			provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(25), &sourceCharge2, &spendCharge1),
			provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-25), &sourceCharge2, &spendCharge1),
			provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(10), nil, &spendCharge1),
			provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-10), nil, &spendCharge1),
		},
	))
	require.NoError(t, err)

	accountID := fbo.AccountID
	balancesBySource, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	require.NoError(t, err)

	requireBalanceBucketAmounts(
		t,
		balancesBySource,
		map[string]float64{
			sourceChargeKey(&sourceCharge1): 150,
			sourceChargeKey(&sourceCharge2): 25,
			sourceChargeKey(nil):            10,
		},
	)

	balancesBySpend, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	require.NoError(t, err)

	requireBalanceBucketAmounts(
		t,
		balancesBySpend,
		map[string]float64{
			spendChargeKey(&spendCharge1): 135,
			spendChargeKey(&spendCharge2): 50,
		},
	)

	balancesBySourceAndSpend, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
		},
	})
	require.NoError(t, err)

	requireBalanceBucketAmounts(
		t,
		balancesBySourceAndSpend,
		map[string]float64{
			sourceSpendChargeKey(&sourceCharge1, &spendCharge1): 100,
			sourceSpendChargeKey(&sourceCharge1, &spendCharge2): 50,
			sourceSpendChargeKey(&sourceCharge2, &spendCharge1): 25,
			sourceSpendChargeKey(nil, &spendCharge1):            10,
		},
	)

	nullSourceBalances, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Provenance: ledger.ProvenanceFilter{
				SourceChargeID: mo.Some[*string](nil),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	require.NoError(t, err)

	requireBalanceBucketAmounts(
		t,
		nullSourceBalances,
		map[string]float64{
			spendChargeKey(&spendCharge1): 10,
		},
	)

	asOfBalances, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			AsOf:      &asOf,
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	require.NoError(t, err)

	requireBalanceBucketAmounts(
		t,
		asOfBalances,
		map[string]float64{
			sourceChargeKey(&sourceCharge1): 150,
		},
	)
}

func TestRepo_GetBalanceBuckets_HydratesCostBasisCurrency(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// given:
	// - an ACME FBO balance whose V3 route is qualified by USD
	// when:
	// - balance buckets hydrate the posting address from SQL
	// then:
	// - the route retains USD and can be reused by ledger collection
	ctx := t.Context()
	namespace := testNamespace()
	costBasis := mustDecimal(t, "0.25")
	sourceCurrency := currencyx.Code("USD")
	customCurrency := currencyx.Code("ACME")
	customCurrencyReference := mustCustomCurrencyReference(t, customCurrency, "test-custom-currency-acme", 2)
	fbo := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerFBO, ledger.Route{
		Currency:          customCurrencyReference,
		CostBasisCurrency: &sourceCurrency,
		CostBasis:         &costBasis,
		CreditPriority:    lo.ToPtr(1),
	})
	counterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeBrokerage, ledger.Route{
		Currency:          customCurrencyReference,
		CostBasisCurrency: &sourceCurrency,
		CostBasis:         &costBasis,
	})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(100), nil, nil),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-100), nil, nil),
	}))
	require.NoError(t, err)

	accountID := fbo.AccountID
	buckets, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency:          customCurrencyReference,
				CostBasisCurrency: mo.Some(&sourceCurrency),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, buckets, 1)

	route := buckets[0].Address.Route().Route()
	require.Equal(t, customCurrency, route.Currency.Code)
	require.Equal(t, &sourceCurrency, route.CostBasisCurrency)
	require.NotNil(t, route.CostBasis)
	require.Equal(t, costBasis.InexactFloat64(), route.CostBasis.InexactFloat64())
}

func TestRepo_GetBalanceBuckets_ExcludesTransactionAnnotations(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	ctx := t.Context()
	namespace := testNamespace()
	fbo := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerFBO, ledger.Route{
		Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
	})
	counterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeWash, ledger.Route{
		Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
	})
	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	visibleInput := mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(5), nil, nil),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-5), nil, nil),
	})
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, visibleInput)
	require.NoError(t, err)

	internalInput := &transactionstestutils.AnyTransactionInput{
		BookedAtValue: time.Now().UTC(),
		EntryInputsValues: []*transactionstestutils.AnyEntryInput{
			provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(3), nil, nil),
			provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-3), nil, nil),
		},
		AnnotationsValue: models.Annotations{
			ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal,
		},
	}
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, internalInput)
	require.NoError(t, err)

	accountID := fbo.AccountID
	buckets, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
		},
		ExcludeAnnotationFilters: map[string]string{
			ledger.AnnotationCustomerBalanceVisibility: ledger.CustomerBalanceVisibilityInternal,
		},
	})
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	require.Equal(t, float64(5), buckets[0].SettledAmount.InexactFloat64())
}

func TestRepo_GetBalanceBuckets_DistinctManagedCurrenciesSameCodeDoNotMerge(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// given:
	// - two managed custom currencies that both use the bare code "ACME"
	//   but have distinct managed IDs (a customer could hold
	//   FBO balances in two differently-managed "ACME" currencies)
	// - both FBO sub-accounts are provisioned under the SAME account, so
	//   exact CurrencyReference identity keeps them separate
	// when:
	// - each currency is funded with a different amount
	// then:
	// - GetBalanceBuckets scoped by one exact reference excludes the other
	ctx := t.Context()
	namespace := testNamespace()
	customCurrency := currencyx.Code("ACME")
	alpha := mustCustomCurrencyReference(t, customCurrency, "custom-currency-alpha", 2)
	beta := mustCustomCurrencyReference(t, customCurrency, "custom-currency-beta", 2)

	acc, err := env.accountRepo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: namespace,
		Type:      ledger.AccountTypeCustomerFBO,
	})
	require.NoError(t, err)

	alphaFBO, err := env.accountRepo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: acc.ID.ID,
		Route: ledger.Route{
			Currency:       alpha,
			CreditPriority: lo.ToPtr(1),
		},
	})
	require.NoError(t, err)

	betaFBO, err := env.accountRepo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: acc.ID.ID,
		Route: ledger.Route{
			Currency:       beta,
			CreditPriority: lo.ToPtr(1),
		},
	})
	require.NoError(t, err)

	alphaCounterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerReceivable, ledger.Route{
		Currency: alpha,
	})
	betaCounterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerReceivable, ledger.Route{
		Currency: beta,
	})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		provenanceEntryInput(t, alphaFBO, alpacadecimal.NewFromInt(100), nil, nil),
		provenanceEntryInput(t, alphaCounterpart, alpacadecimal.NewFromInt(-100), nil, nil),
		provenanceEntryInput(t, betaFBO, alpacadecimal.NewFromInt(30), nil, nil),
		provenanceEntryInput(t, betaCounterpart, alpacadecimal.NewFromInt(-30), nil, nil),
	}))
	require.NoError(t, err)

	buckets, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &acc.ID.ID,
			Route: ledger.RouteFilter{
				Currency: alpha,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	require.True(t, buckets[0].Address.Route().Route().Currency.Equal(alpha))
	require.Equal(t, 100.0, buckets[0].SettledAmount.InexactFloat64())
}

func mustCustomCurrencyReference(t *testing.T, code currencyx.Code, id string, precision int) currencies.CurrencyReference {
	t.Helper()

	reference, err := currencies.ParseCurrencyReference([]byte(fmt.Sprintf("custom|v1|%s|%s|%d", code, id, precision)))
	require.NoError(t, err)

	return reference
}

func provenanceEntryInput(t *testing.T, sub *ledgeraccount.SubAccountData, amount alpacadecimal.Decimal, sourceChargeID, spendChargeID *string) *transactionstestutils.AnyEntryInput {
	t.Helper()

	identityKey, _ := ledger.EntryIdentityParts{
		Provenance: ledger.Provenance{
			SourceChargeID: sourceChargeID,
			SpendChargeID:  spendChargeID,
		},
	}.Text()

	return &transactionstestutils.AnyEntryInput{
		Address:             testAddress(t, sub),
		AmountValue:         amount,
		IdentityKeyValue:    string(identityKey),
		SourceChargeIDValue: sourceChargeID,
		SpendChargeIDValue:  spendChargeID,
	}
}

func requireBalanceBucketAmounts(t *testing.T, balances []ledger.BalanceBucket, expected map[string]float64) {
	t.Helper()

	actual := lo.SliceToMap(balances, func(balance ledger.BalanceBucket) (string, float64) {
		require.NotNil(t, balance.Address)
		require.True(t, balance.SettledAmount.Equal(balance.PendingAmount))

		sourceChargeID := balance.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID]
		spendChargeID := balance.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]

		return sourceSpendChargeKey(sourceChargeID, spendChargeID), balance.SettledAmount.InexactFloat64()
	})
	require.Equal(t, expected, actual)
}

func sourceChargeKey(sourceChargeID *string) string {
	return sourceSpendChargeKey(sourceChargeID, nil)
}

func spendChargeKey(spendChargeID *string) string {
	return sourceSpendChargeKey(nil, spendChargeID)
}

func sourceSpendChargeKey(sourceChargeID, spendChargeID *string) string {
	return fmt.Sprintf("source=%s spend=%s", chargeIDKeyPart(sourceChargeID), chargeIDKeyPart(spendChargeID))
}

func chargeIDKeyPart(chargeID *string) string {
	if chargeID == nil {
		return "<nil>"
	}

	return *chargeID
}

func TestRepo_GetMinimumBalanceBuckets_Timeline(t *testing.T) {
	// All offsets are relative to the requested boundary. Rows are deliberately
	// booked in the supplied order, independently of their effective time.
	type movement struct {
		offset      time.Duration
		amount      int64
		annotations models.Annotations
	}

	ignored := models.Annotations{"test.movement": "ignored"}
	otherIgnored := models.Annotations{"test.other": "ignored"}
	for _, tc := range []struct {
		name      string
		movements []movement
		expected  float64
	}{
		{name: "future spending is reserved", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -7, nil}}, expected: 3},
		{name: "later return cannot hide earlier minimum", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -7, nil}, {2 * time.Hour, 7, nil}}, expected: 3},
		{name: "future only issuance starts at zero", movements: []movement{{time.Hour, 10, nil}}, expected: 0},
		{name: "future only deficit is retained", movements: []movement{{time.Hour, -4, nil}}, expected: -4},
		{name: "future replenishment cannot be spent early", movements: []movement{{-time.Hour, 4, nil}, {time.Hour, 6, nil}, {2 * time.Hour, -6, nil}}, expected: 4},
		{name: "simultaneous postings are netted", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -15, nil}, {time.Hour, 10, nil}}, expected: 5},
		{name: "boundary postings are inclusive", movements: []movement{{-time.Hour, 10, nil}, {0, -6, nil}, {time.Hour, 20, nil}}, expected: 4},
		{name: "out of order history uses effective time", movements: []movement{{2 * time.Hour, 7, nil}, {-time.Hour, 10, nil}, {time.Hour, -7, nil}}, expected: 3},
		{name: "excluded future debits and credits", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -10, ignored}, {2 * time.Hour, 6, ignored}, {3 * time.Hour, -4, nil}}, expected: 6},
		{name: "excluded future entry retains recorded age", movements: []movement{{time.Hour, -10, ignored}, {-time.Hour, 10, nil}}, expected: 10},
		{name: "future only excluded movement preserves the bucket", movements: []movement{{time.Hour, -10, ignored}}, expected: 0},
		{name: "either exclusion omits a future movement", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -4, ignored}, {2 * time.Hour, -3, otherIgnored}, {3 * time.Hour, -2, nil}}, expected: 8},
		{name: "different annotation value is included", movements: []movement{{-time.Hour, 10, nil}, {time.Hour, -4, models.Annotations{"test.movement": "included"}}}, expected: 6},
		{name: "exclusion does not affect opening balance", movements: []movement{{-2 * time.Hour, 10, nil}, {-time.Hour, -4, ignored}}, expected: 6},
		{name: "exclusion does not affect boundary", movements: []movement{{-time.Hour, 10, nil}, {0, -10, ignored}}, expected: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := NewTestEnv(t)
			t.Cleanup(func() { env.Close(t) })
			namespace := testNamespace()
			at := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
			route := ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))}
			account := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerFBO, route)
			counterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeWash, route)
			group, err := env.repo.CreateTransactionGroup(t.Context(), ledgerhistorical.CreateTransactionGroupInput{Namespace: namespace})
			require.NoError(t, err)

			// given: a persisted bucket with past and future movements.
			for _, movement := range tc.movements {
				amount := alpacadecimal.NewFromInt(movement.amount)
				input := &transactionstestutils.AnyTransactionInput{
					BookedAtValue:    at.Add(movement.offset),
					AnnotationsValue: movement.annotations,
					EntryInputsValues: []*transactionstestutils.AnyEntryInput{
						provenanceEntryInput(t, account, amount, nil, nil),
						provenanceEntryInput(t, counterpart, amount.Neg(), nil, nil),
					},
				}
				_, err = env.repo.BookTransaction(t.Context(), models.NamespacedID{Namespace: namespace, ID: group.ID}, input)
				require.NoError(t, err)
			}

			// when: only later matching movements are excluded from the minimum.
			buckets, err := env.repo.GetMinimumBalanceBuckets(t.Context(), ledger.MinimumBalanceBucketQuery{
				BalanceBucketQuery: ledger.BalanceBucketQuery{
					Namespace: namespace,
					Filters:   ledger.Filters{AccountID: &account.AccountID, AsOf: &at},
					GroupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID},
				},
				ExcludeFutureAnnotationFilters: map[string]string{"test.movement": "ignored", "test.other": "ignored"},
			})
			require.NoError(t, err)

			// then: the lowest effective balance includes the opening position.
			require.Len(t, buckets, 1)
			require.Equal(t, tc.expected, buckets[0].SettledAmount.InexactFloat64())
			require.Equal(t, tc.expected, buckets[0].PendingAmount.InexactFloat64())

			oldest, err := env.client.LedgerEntry.Query().Where(ledgerentrydb.SubAccountID(account.ID)).Order(ledgerentrydb.ByCreatedAt()).First(t.Context())
			require.NoError(t, err)
			require.True(t, oldest.CreatedAt.Equal(buckets[0].OldestMatchingEntryCreatedAt))
			require.Nil(t, buckets[0].GroupByValues[ledger.BalanceBucketGroupBySourceChargeID])
		})
	}
}
