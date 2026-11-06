package taxconfigv1

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// AssertTaxConfigs checks that every expected key is present in got with the expected
// taxConfig. Extra keys in got are ignored.
func AssertTaxConfigs(t *testing.T, want, got map[string]*api.TaxConfig) {
	t.Helper()

	for key, wantTaxConfig := range want {
		gotTaxConfig, ok := got[key]
		if assert.True(t, ok, "missing %s", key) {
			assert.Equal(t, wantTaxConfig, gotTaxConfig, "taxConfig of %s", key)
		}
	}
}

// InvoiceLineTaxConfigs returns the taxConfig of each line by name. It also checks that the
// line's rateCard.taxConfig and the taxConfig of its detailed lines match the line's own.
func InvoiceLineTaxConfigs(t *testing.T, lines []api.InvoiceLine) map[string]*api.TaxConfig {
	t.Helper()

	got := make(map[string]*api.TaxConfig, len(lines))
	for _, line := range lines {
		got[line.Name] = line.TaxConfig

		if assert.NotNil(t, line.RateCard, "rate card of line %s", line.Name) {
			assert.Equal(t, line.TaxConfig, line.RateCard.TaxConfig, "rateCard.taxConfig of line %s", line.Name)
		}

		for _, child := range lo.FromPtr(line.Children) {
			assert.Equal(t, line.TaxConfig, child.TaxConfig, "taxConfig of detailed line %s of %s", child.Name, line.Name)

			if assert.NotNil(t, child.RateCard, "rate card of detailed line %s of %s", child.Name, line.Name) {
				assert.Equal(t, line.TaxConfig, child.RateCard.TaxConfig, "rateCard.taxConfig of detailed line %s of %s", child.Name, line.Name)
			}
		}
	}

	return got
}

// StripeOnlyWant returns the taxConfig a write that sent only the given Stripe code must
// render: the code, plus the ID of the tax code the server looked up or created for it. The ID
// is only known after the write, so it is taken from got, which must carry one. Compare got
// against the result in a then step; this doesn't check the rest of got.
func StripeOnlyWant(t *testing.T, stripeCode string, got *api.TaxConfig) *api.TaxConfig {
	t.Helper()

	require.NotNil(t, got, "taxConfig of a stripe-only write")
	require.NotEmpty(t, lo.FromPtr(got.TaxCodeId), "taxCodeId of a stripe-only write")

	return &api.TaxConfig{
		Stripe:    &api.StripeTaxConfig{Code: stripeCode},
		TaxCodeId: lo.ToPtr(*got.TaxCodeId),
	}
}

// AssertTaxCodeStripeMapping checks through v3 that the tax code with the given ID has a
// Stripe app mapping for the given code.
func AssertTaxCodeStripeMapping(t *testing.T, client *v3sdk.Client, taxCodeID, stripeCode string) {
	t.Helper()

	taxCode, err := client.Tax.GetCode(t.Context(), taxCodeID)
	require.NoError(t, err)
	require.NotNil(t, taxCode)

	assert.Equal(t, stripeCode, StripeAppMapping(*taxCode), "stripe app mapping of tax code %s", taxCodeID)
}
