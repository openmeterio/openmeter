package customer

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCustomerMutateCurrencyMustBeFiat(t *testing.T) {
	tests := []struct {
		name     string
		currency *currencyx.Code
		wantErr  bool
	}{
		{
			name: "currency is optional",
		},
		{
			name:     "fiat currency",
			currency: lo.ToPtr(currencyx.Code("USD")),
		},
		{
			name:     "custom currency",
			currency: lo.ToPtr(currencyx.Code("CREDITS")),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a structurally valid customer mutation with an optional currency
			// when:
			// - the customer is validated
			// then:
			// - only fiat currencies are accepted as the customer's billing default
			err := (CustomerMutate{
				Key:      lo.ToPtr("customer"),
				Name:     "Customer",
				Currency: tt.currency,
			}).Validate()
			if tt.wantErr {
				require.ErrorContains(t, err, "must be fiat")
				return
			}

			require.NoError(t, err)
		})
	}
}
