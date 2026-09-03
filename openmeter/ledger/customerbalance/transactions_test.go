package customerbalance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreditTransactionLoaders_InvalidType(t *testing.T) {
	s := &service{}
	invalid := CreditTransactionType("invalid")

	_, err := s.creditTransactionLoaders(&invalid)
	require.Error(t, err)
}

func TestCreditTransactionFromLedgerTransaction_UsesFBOEntry(t *testing.T) {
	usd := currencyx.Code("USD")
	tx := mustHistoricalTransaction(t, []ledgerhistorical.EntryData{
		mustEntryData(t, "entry-usd", ledger.AccountTypeCustomerFBO, usd, alpacadecimal.NewFromInt(-10)),
		mustEntryData(t, "entry-accrued", ledger.AccountTypeCustomerAccrued, usd, alpacadecimal.NewFromInt(10)),
	})

	item, err := creditTransactionFromLedgerTransaction(tx)
	require.NoError(t, err)
	require.Equal(t, CreditTransactionTypeConsumed, item.Type)
	require.Equal(t, currencyx.Code("USD"), item.Currency)
	require.True(t, item.Amount.Equal(alpacadecimal.NewFromInt(-10)))
}

func TestCreditTransactionFromLedgerTransaction_AggregatesScopedFBOEntries(t *testing.T) {
	usd := currencyx.Code("USD")
	tx := mustHistoricalTransaction(t, []ledgerhistorical.EntryData{
		mustEntryData(t, "entry-fbo-1", ledger.AccountTypeCustomerFBO, usd, alpacadecimal.NewFromInt(-10)),
		mustEntryData(t, "entry-fbo-2", ledger.AccountTypeCustomerFBO, usd, alpacadecimal.NewFromInt(-5)),
		mustEntryData(t, "entry-accrued", ledger.AccountTypeCustomerAccrued, usd, alpacadecimal.NewFromInt(15)),
	})

	item, err := creditTransactionFromLedgerTransaction(tx)
	require.NoError(t, err)
	require.Equal(t, CreditTransactionTypeConsumed, item.Type)
	require.Equal(t, currencyx.Code("USD"), item.Currency)
	require.True(t, item.Amount.Equal(alpacadecimal.NewFromInt(-15)))
}

func TestApplyCreditTransactionBalances(t *testing.T) {
	balanceImpact := alpacadecimal.NewFromInt(-7)
	currencyReference := currencies.NewCurrencyReference("USD")
	items := []CreditTransaction{
		{
			Currency:          "USD",
			Amount:            alpacadecimal.NewFromInt(-10),
			balanceImpact:     &balanceImpact,
			currencyReference: currencyReference,
		},
	}

	applyCreditTransactionBalances(items, map[string]alpacadecimal.Decimal{
		currencyReference.IdentityKey(): alpacadecimal.NewFromInt(42),
	})

	require.True(t, items[0].Balance.After.Equal(alpacadecimal.NewFromInt(42)))
	require.True(t, items[0].Balance.Before.Equal(alpacadecimal.NewFromInt(49)))
}

func TestApplyCreditTransactionBalancesSeparatesCustomCurrencyIdentities(t *testing.T) {
	alpha, err := currencies.ParseCurrencyReference([]byte("custom|v1|CREDITS|currency-alpha|2"))
	require.NoError(t, err)
	beta, err := currencies.ParseCurrencyReference([]byte("custom|v1|CREDITS|currency-beta|2"))
	require.NoError(t, err)

	items := []CreditTransaction{
		{Currency: "CREDITS", Amount: alpacadecimal.NewFromInt(10), currencyReference: alpha},
		{Currency: "CREDITS", Amount: alpacadecimal.NewFromInt(20), currencyReference: beta},
		{Currency: "CREDITS", Amount: alpacadecimal.NewFromInt(30), currencyReference: alpha},
	}

	applyCreditTransactionBalances(items, map[string]alpacadecimal.Decimal{
		alpha.IdentityKey(): alpacadecimal.NewFromInt(100),
		beta.IdentityKey():  alpacadecimal.NewFromInt(200),
	})

	require.Equal(t, float64(90), items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(100), items[0].Balance.After.InexactFloat64())
	require.Equal(t, float64(180), items[1].Balance.Before.InexactFloat64())
	require.Equal(t, float64(200), items[1].Balance.After.InexactFloat64())
	require.Equal(t, float64(60), items[2].Balance.Before.InexactFloat64())
	require.Equal(t, float64(90), items[2].Balance.After.InexactFloat64())
}

