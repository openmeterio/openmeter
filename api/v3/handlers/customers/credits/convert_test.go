package customerscredits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingCreditGrantPromotional(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)

	charge := creditpurchase.Charge{
		ChargeBase: creditpurchase.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "ns",
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "grant-1",
			},
			Intent: creditpurchase.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name: "Promo credits",
					},
					CreditAmount: alpacadecimal.RequireFromString("25"),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
			},
			Status: creditpurchase.StatusActive,
		},
	}

	grant, err := toAPIBillingCreditGrant(charge)
	require.NoError(t, err)
	require.Equal(t, api.BillingCreditFundingMethodNone, grant.FundingMethod)
	require.Nil(t, grant.Purchase)
	require.Equal(t, "25", grant.Amount)
	require.Equal(t, api.BillingCreditGrantStatusActive, grant.Status)
	require.Nil(t, grant.VoidedAt)

	t.Run("ledger-derived voiding overrides the charge status", func(t *testing.T) {
		voidedAt := now.Add(time.Hour)

		voidedCharge := charge
		voidedCharge.State.VoidedAt = &voidedAt

		grant, err := toAPIBillingCreditGrant(voidedCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusVoided, grant.Status)
		require.Equal(t, lo.ToPtr(voidedAt), grant.VoidedAt)
	})
}

func TestToAPIBillingCreditGrantStatusUsesExpiry(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	charge := creditpurchase.Charge{
		ChargeBase: creditpurchase.ChargeBase{
			Intent: creditpurchase.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					CreditAmount: alpacadecimal.RequireFromString("25"),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				},
			},
			Status: creditpurchase.StatusActive,
		},
	}

	t.Run("past expiry is public expired", func(t *testing.T) {
		expiredAt := now.Add(-time.Nanosecond)
		expiredCharge := charge
		expiredCharge.Intent.ExpiresAt = &expiredAt

		grant, err := toAPIBillingCreditGrant(expiredCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusExpired, grant.Status)
	})

	t.Run("expiry at now is public expired", func(t *testing.T) {
		expiredCharge := charge
		expiredCharge.Intent.ExpiresAt = &now

		grant, err := toAPIBillingCreditGrant(expiredCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusExpired, grant.Status)
	})

	t.Run("future expiry stays active", func(t *testing.T) {
		expiresAt := now.Add(time.Nanosecond)
		activeCharge := charge
		activeCharge.Intent.ExpiresAt = &expiresAt

		grant, err := toAPIBillingCreditGrant(activeCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusActive, grant.Status)
	})

	t.Run("voided wins over expired", func(t *testing.T) {
		expiredAt := now.Add(-time.Nanosecond)
		voidedAt := now
		voidedCharge := charge
		voidedCharge.Intent.ExpiresAt = &expiredAt
		voidedCharge.State.VoidedAt = &voidedAt

		grant, err := toAPIBillingCreditGrant(voidedCharge)
		require.NoError(t, err)
		require.Equal(t, api.BillingCreditGrantStatusVoided, grant.Status)
	})
}

func TestToAPIBillingCreditGrantKey(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)

	newCharge := func(key *string) creditpurchase.Charge {
		return creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				ManagedResource: meta.ManagedResource{
					NamespacedModel: models.NamespacedModel{
						Namespace: "ns",
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
					ID: "grant-1",
				},
				Intent: creditpurchase.Intent{
					Intent: meta.Intent{
						CustomerID: "cust-1",
						Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
					},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name: "Promo credits",
						},
						CreditAmount: alpacadecimal.RequireFromString("25"),
						Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
					},
					Key: key,
				},
				Status: creditpurchase.StatusActive,
			},
		}
	}

	t.Run("maps the idempotency key into the read response", func(t *testing.T) {
		grant, err := toAPIBillingCreditGrant(newCharge(lo.ToPtr("welcome-bonus")))
		require.NoError(t, err)
		require.Equal(t, lo.ToPtr("welcome-bonus"), grant.Key)
	})

	t.Run("omits the key when the grant has none", func(t *testing.T) {
		grant, err := toAPIBillingCreditGrant(newCharge(nil))
		require.NoError(t, err)
		require.Nil(t, grant.Key)
	})
}

