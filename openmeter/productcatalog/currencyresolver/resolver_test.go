package currencyresolver_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/currencyresolver"
	productcatalogtestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestResolver(t *testing.T) {
	fixture := newResolverFixture(t)

	t.Run("CurrencyReference", func(t *testing.T) {
		testResolveCurrency(t, fixture)
	})

	t.Run("Plan", func(t *testing.T) {
		testResolveCurrenciesForPlan(t, fixture)
	})

	t.Run("Addon", func(t *testing.T) {
		testResolveCurrenciesForAddon(t, fixture)
	})
}

func testResolveCurrency(t *testing.T, fixture resolverFixture) {
	t.Run("fiat currency", func(t *testing.T) {
		// given:
		// - a fiat currency reference
		reference := currencies.NewCurrencyReference("USD")

		// when:
		// - the reference is resolved
		err := currencyresolver.ResolveCurrency(t.Context(), fixture.resolver, &reference)

		// then:
		// - the fiat code is sufficient to consider the reference resolved
		require.NoError(t, err)
		assertResolvedFiatReference(t, reference, "USD")
	})

	t.Run("custom currency without id", func(t *testing.T) {
		// given:
		// - a code-only custom currency reference
		reference := currencies.NewCurrencyReference(fixture.credits.GetCode())

		// when:
		// - the reference is resolved from the database
		err := currencyresolver.ResolveCurrency(t.Context(), fixture.resolver, &reference)

		// then:
		// - the managed identity and expanded cost basis are populated
		require.NoError(t, err)
		assertResolvedCustomReference(t, reference, fixture.credits)
	})

	t.Run("custom currency with id", func(t *testing.T) {
		// given:
		// - a custom currency reference carrying its stable managed identity
		reference := currencies.NewCurrencyReference(fixture.credits.GetCode())
		reference.CustomCurrencyID = &fixture.credits.ID

		// when:
		// - the reference is resolved by identity from the database
		err := currencyresolver.ResolveCurrency(t.Context(), fixture.resolver, &reference)

		// then:
		// - the same managed currency and its cost basis are populated
		require.NoError(t, err)
		assertResolvedCustomReference(t, reference, fixture.credits)
	})

	t.Run("invalid", func(t *testing.T) {
		testCases := []struct {
			name          string
			reference     currencies.CurrencyReference
			expectedError error
			errorContains string
		}{
			{
				name:          "empty reference",
				reference:     currencies.CurrencyReference{},
				errorContains: "invalid currency reference",
			},
			{
				name:          "unknown custom code",
				reference:     currencies.NewCurrencyReference("UNKNOWN"),
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name:          "unknown custom id",
				reference:     *currencyReferencePointer(fixture.credits.GetCode(), "01J00000000000000000000000"),
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name:          "deleted custom currency by code",
				reference:     currencies.NewCurrencyReference(fixture.archived.GetCode()),
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name:          "deleted custom currency by id",
				reference:     *currencyReferencePointer(fixture.archived.GetCode(), fixture.archived.ID),
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name: "custom code and id identify different currencies",
				reference: currencies.CurrencyReference{
					Code:             fixture.credits.GetCode(),
					CustomCurrencyID: &fixture.tokens.ID,
				},
				errorContains: "code mismatch between reference and currency",
			},
			{
				name: "fiat currency with custom id",
				reference: currencies.CurrencyReference{
					Code:             "USD",
					CustomCurrencyID: &fixture.credits.ID,
				},
				errorContains: "fiat currency cannot have a custom currency id",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				// when:
				// - an invalid or missing reference is resolved
				err := currencyresolver.ResolveCurrency(t.Context(), fixture.resolver, &testCase.reference)

				// then:
				// - resolution reports the invalid reference
				require.Error(t, err)
				if testCase.expectedError != nil {
					assert.ErrorIs(t, err, testCase.expectedError)
				}
				if testCase.errorContains != "" {
					assert.ErrorContains(t, err, testCase.errorContains)
				}
			})
		}
	})
}

