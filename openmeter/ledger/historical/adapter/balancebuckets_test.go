package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

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
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	fbo := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerFBO, ledger.Route{
		Currency:       currencyx.Code("USD"),
		CostBasis:      lo.ToPtr(mustDecimal(t, "0.70")),
		CreditPriority: lo.ToPtr(1),
	})
	counterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeWash, ledger.Route{
		Currency: currencyx.Code("USD"),
	})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	sourceCharge1 := "01JABCDEF0123456789ABCDEFG"
	sourceCharge2 := "01JBCDEFG0123456789ABCDEFG"
	spendCharge1 := "01JCDEFGH0123456789ABCDEFG"
	spendCharge2 := "01JDEFGHI0123456789ABCDEFG"

	bookedAtEarly := time.Now().UTC().Add(-2 * time.Hour)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, mustSetUpHistoricalTransactionInput(t, bookedAtEarly, []*transactionstestutils.AnyEntryInput{
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(100), &sourceCharge1, &spendCharge1),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-100), &sourceCharge1, &spendCharge1),
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(50), &sourceCharge1, &spendCharge2),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-50), &sourceCharge1, &spendCharge2),
	}))
	require.NoError(t, err)

	asOf := bookedAtEarly.Add(time.Hour)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, mustSetUpHistoricalTransactionInput(t, asOf.Add(time.Hour), []*transactionstestutils.AnyEntryInput{
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(25), &sourceCharge2, &spendCharge1),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-25), &sourceCharge2, &spendCharge1),
		provenanceEntryInput(t, fbo, alpacadecimal.NewFromInt(10), nil, &spendCharge1),
		provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-10), nil, &spendCharge1),
	}))
	require.NoError(t, err)

	accountID := fbo.AccountID
	balancesBySource, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencyx.Code("USD"),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	require.NoError(t, err)
	requireBalanceBucketAmounts(t, balancesBySource, map[string]float64{
		sourceChargeKey(&sourceCharge1): 150,
		sourceChargeKey(&sourceCharge2): 25,
		sourceChargeKey(nil):            10,
	})

	balancesBySpend, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencyx.Code("USD"),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	require.NoError(t, err)
	requireBalanceBucketAmounts(t, balancesBySpend, map[string]float64{
		spendChargeKey(&spendCharge1): 135,
		spendChargeKey(&spendCharge2): 50,
	})

	balancesBySourceAndSpend, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: currencyx.Code("USD"),
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
		},
	})
	require.NoError(t, err)
	requireBalanceBucketAmounts(t, balancesBySourceAndSpend, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, &spendCharge1): 100,
		sourceSpendChargeKey(&sourceCharge1, &spendCharge2): 50,
		sourceSpendChargeKey(&sourceCharge2, &spendCharge1): 25,
		sourceSpendChargeKey(nil, &spendCharge1):            10,
	})

	nullSourceBalances, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID:      &accountID,
			SourceChargeID: mo.Some[*string](nil),
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
	})
	require.NoError(t, err)
	requireBalanceBucketAmounts(t, nullSourceBalances, map[string]float64{
		spendChargeKey(&spendCharge1): 10,
	})

	asOfBalances, err := env.repo.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			AsOf:      &asOf,
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	require.NoError(t, err)
	requireBalanceBucketAmounts(t, asOfBalances, map[string]float64{
		sourceChargeKey(&sourceCharge1): 150,
	})
}

func provenanceEntryInput(t *testing.T, sub *ledgeraccount.SubAccountData, amount alpacadecimal.Decimal, sourceChargeID, spendChargeID *string) *transactionstestutils.AnyEntryInput {
	t.Helper()

	identityKey, _ := ledger.EntryIdentityParts{
		SourceChargeID: sourceChargeID,
		SpendChargeID:  spendChargeID,
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