func TestFromAPICreateChargeCostBasis(t *testing.T) {
	usd := api.CurrencyCode("USD")

	manual := func(fiatCurrency *api.CurrencyCode) *api.CreateChargeCostBasis {
		var out api.CreateChargeCostBasis
		require.NoError(t, out.FromCreateChargeCostBasisManual(api.CreateChargeCostBasisManual{
			Type:         api.CreateChargeCostBasisManualTypeManual,
			FiatCurrency: fiatCurrency,
			Rate:         "0.5",
		}))

		return &out
	}

	t.Run("nil", func(t *testing.T) {
		out, err := fromAPICreateChargeCostBasis(nil)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("manual without fiat currency folds into the fiat rate", func(t *testing.T) {
		out, err := fromAPICreateChargeCostBasis(manual(nil))
		require.NoError(t, err)
		require.Equal(t, creditpurchase.CostBasisTypeFiat, out.Type())

		fiat, err := out.AsFiat()
		require.NoError(t, err)
		require.Equal(t, 0.5, fiat.Rate.InexactFloat64())
	})

	t.Run("manual with fiat currency keeps the intent", func(t *testing.T) {
		out, err := fromAPICreateChargeCostBasis(manual(&usd))
		require.NoError(t, err)
		require.Equal(t, creditpurchase.CostBasisTypeCustomCurrency, out.Type())
		require.Equal(t, costbasis.ModeManual, out.GetCustomCurrencyModeOrEmpty())

		intent, err := out.AsCustomCurrency()
		require.NoError(t, err)

		fiat, err := intent.GetFiatCurrency()
		require.NoError(t, err)
		require.Equal(t, currencyx.Code("USD"), fiat.Details().Code)
	})

	t.Run("dynamic keeps the intent", func(t *testing.T) {
		var in api.CreateChargeCostBasis
		require.NoError(t, in.FromCreateChargeCostBasisDynamic(api.CreateChargeCostBasisDynamic{
			Type:         api.CreateChargeCostBasisDynamicTypeDynamic,
			FiatCurrency: usd,
		}))

		out, err := fromAPICreateChargeCostBasis(&in)
		require.NoError(t, err)
		require.Equal(t, costbasis.ModeDynamic, out.GetCustomCurrencyModeOrEmpty())
	})

	t.Run("invalid rate", func(t *testing.T) {
		var in api.CreateChargeCostBasis
		require.NoError(t, in.FromCreateChargeCostBasisManual(api.CreateChargeCostBasisManual{
			Type: api.CreateChargeCostBasisManualTypeManual,
			Rate: "not-a-number",
		}))

		_, err := fromAPICreateChargeCostBasis(&in)
		require.ErrorContains(t, err, "invalid cost basis rate")
	})
}

func TestToAPICreditGrantPurchase(t *testing.T) {
	now := time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC)

	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	newCharge := func(currency currencies.Currency, costBasis creditpurchase.CostBasis, rate alpacadecimal.Decimal) creditpurchase.Charge {
		return creditpurchase.Charge{
			ChargeBase: creditpurchase.ChargeBase{
				Intent: creditpurchase.Intent{
					Intent: meta.Intent{
						CustomerID: "cust-1",
						Currency:   currency,
					},
					IntentMutableFields: creditpurchase.IntentMutableFields{
						CreditAmount: alpacadecimal.NewFromInt(100),
						Settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
							InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
						}),
					},
					CostBasis: costBasis,
				},
				State: creditpurchase.State{
					ResolvedCostBasis: &costbasis.State{CostBasis: rate, ResolvedAt: now},
				},
				Status: creditpurchase.StatusActivePaymentPending,
			},
		}
	}

	t.Run("fiat grant echoes the rate through the deprecated field", func(t *testing.T) {
		rate := alpacadecimal.NewFromFloat(0.5)
		charge := newCharge(currenciestestutils.NewFiatCurrency(t, "USD"), creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: rate}), rate)

		purchase, err := toAPICreditGrantPurchase(charge)
		require.NoError(t, err)
		require.NotNil(t, purchase)
		require.Equal(t, api.CurrencyCode("USD"), purchase.Currency)
		require.Equal(t, lo.ToPtr("50"), purchase.Amount)
		require.Equal(t, lo.ToPtr("0.5"), purchase.PerUnitCostBasis)
		require.NotNil(t, purchase.ResolvedCostBasis)
		require.Equal(t, "0.5", purchase.ResolvedCostBasis.Rate)
	})

	t.Run("custom currency grant exposes the rate only as resolved cost basis", func(t *testing.T) {
		rate := alpacadecimal.NewFromFloat(0.25)
		charge := newCharge(
			currenciestestutils.NewCustomCurrency(t, "TOKENS", 3),
			creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: usd, Rate: rate})),
			rate,
		)

		purchase, err := toAPICreditGrantPurchase(charge)
		require.NoError(t, err)
		require.NotNil(t, purchase)
		require.Equal(t, api.CurrencyCode("USD"), purchase.Currency)
		require.Equal(t, lo.ToPtr("25"), purchase.Amount)
		require.Nil(t, purchase.PerUnitCostBasis)
		require.NotNil(t, purchase.ResolvedCostBasis)
		require.Equal(t, api.CurrencyCode("USD"), purchase.ResolvedCostBasis.FiatCurrency)
		require.Equal(t, "0.25", purchase.ResolvedCostBasis.Rate)
	})
}
