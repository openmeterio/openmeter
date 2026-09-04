package creditgrant

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCreateInputValidatePurchaseCurrency(t *testing.T) {
	for _, fundingMethod := range []FundingMethod{
		FundingMethodInvoice,
		FundingMethodExternal,
	} {
		t.Run(string(fundingMethod), func(t *testing.T) {
			base := CreateInput{
				Namespace:     "default",
				CustomerID:    "customer-id",
				Name:          "credit grant",
				Currency:      currencyx.Code("USD"),
				Amount:        alpacadecimal.NewFromInt(100),
				FundingMethod: fundingMethod,
			}

			t.Run("fiat grant requires matching purchase currency", func(t *testing.T) {
				input := base
				input.Purchase = &PurchaseTerms{
					Currency: currencyx.Code("EUR"),
				}

				err := input.Validate()
				require.ErrorContains(t, err, `purchase currency "EUR" must match fiat credit currency "USD"`)
			})

			t.Run("custom grant allows a distinct fiat purchase currency", func(t *testing.T) {
				input := base
				input.Currency = currencyx.Code("TOKENS")
				input.Purchase = &PurchaseTerms{
					Currency: currencyx.Code("USD"),
				}

				require.NoError(t, input.Validate())
			})

			t.Run("purchase currency must be fiat", func(t *testing.T) {
				input := base
				input.Currency = currencyx.Code("TOKENS")
				input.Purchase = &PurchaseTerms{
					Currency: currencyx.Code("POINTS"),
				}

				err := input.Validate()
				require.ErrorContains(t, err, `purchase currency "POINTS" must be fiat`)
			})
		})
	}
}
