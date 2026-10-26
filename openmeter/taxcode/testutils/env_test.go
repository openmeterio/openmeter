package testutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestProvisionDefaultTaxCodes(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)

	t.Run("StoresOrganizationDefaults", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key

		defaults := env.ProvisionDefaultTaxCodes(t, ns)

		stored, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: ns})
		require.NoError(t, err)
		assert.Equal(t, defaults.InvoicingTaxCodeID, stored.InvoicingTaxCodeID)
		assert.Equal(t, defaults.CreditGrantTaxCodeID, stored.CreditGrantTaxCodeID)

		invoicing, err := env.Service.GetTaxCodeByKey(t.Context(), taxcode.GetTaxCodeByKeyInput{Namespace: ns, Key: taxcode.ProviderDefaultTaxCodeKey})
		require.NoError(t, err)
		assert.Equal(t, invoicing.ID, defaults.InvoicingTaxCodeID)
	})

	t.Run("Idempotent", func(t *testing.T) {
		// given: a namespace whose defaults were already provisioned
		ns := testutils.NameGenerator.Generate().Key
		first := env.ProvisionDefaultTaxCodes(t, ns)

		// when: provisioning again
		second := env.ProvisionDefaultTaxCodes(t, ns)

		// then: the same tax codes are reused
		assert.Equal(t, first.InvoicingTaxCodeID, second.InvoicingTaxCodeID)
		assert.Equal(t, first.CreditGrantTaxCodeID, second.CreditGrantTaxCodeID)
	})

	t.Run("ReusesStandaloneProviderDefault", func(t *testing.T) {
		// given: the provider default tax code exists without organization defaults
		ns := testutils.NameGenerator.Generate().Key
		providerDefault := env.ProvisionProviderDefaultTaxCode(t, ns)

		// when: provisioning organization defaults
		defaults := env.ProvisionDefaultTaxCodes(t, ns)

		// then: the provider default becomes the invoicing default
		assert.Equal(t, providerDefault.ID, defaults.InvoicingTaxCodeID)
	})

	t.Run("AppMappingOverrides", func(t *testing.T) {
		ns := testutils.NameGenerator.Generate().Key

		defaults := env.ProvisionDefaultTaxCodes(t, ns, taxcodetestutils.ProvisionDefaultTaxCodesInput{
			InvoicingAppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
			},
		})

		invoicing, err := env.Service.GetTaxCode(t.Context(), taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: defaults.InvoicingTaxCodeID},
		})
		require.NoError(t, err)
		code, ok := invoicing.GetAppMapping(app.AppTypeStripe)
		require.True(t, ok)
		assert.Equal(t, "txcd_10000000", code.TaxCode)
	})
}
