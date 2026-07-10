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

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	require.NotNilf(t, e.db, "database must be initialized")

	err := e.db.EntDriver.Client().Schema.Create(t.Context())
	require.NoErrorf(t, err, "schema migration must not fail")
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
// generated invoicing/credit-grant tax codes. Zero-value fields keep the built-in defaults.
type ProvisionDefaultTaxCodesInput struct {
	InvoicingAppMappings   taxcode.TaxCodeAppMappings
	CreditGrantAppMappings taxcode.TaxCodeAppMappings
}

// ProvisionDefaultTaxCodes creates (if necessary) the invoicing and credit-grant tax codes for
// namespace and stores them as the namespace's organization defaults. Tests that create charges
// via the real charges service must call this for the namespace, because charge creation
// auto-stamps the namespace's default tax code when the caller's TaxConfig has no TaxCodeID.
// Idempotent — safe to call more than once for the same namespace, e.g. after
// ProvisionProviderDefaultTaxCode was already called standalone for it.
//
// opts optionally overrides the app mappings placed on the invoicing/credit-grant tax codes, e.g.
// for tests asserting that TaxConfig.Stripe is backfilled from the namespace's default tax codes.
// App mappings apply only when this call creates the tax code; a pre-existing row is reused as-is.
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
// invoicing app omits an app-specific provider code for namespace. This is distinct from
// organization default tax-code settings: API invoice edit diffing needs the provider-default tax
// code row to resolve empty provider tax config, but it must not imply that the namespace has
// configured org-level default tax codes. Idempotent.
func (e *TestEnv) ProvisionProviderDefaultTaxCode(t *testing.T, namespace string) taxcode.TaxCode {
	t.Helper()
	return e.getOrCreateTaxCodeByKey(t, namespace, taxcode.ProviderDefaultTaxCodeKey, "Provider Default", nil)
}

// getOrCreateTaxCodeByKey returns the tax code identified by (namespace, key), creating it with
// name and appMappings if it doesn't exist yet. Idempotent: the (namespace, key) unique index only
// applies to non-deleted rows, so this is safe to call more than once for the same namespace/key.
func (e *TestEnv) getOrCreateTaxCodeByKey(t *testing.T, namespace, key, name string, appMappings taxcode.TaxCodeAppMappings) taxcode.TaxCode {
	t.Helper()
	ctx := t.Context()

	tc, err := e.Service.GetTaxCodeByKey(ctx, taxcode.GetTaxCodeByKeyInput{
		Namespace: namespace,
		Key:       key,
	})
	if err == nil {
		return tc
	}
	require.True(t, taxcode.IsTaxCodeNotFoundError(err), "getting tax code by key should either succeed or return not found")

	tc, err = e.Service.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
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

	db := testutils.InitPostgresDB(t)
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
