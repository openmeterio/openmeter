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

// ProvisionDefaultTaxCodesInput overrides the app mappings ProvisionDefaultTaxCodes puts on the
// invoicing/credit-grant tax codes. Nil fields keep the built-in defaults.
type ProvisionDefaultTaxCodesInput struct {
	InvoicingAppMappings   taxcode.TaxCodeAppMappings
	CreditGrantAppMappings taxcode.TaxCodeAppMappings
}

// ProvisionDefaultTaxCodes creates (if necessary) the invoicing and credit-grant tax codes for
// namespace and stores them as the namespace's organization defaults. Charge creation stamps the
// namespace's default tax code when the caller provides none, so tests creating charges through
// the real service must call this first. Idempotent: existing tax codes are reused by key, so it
// is safe after ProvisionProviderDefaultTaxCode or a previous call for the same namespace.
//
// App mapping overrides apply only when this call creates the tax code.
func (e *TestEnv) ProvisionDefaultTaxCodes(t *testing.T, namespace string, opts ...ProvisionDefaultTaxCodesInput) taxcode.OrganizationDefaultTaxCodes {
	t.Helper()

	var input ProvisionDefaultTaxCodesInput
	if len(opts) > 0 {
		input = opts[0]
	}

	creditGrantAppMappings := input.CreditGrantAppMappings
	if creditGrantAppMappings == nil {
		creditGrantAppMappings = taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_00000000"},
		}
	}

	invoicing := e.getOrCreateTaxCodeByKey(t, namespace, taxcode.ProviderDefaultTaxCodeKey, "Provider Default", input.InvoicingAppMappings)
	creditGrant := e.getOrCreateTaxCodeByKey(t, namespace, taxcode.CreditGrantTaxCodeKey, "Non-Taxable", creditGrantAppMappings)

	defaults, err := e.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
		Namespace:            namespace,
		InvoicingTaxCodeID:   invoicing.ID,
		CreditGrantTaxCodeID: creditGrant.ID,
	})
	require.NoError(t, err, "upserting organization default tax codes")

	return defaults
}

// ProvisionProviderDefaultTaxCode returns (creating if necessary) the tax code used when an
// invoicing app omits an app-specific provider code. Unlike ProvisionDefaultTaxCodes it does not
// configure organization defaults: invoice edit diffing needs this row to resolve an empty provider
// tax config, but the namespace must still look like it has no org-level defaults. Idempotent.
func (e *TestEnv) ProvisionProviderDefaultTaxCode(t *testing.T, namespace string) taxcode.TaxCode {
	t.Helper()

	return e.getOrCreateTaxCodeByKey(t, namespace, taxcode.ProviderDefaultTaxCodeKey, "Provider Default", nil)
}

func (e *TestEnv) getOrCreateTaxCodeByKey(t *testing.T, namespace, key, name string, appMappings taxcode.TaxCodeAppMappings) taxcode.TaxCode {
	t.Helper()

	tc, err := e.Service.GetTaxCodeByKey(t.Context(), taxcode.GetTaxCodeByKeyInput{
		Namespace: namespace,
		Key:       key,
	})
	if err == nil {
		return tc
	}

	require.True(t, taxcode.IsTaxCodeNotFoundError(err), "getting tax code by key should either succeed or return not found: %v", err)

	tc, err = e.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace:   namespace,
		Key:         key,
		Name:        name,
		AppMappings: appMappings,
	})
	require.NoError(t, err, "creating tax code")

	return tc
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
