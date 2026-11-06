package taxconfigv1

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	api "github.com/openmeterio/openmeter/api/client/go"
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
