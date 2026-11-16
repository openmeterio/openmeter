package tax_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/handlers/tax"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPITaxCodeConfig(t *testing.T) {
	t.Run("nil when tax config is nil", func(t *testing.T) {
		assert.Nil(t, tax.ToAPITaxCodeConfig(nil))
	})

	t.Run("nil when tax config has no shared fields", func(t *testing.T) {
		assert.Nil(t, tax.ToAPITaxCodeConfig(&productcatalog.TaxCodeConfig{}))
	})

	t.Run("maps tax code ID", func(t *testing.T) {
		result := tax.ToAPITaxCodeConfig(&productcatalog.TaxCodeConfig{
			TaxCodeID: "01TAXCODE000",
		})

		require.NotNil(t, result)
		require.NotNil(t, result.Code)
		assert.Equal(t, "01TAXCODE000", result.Code.Id)
		assert.Nil(t, result.Behavior)
	})

	t.Run("maps behavior", func(t *testing.T) {
		result := tax.ToAPITaxCodeConfig(&productcatalog.TaxCodeConfig{
			TaxCodeID: "01TAXCODE000",
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		})

		require.NotNil(t, result)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, apiv3models.BillingTaxBehaviorInclusive, *result.Behavior)
	})

	t.Run("maps behavior-only config without code", func(t *testing.T) {
		result := tax.ToAPITaxCodeConfig(&productcatalog.TaxCodeConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		})

		require.NotNil(t, result)
		assert.Nil(t, result.Code)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, apiv3models.BillingTaxBehaviorExclusive, *result.Behavior)
	})
}

func TestFromAPITaxCodeConfig(t *testing.T) {
	t.Run("nil when tax config is nil", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("maps tax code ID", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{
			Code: &apiv3models.TaxCodeReference{Id: "01TAXCODE000"},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "01TAXCODE000", result.TaxCodeID)
		assert.Nil(t, result.Behavior)
	})

	t.Run("maps behavior", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{
			Code:     &apiv3models.TaxCodeReference{Id: "01TAXCODE000"},
			Behavior: lo.ToPtr(apiv3models.BillingTaxBehaviorInclusive),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, productcatalog.InclusiveTaxBehavior, *result.Behavior)
	})

	t.Run("maps behavior-only config", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{
			Behavior: lo.ToPtr(apiv3models.BillingTaxBehaviorInclusive),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.TaxCodeID)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, productcatalog.InclusiveTaxBehavior, *result.Behavior)
	})

	t.Run("rejects empty code ID", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{
			Code: &apiv3models.TaxCodeReference{Id: ""},
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, models.IsGenericValidationError(err), "an explicit code reference without an id must surface as a validation error")
	})

	t.Run("rejects empty code ID even with behavior", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{
			Code:     &apiv3models.TaxCodeReference{Id: ""},
			Behavior: lo.ToPtr(apiv3models.BillingTaxBehaviorInclusive),
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, models.IsGenericValidationError(err), "an explicit code reference without an id must surface as a validation error")
	})

	t.Run("rejects empty tax config", func(t *testing.T) {
		result, err := tax.FromAPITaxCodeConfig(&apiv3models.TaxCodeConfig{})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, models.IsGenericValidationError(err), "tax config without code or behavior must surface as a validation error")
	})
}
