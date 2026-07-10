package taxcode_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// makeTestSeeds returns two seed entries: "default" (invoicing) and "nontaxable"
// (credit grant). Each test that needs different seeds can build its own, but
// most tests share this helper.
func makeTestSeeds() []taxcode.SeedEntry {
	return []taxcode.SeedEntry{
		{
			Key:              taxcode.ProviderDefaultTaxCodeKey,
			Name:             "Default Tax",
			DefaultInvoicing: true,
		},
		{
			Key:                taxcode.CreditGrantTaxCodeKey,
			Name:               "Non-Taxable",
			DefaultCreditGrant: true,
		},
	}
}

func TestNamespaceHandler(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	makeHandler := func(t *testing.T, seeds []taxcode.SeedEntry) *taxcode.NamespaceHandler {
		t.Helper()
		h, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Logger:             env.Logger,
			Service:            env.Service,
			Seeds:              seeds,
			TransactionManager: env.Adapter,
		})
		require.NoError(t, err)
		return h
	}

	t.Run("FreshNamespace", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key
		h := makeHandler(t, makeTestSeeds())

		err := h.CreateNamespace(t.Context(), ns)
		require.NoError(t, err)

		// Both tax codes must exist.
		result, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
			Namespace: ns,
			Page:      pagination.Page{PageSize: 100, PageNumber: 1},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)

		keyToTC := make(map[string]taxcode.TaxCode, 2)
		for _, tc := range result.Items {
			keyToTC[tc.Key] = tc
		}

		defaultTC, ok := keyToTC[taxcode.ProviderDefaultTaxCodeKey]
		require.True(t, ok, "default tax code must exist")
		assert.True(t, defaultTC.IsManagedBySystem(), "default tax code must be managed by system")

		nontaxableTC, ok := keyToTC[taxcode.CreditGrantTaxCodeKey]
		require.True(t, ok, "nontaxable tax code must exist")
		assert.True(t, nontaxableTC.IsManagedBySystem(), "nontaxable tax code must be managed by system")

		// Org defaults must reference both.
		defaults, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		assert.Equal(t, defaultTC.ID, defaults.InvoicingTaxCodeID)
		assert.Equal(t, nontaxableTC.ID, defaults.CreditGrantTaxCodeID)
	})

	t.Run("PreExistingTaxCode", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key

		// Pre-seed a "default" tax code with a different name and no annotations.
		preExisting, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       taxcode.ProviderDefaultTaxCodeKey,
			Name:      "Pre-Existing Default",
		})
		require.NoError(t, err)
		assert.False(t, preExisting.IsManagedBySystem())

		h := makeHandler(t, makeTestSeeds())
		err = h.CreateNamespace(t.Context(), ns)
		require.NoError(t, err)

		// The pre-existing "default" must be untouched (same ID, same name).
		result, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
			Namespace: ns,
			Page:      pagination.Page{PageSize: 100, PageNumber: 1},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)

		keyToTC := make(map[string]taxcode.TaxCode, 2)
		for _, tc := range result.Items {
			keyToTC[tc.Key] = tc
		}

		gotDefault, ok := keyToTC[taxcode.ProviderDefaultTaxCodeKey]
		require.True(t, ok)
		assert.Equal(t, preExisting.ID, gotDefault.ID, "pre-existing ID must not change")
		assert.Equal(t, "Pre-Existing Default", gotDefault.Name, "pre-existing name must not change")
		// Note: not managed by system because we didn't add annotation when pre-creating.
		assert.False(t, gotDefault.IsManagedBySystem())

		gotNontaxable, ok := keyToTC[taxcode.CreditGrantTaxCodeKey]
		require.True(t, ok, "nontaxable must be freshly created")
		assert.True(t, gotNontaxable.IsManagedBySystem())

		// Org defaults must point at the pre-existing default and the new nontaxable.
		defaults, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		assert.Equal(t, preExisting.ID, defaults.InvoicingTaxCodeID)
		assert.Equal(t, gotNontaxable.ID, defaults.CreditGrantTaxCodeID)
	})

	t.Run("PreExistingOrgDefaults", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key

		// Pre-seed both tax codes and a complete org defaults row.
		defaultTC, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       taxcode.ProviderDefaultTaxCodeKey,
			Name:      "Default Tax",
			Annotations: models.Annotations{
				taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
			},
		})
		require.NoError(t, err)

		nontaxableTC, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       taxcode.CreditGrantTaxCodeKey,
			Name:      "Non-Taxable",
			Annotations: models.Annotations{
				taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
			},
		})
		require.NoError(t, err)

		preDefaults, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
			Namespace:            ns,
			InvoicingTaxCodeID:   defaultTC.ID,
			CreditGrantTaxCodeID: nontaxableTC.ID,
		})
		require.NoError(t, err)

		h := makeHandler(t, makeTestSeeds())
		err = h.CreateNamespace(t.Context(), ns)
		require.NoError(t, err)

		// Org defaults must be unchanged.
		afterDefaults, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		assert.Equal(t, preDefaults.ID, afterDefaults.ID, "org defaults row ID must not change")
		assert.Equal(t, preDefaults.InvoicingTaxCodeID, afterDefaults.InvoicingTaxCodeID)
		assert.Equal(t, preDefaults.CreditGrantTaxCodeID, afterDefaults.CreditGrantTaxCodeID)
	})

	t.Run("Idempotency", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key
		h := makeHandler(t, makeTestSeeds())

		// First call.
		err := h.CreateNamespace(t.Context(), ns)
		require.NoError(t, err)

		firstDefaults, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		require.NoError(t, err)

		// Second call must be a no-op.
		err = h.CreateNamespace(t.Context(), ns)
		require.NoError(t, err)

		secondDefaults, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		require.NoError(t, err)

		assert.Equal(t, firstDefaults.ID, secondDefaults.ID)
		assert.Equal(t, firstDefaults.InvoicingTaxCodeID, secondDefaults.InvoicingTaxCodeID)
		assert.Equal(t, firstDefaults.CreditGrantTaxCodeID, secondDefaults.CreditGrantTaxCodeID)
		assert.Equal(t, firstDefaults.CreatedAt, secondDefaults.CreatedAt, "created_at must not move on second call")

		// Only 2 tax codes must exist.
		result, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
			Namespace: ns,
			Page:      pagination.Page{PageSize: 100, PageNumber: 1},
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})
}

func TestNewNamespaceHandler_Validation(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	validSeeds := makeTestSeeds()

	t.Run("MissingLogger", func(t *testing.T) {
		_, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Service: env.Service,
			Seeds:   validSeeds,
		})
		require.Error(t, err)
	})

	t.Run("MissingService", func(t *testing.T) {
		_, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Logger: env.Logger,
			Seeds:  validSeeds,
		})
		require.Error(t, err)
	})

	t.Run("EmptySeeds", func(t *testing.T) {
		_, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Logger:  env.Logger,
			Service: env.Service,
			Seeds:   nil,
		})
		require.Error(t, err)
	})

	t.Run("MissingTransactionManager", func(t *testing.T) {
		_, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Logger:  env.Logger,
			Service: env.Service,
			Seeds:   validSeeds,
		})
		require.Error(t, err)
	})

	t.Run("ValidConfig", func(t *testing.T) {
		h, err := taxcode.NewNamespaceHandler(taxcode.NamespaceHandlerConfig{
			Logger:             env.Logger,
			Service:            env.Service,
			Seeds:              validSeeds,
			TransactionManager: env.Adapter,
		})
		require.NoError(t, err)
		require.NotNil(t, h)
	})
}
