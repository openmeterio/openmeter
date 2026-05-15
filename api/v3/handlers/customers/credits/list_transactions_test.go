package customerscredits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIBillingCreditTransactionType_Funded(t *testing.T) {
	filter := api.BillingCreditTransactionTypeFunded

	txType := fromAPIBillingCreditTransactionType(&filter)

	require.NotNil(t, txType)
	require.Equal(t, customerbalance.CreditTransactionTypeFunded, *txType)
}

func TestFromAPIBillingCreditTransactionType_Expired(t *testing.T) {
	filter := api.BillingCreditTransactionTypeExpired

	txType := fromAPIBillingCreditTransactionType(&filter)

	require.NotNil(t, txType)
	require.Equal(t, customerbalance.CreditTransactionTypeExpired, *txType)
}

func TestToAPIBillingCreditTransaction(t *testing.T) {
	createdAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	bookedAt := createdAt.Add(time.Second)
	description := "Welcome credits"

	tx := toAPIBillingCreditTransaction(customerbalance.CreditTransaction{
		ID: models.NamespacedID{
			Namespace: "ns",
			ID:        "tx-1",
		},
		CreatedAt: createdAt,
		BookedAt:  bookedAt,
		Type:      customerbalance.CreditTransactionTypeConsumed,
		Currency:  currencyx.Code("USD"),
		Amount:    alpacadecimal.NewFromInt(-10),
		Balance: customerbalance.CreditTransactionBalance{
			Before: alpacadecimal.NewFromInt(52),
			After:  alpacadecimal.NewFromInt(42),
		},
		Name:        "credit_transaction",
		Description: &description,
		Annotations: models.Annotations{
			ledger.AnnotationChargeID: "charge-1",
		},
	})

	require.Equal(t, api.ULID("tx-1"), tx.Id)
	require.Equal(t, api.BillingCreditTransactionTypeConsumed, tx.Type)
	require.Equal(t, api.BillingCurrencyCode("USD"), tx.Currency)
	require.Equal(t, api.Numeric("-10"), tx.Amount)
	require.Equal(t, api.Numeric("52"), tx.AvailableBalance.Before)
	require.Equal(t, api.Numeric("42"), tx.AvailableBalance.After)
	require.NotNil(t, tx.Description)
	require.Equal(t, description, *tx.Description)
	require.NotNil(t, tx.Labels)
	require.Equal(t, "charge-1", (*tx.Labels)["charge_id"])
}

func TestToAPIBillingCreditTransaction_Expired(t *testing.T) {
	tx := toAPIBillingCreditTransaction(customerbalance.CreditTransaction{
		ID: models.NamespacedID{
			Namespace: "ns",
			ID:        "tx-1",
		},
		CreatedAt: time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
		BookedAt:  time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC),
		Type:      customerbalance.CreditTransactionTypeExpired,
		Currency:  currencyx.Code("USD"),
		Amount:    alpacadecimal.NewFromInt(-4),
	})

	require.Equal(t, api.BillingCreditTransactionTypeExpired, tx.Type)
}

func TestCreditTransactionCursorConversion(t *testing.T) {
	bookedAt := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	createdAt := bookedAt.Add(-time.Second)
	namespace := "ns"

	ledgerCursor := ledger.TransactionCursor{
		BookedAt:  bookedAt,
		CreatedAt: createdAt,
		ID: models.NamespacedID{
			Namespace: namespace,
			ID:        "01J7JABCDXYZ1234567890ABCD",
		},
	}

	encoded, err := encodeBillingCreditTransactionCursor(ledgerCursor)
	require.NoError(t, err)

	decoded, err := decodeBillingCreditTransactionCursor(encoded, namespace)
	require.NoError(t, err)
	require.Equal(t, bookedAt, decoded.BookedAt)
	require.Equal(t, createdAt, decoded.CreatedAt)
	require.Equal(t, namespace, decoded.ID.Namespace)
	require.Equal(t, ledgerCursor.ID.ID, decoded.ID.ID)
}

func TestCreditTransactionCursorConversion_InvalidCursor(t *testing.T) {
	_, err := decodeBillingCreditTransactionCursor("not-base64", "ns")
	require.Error(t, err)
}
