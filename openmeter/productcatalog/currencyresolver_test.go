package productcatalog

import (
	"context"
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestResolvePlanCurrencies(t *testing.T) {
	// given:
	// - a fiat plan with a code-only custom-currency rate card override
	// when:
	// - authoring identities are resolved before persistence
	// then:
	// - both identities are materialized and validation reuses them
	customCode := currencyx.Code("CREDITS")
	customID := "01J00000000000000000000000"
	resolver := &recordingCurrencyResolver{
		identities: map[currencyx.Code]currencyx.CurrencyIdentity{
			currencyx.Code(currency.USD): mustFiatCurrency(t, currencyx.Code(currency.USD)),
			customCode: currencies.Currency{
				NamespacedID: models.NamespacedID{ID: customID},
				Code:         customCode.String(),
			},
		},
	}
	plan := Plan{
		PlanMeta: PlanMeta{Currency: currencyx.Code(currency.USD)},
		Phases: []Phase{{
			PhaseMeta: PhaseMeta{Key: "default"},
			RateCards: RateCards{&FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key:      "fee",
					Name:     "Fee",
					Currency: customCode,
					Price: NewPriceFrom(FlatPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				},
			}},
		}},
	}

	err := ResolvePlanCurrencies(t.Context(), "namespace", resolver, &plan)
	require.NoError(t, err)
	require.IsType(t, &currencyx.FiatCurrency{}, plan.Currency)

	rateCardCurrency, ok := plan.Phases[0].RateCards[0].AsMeta().Currency.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, customID, rateCardCurrency.GetID())
	require.Equal(t, 2, resolver.resolveCalls)

	err = ValidatePlanWithCurrencies(t.Context(), "namespace", resolver)(plan)
	require.NoError(t, err)
	require.Equal(t, 2, resolver.resolveCalls, "resolved identities must avoid code lookups during validation")
}

func TestResolveAddonCurrenciesLeavesInheritedRateCardsUnresolved(t *testing.T) {
	// given:
	// - an add-on whose priced rate card inherits a custom default currency
	// when:
	// - the add-on is prepared for persistence
	// then:
	// - identity stays on the add-on and the rate card remains an inheritance
	customCode := currencyx.Code("CREDITS")
	customID := "01J00000000000000000000000"
	resolver := &recordingCurrencyResolver{
		identities: map[currencyx.Code]currencyx.CurrencyIdentity{
			customCode: currencies.Currency{
				NamespacedID: models.NamespacedID{ID: customID},
				Code:         customCode.String(),
			},
		},
	}
	addon := Addon{
		AddonMeta: AddonMeta{Currency: customCode},
		RateCards: RateCards{&FlatFeeRateCard{
			RateCardMeta: RateCardMeta{
				Key:  "fee",
				Name: "Fee",
				Price: NewPriceFrom(FlatPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
			},
		}},
	}

	err := ResolveAddonCurrencies(t.Context(), "namespace", resolver, &addon)
	require.NoError(t, err)
	managed, ok := addon.Currency.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, customID, managed.GetID())
	require.Nil(t, addon.RateCards[0].AsMeta().Currency)
	require.Equal(t, 1, resolver.resolveCalls)
}

func TestResolvePlanCurrenciesReusesManagedCustomIdentity(t *testing.T) {
	// given:
	// - a persisted plan identity whose custom resource may now be archived
	// when:
	// - the plan is prepared during a later lifecycle operation
	// then:
	// - the stable resource identity is reused without resolving its code again
	customCode := currencyx.Code("CREDITS")
	customID := "01J00000000000000000000000"
	identity := currencies.Currency{
		NamespacedID: models.NamespacedID{ID: customID},
		Code:         customCode.String(),
	}
	resolver := &recordingCurrencyResolver{}
	plan := Plan{PlanMeta: PlanMeta{Currency: identity}}

	err := ResolvePlanCurrencies(t.Context(), "namespace", resolver, &plan)
	require.NoError(t, err)
	require.Equal(t, 0, resolver.resolveCalls)
	require.True(t, identity.Equal(plan.Currency))
}

func TestResolvePlanCurrenciesDoesNotRetargetReusedCode(t *testing.T) {
	// given:
	// - one persisted override points at an archived custom resource
	// - a new authoring override uses a new resource with the same code
	// when:
	// - both identities are prepared together
	// then:
	// - the persisted identity stays unchanged and only the new input resolves
	customCode := currencyx.Code("CREDITS")
	oldID := "01J00000000000000000000000"
	newID := "01J00000000000000000000001"
	oldIdentity := currencies.Currency{
		NamespacedID: models.NamespacedID{ID: oldID},
		Code:         customCode.String(),
	}
	resolver := &recordingCurrencyResolver{
		identities: map[currencyx.Code]currencyx.CurrencyIdentity{
			customCode: currencies.Currency{
				NamespacedID: models.NamespacedID{ID: newID},
				Code:         customCode.String(),
			},
		},
	}
	plan := Plan{
		PlanMeta: PlanMeta{Currency: mustFiatCurrency(t, currencyx.Code(currency.USD))},
		Phases: []Phase{{
			PhaseMeta: PhaseMeta{Key: "default"},
			RateCards: RateCards{
				&FlatFeeRateCard{RateCardMeta: RateCardMeta{
					Key:      "persisted",
					Name:     "Persisted",
					Currency: oldIdentity,
					Price: NewPriceFrom(FlatPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				}},
				&FlatFeeRateCard{RateCardMeta: RateCardMeta{
					Key:      "new",
					Name:     "New",
					Currency: customCode,
					Price: NewPriceFrom(FlatPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				}},
			},
		}},
	}

	err := ResolvePlanCurrencies(t.Context(), "namespace", resolver, &plan)
	require.NoError(t, err)
	persisted := plan.Phases[0].RateCards[0].AsMeta().Currency.(currencyx.ManagedCurrency)
	newCurrency := plan.Phases[0].RateCards[1].AsMeta().Currency.(currencyx.ManagedCurrency)
	require.Equal(t, oldID, persisted.GetID())
	require.Equal(t, newID, newCurrency.GetID())
	require.Equal(t, 1, resolver.resolveCalls)
}

func mustFiatCurrency(t *testing.T, code currencyx.Code) currencyx.Currency {
	t.Helper()

	resolved, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(code).
		Build()
	require.NoError(t, err)

	return resolved
}

type recordingCurrencyResolver struct {
	identities   map[currencyx.Code]currencyx.CurrencyIdentity
	resolveCalls int
}

func (r *recordingCurrencyResolver) Resolve(_ context.Context, _ string, code currencyx.Code) (currencyx.CurrencyIdentity, error) {
	r.resolveCalls++

	identity, ok := r.identities[code]
	if !ok {
		return nil, models.NewGenericNotFoundError(fmt.Errorf("currency %q", code))
	}

	return identity, nil
}

func (r *recordingCurrencyResolver) HasCostBasis(_ context.Context, _ string, _ currencyx.ManagedCurrency, _ currencyx.CurrencyIdentity) (bool, error) {
	return true, nil
}

var _ CurrencyResolver = (*recordingCurrencyResolver)(nil)
