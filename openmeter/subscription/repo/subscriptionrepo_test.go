package repo_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionCostBasisPinPersistence(t *testing.T) {
	// given:
	// - a pinned subscription, custom currency, and exact cost-basis resource
	// when:
	// - the pair is pinned and the subscription is loaded again
	// then:
	// - the hydrated pin preserves the exact resource ID and protects its references
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	db := testDB.EntDriver.Client()
	t.Cleanup(func() {
		_ = db.Close()
		testDB.Close(t)
	})

	const namespace = "default"
	at := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	customerID := ulid.Make().String()
	customCurrencyID := ulid.Make().String()
	costBasisID := ulid.Make().String()

	_, err := db.Customer.Create().
		SetID(customerID).
		SetNamespace(namespace).
		SetName("Pin Test Customer").
		SetCurrency(currencyx.Code("USD")).
		Save(t.Context())
	require.NoError(t, err)

	_, err = db.CustomCurrency.Create().
		SetID(customCurrencyID).
		SetNamespace(namespace).
		SetCode("CREDITS").
		SetName("Credits").
		SetSymbol("CR").
		Save(t.Context())
	require.NoError(t, err)

	costBasis, err := db.CurrencyCostBasis.Create().
		SetID(costBasisID).
		SetNamespace(namespace).
		SetCurrencyID(customCurrencyID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.25)).
		SetEffectiveFrom(at.Add(-time.Hour)).
		Save(t.Context())
	require.NoError(t, err)

	repository := subscriptionrepo.NewSubscriptionRepo(db)
	sub, err := repository.Create(t.Context(), subscription.CreateSubscriptionEntityInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		CadencedModel:   models.CadencedModel{ActiveFrom: at},
		Name:            "Pinned Subscription",
		CustomerId:      customerID,
		InvoiceCurrency: currencyx.Code("USD"),
		CostBasisMode:   subscription.CostBasisModePinned,
		BillingCadence:  datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
		BillingAnchor:   at,
		SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
	})
	require.NoError(t, err)
	require.Empty(t, sub.CostBasisPins)

	pinInput := subscription.CreateCostBasisPinEntityInput{
		Namespace:        namespace,
		SubscriptionID:   sub.ID,
		CustomCurrencyID: customCurrencyID,
		InvoiceCurrency:  currencyx.Code("USD"),
		CostBasisID:      costBasisID,
	}
	require.NoError(t, repository.CreateCostBasisPins(t.Context(), []subscription.CreateCostBasisPinEntityInput{pinInput}))

	loaded, err := repository.GetByID(t.Context(), sub.NamespacedID)
	require.NoError(t, err)
	require.Len(t, loaded.CostBasisPins, 1)
	require.Equal(t, customCurrencyID, loaded.CostBasisPins[0].CustomCurrencyID)
	require.Equal(t, currencyx.Code("USD"), loaded.CostBasisPins[0].InvoiceCurrency)
	require.Equal(t, costBasisID, loaded.CostBasisPins[0].CostBasis.ID)
	require.Equal(t, costBasis.Rate, loaded.CostBasisPins[0].CostBasis.Rate)

	err = repository.CreateCostBasisPins(t.Context(), []subscription.CreateCostBasisPinEntityInput{pinInput})
	require.Error(t, err, "a subscription can only pin one resource for a custom/fiat pair")

	err = db.CurrencyCostBasis.DeleteOneID(costBasisID).Exec(t.Context())
	require.Error(t, err, "a pinned cost basis must remain lifecycle-protected")

	err = db.CustomCurrency.DeleteOneID(customCurrencyID).Exec(t.Context())
	require.Error(t, err, "a currency referenced by a pin must remain lifecycle-protected")
}
