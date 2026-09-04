package creditgrant

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
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

func TestCreateInputValidateCostBasis(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	eur, err := currencyx.NewFiatCurrency("EUR")
	require.NoError(t, err)

	fiatRate := lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: alpacadecimal.NewFromFloat(0.5)}))

	manual := func(fiat *currencyx.FiatCurrency) *creditpurchase.CostBasis {
		return lo.ToPtr(creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: alpacadecimal.NewFromFloat(0.5)})))
	}

	dynamic := lo.ToPtr(creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: usd})))

	tests := []struct {
		name           string
		creditCurrency currencyx.Code
		purchase       PurchaseTerms
		wantErr        string
	}{
		{
			name:           "fiat grant with fiat rate",
			creditCurrency: "USD",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: fiatRate},
		},
		{
			name:           "fiat grant with custom currency manual cost basis",
			creditCurrency: "USD",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: manual(usd)},
			wantErr:        "validation error: purchase.cost_basis is not supported for fiat credit grants",
		},
		{
			name:           "fiat grant with dynamic cost basis",
			creditCurrency: "USD",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: dynamic},
			wantErr:        "validation error: purchase.cost_basis is not supported for fiat credit grants",
		},
		{
			name:           "custom grant without cost basis",
			creditCurrency: "TOKENS",
			purchase:       PurchaseTerms{Currency: "USD"},
			wantErr:        "purchase.cost_basis is required for custom currency credit grants",
		},
		{
			name:           "custom grant with manual cost basis without fiat currency",
			creditCurrency: "TOKENS",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: manual(nil)},
			wantErr:        "purchase.cost_basis: validation error: fiat currency",
		},
		{
			name:           "custom grant with manual cost basis",
			creditCurrency: "TOKENS",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: manual(usd)},
		},
		{
			name:           "custom grant with dynamic cost basis",
			creditCurrency: "TOKENS",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: dynamic},
		},
		{
			name:           "custom grant with cost basis in another currency",
			creditCurrency: "TOKENS",
			purchase:       PurchaseTerms{Currency: "USD", CostBasis: manual(eur)},
			wantErr:        `purchase currency "USD" must match purchase.cost_basis fiat currency "EUR"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (CreateInput{
				Namespace:     "default",
				CustomerID:    "customer-id",
				Name:          "credit grant",
				Currency:      tt.creditCurrency,
				Amount:        alpacadecimal.NewFromInt(100),
				FundingMethod: FundingMethodExternal,
				Purchase:      &tt.purchase,
			}).Validate()

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
