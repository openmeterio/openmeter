//go:build rte_e2e

package e2e

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wmkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteclient "github.com/openmeterio/openmeter/openmeter/rte/client"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRTEClient(t *testing.T) {
	api := newV3Client(t)
	ctx := t.Context()
	pool := initE2EPostgresPool(t)
	driver := entdriver.NewEntPostgresDriver(stdlib.OpenDBFromPool(pool))
	t.Cleanup(func() { require.NoError(t, driver.Close()) })

	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	publisher, err := wmkafka.NewPublisher(wmkafka.PublisherConfig{
		Brokers: []string{"localhost:39092"}, Marshaler: wmkafka.DefaultMarshaler{},
	}, watermill.NewSlogLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, publisher.Close()) })
	bus, err := eventbus.New(eventbus.Options{
		Publisher: publisher, Logger: logger,
		TopicMapping: eventbus.TopicMapping{SystemEventsTopic: "om_sys.api_events", IngestEventsTopic: "om_sys.ingest_events", BalanceWorkerEventsTopic: "om_sys.balance_worker_events"},
	})
	require.NoError(t, err)
	client, err := rteclient.New(rteclient.Config{DB: driver.Client(), Publisher: bus, Logger: logger})
	require.NoError(t, err)
	credits := client.CreditBalances()

	// Given a new customer with no stored RTE balance.
	customerKey := uniqueKey("rte-client")
	cust, err := api.Customers.Create(ctx, v3sdk.CreateCustomerRequest{Key: customerKey, Name: customerKey, Currency: lo.ToPtr("USD")})
	api.requireStatus(http.StatusCreated, err)
	key := creditbalance.Key{
		CustomerID: customer.CustomerID{Namespace: getCustomerNamespaceByKey(t, pool, customerKey), ID: cust.ID},
		Currency:   currencies.NewCurrencyReference("USD"),
	}
	_, err = credits.Get(ctx, key)
	require.True(t, models.IsGenericNotFoundError(err), "%v", err)

	// When the client asks for a calculation, the separate worker stores a real zero.
	require.NoError(t, credits.RequestCalculation(ctx, key))
	var initial rte.State[creditbalance.Balance]
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		initial, err = credits.Get(ctx, key)
		require.NoError(c, err)
		require.False(c, initial.CalculatedAt.IsZero())
		require.Zero(c, initial.Value.Live.InexactFloat64())
	}, 30*time.Second, 100*time.Millisecond)

	// Then a posted grant is visible after the worker processes a queued pull.
	_, err = api.Customers.Credits.Grants.Create(ctx, cust.ID, v3sdk.CreateCreditGrantRequest{
		Name: "RTE client balance", Currency: "USD", Amount: "25", FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	api.requireStatus(http.StatusCreated, err)
	require.NoError(t, credits.RequestRefresh(ctx, key, rte.FreshnessPolicy{TrackForSeconds: 60}))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		state, err := credits.Get(ctx, key)
		require.NoError(c, err)
		require.True(c, state.CalculatedAt.After(initial.CalculatedAt))
		require.Equal(c, float64(25), state.Value.Live.InexactFloat64())
	}, 30*time.Second, 100*time.Millisecond)
	// Activating the key makes subsequent grants refresh state without another pull.
	require.NoError(t, credits.Activate(ctx, key, 0))
	_, err = api.Customers.Credits.Grants.Create(ctx, cust.ID, v3sdk.CreateCreditGrantRequest{
		Name: "tracked credit", Currency: "USD", Amount: "10", FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	api.requireStatus(http.StatusCreated, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		state, err := credits.Get(ctx, key)
		require.NoError(c, err)
		require.Equal(c, float64(35), state.Value.Live.InexactFloat64())
	}, 30*time.Second, 100*time.Millisecond)

	require.NoError(t, credits.Deactivate(ctx, key))
	retained, err := credits.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, float64(35), retained.Value.Live.InexactFloat64())
}
