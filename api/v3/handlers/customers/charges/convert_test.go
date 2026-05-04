package charges

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestConvertTaxCodeConfigToAPI(t *testing.T) {
	tests := []struct {
		name  string
		input *productcatalog.TaxCodeConfig
		want  *api.BillingTaxConfig
	}{
		{
			name:  "nil returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty config (both fields nil) returns nil",
			input: &productcatalog.TaxCodeConfig{},
			want:  nil,
		},
		{
			name: "behavior only",
			input: &productcatalog.TaxCodeConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			},
			want: &api.BillingTaxConfig{
				Behavior: lo.ToPtr(api.BillingTaxBehaviorInclusive),
			},
		},
		{
			name: "exclusive behavior",
			input: &productcatalog.TaxCodeConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			},
			want: &api.BillingTaxConfig{
				Behavior: lo.ToPtr(api.BillingTaxBehaviorExclusive),
			},
		},
		{
			name: "tax code ID only",
			input: &productcatalog.TaxCodeConfig{
				TaxCodeID: lo.ToPtr("01JTEST00000000000000000001"),
			},
			want: &api.BillingTaxConfig{
				TaxCode: &api.BillingTaxCodeReference{Id: "01JTEST00000000000000000001"},
			},
		},
		{
			name: "both behavior and tax code ID",
			input: &productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01JTEST00000000000000000002"),
			},
			want: &api.BillingTaxConfig{
				Behavior: lo.ToPtr(api.BillingTaxBehaviorExclusive),
				TaxCode:  &api.BillingTaxCodeReference{Id: "01JTEST00000000000000000002"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTaxCodeConfigToAPI(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
