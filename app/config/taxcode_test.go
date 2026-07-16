package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

func TestTaxCodeConfigurationValidate(t *testing.T) {
	validBase := func() TaxCodeConfiguration {
		return TaxCodeConfiguration{
			Seeds: []TaxCodeSeed{
				{Key: taxcode.ProviderDefaultTaxCodeKey, Name: "Provider default", DefaultInvoicing: true},
				{Key: taxcode.CreditGrantTaxCodeKey, Name: "Nontaxable", DefaultCreditGrant: true, AppMappings: []TaxCodeAppMapping{
					{AppType: "stripe", TaxCode: "txcd_00000000"},
				}},
			},
		}
	}

	t.Run("Valid", func(t *testing.T) {
		require.NoError(t, validBase().Validate())
	})

	t.Run("EmptySeeds", func(t *testing.T) {
		err := TaxCodeConfiguration{}.Validate()
		assert.ErrorContains(t, err, "seeds must not be empty")
	})

	t.Run("EmptyKey", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[0].Key = ""
		err := cfg.Validate()
		assert.ErrorContains(t, err, "key must not be empty")
	})

	t.Run("EmptyName", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[0].Name = ""
		err := cfg.Validate()
		assert.ErrorContains(t, err, "name must not be empty")
	})

	t.Run("DuplicateKeys", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[1].Key = cfg.Seeds[0].Key
		err := cfg.Validate()
		assert.ErrorContains(t, err, "duplicate key")
	})

	t.Run("NoDefaultInvoicing", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[0].DefaultInvoicing = false
		err := cfg.Validate()
		assert.ErrorContains(t, err, "defaultInvoicing=true")
	})

	t.Run("MultipleDefaultInvoicing", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[1].DefaultInvoicing = true
		err := cfg.Validate()
		assert.ErrorContains(t, err, "defaultInvoicing=true")
	})

	t.Run("NoDefaultCreditGrant", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[1].DefaultCreditGrant = false
		err := cfg.Validate()
		assert.ErrorContains(t, err, "defaultCreditGrant=true")
	})

	t.Run("MultipleDefaultCreditGrant", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[0].DefaultCreditGrant = true
		err := cfg.Validate()
		assert.ErrorContains(t, err, "defaultCreditGrant=true")
	})

	t.Run("SingleSeedCarriesBothFlags", func(t *testing.T) {
		// A single seed may carry both flags; this is legal.
		cfg := TaxCodeConfiguration{
			Seeds: []TaxCodeSeed{
				{Key: "all", Name: "All", DefaultInvoicing: true, DefaultCreditGrant: true},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("AppMappingEmptyAppType", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[1].AppMappings[0].AppType = ""
		err := cfg.Validate()
		assert.ErrorContains(t, err, "appType must not be empty")
	})

	t.Run("AppMappingEmptyTaxCode", func(t *testing.T) {
		cfg := validBase()
		cfg.Seeds[1].AppMappings[0].TaxCode = ""
		err := cfg.Validate()
		assert.ErrorContains(t, err, "taxCode must not be empty")
	})
}
