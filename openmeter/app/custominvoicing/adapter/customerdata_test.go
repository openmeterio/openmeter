package adapter

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustominvoicingcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicingcustomer"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type customerDataTestEnv struct {
	db      *db.Client
	adapter *adapter
}

func newCustomerDataTestEnv(t *testing.T) customerDataTestEnv {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	customInvoicingAdapter, err := New(Config{
		Client: dbClient,
		Logger: testutils.NewLogger(t),
	})
	require.NoError(t, err)

	return customerDataTestEnv{
		db:      dbClient,
		adapter: customInvoicingAdapter.(*adapter),
	}
}

func (e customerDataTestEnv) createApp(t *testing.T, namespace string) app.AppID {
	t.Helper()

	appEntity, err := e.db.App.Create().
		SetNamespace(namespace).
		SetName("custom-invoicing-" + ulid.Make().String()).
		SetType(app.AppTypeCustomInvoicing).
		SetStatus(app.AppStatusReady).
		Save(t.Context())
	require.NoError(t, err)

	_, err = e.db.AppCustomInvoicing.Create().
		SetID(appEntity.ID).
		SetNamespace(namespace).
		Save(t.Context())
	require.NoError(t, err)

	return app.AppID{Namespace: namespace, ID: appEntity.ID}
}

func (e customerDataTestEnv) createCustomer(t *testing.T, namespace string) customer.CustomerID {
	t.Helper()

	entity, err := e.db.Customer.Create().
		SetNamespace(namespace).
		SetName("customer-" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	return customer.CustomerID{Namespace: namespace, ID: entity.ID}
}

func TestUpsertCustomerDataResolvesReferencesInNamespace(t *testing.T) {
	// given:
	// - a custom-invoicing app and customer in a target namespace
	// - another app and customer whose IDs exist only in a foreign namespace
	// when:
	// - custom-invoicing customer data is upserted with target-namespace wrappers
	// then:
	// - only references whose actual rows belong to the target namespace are accepted
	env := newCustomerDataTestEnv(t)
	targetNamespace := ulid.Make().String()
	foreignNamespace := ulid.Make().String()
	targetApp := env.createApp(t, targetNamespace)
	foreignApp := env.createApp(t, foreignNamespace)
	targetCustomer := env.createCustomer(t, targetNamespace)
	foreignCustomer := env.createCustomer(t, foreignNamespace)

	err := env.adapter.UpsertCustomerData(t.Context(), appcustominvoicing.UpsertCustomerDataInput{
		CustomerDataID: appcustominvoicing.CustomerDataID{
			Namespace:  targetNamespace,
			AppID:      targetApp.ID,
			CustomerID: targetCustomer.ID,
		},
		Data: appcustominvoicing.CustomerData{
			Metadata: models.Metadata{"reference": "target"},
		},
	})
	require.NoError(t, err)

	exists, err := env.db.AppCustomInvoicingCustomer.Query().
		Where(appcustominvoicingcustomerdb.Namespace(targetNamespace)).
		Where(appcustominvoicingcustomerdb.AppID(targetApp.ID)).
		Where(appcustominvoicingcustomerdb.CustomerID(targetCustomer.ID)).
		Exist(t.Context())
	require.NoError(t, err)
	require.True(t, exists)

	for _, testCase := range []struct {
		name        string
		appID       string
		customerID  string
		appNotFound bool
	}{
		{
			name:        "foreign app",
			appID:       foreignApp.ID,
			customerID:  targetCustomer.ID,
			appNotFound: true,
		},
		{
			name:        "missing app",
			appID:       ulid.Make().String(),
			customerID:  targetCustomer.ID,
			appNotFound: true,
		},
		{
			name:       "foreign customer",
			appID:      targetApp.ID,
			customerID: foreignCustomer.ID,
		},
		{
			name:       "missing customer",
			appID:      targetApp.ID,
			customerID: ulid.Make().String(),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := env.adapter.UpsertCustomerData(t.Context(), appcustominvoicing.UpsertCustomerDataInput{
				CustomerDataID: appcustominvoicing.CustomerDataID{
					Namespace:  targetNamespace,
					AppID:      testCase.appID,
					CustomerID: testCase.customerID,
				},
				Data: appcustominvoicing.CustomerData{},
			})
			require.Error(t, err)
			require.True(t, models.IsGenericNotFoundError(err))
			if testCase.appNotFound {
				require.True(t, app.IsAppNotFoundError(err))
			}

			exists, err := env.db.AppCustomInvoicingCustomer.Query().
				Where(appcustominvoicingcustomerdb.Namespace(targetNamespace)).
				Where(appcustominvoicingcustomerdb.AppID(testCase.appID)).
				Where(appcustominvoicingcustomerdb.CustomerID(testCase.customerID)).
				Exist(t.Context())
			require.NoError(t, err)
			require.False(t, exists)
		})
	}
}
