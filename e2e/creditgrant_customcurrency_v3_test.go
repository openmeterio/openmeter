package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CustomCurrencyCreditGrantLifecycle(t *testing.T) {
	c := newV3Client(t)
	code := uniqueKey("cgr")
	custom, err := c.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Name: "Grant Currency " + code, Code: code, Precision: 3, DecimalMark: ".", ThousandSeparator: ",",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, custom)
	currencyCostBasis, err := c.Currencies.CreateCostBasis(t.Context(), custom.ID, v3sdk.CreateCostBasisRequest{
		FiatCode: "USD", Rate: "0.25",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, currencyCostBasis)
	require.NotNil(t, currencyCostBasis.EffectiveFrom)
	// Grant timestamps use whole seconds; the rate must be effective before that
	// second starts, including when the fixture is created within the same second.
	require.Eventually(t, func() bool {
		return time.Now().UTC().Truncate(time.Second).After(*currencyCostBasis.EffectiveFrom)
	}, 5*time.Second, 10*time.Millisecond)

	for _, tc := range []struct {
		name      string
		costBasis *v3sdk.ChargeCostBasis
	}{
		{name: "promotional"},
		{name: "manual", costBasis: lo.ToPtr(lo.Must(v3sdk.ChargeCostBasisFromChargeCostBasisManual(v3sdk.ChargeCostBasisManual{
			FiatCurrency: lo.ToPtr("USD"), Rate: "0.25",
		})))},
		{name: "pinned", costBasis: lo.ToPtr(lo.Must(v3sdk.ChargeCostBasisFromChargeCostBasisPinned(v3sdk.ChargeCostBasisPinned{
			FiatCurrency: "USD", CostBasisID: currencyCostBasis.ID,
		})))},
		{name: "dynamic", costBasis: lo.ToPtr(lo.Must(v3sdk.ChargeCostBasisFromChargeCostBasisDynamic(v3sdk.ChargeCostBasisDynamic{
			FiatCurrency: "USD",
		})))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a custom-currency grant and a fiat grant for the same customer
			c := newV3Client(t)
			customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
				Key: uniqueKey("custom_grant_customer"), Name: "Custom Grant Customer", Currency: lo.ToPtr("USD"),
			})
			c.requireStatus(http.StatusCreated, err)
			require.NotNil(t, customer)
			body := v3sdk.CreateCreditGrantRequest{
				Name: tc.name + " custom credits", Currency: v3sdk.BillingCurrencyCode(code), Amount: "100",
				FundingMethod: v3sdk.CreditFundingMethodNone,
			}
			if tc.costBasis != nil {
				body.FundingMethod = v3sdk.CreditFundingMethodExternal
				body.Purchase = &v3sdk.CreateCreditGrantPurchase{Currency: "USD", CostBasis: tc.costBasis}
			}

			created, err := c.Customers.Credits.Grants.Create(t.Context(), customer.ID, body)
			c.requireStatus(http.StatusCreated, err)
			require.NotNil(t, created)
			requireGrant := func(grant *v3sdk.CreditGrant) {
				t.Helper()
				require.NotNil(t, grant)
				require.Equal(t, created.ID, grant.ID)
				require.Equal(t, v3sdk.BillingCurrencyCode(code), grant.Currency)
				require.NotNil(t, grant.CustomCurrency)
				require.Equal(t, custom.ID, grant.CustomCurrency.ID)
				if tc.costBasis == nil {
					require.Nil(t, grant.Purchase)
					return
				}

				require.NotNil(t, grant.Purchase)
				require.Equal(t, "USD", grant.Purchase.Currency)
				require.Equal(t, lo.ToPtr(v3sdk.Numeric("25")), grant.Purchase.Amount)
				require.Nil(t, grant.Purchase.PerUnitCostBasis)
				require.NotNil(t, grant.Purchase.ResolvedCostBasis)
				require.Equal(t, "USD", grant.Purchase.ResolvedCostBasis.FiatCurrency)
				require.Equal(t, v3sdk.Numeric("0.25"), grant.Purchase.ResolvedCostBasis.Rate)
			}
			requireGrant(created)
			fiatGrant, err := c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
				Name: "fiat credits", Currency: "USD", Amount: "10", FundingMethod: v3sdk.CreditFundingMethodNone,
			})
			c.requireStatus(http.StatusCreated, err)
			require.Nil(t, fiatGrant.CustomCurrency)

			// when: reading, filtering, settling, and voiding the custom grant
			got, err := c.Customers.Credits.Grants.Get(t.Context(), customer.ID, created.ID)
			c.requireStatus(http.StatusOK, err)
			requireGrant(got)
			page, err := c.Customers.Credits.Grants.List(t.Context(), customer.ID, v3sdk.CreditGrantListParams{
				Filter: &v3sdk.CreditGrantFilter{Currency: lo.ToPtr(v3sdk.BillingCurrencyCode(code))},
			})
			c.requireStatus(http.StatusOK, err)
			require.Len(t, page.Data, 1)
			requireGrant(&page.Data[0])
			if tc.costBasis != nil {
				settled, err := c.Customers.Credits.Grants.UpdateExternalSettlement(t.Context(), customer.ID, created.ID, v3sdk.UpdateCreditGrantExternalSettlementRequest{
					Status: v3sdk.CreditPurchasePaymentSettlementStatusSettled,
				})
				c.requireStatus(http.StatusOK, err)
				requireGrant(settled)
				require.Equal(t, lo.ToPtr(v3sdk.CreditPurchasePaymentSettlementStatusSettled), settled.Purchase.SettlementStatus)
			}

			voided, err := c.Customers.Credits.Grants.Void(t.Context(), customer.ID, created.ID, &v3sdk.VoidCreditGrantRequest{})
			c.requireStatus(http.StatusOK, err)

			// then: every grant response retains its native currency and purchase terms
			requireGrant(voided)
			require.Equal(t, v3sdk.CreditGrantStatusVoided, voided.Status)
		})
	}
}
