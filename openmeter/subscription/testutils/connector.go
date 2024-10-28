package subscriptiontestutils

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/meter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

type deps struct {
	PlanAdapter        *planAdapter
	CustomerAdapter    *testCustomerRepo
	FeatureConnector   *testFeatureConnector
	EntitlementAdapter subscription.EntitlementAdapter
}

func NewCommandAndQuery(t *testing.T, dbDeps *DBDeps) (subscription.Command, subscription.Query, *deps) {
	t.Helper()
	logger := testutils.NewLogger(t)
	subRepo := NewRepo(t, dbDeps)
	priceConnector := NewPriceConnector(t, dbDeps)
	meterRepo := meter.NewInMemoryRepository([]models.Meter{{
		Slug:        ExampleFeatureMeterSlug,
		Namespace:   ExampleNamespace,
		Aggregation: models.MeterAggregationSum,
		WindowSize:  models.WindowSizeMinute,
	}})

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.dbClient,
		StreamingConnector: streamingtestutils.NewMockStreamingConnector(t),
		Logger:             logger,
		MeterRepository:    meterRepo,
		Publisher:          eventbus.NewMock(t),
	})

	subscriptionEntitlementRepo := subscriptionentitlement.NewRepository(dbDeps.dbClient)
	entitlementAdapter := subscriptionentitlement.NewEntitlementSubscriptionAdapter(
		entitlementRegistry.Entitlement,
		subscriptionEntitlementRepo,
		subscriptionEntitlementRepo,
	)

	planAdapter := NewMockPlanAdapter(t)

	customerAdapter := NewCustomerAdapter(t, dbDeps)
	customer := NewCustomerService(t, dbDeps)

	connector := subscription.NewConnector(
		subRepo,
		priceConnector,
		customer,
		planAdapter,
		entitlementAdapter,
		subscriptionEntitlementRepo,
	)

	return connector, connector, &deps{
		PlanAdapter:        planAdapter,
		CustomerAdapter:    customerAdapter,
		FeatureConnector:   NewTestFeatureConnector(entitlementRegistry.Feature),
		EntitlementAdapter: entitlementAdapter,
	}
}
