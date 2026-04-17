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

func TestFromAPIBillingCreditTransactionType_Adjusted(t *testing.T) {
	filter := api.BillingCreditTransactionTypeAdjusted

	txType := fromAPIBillingCreditTransactionType(&filter)

	require.NotNil(t, txType)
	require.Equal(t, customerbalance.CreditTransactionTypeAdjusted, *txType)
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
