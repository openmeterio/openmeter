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

func TestIntentGetSettlementFiatCurrency(t *testing.T) {
	t.Run("fiat purchase settles in its purchase currency", func(t *testing.T) {
		intent := Intent{
			Intent: meta.Intent{Currency: currenciestestutils.NewFiatCurrency(t, "USD")},
			IntentMutableFields: IntentMutableFields{
				Settlement: NewInvoiceSettlement(),
			},
			CostBasis: NewCostBasis(FiatCostBasis{Rate: alpacadecimal.NewFromInt(1)}),
		}

		fiatCurrency, err := intent.GetSettlementFiatCurrency()
		require.NoError(t, err)
		require.Equal(t, currencyx.FiatCode("USD"), fiatCurrency.GetFiatCode())
	})

	t.Run("custom currency purchase settles in its cost basis currency", func(t *testing.T) {
		usd, err := currencyx.NewFiatCurrency("USD")
		require.NoError(t, err)

		intent := Intent{
			Intent: meta.Intent{Currency: currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)},
			IntentMutableFields: IntentMutableFields{
				Settlement: NewSettlement(ExternalSettlement{
					InitialStatus: CreatedInitialPaymentSettlementStatus,
				}),
			},
			CostBasis: NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
				FiatCurrency: usd,
				Rate:         alpacadecimal.NewFromInt(1),
			})),
		}

		fiatCurrency, err := intent.GetSettlementFiatCurrency()
		require.NoError(t, err)
		require.Equal(t, currencyx.FiatCode("USD"), fiatCurrency.GetFiatCode())
	})

	t.Run("promotional purchase has no settlement fiat currency", func(t *testing.T) {
		intent := Intent{
			Intent: meta.Intent{Currency: currenciestestutils.NewFiatCurrency(t, "USD")},
			IntentMutableFields: IntentMutableFields{
				Settlement: NewSettlement(PromotionalSettlement{}),
			},
		}

		_, err := intent.GetSettlementFiatCurrency()
		require.ErrorContains(t, err, "does not have a settlement fiat currency")
	})

	t.Run("payment-backed purchase requires a cost basis", func(t *testing.T) {
		intent := Intent{
			Intent: meta.Intent{Currency: currenciestestutils.NewFiatCurrency(t, "USD")},
			IntentMutableFields: IntentMutableFields{
				Settlement: NewInvoiceSettlement(),
			},
		}

		_, err := intent.GetSettlementFiatCurrency()
		require.ErrorContains(t, err, "cost basis is required")
	})
}

func TestChargeGetFiatSettlementAmount(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	testCases := map[string]Intent{
		"fiat purchase": {
			Intent: meta.Intent{Currency: currenciestestutils.NewFiatCurrency(t, "USD")},
			IntentMutableFields: IntentMutableFields{
				CreditAmount: alpacadecimal.NewFromFloat(100.12),
				Settlement:   NewInvoiceSettlement(),
			},
			CostBasis: NewCostBasis(FiatCostBasis{Rate: alpacadecimal.NewFromFloat(0.5)}),
		},
		"custom currency purchase": {
			Intent: meta.Intent{Currency: currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)},
			IntentMutableFields: IntentMutableFields{
				CreditAmount: alpacadecimal.NewFromFloat(100.12),
				Settlement:   NewInvoiceSettlement(),
			},
			CostBasis: NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
				FiatCurrency: usd,
				Rate:         alpacadecimal.NewFromFloat(0.5),
			})),
		},
	}

	for name, intent := range testCases {
		t.Run(name, func(t *testing.T) {
			charge := Charge{ChargeBase: ChargeBase{
				Intent: intent,
				State: State{ResolvedCostBasis: &chargecostbasis.State{
					CostBasis:  alpacadecimal.NewFromFloat(0.5),
					ResolvedAt: time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
				}},
			}}

			amount, err := charge.GetFiatSettlementAmount()
			require.NoError(t, err)
			require.Equal(t, float64(50.06), amount.InexactFloat64())
		})
	}
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
					CostBasis: NewCostBasis(FiatCostBasis{
						Rate: alpacadecimal.NewFromInt(1),
					}),
				},
			}

			// when: the create input is validated
			err := input.Validate()

			// then: expiration at or before creation is rejected
			require.ErrorContains(t, err, "expires at must be after effective at")
		})
	}
}

