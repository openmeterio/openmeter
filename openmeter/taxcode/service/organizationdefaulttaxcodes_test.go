package service_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestOrganizationDefaultTaxCodesService(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	ns := testutils.NameGenerator.Generate().Key

	// Helper: create a tax code in the given namespace.
	createTaxCode := func(t *testing.T, namespace string) taxcode.TaxCode {
		t.Helper()
		name := testutils.NameGenerator.Generate()
		tc, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: namespace,
			Key:       name.Key,
			Name:      name.Name,
		})
		require.NoError(t, err)
		return tc
	}

	t.Run("Get", func(t *testing.T) {
		t.Run("ValidationError/EmptyNamespace", func(t *testing.T) {
			_, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{})
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})

		t.Run("NotFound", func(t *testing.T) {
			_, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
				Namespace: testutils.NameGenerator.Generate().Key,
			})
			require.Error(t, err)
			assert.True(t, taxcode.IsOrganizationDefaultTaxCodesNotFoundError(err))
		})
	})

	t.Run("Upsert", func(t *testing.T) {
		t.Run("ValidationError/EmptyNamespace", func(t *testing.T) {
			tc := createTaxCode(t, ns)
			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				InvoicingTaxCodeID:   tc.ID,
				CreditGrantTaxCodeID: tc.ID,
			})
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})

		t.Run("ValidationError/EmptyInvoicingTaxCodeID", func(t *testing.T) {
			tc := createTaxCode(t, ns)
			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns,
				CreditGrantTaxCodeID: tc.ID,
			})
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})

		t.Run("ValidationError/EmptyCreditGrantTaxCodeID", func(t *testing.T) {
			tc := createTaxCode(t, ns)
			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:          ns,
				InvoicingTaxCodeID: tc.ID,
			})
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})

		t.Run("NotFound/NonExistentInvoicingTaxCode", func(t *testing.T) {
			tc := createTaxCode(t, ns)
			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns,
				InvoicingTaxCodeID:   ulid.Make().String(),
				CreditGrantTaxCodeID: tc.ID,
			})
			require.Error(t, err)
			assert.True(t, taxcode.IsTaxCodeNotFoundError(err))
		})

		t.Run("NotFound/NonExistentCreditGrantTaxCode", func(t *testing.T) {
			tc := createTaxCode(t, ns)
			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns,
				InvoicingTaxCodeID:   tc.ID,
				CreditGrantTaxCodeID: ulid.Make().String(),
			})
			require.Error(t, err)
			assert.True(t, taxcode.IsTaxCodeNotFoundError(err))
		})

		t.Run("CrossNamespace/InvoicingTaxCodeFromOtherNamespace", func(t *testing.T) {
			otherNs := testutils.NameGenerator.Generate().Key
			otherTC := createTaxCode(t, otherNs)
			localTC := createTaxCode(t, ns)

			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns,
				InvoicingTaxCodeID:   otherTC.ID,
				CreditGrantTaxCodeID: localTC.ID,
			})
			require.Error(t, err)
			assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "tax code from another namespace must not resolve")
		})

		t.Run("CrossNamespace/CreditGrantTaxCodeFromOtherNamespace", func(t *testing.T) {
			otherNs := testutils.NameGenerator.Generate().Key
			otherTC := createTaxCode(t, otherNs)
			localTC := createTaxCode(t, ns)

			_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns,
				InvoicingTaxCodeID:   localTC.ID,
				CreditGrantTaxCodeID: otherTC.ID,
			})
			require.Error(t, err)
			assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "tax code from another namespace must not resolve")
		})

		t.Run("Create", func(t *testing.T) {
			ns2 := testutils.NameGenerator.Generate().Key
			invoicing := createTaxCode(t, ns2)
			creditGrant := createTaxCode(t, ns2)

			result, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns2,
				InvoicingTaxCodeID:   invoicing.ID,
				CreditGrantTaxCodeID: creditGrant.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, ns2, result.Namespace)
			assert.Equal(t, invoicing.ID, result.InvoicingTaxCodeID)
			assert.Equal(t, creditGrant.ID, result.CreditGrantTaxCodeID)

			t.Run("Get", func(t *testing.T) {
				got, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
					Namespace: ns2,
				})
				require.NoError(t, err)
				assert.Equal(t, result.ID, got.ID)
				assert.Equal(t, invoicing.ID, got.InvoicingTaxCodeID)
				assert.Equal(t, creditGrant.ID, got.CreditGrantTaxCodeID)
			})

			t.Run("Update", func(t *testing.T) {
				newInvoicing := createTaxCode(t, ns2)

				updated, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
					Namespace:            ns2,
					InvoicingTaxCodeID:   newInvoicing.ID,
					CreditGrantTaxCodeID: creditGrant.ID,
				})
				require.NoError(t, err)
				assert.Equal(t, result.ID, updated.ID, "record ID must not change on update")
				assert.Equal(t, newInvoicing.ID, updated.InvoicingTaxCodeID)
				assert.Equal(t, creditGrant.ID, updated.CreditGrantTaxCodeID)
			})
		})

		t.Run("Idempotent", func(t *testing.T) {
			ns3 := testutils.NameGenerator.Generate().Key
			invoicing := createTaxCode(t, ns3)
			creditGrant := createTaxCode(t, ns3)

			input := taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns3,
				InvoicingTaxCodeID:   invoicing.ID,
				CreditGrantTaxCodeID: creditGrant.ID,
			}

			first, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), input)
			require.NoError(t, err)

			second, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), input)
			require.NoError(t, err)

			assert.Equal(t, first.ID, second.ID, "row ID must be stable across identical upserts")
			assert.Equal(t, first.InvoicingTaxCodeID, second.InvoicingTaxCodeID)
			assert.Equal(t, first.CreditGrantTaxCodeID, second.CreditGrantTaxCodeID)
			assert.Equal(t, first.CreatedAt, second.CreatedAt, "created_at must not move on a no-op upsert")
		})

		t.Run("SameTaxCodeForBothFields", func(t *testing.T) {
			ns4 := testutils.NameGenerator.Generate().Key
			tc := createTaxCode(t, ns4)

			result, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            ns4,
				InvoicingTaxCodeID:   tc.ID,
				CreditGrantTaxCodeID: tc.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.ID, result.InvoicingTaxCodeID)
			assert.Equal(t, tc.ID, result.CreditGrantTaxCodeID)
		})
	})

	t.Run("Expand", func(t *testing.T) {
		nsExp := testutils.NameGenerator.Generate().Key
		invoicing := createTaxCode(t, nsExp)
		creditGrant := createTaxCode(t, nsExp)

		_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
			Namespace:            nsExp,
			InvoicingTaxCodeID:   invoicing.ID,
			CreditGrantTaxCodeID: creditGrant.ID,
		})
		require.NoError(t, err)

		t.Run("NoExpand/TaxCodesAreNil", func(t *testing.T) {
			result, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
				Namespace: nsExp,
			})
			require.NoError(t, err)
			assert.Nil(t, result.InvoicingTaxCode, "InvoicingTaxCode must be nil without expand")
			assert.Nil(t, result.CreditGrantTaxCode, "CreditGrantTaxCode must be nil without expand")
			assert.Equal(t, invoicing.ID, result.InvoicingTaxCodeID)
			assert.Equal(t, creditGrant.ID, result.CreditGrantTaxCodeID)
		})

		t.Run("ExpandAll/TaxCodesArePopulated", func(t *testing.T) {
			result, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
				Namespace: nsExp,
				Expand:    taxcode.OrganizationDefaultTaxCodesExpandAll,
			})
			require.NoError(t, err)
			require.NotNil(t, result.InvoicingTaxCode)
			require.NotNil(t, result.CreditGrantTaxCode)
			assert.Equal(t, invoicing.ID, result.InvoicingTaxCode.ID)
			assert.Equal(t, creditGrant.ID, result.CreditGrantTaxCode.ID)
		})

		t.Run("ExpandInvoicingOnly", func(t *testing.T) {
			result, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
				Namespace: nsExp,
				Expand:    taxcode.OrganizationDefaultTaxCodesExpand{InvoicingTaxCode: true},
			})
			require.NoError(t, err)
			require.NotNil(t, result.InvoicingTaxCode)
			assert.Equal(t, invoicing.ID, result.InvoicingTaxCode.ID)
			assert.Nil(t, result.CreditGrantTaxCode, "CreditGrantTaxCode must be nil when not expanded")
		})

		t.Run("ExpandCreditGrantOnly", func(t *testing.T) {
			result, err := env.Service.GetOrganizationDefaultTaxCodes(t.Context(), taxcode.GetOrganizationDefaultTaxCodesInput{
				Namespace: nsExp,
				Expand:    taxcode.OrganizationDefaultTaxCodesExpand{CreditGrantTaxCode: true},
			})
			require.NoError(t, err)
			assert.Nil(t, result.InvoicingTaxCode, "InvoicingTaxCode must be nil when not expanded")
			require.NotNil(t, result.CreditGrantTaxCode)
			assert.Equal(t, creditGrant.ID, result.CreditGrantTaxCode.ID)
		})

		t.Run("UpsertWithExpand/ResponseContainsTaxCodes", func(t *testing.T) {
			nsExp2 := testutils.NameGenerator.Generate().Key
			inv := createTaxCode(t, nsExp2)
			cg := createTaxCode(t, nsExp2)

			result, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
				Namespace:            nsExp2,
				InvoicingTaxCodeID:   inv.ID,
				CreditGrantTaxCodeID: cg.ID,
				Expand:               taxcode.OrganizationDefaultTaxCodesExpandAll,
			})
			require.NoError(t, err)
			require.NotNil(t, result.InvoicingTaxCode)
			require.NotNil(t, result.CreditGrantTaxCode)
			assert.Equal(t, inv.ID, result.InvoicingTaxCode.ID)
			assert.Equal(t, cg.ID, result.CreditGrantTaxCode.ID)
		})
	})
}
