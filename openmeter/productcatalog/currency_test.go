package productcatalog

import (
	"context"
	"fmt"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidatePlanRateCardCurrencies(t *testing.T) {
	custom := currencyx.Code("CREDITS")

	tests := []struct {
		name         string
		planCurrency currencyx.CurrencyIdentity
		override     currencyx.CurrencyIdentity
		expected     error
	}{
		{
			name:     "missing plan currency",
			expected: ErrCurrencyInvalid,
		},
		{
			name:         "fiat plan with inherited currency",
			planCurrency: currencyx.Code(currency.USD),
		},
		{
			name:         "fiat plan with custom override",
			planCurrency: currencyx.Code(currency.USD),
			override:     custom,
		},
		{
			name:         "fiat plan with redundant override",
			planCurrency: currencyx.Code(currency.USD),
			override:     currencyx.Code(currency.USD),
			expected:     ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:         "fiat plan with second fiat",
			planCurrency: currencyx.Code(currency.USD),
			override:     currencyx.Code(currency.EUR),
			expected:     ErrPlanMultipleFiatCurrencies,
		},
		{
			name:         "custom plan with inherited currency",
			planCurrency: custom,
		},
		{
			name:         "custom plan with override",
			planCurrency: custom,
			override:     currencyx.Code("TOKENS"),
			expected:     ErrRateCardCurrencyOverrideNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := Plan{
				PlanMeta: PlanMeta{Currency: tt.planCurrency},
				Phases: []Phase{{
					PhaseMeta: PhaseMeta{Key: "default"},
					RateCards: RateCards{&FlatFeeRateCard{
						RateCardMeta: RateCardMeta{Key: "base", Currency: tt.override},
					}},
				}},
			}

			err := ValidatePlanRateCardCurrencies()(plan)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestValidatePlanWithCurrencies(t *testing.T) {
	custom := currencyx.Code("CREDITS")
	customCurrencyID := "currency-id"
	customCurrency := currencies.Currency{
		NamespacedID: models.NamespacedID{ID: customCurrencyID},
		Code:         custom.String(),
	}

	tests := []struct {
		name     string
		resolver *testCurrencyResolver
		expected error
	}{
		{
			name: "matching cost basis",
			resolver: &testCurrencyResolver{
				currencies: map[currencyx.Code]currencyx.CurrencyIdentity{custom: customCurrency},
				costBases:  map[string]bool{"currency-id|USD": true},
			},
		},
		{
			name:     "unknown custom currency",
			resolver: &testCurrencyResolver{},
			expected: ErrCurrencyNotFound,
		},
		{
			name: "missing cost basis",
			resolver: &testCurrencyResolver{
				currencies: map[currencyx.Code]currencyx.CurrencyIdentity{custom: customCurrency},
			},
			expected: ErrCurrencyCostBasisNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a fiat plan with one custom-currency rate card
			// when:
			// - managed currency and cost-basis identities are validated
			// then:
			// - only a known custom currency with a USD cost-basis pair is valid
			plan := Plan{
				PlanMeta: PlanMeta{Currency: currencyx.Code(currency.USD)},
				Phases: []Phase{{
					PhaseMeta: PhaseMeta{Key: "default"},
					RateCards: RateCards{&FlatFeeRateCard{
						RateCardMeta: RateCardMeta{Key: "base", Currency: custom},
					}},
				}},
			}

			err := ValidatePlanWithCurrencies(t.Context(), "namespace", tt.resolver)(plan)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestCostBasisValidationCachesByManagedCurrencyIdentity(t *testing.T) {
	// given:
	// - two managed custom currency resources reuse the same code
	// - only the older resource has a cost-basis pair with USD
	// when:
	// - catalog cost-basis validation checks both priced rate cards
	// then:
	// - each managed identity is checked independently despite the shared code
	oldCredits := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "old-credits-id"},
		Code:         "CREDITS",
	}
	newCredits := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "new-credits-id"},
		Code:         "CREDITS",
	}

	newRateCard := func(key string, identity currencyx.CurrencyIdentity) RateCard {
		return &FlatFeeRateCard{RateCardMeta: RateCardMeta{
			Key:      key,
			Name:     key,
			Currency: identity,
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		}}
	}

	rateCards := RateCards{
		newRateCard("old", oldCredits),
		newRateCard("new", newCredits),
	}

	tests := []struct {
		name     string
		validate func(*testCurrencyResolver) error
	}{
		{
			name: "plan",
			validate: func(resolver *testCurrencyResolver) error {
				return ValidatePlanWithCurrencies(t.Context(), "namespace", resolver)(Plan{
					PlanMeta: PlanMeta{Currency: currencyx.Code(currency.USD)},
					Phases: []Phase{{
						PhaseMeta: PhaseMeta{Key: "default"},
						RateCards: rateCards,
					}},
				})
			},
		},
		{
			name: "addon",
			validate: func(resolver *testCurrencyResolver) error {
				return ValidateAddonWithCurrencies(t.Context(), "namespace", resolver)(Addon{
					AddonMeta: AddonMeta{Currency: currencyx.Code(currency.USD)},
					RateCards: rateCards,
				})
			},
		},
		{
			name: "plan addon assignment",
			validate: func(resolver *testCurrencyResolver) error {
				return ValidatePlanAddonWithCurrencies(t.Context(), "namespace", resolver)(PlanAddon{
					Plan: Plan{PlanMeta: PlanMeta{Currency: currencyx.Code(currency.USD)}},
					Addon: Addon{
						AddonMeta: AddonMeta{Currency: currencyx.Code(currency.USD)},
						RateCards: rateCards,
					},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &testCurrencyResolver{
				costBases: map[string]bool{"old-credits-id|USD": true},
			}

			err := tt.validate(resolver)
			require.ErrorIs(t, err, ErrCurrencyCostBasisNotFound)
			require.ElementsMatch(t, []string{"old-credits-id|USD", "new-credits-id|USD"}, resolver.costBasisCalls)
		})
	}
}

func TestRateCardCurrencyRequiresPrice(t *testing.T) {
	custom := currencyx.Code("CREDITS")

	t.Run("currency without price is invalid", func(t *testing.T) {
		err := (RateCardMeta{Currency: custom}).Validate()
		require.ErrorIs(t, err, ErrRateCardCurrencyRequiresPrice)
	})

	t.Run("currency with price is valid", func(t *testing.T) {
		err := (RateCardMeta{
			Currency: custom,
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		}).Validate()
		require.NoError(t, err)
	})
}

type testCurrencyResolver struct {
	currencies     map[currencyx.Code]currencyx.CurrencyIdentity
	costBases      map[string]bool
	costBasisCalls []string
}

func (r *testCurrencyResolver) Resolve(_ context.Context, _ string, code currencyx.Code) (currencyx.CurrencyIdentity, error) {
	if code.IsFiat() {
		return code, nil
	}

	resolved, ok := r.currencies[code]
	if !ok {
		return nil, models.NewGenericNotFoundError(fmt.Errorf("currency %q", code))
	}

	return resolved, nil
}

func (r *testCurrencyResolver) HasCostBasis(_ context.Context, _ string, customCurrency currencyx.ManagedCurrency, fiatCurrency currencyx.CurrencyIdentity) (bool, error) {
	key := customCurrency.GetID() + "|" + fiatCurrency.GetCode().String()
	r.costBasisCalls = append(r.costBasisCalls, key)

	return r.costBases[key], nil
}

var _ CurrencyResolver = (*testCurrencyResolver)(nil)
