package customerbalance

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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

func TestApplyCreditTransactionBalances(t *testing.T) {
	items := []CreditTransaction{
		{
			Amount: alpacadecimal.NewFromInt(-10),
		},
	}

	applyCreditTransactionBalances(items, alpacadecimal.NewFromInt(42))

	require.True(t, items[0].Balance.After.Equal(alpacadecimal.NewFromInt(42)))
	require.True(t, items[0].Balance.Before.Equal(alpacadecimal.NewFromInt(52)))
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
							Intent: chargemeta.Intent{
								Name:        "Intro Credits",
								Description: lo.ToPtr(description),
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

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, route)
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