func TestCreateChargeInputValidateInitialCostBasisState(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	initialState := &chargecostbasis.State{
		CostBasis:  alpacadecimal.NewFromInt(1),
		ResolvedAt: period.From,
	}
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
					Name:              "credit purchase",
					ServicePeriod:     period,
					FullServicePeriod: period,
					BillingPeriod:     period,
				},
				CreditAmount: alpacadecimal.NewFromInt(1),
				Settlement:   NewInvoiceSettlement(),
			},
			CostBasis: NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.DynamicIntent{
				FiatCurrency: fiatCurrency,
			})),
		},
	}}

	t.Run("dynamic custom currency requires empty initial state", func(t *testing.T) {
		// given: a dynamic custom-currency purchase with pre-resolved state
		input := input
		input.InitialCostBasisState = initialState

		// when: the adapter create input is validated
		err := input.Validate()

		// then: creation rejects the state because resolution belongs to realization
		require.ErrorContains(t, err, "dynamic credit purchase cannot have initial cost-basis state")

		input.InitialCostBasisState = nil
		require.NoError(t, input.Validate())
	})

	for name, tc := range map[string]struct {
		costBasis    CostBasis
		initialState *chargecostbasis.State
	}{
		"manual": {
			costBasis: NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         alpacadecimal.NewFromInt(1),
			})),
			initialState: initialState,
		},
		"pinned": {
			costBasis: NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.PinnedIntent{
				FiatCurrency:        fiatCurrency,
				CurrencyCostBasisID: "currency-cost-basis-1",
			})),
			initialState: &chargecostbasis.State{
				CostBasisID: lo.ToPtr("currency-cost-basis-1"),
				CostBasis:   alpacadecimal.NewFromInt(1),
				ResolvedAt:  period.From,
			},
		},
	} {
		t.Run(name+" custom currency requires initial state", func(t *testing.T) {
			// given: a resolvable custom-currency purchase without its initial state
			input := input
			input.Intent.CostBasis = tc.costBasis

			// when: the adapter create input is validated
			err := input.Validate()

			// then: creation requires the resolver output for persistence
			require.ErrorContains(t, err, "manual or pinned credit purchase requires initial cost-basis state")

			input.InitialCostBasisState = tc.initialState
			require.NoError(t, input.Validate())
		})
	}

	t.Run("fiat requires empty initial state", func(t *testing.T) {
		// given: a fiat purchase with redundant derived state
		input := input
		input.Intent.Currency = currenciestestutils.NewFiatCurrency(t, "USD")
		input.Intent.CostBasis = NewCostBasis(FiatCostBasis{Rate: alpacadecimal.NewFromInt(1)})
		input.InitialCostBasisState = initialState

		// when: the adapter create input is validated
		err := input.Validate()

		// then: creation rejects state that the read mapper reconstructs from intent
		require.ErrorContains(t, err, "fiat credit purchase cannot have initial cost-basis state")

		input.InitialCostBasisState = nil
		require.NoError(t, input.Validate())
	})

	t.Run("promotional requires empty initial state", func(t *testing.T) {
		// given: a promotional purchase with cost-basis state
		input := input
		input.Intent.Currency = currenciestestutils.NewFiatCurrency(t, "USD")
		input.Intent.Settlement = NewSettlement(PromotionalSettlement{})
		input.Intent.CostBasis = CostBasis{}
		input.InitialCostBasisState = initialState

		// when: the adapter create input is validated
		err := input.Validate()

		// then: creation rejects monetary state for a free grant
		require.ErrorContains(t, err, "promotional credit purchase cannot have initial cost-basis state")

		input.InitialCostBasisState = nil
		require.NoError(t, input.Validate())
	})
}

func TestSetResolvedCostBasisInputValidate(t *testing.T) {
	valid := SetResolvedCostBasisInput{
		ChargeID: meta.ChargeID{
			Namespace: "test",
			ID:        "charge-1",
		},
		ChargeCostBasisID: "charge-cost-basis-1",
		State: chargecostbasis.State{
			CostBasis:   alpacadecimal.NewFromInt(1),
			CostBasisID: lo.ToPtr("currency-cost-basis-1"),
			ResolvedAt:  time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	require.NoError(t, valid.Validate())

	t.Run("requires charge ID", func(t *testing.T) {
		input := valid
		input.ChargeID = meta.ChargeID{}

		require.ErrorContains(t, input.Validate(), "charge ID")
	})

	t.Run("requires charge cost basis ID", func(t *testing.T) {
		input := valid
		input.ChargeCostBasisID = ""

		require.ErrorContains(t, input.Validate(), "charge cost basis ID is required")
	})

	t.Run("validates state", func(t *testing.T) {
		input := valid
		input.State.CostBasis = alpacadecimal.Zero

		require.ErrorContains(t, input.Validate(), "state")
	})
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
