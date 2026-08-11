package appstripeadapter

import (
	"context"
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type stripeCustomerTestEnv struct {
	db           *db.Client
	sqlDB        *sql.DB
	adapter      *adapter
	stripeClient *stripeCustomerClientStub
}

func newStripeCustomerTestEnv(t *testing.T) stripeCustomerTestEnv {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	logger := testutils.NewLogger(t)
	publisher := eventbus.NewMock(t)

	appAdapter, err := appadapter.New(appadapter.Config{Client: dbClient})
	require.NoError(t, err)

	appService, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	require.NoError(t, err)

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: logger,
	})
	require.NoError(t, err)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: publisher,
	})
	require.NoError(t, err)

	secretService, err := secretservice.New(secretservice.Config{Adapter: secretadapter.New()})
	require.NoError(t, err)

	stripeClient := &stripeCustomerClientStub{}
	stripeAdapter, err := New(Config{
		Client:               dbClient,
		AppService:           appService,
		CustomerService:      customerService,
		SecretService:        secretService,
		WebhookSecretService: secretService,
		StripeAppClientFactory: func(stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return stripeClient, nil
		},
		Logger: logger,
	})
	require.NoError(t, err)

	return stripeCustomerTestEnv{
		db:           dbClient,
		sqlDB:        testDB.PGDriver.DB(),
		adapter:      stripeAdapter.(*adapter),
		stripeClient: stripeClient,
	}
}

func (e stripeCustomerTestEnv) createStripeApp(t *testing.T, namespace string) app.AppID {
	t.Helper()

	appEntity, err := e.db.App.Create().
		SetNamespace(namespace).
		SetName("stripe-app-" + ulid.Make().String()).
		SetType(app.AppTypeStripe).
		SetStatus(app.AppStatusReady).
		Save(t.Context())
	require.NoError(t, err)

	_, err = e.db.AppStripe.Create().
		SetID(appEntity.ID).
		SetNamespace(namespace).
		SetStripeAccountID("acct_" + ulid.Make().String()).
		SetStripeLivemode(false).
		SetAPIKey("sk_test_" + ulid.Make().String()).
		SetMaskedAPIKey("****").
		SetStripeWebhookID("we_" + ulid.Make().String()).
		SetWebhookSecret("whsec_" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	return app.AppID{Namespace: namespace, ID: appEntity.ID}
}

func (e stripeCustomerTestEnv) createCustomer(t *testing.T, namespace string) customer.CustomerID {
	t.Helper()

	entity, err := e.db.Customer.Create().
		SetNamespace(namespace).
		SetName("customer-" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	return customer.CustomerID{Namespace: namespace, ID: entity.ID}
}

func TestUpsertStripeCustomerDataRevalidatesStripeAppNamespaceBeforeWrite(t *testing.T) {
	// given:
	// - a Stripe app and customer in the same namespace
	// - the Stripe provider row moves to another namespace after client resolution
	// when:
	// - Stripe customer data is upserted
	// then:
	// - the provider app is reported as not found and no customer relationship is written
	env := newStripeCustomerTestEnv(t)
	targetNamespace := ulid.Make().String()
	foreignNamespace := ulid.Make().String()
	appID := env.createStripeApp(t, targetNamespace)
	customerID := env.createCustomer(t, targetNamespace)
	stripeCustomerID := "cus_" + ulid.Make().String()

	env.stripeClient.getCustomer = func(ctx context.Context, id string) (stripeclient.StripeCustomer, error) {
		result, err := env.sqlDB.ExecContext(ctx, `UPDATE app_stripes SET namespace = $1 WHERE id = $2`, foreignNamespace, appID.ID)
		require.NoError(t, err)
		rowsAffected, err := result.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, rowsAffected)

		return stripeclient.StripeCustomer{StripeCustomerID: id}, nil
	}

	err := env.adapter.UpsertStripeCustomerData(t.Context(), appstripe.UpsertStripeCustomerDataInput{
		AppID:            appID,
		CustomerID:       customerID,
		StripeCustomerID: stripeCustomerID,
	})
	require.Error(t, err)
	require.True(t, app.IsAppNotFoundError(err))

	stripeCustomerCount, err := env.db.AppStripeCustomer.Query().
		Where(appstripecustomerdb.AppID(appID.ID)).
		Where(appstripecustomerdb.CustomerID(customerID.ID)).
		Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, stripeCustomerCount)

	appCustomerCount, err := env.db.AppCustomer.Query().
		Where(appcustomerdb.AppID(appID.ID)).
		Where(appcustomerdb.CustomerID(customerID.ID)).
		Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, appCustomerCount)
}

type stripeCustomerClientStub struct {
	stripeclient.StripeAppClient

	getCustomer func(context.Context, string) (stripeclient.StripeCustomer, error)
}

func (c stripeCustomerClientStub) GetCustomer(ctx context.Context, stripeCustomerID string) (stripeclient.StripeCustomer, error) {
	return c.getCustomer(ctx, stripeCustomerID)
}