func testResolveCurrenciesForPlan(t *testing.T, fixture resolverFixture) {
	t.Run("fiat currency without rate card override", func(t *testing.T) {
		// given:
		// - a USD plan whose rate card inherits the plan currency
		plan := &productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Currency: currencies.NewCurrencyReference("USD"),
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
				RateCards: productcatalog.RateCards{
					newRateCard("base", nil),
				},
			}},
		}

		// when:
		// - all plan currencies are resolved
		err := currencyresolver.ResolveCurrenciesForPlan(t.Context(), fixture.resolver, plan)

		// then:
		// - the plan retains its resolved fiat default and no override is introduced
		require.NoError(t, err)
		assertResolvedFiatReference(t, plan.Currency, "USD")
		assert.Nil(t, plan.Phases[0].RateCards[0].AsMeta().Currency)
	})

	t.Run("fiat currency with custom rate card override", func(t *testing.T) {
		// given:
		// - a USD plan with a code-only custom currency override
		override := currencies.NewCurrencyReference(fixture.credits.GetCode())
		plan := &productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Currency: currencies.NewCurrencyReference("USD"),
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
				RateCards: productcatalog.RateCards{
					newRateCard("base", &override),
				},
			}},
		}

		// when:
		// - all plan currencies are resolved
		err := currencyresolver.ResolveCurrenciesForPlan(t.Context(), fixture.resolver, plan)

		// then:
		// - the custom override contains its database identity and USD cost basis
		require.NoError(t, err)
		assertResolvedFiatReference(t, plan.Currency, "USD")
		resolvedOverride := requireRateCardCurrency(t, plan.Phases[0].RateCards[0])
		assertResolvedCustomReference(t, resolvedOverride, fixture.credits)
	})

	t.Run("custom currency without rate card override", func(t *testing.T) {
		// given:
		// - a code-only custom-currency plan whose rate card inherits that currency
		plan := &productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Currency: currencies.NewCurrencyReference(fixture.credits.GetCode()),
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
				RateCards: productcatalog.RateCards{
					newRateCard("base", nil),
				},
			}},
		}

		// when:
		// - all plan currencies are resolved
		err := currencyresolver.ResolveCurrenciesForPlan(t.Context(), fixture.resolver, plan)

		// then:
		// - the plan default contains the managed custom currency
		require.NoError(t, err)
		assertResolvedCustomReference(t, plan.Currency, fixture.credits)
		assert.Nil(t, plan.Phases[0].RateCards[0].AsMeta().Currency)
	})

	t.Run("invalid", func(t *testing.T) {
		testCases := []struct {
			name          string
			plan          *productcatalog.Plan
			expectedError error
			errorContains string
		}{
			{
				name:          "nil plan",
				errorContains: "plan is required",
			},
			{
				name: "unknown plan currency",
				plan: &productcatalog.Plan{
					PlanMeta: productcatalog.PlanMeta{
						Currency: currencies.NewCurrencyReference("UNKNOWN"),
					},
				},
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name: "unknown rate card override",
				plan: &productcatalog.Plan{
					PlanMeta: productcatalog.PlanMeta{
						Currency: currencies.NewCurrencyReference("USD"),
					},
					Phases: []productcatalog.Phase{{
						PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
						RateCards: productcatalog.RateCards{
							newRateCard("base", currencyReferencePointer("UNKNOWN", "")),
						},
					}},
				},
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name: "rate card override code and id identify different currencies",
				plan: &productcatalog.Plan{
					PlanMeta: productcatalog.PlanMeta{
						Currency: currencies.NewCurrencyReference("USD"),
					},
					Phases: []productcatalog.Phase{{
						PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
						RateCards: productcatalog.RateCards{
							newRateCard("base", currencyReferencePointer(fixture.credits.GetCode(), fixture.tokens.ID)),
						},
					}},
				},
				errorContains: "code mismatch between reference and currency",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				// when:
				// - a plan with an invalid or missing currency is resolved
				err := currencyresolver.ResolveCurrenciesForPlan(t.Context(), fixture.resolver, testCase.plan)

				// then:
				// - aggregate resolution reports the invalid catalog field
				require.Error(t, err)
				if testCase.expectedError != nil {
					assert.ErrorIs(t, err, testCase.expectedError)
				}
				if testCase.errorContains != "" {
					assert.ErrorContains(t, err, testCase.errorContains)
				}
			})
		}
	})
}

