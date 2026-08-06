package creditgrant

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCreateInputValidateRejectsMismatchedPurchaseCurrency(t *testing.T) {
	for _, fundingMethod := range []FundingMethod{
		FundingMethodInvoice,
		FundingMethodExternal,
	} {
		t.Run(string(fundingMethod), func(t *testing.T) {
			err := (CreateInput{
				Namespace:     "default",
				CustomerID:    "customer-id",
				Name:          "credit grant",
				Currency:      currencyx.Code("USD"),
				Amount:        alpacadecimal.NewFromInt(100),
				FundingMethod: fundingMethod,
				Purchase: &PurchaseTerms{
					Currency: currencyx.Code("EUR"),
				},
			}).Validate()

			require.ErrorContains(t, err, `purchase currency "EUR" must match credit currency "USD"`)
		})
	}
}
