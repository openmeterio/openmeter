package creditpurchase

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestIntentNormalizedPinsServicePeriodsToEffectiveAt(t *testing.T) {
	effectiveAt := time.Date(2026, 4, 17, 11, 23, 0, 0, time.UTC)
	originalPeriod := timeutil.ClosedPeriod{
		From: effectiveAt.Add(-time.Hour),
		To:   effectiveAt.Add(time.Hour),
	}

	intent := Intent{
		IntentMutableFields: IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				ServicePeriod:     originalPeriod,
				FullServicePeriod: originalPeriod,
				BillingPeriod:     originalPeriod,
			},
			EffectiveAt: &effectiveAt,
		},
	}

	got := intent.Normalized()

	expectedPeriod := timeutil.ClosedPeriod{From: effectiveAt, To: effectiveAt}
	require.Equal(t, expectedPeriod, got.ServicePeriod)
	require.Equal(t, expectedPeriod, got.FullServicePeriod)
	require.Equal(t, expectedPeriod, got.BillingPeriod)
}

func TestFeatureFiltersNormalize(t *testing.T) {
	require.Equal(t, FeatureFilters{"api-calls", "storage"}, FeatureFilters([]string{"storage", "api-calls", "storage"}).Normalize())
}

func TestFeatureFiltersValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, FeatureFilters([]string{"api-calls", "storage"}).Validate())
	})

	t.Run("empty key", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{""}).Validate())
	})

	t.Run("duplicate key", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{"api-calls", "api-calls"}).Validate())
	})
}

func TestFeatureFiltersValidateAsFeatureFilter(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, FeatureFilters([]string{"api-calls"}).ValidateAsFeatureFilter())
	})

	t.Run("empty", func(t *testing.T) {
		require.Error(t, FeatureFilters(nil).ValidateAsFeatureFilter())
	})

	t.Run("multiple", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{"api-calls", "storage"}).ValidateAsFeatureFilter())
	})

	t.Run("invalid feature", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{""}).ValidateAsFeatureFilter())
	})
}

func TestListFundedCreditActivitiesInputValidateAllowsCustomCurrency(t *testing.T) {
	currency := currencyx.Code("CREDITS")

	input := ListFundedCreditActivitiesInput{
		Customer: customer.CustomerID{
			Namespace: "ns",
			ID:        "customer-id",
		},
		Limit:    1,
		Currency: &currency,
	}

	require.NoError(t, input.Validate())
}

func TestIntentValidateCustomCreditWithFiatSettlement(t *testing.T) {
	t.Run("allows custom credit currency with fiat settlement", func(t *testing.T) {
		intent := validIntentForValidation()
		intent.Currency = currencyx.Code("ACME")
		intent.Settlement = NewSettlement(ExternalSettlement{
			InitialStatus: CreatedInitialPaymentSettlementStatus,
			GenericSettlement: GenericSettlement{
				Currency:  currencyx.Code("USD"),
				CostBasis: alpacadecimal.RequireFromString("0.5"),
			},
		})

		require.NoError(t, intent.Validate())
	})

	t.Run("rejects fiat credit currency with different fiat settlement", func(t *testing.T) {
		intent := validIntentForValidation()
		intent.Currency = currencyx.Code("USD")
		intent.Settlement = NewSettlement(ExternalSettlement{
			InitialStatus: CreatedInitialPaymentSettlementStatus,
			GenericSettlement: GenericSettlement{
				Currency:  currencyx.Code("EUR"),
				CostBasis: alpacadecimal.RequireFromString("0.5"),
			},
		})

		require.ErrorContains(t, intent.Validate(), `settlement currency "EUR" must match credit currency "USD"`)
	})

	t.Run("rejects custom settlement currency", func(t *testing.T) {
		intent := validIntentForValidation()
		intent.Currency = currencyx.Code("ACME")
		intent.Settlement = NewSettlement(ExternalSettlement{
			InitialStatus: CreatedInitialPaymentSettlementStatus,
			GenericSettlement: GenericSettlement{
				Currency:  currencyx.Code("CREDITS"),
				CostBasis: alpacadecimal.RequireFromString("0.5"),
			},
		})

		require.ErrorContains(t, intent.Validate(), "settlement currency must be a known fiat currency")
	})
}

func TestSettlementAmount(t *testing.T) {
	t.Run("calculates and rounds funded amount in settlement currency", func(t *testing.T) {
		settlement := NewSettlement(ExternalSettlement{
			InitialStatus: CreatedInitialPaymentSettlementStatus,
			GenericSettlement: GenericSettlement{
				Currency:  currencyx.Code("USD"),
				CostBasis: alpacadecimal.RequireFromString("0.3333"),
			},
		})

		currency, amount, err := SettlementAmount(settlement, alpacadecimal.NewFromInt(3))
		require.NoError(t, err)
		require.Equal(t, currencyx.Code("USD"), currency)
		require.Equal(t, float64(1), amount.InexactFloat64())
	})

	t.Run("rejects promotional settlement", func(t *testing.T) {
		_, _, err := SettlementAmount(
			NewSettlement(PromotionalSettlement{}),
			alpacadecimal.NewFromInt(100),
		)

		require.ErrorContains(t, err, "settlement amount is not available for promotional credit purchase")
	})

	t.Run("rejects amount rounded to zero in settlement currency", func(t *testing.T) {
		_, _, err := SettlementAmount(
			NewSettlement(ExternalSettlement{
				InitialStatus: CreatedInitialPaymentSettlementStatus,
				GenericSettlement: GenericSettlement{
					Currency:  currencyx.Code("USD"),
					CostBasis: alpacadecimal.RequireFromString("0.001"),
				},
			}),
			alpacadecimal.NewFromInt(1),
		)

		require.ErrorContains(t, err, "settlement amount in USD must be positive after rounding")
	})
}

func validIntentForValidation() Intent {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(time.Hour),
	}

	return Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.SystemManagedLine,
			CustomerID: "customer-id",
			Currency:   currencyx.Code("USD"),
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-id",
			},
		},
		IntentMutableFields: IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "Credits",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			CreditAmount: alpacadecimal.NewFromInt(100),
			Settlement: NewSettlement(ExternalSettlement{
				InitialStatus: CreatedInitialPaymentSettlementStatus,
				GenericSettlement: GenericSettlement{
					Currency:  currencyx.Code("USD"),
					CostBasis: alpacadecimal.NewFromInt(1),
				},
			}),
		},
	}
}