func testResolveCurrenciesForAddon(t *testing.T, fixture resolverFixture) {
	t.Run("fiat currency without rate card override", func(t *testing.T) {
		// given:
		// - a USD add-on whose rate card inherits the add-on currency
		addon := &productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Currency: currencies.NewCurrencyReference("USD"),
			},
			RateCards: productcatalog.RateCards{
				newRateCard("base", nil),
			},
		}

		// when:
		// - all add-on currencies are resolved
		err := currencyresolver.ResolveCurrenciesForAddon(t.Context(), fixture.resolver, addon)

		// then:
		// - the add-on retains its resolved fiat default and no override is introduced
		require.NoError(t, err)
		assertResolvedFiatReference(t, addon.Currency, "USD")
		assert.Nil(t, addon.RateCards[0].AsMeta().Currency)
	})

	t.Run("fiat currency with custom rate card override", func(t *testing.T) {
		// given:
		// - a USD add-on with a code-only custom currency override
		override := currencies.NewCurrencyReference(fixture.credits.GetCode())
		addon := &productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Currency: currencies.NewCurrencyReference("USD"),
			},
			RateCards: productcatalog.RateCards{
				newRateCard("base", &override),
			},
		}

		// when:
		// - all add-on currencies are resolved
		err := currencyresolver.ResolveCurrenciesForAddon(t.Context(), fixture.resolver, addon)

		// then:
		// - the custom override contains its database identity and USD cost basis
		require.NoError(t, err)
		assertResolvedFiatReference(t, addon.Currency, "USD")
		resolvedOverride := requireRateCardCurrency(t, addon.RateCards[0])
		assertResolvedCustomReference(t, resolvedOverride, fixture.credits)
	})

	t.Run("custom currency without rate card override", func(t *testing.T) {
		// given:
		// - a code-only custom-currency add-on whose rate card inherits that currency
		addon := &productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Currency: currencies.NewCurrencyReference(fixture.credits.GetCode()),
			},
			RateCards: productcatalog.RateCards{
				newRateCard("base", nil),
			},
		}

		// when:
		// - all add-on currencies are resolved
		err := currencyresolver.ResolveCurrenciesForAddon(t.Context(), fixture.resolver, addon)

		// then:
		// - the add-on default contains the managed custom currency
		require.NoError(t, err)
		assertResolvedCustomReference(t, addon.Currency, fixture.credits)
		assert.Nil(t, addon.RateCards[0].AsMeta().Currency)
	})

	t.Run("invalid", func(t *testing.T) {
		testCases := []struct {
			name          string
			addon         *productcatalog.Addon
			expectedError error
			errorContains string
		}{
			{
				name:          "nil add-on",
				errorContains: "add-on is required",
			},
			{
				name: "unknown add-on currency",
				addon: &productcatalog.Addon{
					AddonMeta: productcatalog.AddonMeta{
						Currency: currencies.NewCurrencyReference("UNKNOWN"),
					},
				},
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name: "unknown rate card override",
				addon: &productcatalog.Addon{
					AddonMeta: productcatalog.AddonMeta{
						Currency: currencies.NewCurrencyReference("USD"),
					},
					RateCards: productcatalog.RateCards{
						newRateCard("base", currencyReferencePointer("UNKNOWN", "")),
					},
				},
				expectedError: productcatalog.ErrCurrencyNotFound,
			},
			{
				name: "rate card override code and id identify different currencies",
				addon: &productcatalog.Addon{
					AddonMeta: productcatalog.AddonMeta{
						Currency: currencies.NewCurrencyReference("USD"),
					},
					RateCards: productcatalog.RateCards{
						newRateCard("base", currencyReferencePointer(fixture.credits.GetCode(), fixture.tokens.ID)),
					},
				},
				errorContains: "code mismatch between reference and currency",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				// when:
				// - an add-on with an invalid or missing currency is resolved
				err := currencyresolver.ResolveCurrenciesForAddon(t.Context(), fixture.resolver, testCase.addon)

				// then:
				// - aggregate resolution reports the invalid catalog field
				require.Error(t, err)
				if testCase.expectedError != nil {
					assert.ErrorIs(t, err, testCase.expectedError)
				}
				if testCase.errorContains != "" {
					assert.ErrorContains(t, err, testCase.errorContains)
				}
			})
		}
	})
}

