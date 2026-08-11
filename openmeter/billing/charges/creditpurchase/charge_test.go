package creditpurchase

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
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
		Intent: meta.Intent{
			Currency: currenciestestutils.NewFiatCurrency(t, "USD"),
		},
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

func TestIntentMutableFieldsCalculateEffectiveAtDoesNotBackdateToServicePeriod(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	fields := IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			ServicePeriod: timeutil.ClosedPeriod{
				From: now.Add(-24 * time.Hour),
				To:   now,
			},
		},
	}

	require.Equal(t, now, fields.CalculateEffectiveAt())
}

func TestIntentMutableFieldsValidateAllowsHistoricalExpiryWithoutEffectiveAt(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	period := timeutil.ClosedPeriod{
		From: now.Add(-2 * time.Hour),
		To:   now.Add(-time.Hour),
	}

	require.NoError(t, (IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:              "historical credit purchase",
			ServicePeriod:     period,
			FullServicePeriod: period,
			BillingPeriod:     period,
		},
		CreditAmount: alpacadecimal.NewFromInt(1),
		ExpiresAt:    lo.ToPtr(period.To),
		Settlement:   NewInvoiceSettlement(),
	}).Validate())
}

func TestCreateInputValidateRejectsExpiryWithoutEffectiveAt(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	period := timeutil.ClosedPeriod{
		From: now.Add(-2 * time.Hour),
		To:   now.Add(-time.Hour),
	}

	for _, tc := range []struct {
		name      string
		expiresAt time.Time
	}{
		{
			name:      "before now",
			expiresAt: period.To,
		},
		{
			name:      "equal to now",
			expiresAt: now,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a credit purchase without an explicit effective time that
			// expires no later than its creation time
			input := CreateInput{
				Namespace: "test",
				Intent: Intent{
					Intent: meta.Intent{
						ManagedBy:  billing.ManuallyManagedLine,
						CustomerID: "customer-1",
						Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: "tax-code-1",
						},
					},
					IntentMutableFields: IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "historical credit purchase",
							ServicePeriod:     period,
							FullServicePeriod: period,
							BillingPeriod:     period,
						},
						CreditAmount: alpacadecimal.NewFromInt(1),
						ExpiresAt:    lo.ToPtr(tc.expiresAt),
						Settlement:   NewInvoiceSettlement(),
					},
					CostBasis: lo.ToPtr(NewCostBasis(FiatCostBasis{
						Rate: alpacadecimal.NewFromInt(1),
					})),
				},
			}

			// when: the create input is validated
			err := input.Validate()

			// then: expiration at or before creation is rejected
			require.ErrorContains(t, err, "expires at must be after effective at")
		})
	}
}

func TestCreateChargeInputValidateRejectsDynamicCostBasis(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	input := CreateChargeInput{CreateInput: CreateInput{
		Namespace: "test",
		Intent: Intent{
			Intent: meta.Intent{
				ManagedBy:  billing.ManuallyManagedLine,
				CustomerID: "customer-1",
				Currency:   currenciestestutils.NewCustomCurrency(t, "TOKENS", 2),
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: "tax-code-1",
				},
			},
			IntentMutableFields: IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "dynamic credit purchase",
					ServicePeriod:     period,
					FullServicePeriod: period,
					BillingPeriod:     period,
				},
				CreditAmount: alpacadecimal.NewFromInt(1),
				Settlement:   NewInvoiceSettlement(),
			},
			CostBasis: lo.ToPtr(NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.DynamicIntent{
				FiatCurrency: fiatCurrency,
			}))),
		},
	}}

	err = input.Validate()
	require.ErrorContains(t, err, "dynamic cost basis is not supported for credit purchases")
	require.NotContains(t, err.Error(), "requires resolved cost-basis state")
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