func TestListCreditTransactionsBalancesByCurrency(t *testing.T) {
	env := newTestEnv(t)

	// given:
	// - funded and consumed USD/EUR movements are interleaved by booked time
	issuedAt := clock.Now()
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), "USD", nil)

	eurFundedAt := issuedAt.Add(time.Hour)
	clock.FreezeTime(eurFundedAt)
	t.Cleanup(clock.UnFreeze)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(200), "EUR", nil)

	usdConsumedAt := issuedAt.Add(2 * time.Hour)
	clock.FreezeTime(usdConsumedAt.Add(-time.Minute))
	t.Cleanup(clock.UnFreeze)
	usdCharge := env.createFlatFeeChargeInCurrency(
		t,
		alpacadecimal.NewFromInt(30),
		productcatalog.CreditOnlySettlementMode,
		timeutil.ClosedPeriod{From: usdConsumedAt, To: usdConsumedAt},
		"USD",
	)
	clock.FreezeTime(usdConsumedAt.Add(time.Second))
	t.Cleanup(clock.UnFreeze)
	env.advanceFlatFeeCharge(t, usdCharge)

	eurConsumedAt := issuedAt.Add(3 * time.Hour)
	clock.FreezeTime(eurConsumedAt.Add(-time.Minute))
	t.Cleanup(clock.UnFreeze)
	eurCharge := env.createFlatFeeChargeInCurrency(
		t,
		alpacadecimal.NewFromInt(50),
		productcatalog.CreditOnlySettlementMode,
		timeutil.ClosedPeriod{From: eurConsumedAt, To: eurConsumedAt},
		"EUR",
	)
	clock.FreezeTime(eurConsumedAt.Add(time.Second))
	t.Cleanup(clock.UnFreeze)
	env.advanceFlatFeeCharge(t, eurCharge)

	// when:
	// - an unfiltered history interleaves two currency balance chains
	result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	// then:
	// - rows stay globally ordered while balances advance only within their currency
	expected := []struct {
		currency currencyx.Code
		txType   CreditTransactionType
		bookedAt time.Time
		amount   int64
		before   int64
		after    int64
	}{
		{currency: "EUR", txType: CreditTransactionTypeConsumed, bookedAt: eurConsumedAt, amount: -50, before: 200, after: 150},
		{currency: "USD", txType: CreditTransactionTypeConsumed, bookedAt: usdConsumedAt, amount: -30, before: 100, after: 70},
		{currency: "EUR", txType: CreditTransactionTypeFunded, bookedAt: eurFundedAt, amount: 200, before: 0, after: 200},
		{currency: "USD", txType: CreditTransactionTypeFunded, bookedAt: issuedAt, amount: 100, before: 0, after: 100},
	}
	require.Len(t, result.Items, len(expected))
	for i, want := range expected {
		item := result.Items[i]
		require.Equal(t, want.currency, item.Currency)
		require.Equal(t, want.txType, item.Type)
		require.True(t, want.bookedAt.Equal(item.BookedAt))
		require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
		require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
		require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())
	}

	// when:
	// - the same history is filtered to one currency
	usd := currencyx.Code("USD")
	filtered, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
		CustomerID:    env.CustomerID,
		Limit:         10,
		Currency:      &usd,
		FeatureFilter: AllFeatureFilter(),
	})
	require.NoError(t, err)

	// then:
	// - the filtered rows retain the same USD balance chain
	require.Len(t, filtered.Items, 2)
	require.Equal(t, CreditTransactionTypeConsumed, filtered.Items[0].Type)
	require.Equal(t, float64(100), filtered.Items[0].Balance.Before.InexactFloat64())
	require.Equal(t, float64(70), filtered.Items[0].Balance.After.InexactFloat64())
	require.Equal(t, CreditTransactionTypeFunded, filtered.Items[1].Type)
	require.Equal(t, float64(0), filtered.Items[1].Balance.Before.InexactFloat64())
	require.Equal(t, float64(100), filtered.Items[1].Balance.After.InexactFloat64())
}