type resolverFixture struct {
	resolver currencies.NamespacedCurrencyResolver
	credits  currencies.Currency
	tokens   currencies.Currency
	archived currencies.Currency
}

func newResolverFixture(t *testing.T) resolverFixture {
	t.Helper()

	env := productcatalogtestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)

	credits, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		namespace,
		"CREDITS",
		"Credits",
		"CR",
	))
	require.NoError(t, err)

	tokens, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		namespace,
		"TOKENS",
		"Tokens",
		"T",
	))
	require.NoError(t, err)

	archived, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		namespace,
		"ARCHIVED",
		"Archived",
		"A",
	))
	require.NoError(t, err)

	_, err = env.Client.CustomCurrency.UpdateOneID(archived.ID).
		SetDeletedAt(clock.Now()).
		Save(t.Context())
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: credits.ID,
		FiatCode:   "USD",
		Rate:       alpacadecimal.RequireFromString("0.01"),
	})
	require.NoError(t, err)

	return resolverFixture{
		resolver: env.CurrencyResolver.WithNamespace(namespace),
		credits:  credits,
		tokens:   tokens,
		archived: archived,
	}
}

func newRateCard(key string, reference *currencies.CurrencyReference) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      key,
			Name:     key,
			Currency: reference,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
}

func assertResolvedFiatReference(t *testing.T, reference currencies.CurrencyReference, expectedCode currencyx.Code) {
	t.Helper()

	require.Truef(t, reference.IsResolved(), "it should be resolved")
	assert.Equalf(t, expectedCode, reference.Code, "code must match")
	assert.Nil(t, reference.CustomCurrencyID, "custom currency id should not be nil")
}

func assertResolvedCustomReference(t *testing.T, reference currencies.CurrencyReference, expected currencies.Currency) {
	t.Helper()

	require.True(t, reference.IsResolved())
	require.NotNil(t, reference.CustomCurrencyID)
	assert.Equal(t, expected.ID, *reference.CustomCurrencyID)
	assert.Equal(t, expected.GetCode(), reference.Code)

	resolved, ok := reference.CustomCurrency()
	require.True(t, ok)
	require.NotNil(t, resolved)
	assert.Equal(t, expected.ID, resolved.ID)
	assert.Equal(t, expected.Namespace, resolved.Namespace)
	assert.Equal(t, expected.GetCode(), resolved.GetCode())
	require.NotNil(t, resolved.CostBasis)
	require.Len(t, *resolved.CostBasis, 1)
	assert.Equal(t, currencyx.Code("USD"), (*resolved.CostBasis)[0].FiatCode)
	assert.Equal(t, float64(0.01), (*resolved.CostBasis)[0].Rate.InexactFloat64())
}

func requireRateCardCurrency(t *testing.T, rateCard productcatalog.RateCard) currencies.CurrencyReference {
	t.Helper()

	reference := rateCard.AsMeta().Currency
	require.NotNil(t, reference)

	return *reference
}

func currencyReferencePointer(code currencyx.Code, id string) *currencies.CurrencyReference {
	reference := currencies.NewCurrencyReference(code)
	if id != "" {
		reference.CustomCurrencyID = &id
	}

	return &reference
}
