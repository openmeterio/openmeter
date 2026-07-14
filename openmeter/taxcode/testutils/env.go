package testutils

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
	taxcodeservice "github.com/openmeterio/openmeter/openmeter/taxcode/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type TestEnv struct {
	Logger  *slog.Logger
	Service taxcode.Service
	Adapter taxcode.Repository
	Client  *entdb.Client
	db      *testutils.TestDB
	close   sync.Once
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		// If we are not owning the test database, we should not do cleanup here.
		if e.db == nil {
			return
		}
		if e.Client != nil {
			if err := e.Client.Close(); err != nil {
				t.Errorf("failed to close ent client: %v", err)
			}
		}

		if err := e.db.EntDriver.Close(); err != nil {
			t.Errorf("failed to close ent driver: %v", err)
		}

		if err := e.db.PGDriver.Close(); err != nil {
			t.Errorf("failed to close postgres driver: %v", err)
		}
	})
}

// CreateTaxCode creates a tax code; if opts is provided its first element overrides the
// input — Namespace, Key, and Name are generated when empty.
func (e *TestEnv) CreateTaxCode(t *testing.T, namespace string, opts ...taxcode.CreateTaxCodeInput) taxcode.TaxCode {
	t.Helper()
	var input taxcode.CreateTaxCodeInput
	if len(opts) > 0 {
		input = opts[0]
	}
	generated := testutils.NameGenerator.Generate()
	input.Namespace = namespace
	if input.Key == "" {
		input.Key = generated.Key
	}
	if input.Name == "" {
		input.Name = generated.Name
	}
	tc, err := e.Service.CreateTaxCode(t.Context(), input)
	require.NoError(t, err)
	return tc
}

// SetupNamespaceDefaults provisions two seed tax codes and upserts the org-default tax codes for namespace.
func (e *TestEnv) SetupNamespaceDefaults(t *testing.T, namespace string) {
	t.Helper()
	invoicing := e.CreateTaxCode(t, namespace, taxcode.CreateTaxCodeInput{
		Key:  taxcode.ProviderDefaultTaxCodeKey,
		Name: "Provider Default",
	})
	creditGrant := e.CreateTaxCode(t, namespace, taxcode.CreateTaxCodeInput{
		Name: "Non-Taxable",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_00000000"},
		},
	})
	_, err := e.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
		Namespace:            namespace,
		InvoicingTaxCodeID:   invoicing.ID,
		CreditGrantTaxCodeID: creditGrant.ID,
	})
	require.NoError(t, err)
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	logger := testutils.NewDiscardLogger(t)

	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	client := db.EntDriver.Client()

	env := NewTestEnvFromClient(t, client, logger)
	env.db = db
	t.Cleanup(func() { env.Close(t) })

	return env
}

func NewTestEnvFromClient(t *testing.T, client *entdb.Client, logger *slog.Logger) *TestEnv {
	t.Helper()

	require.NotNil(t, client)
	if logger == nil {
		logger = testutils.NewDiscardLogger(t)
	}

	env := &TestEnv{
		Logger: logger,
		Client: client,
	}

	adapter, err := taxcodeadapter.New(taxcodeadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing taxcode adapter must not fail")

	svc, err := taxcodeservice.New(taxcodeservice.Config{
		Adapter: adapter,
		Logger:  logger,
	})
	require.NoErrorf(t, err, "initializing taxcode service must not fail")

	env.Adapter = adapter
	env.Service = svc

	return env
}
