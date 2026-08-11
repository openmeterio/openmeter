package appadapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const customerTestAppType = app.AppTypeSandbox

type customerTestEnv struct {
	db      *db.Client
	adapter app.Adapter
}

func newCustomerTestEnv(t *testing.T) customerTestEnv {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	adapter, err := appadapter.New(appadapter.Config{Client: dbClient})
	require.NoError(t, err)

	err = adapter.RegisterMarketplaceListing(app.RegistryItem{
		Listing: app.MarketplaceListing{
			Type:        customerTestAppType,
			Name:        "Customer test",
			Description: "App customer namespace-isolation tests",
		},
		Factory: customerTestFactory{},
	})
	require.NoError(t, err)

	return customerTestEnv{
		db:      dbClient,
		adapter: adapter,
	}
}

func (e customerTestEnv) createApp(t *testing.T, namespace string) app.AppID {
	t.Helper()

	entity, err := e.db.App.Create().
		SetNamespace(namespace).
		SetName("app-" + ulid.Make().String()).
		SetType(customerTestAppType).
		SetStatus(app.AppStatusReady).
		Save(t.Context())
	require.NoError(t, err)

	return app.AppID{Namespace: namespace, ID: entity.ID}
}

func (e customerTestEnv) createCustomer(t *testing.T, namespace string) customer.CustomerID {
	t.Helper()

	entity, err := e.db.Customer.Create().
		SetNamespace(namespace).
		SetName("customer-" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	return customer.CustomerID{Namespace: namespace, ID: entity.ID}
}

func TestEnsureCustomerResolvesReferencesInNamespace(t *testing.T) {
	// given:
	// - an app and a customer in a target namespace
	// - another app and customer whose IDs exist only in a foreign namespace
	// when:
	// - app-customer relationships are created with target-namespace wrappers
	// then:
	// - only references whose actual rows belong to the target namespace are accepted
	env := newCustomerTestEnv(t)
	targetNamespace := ulid.Make().String()
	foreignNamespace := ulid.Make().String()
	targetApp := env.createApp(t, targetNamespace)
	targetCustomer := env.createCustomer(t, targetNamespace)
	foreignApp := env.createApp(t, foreignNamespace)
	foreignCustomer := env.createCustomer(t, foreignNamespace)

	err := env.adapter.EnsureCustomer(t.Context(), app.EnsureCustomerInput{
		AppID:      targetApp,
		CustomerID: targetCustomer,
	})
	require.NoError(t, err)

	for _, testCase := range []struct {
		name       string
		appID      app.AppID
		customerID customer.CustomerID
	}{
		{
			name:       "foreign app",
			appID:      app.AppID{Namespace: targetNamespace, ID: foreignApp.ID},
			customerID: targetCustomer,
		},
		{
			name:  "foreign customer",
			appID: targetApp,
			customerID: customer.CustomerID{
				Namespace: targetNamespace,
				ID:        foreignCustomer.ID,
			},
		},
		{
			name:       "missing app",
			appID:      app.AppID{Namespace: targetNamespace, ID: ulid.Make().String()},
			customerID: targetCustomer,
		},
		{
			name:  "missing customer",
			appID: targetApp,
			customerID: customer.CustomerID{
				Namespace: targetNamespace,
				ID:        ulid.Make().String(),
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := env.adapter.EnsureCustomer(t.Context(), app.EnsureCustomerInput{
				AppID:      testCase.appID,
				CustomerID: testCase.customerID,
			})
			require.Error(t, err)
			require.True(t, models.IsGenericNotFoundError(err))

			exists, err := env.db.AppCustomer.Query().
				Where(appcustomerdb.Namespace(targetNamespace)).
				Where(appcustomerdb.AppID(testCase.appID.ID)).
				Where(appcustomerdb.CustomerID(testCase.customerID.ID)).
				Exist(t.Context())
			require.NoError(t, err)
			require.False(t, exists)
		})
	}
}

func TestEnsureCustomerRestoresDeletedRelationship(t *testing.T) {
	// given:
	// - an app and customer with a soft-deleted AppCustomer relationship
	// when:
	// - the relationship is ensured again
	// then:
	// - exactly one active relationship exists
	env := newCustomerTestEnv(t)
	namespace := ulid.Make().String()
	appID := env.createApp(t, namespace)
	customerID := env.createCustomer(t, namespace)

	_, err := env.db.AppCustomer.Create().
		SetNamespace(namespace).
		SetAppID(appID.ID).
		SetCustomerID(customerID.ID).
		SetDeletedAt(time.Now()).
		Save(t.Context())
	require.NoError(t, err)

	err = env.adapter.EnsureCustomer(t.Context(), app.EnsureCustomerInput{
		AppID:      appID,
		CustomerID: customerID,
	})
	require.NoError(t, err)

	activeRelationshipCount, err := env.db.AppCustomer.Query().
		Where(appcustomerdb.Namespace(namespace)).
		Where(appcustomerdb.AppID(appID.ID)).
		Where(appcustomerdb.CustomerID(customerID.ID)).
		Where(appcustomerdb.DeletedAtIsNil()).
		Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, activeRelationshipCount)
}

func TestListAppsIgnoresForeignNamespaceCustomerRelationships(t *testing.T) {
	// given:
	// - an app and customer in a target namespace
	// - a deliberately corrupted AppCustomer row carrying another namespace
	// when:
	// - apps are listed for the target customer
	// then:
	// - the foreign-namespace relationship does not make the app eligible
	env := newCustomerTestEnv(t)
	targetNamespace := ulid.Make().String()
	foreignNamespace := ulid.Make().String()
	targetApp := env.createApp(t, targetNamespace)
	targetCustomer := env.createCustomer(t, targetNamespace)

	_, err := env.db.AppCustomer.Create().
		SetNamespace(foreignNamespace).
		SetAppID(targetApp.ID).
		SetCustomerID(targetCustomer.ID).
		Save(t.Context())
	require.NoError(t, err)

	result, err := env.adapter.ListApps(t.Context(), app.ListAppInput{
		Namespace:  targetNamespace,
		CustomerID: &targetCustomer,
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10,
		},
	})
	require.NoError(t, err)
	require.Empty(t, result.Items)
	require.Zero(t, result.TotalCount)
}

type customerTestFactory struct{}

func (customerTestFactory) NewApp(_ context.Context, appBase app.AppBase) (app.App, error) {
	return customerTestApp{AppBase: appBase}, nil
}

func (customerTestFactory) UninstallApp(context.Context, app.UninstallAppInput) error {
	return nil
}

type customerTestApp struct {
	app.AppBase
}

func (customerTestApp) GetEventAppData() (app.EventAppData, error) {
	return app.EventAppData{}, nil
}

func (customerTestApp) UpdateAppConfig(context.Context, app.AppConfigUpdate) error {
	return nil
}

func (customerTestApp) GetCustomerData(context.Context, app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return nil, nil
}

func (customerTestApp) UpsertCustomerData(context.Context, app.UpsertAppInstanceCustomerDataInput) error {
	return nil
}

func (customerTestApp) DeleteCustomerData(context.Context, app.DeleteAppInstanceCustomerDataInput) error {
	return nil
}