func TestApplyChargeMetadataToCreditTransactions(t *testing.T) {
	const (
		namespace = "ns"
		chargeID  = "charge-1"
	)

	description := "Welcome credits"

	service := service{
		ChargesService: staticChargeService{
			chargesByID: map[string]charges.Charge{
				chargeID: charges.NewCharge(creditpurchase.Charge{
					ChargeBase: creditpurchase.ChargeBase{
						ManagedResource: chargemeta.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: namespace,
							},
							ID: chargeID,
						},
						Intent: creditpurchase.Intent{
							Intent: chargemeta.Intent{},
							IntentMutableFields: creditpurchase.IntentMutableFields{
								IntentMutableFields: chargemeta.IntentMutableFields{
									Name:        "Intro Credits",
									Description: lo.ToPtr(description),
								},
							},
						},
					},
				}),
			},
		},
	}

	items := []CreditTransaction{
		{
			Name: "IssueCustomerReceivableTemplate",
			Annotations: models.Annotations{
				ledger.AnnotationChargeID: chargeID,
			},
		},
		{
			Name: "",
		},
	}

	service.applyChargeMetadataToCreditTransactions(t.Context(), namespace, items)

	require.Equal(t, "Intro Credits", items[0].Name)
	require.NotNil(t, items[0].Description)
	require.Equal(t, description, *items[0].Description)
	require.Equal(t, "", items[1].Name)
	require.Nil(t, items[1].Description)
}

func TestMergeSortedLists_ByCursorDesc(t *testing.T) {
	base := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	toTx := func(id string, bookedAt, createdAt time.Time) CreditTransaction {
		return CreditTransaction{
			ID: models.NamespacedID{
				Namespace: "ns",
				ID:        id,
			},
			BookedAt:  bookedAt,
			CreatedAt: createdAt,
		}
	}

	funded := []CreditTransaction{
		toTx("tx-6", base.Add(-10*time.Second), base.Add(-10*time.Second)),
		toTx("tx-4", base.Add(-30*time.Second), base.Add(-30*time.Second)),
		toTx("tx-2", base.Add(-50*time.Second), base.Add(-50*time.Second)),
	}

	consumed := []CreditTransaction{
		toTx("tx-5", base.Add(-20*time.Second), base.Add(-20*time.Second)),
		toTx("tx-3", base.Add(-40*time.Second), base.Add(-40*time.Second)),
		toTx("tx-1", base.Add(-60*time.Second), base.Add(-60*time.Second)),
	}

	merged, hasMore := mergeSortedLists(
		[][]CreditTransaction{funded, consumed},
		4,
		compareCreditTransactionsByCursor,
	)

	require.True(t, hasMore)
	require.Len(t, merged, 4)
	require.Equal(t, "tx-6", merged[0].ID.ID)
	require.Equal(t, "tx-5", merged[1].ID.ID)
	require.Equal(t, "tx-4", merged[2].ID.ID)
	require.Equal(t, "tx-3", merged[3].ID.ID)
}

func mustHistoricalTransaction(t *testing.T, entries []ledgerhistorical.EntryData) ledger.Transaction {
	t.Helper()

	tx, err := ledgerhistorical.NewTransactionFromData(ledgerhistorical.TransactionData{
		ID:        "tx-1",
		Namespace: "ns",
		CreatedAt: time.Now().UTC(),
		BookedAt:  time.Now().UTC(),
	}, entries)
	require.NoError(t, err)

	return tx
}

func mustEntryData(t *testing.T, id string, accountType ledger.AccountType, currency currencyx.Code, amount alpacadecimal.Decimal) ledgerhistorical.EntryData {
	t.Helper()

	route := ledger.Route{Currency: currencies.NewCurrencyReference(currency)}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	return ledgerhistorical.EntryData{
		ID:            id,
		Namespace:     "ns",
		CreatedAt:     time.Now().UTC(),
		SubAccountID:  id + "-subaccount",
		AccountType:   accountType,
		Route:         route,
		RouteID:       id + "-route",
		RouteKey:      key.Value(),
		RouteKeyVer:   key.Version(),
		Amount:        amount,
		TransactionID: "tx-1",
	}
}

type staticChargeService struct {
	chargesByID map[string]charges.Charge
}

func (s staticChargeService) GetByIDs(_ context.Context, input charges.GetByIDsInput) (charges.Charges, error) {
	items := make(charges.Charges, 0, len(input.IDs))
	for _, id := range input.IDs {
		charge, ok := s.chargesByID[id]
		if !ok {
			continue
		}

		items = append(items, charge)
	}

	return items, nil
}

func (s staticChargeService) ListCharges(context.Context, charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	return pagination.Result[charges.Charge]{}, nil
}

type noListChargesService struct {
	ChargesService chargesService
}

func (s noListChargesService) GetByIDs(ctx context.Context, input charges.GetByIDsInput) (charges.Charges, error) {
	return s.ChargesService.GetByIDs(ctx, input)
}

func (s noListChargesService) ListCharges(context.Context, charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	return pagination.Result[charges.Charge]{}, errors.New("ListCharges must not be called")
}
