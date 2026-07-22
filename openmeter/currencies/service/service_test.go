package service_test

import (
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestCurrenciesService(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := currenciestestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)

	t.Run("CustomCurrency", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			// given:
			// - valid custom currency details
			input := currencies.CreateCurrencyInput{
				Namespace: namespace,
				CurrencyDetails: currencyx.CurrencyDetails{
					Code:               "TOKENS",
					Name:               "Tokens",
					Symbol:             "T",
					Precision:          2,
					DecimalMark:        ".",
					ThousandsSeparator: ",",
				},
			}

			// when:
			// - the custom currency is created
			createdCurrency, err := env.Service.CreateCurrency(t.Context(), input)

			// then:
			// - its managed identity and formatting details are persisted
			require.NoError(t, err)
			require.NotEmpty(t, createdCurrency.ID)
			assert.Equal(t, namespace, createdCurrency.Namespace)
			assert.Equal(t, currencyx.CurrencyTypeCustom, createdCurrency.Type())
			assert.Equal(t, input.CurrencyDetails, createdCurrency.Details())
			assert.Nil(t, createdCurrency.CostBasis)

			t.Run("Get", func(t *testing.T) {
				// when:
				// - the newly created custom currency is retrieved without expansions
				result, err := env.Service.GetCurrency(t.Context(), currencies.GetCurrencyInput{
					NamespacedID: createdCurrency.NamespacedID,
				})

				// then:
				// - the same currency is returned without cost-basis data
				require.NoError(t, err)
				assert.Equal(t, createdCurrency.ID, result.ID)
				assert.Equal(t, input.CurrencyDetails, result.Details())
				assert.Nil(t, result.CostBasis)
			})

			t.Run("Invalid", func(t *testing.T) {
				// given:
				// - custom currency details with an invalid code and missing name
				invalidInput := currencies.CreateCurrencyInput{
					Namespace: namespace,
					CurrencyDetails: currencyx.CurrencyDetails{
						Code:               "BAD",
						Precision:          2,
						DecimalMark:        ".",
						ThousandsSeparator: ",",
					},
				}

				// when:
				// - the invalid custom currency is created
				_, err := env.Service.CreateCurrency(t.Context(), invalidInput)

				// then:
				// - validation fails before persistence
				require.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err))
				assert.Contains(t, err.Error(), "currency code")
				assert.Contains(t, err.Error(), "name is required")
			})

			t.Run("CostBasis", func(t *testing.T) {
				// given:
				// - the newly created custom currency
				// when:
				// - its initial USD cost basis is created
				usd, err := env.Service.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
					Namespace:  namespace,
					CurrencyID: createdCurrency.ID,
					FiatCode:   "USD",
					Rate:       alpacadecimal.RequireFromString("0.01"),
				})

				// then:
				// - the cost basis is immediately effective and open-ended
				require.NoError(t, err)
				require.NotEmpty(t, usd.ID)
				assert.Equal(t, createdCurrency.ID, usd.CurrencyID)
				assert.Equal(t, currencyx.Code("USD"), usd.FiatCode)
				assert.Equal(t, float64(0.01), usd.Rate.InexactFloat64())
				assert.Equal(t, now, usd.EffectiveFrom)
				assert.Nil(t, usd.EffectiveTo)

				t.Run("Get", func(t *testing.T) {
					// when:
					// - the newly created cost basis is retrieved without expansions
					result, err := env.Service.GetCostBasis(t.Context(), currencies.GetCostBasisInput{
						NamespacedID: usd.NamespacedID,
					})

					// then:
					// - the persisted cost basis is returned without its custom currency
					require.NoError(t, err)
					assert.Equal(t, usd.ID, result.ID)
					assert.Equal(t, usd.Namespace, result.Namespace)
					assert.Equal(t, usd.CurrencyID, result.CurrencyID)
					assert.Equal(t, usd.FiatCode, result.FiatCode)
					assert.Equal(t, usd.Rate.InexactFloat64(), result.Rate.InexactFloat64())
					assert.Equal(t, usd.EffectiveFrom, result.EffectiveFrom)
					assert.Equal(t, usd.EffectiveTo, result.EffectiveTo)
					assert.Nil(t, result.CustomCurrency)

					t.Run("WithCustomCurrency", func(t *testing.T) {
						// when:
						// - the cost basis is retrieved with its custom currency expanded
						result, err := env.Service.GetCostBasis(t.Context(), currencies.GetCostBasisInput{
							NamespacedID: usd.NamespacedID,
							CostBasisExpandOptions: currencies.CostBasisExpandOptions{
								CustomCurrency: true,
							},
						})

						// then:
						// - the cost basis includes the custom currency details
						require.NoError(t, err)
						assert.Equal(t, usd.ID, result.ID)
						require.NotNil(t, result.CustomCurrency)
						assert.Equal(t, createdCurrency.ID, result.CustomCurrency.ID)
						assert.Equal(t, createdCurrency.Namespace, result.CustomCurrency.Namespace)
						assert.Equal(t, createdCurrency.Details(), result.CustomCurrency.Details())
						assert.Nil(t, result.CustomCurrency.CostBasis)
					})

					t.Run("NotFound", func(t *testing.T) {
						// when:
						// - the cost basis is retrieved from another namespace
						_, err := env.Service.GetCostBasis(t.Context(), currencies.GetCostBasisInput{
							NamespacedID: models.NamespacedID{
								Namespace: currenciestestutils.NewTestNamespace(t),
								ID:        usd.ID,
							},
						})

						// then:
						// - the namespace boundary is enforced
						require.Error(t, err)
						assert.True(t, models.IsGenericNotFoundError(err))
					})

					t.Run("Invalid", func(t *testing.T) {
						// when:
						// - a cost basis is retrieved without an identity
						_, err := env.Service.GetCostBasis(t.Context(), currencies.GetCostBasisInput{})

						// then:
						// - validation fails before querying the repository
						require.Error(t, err)
						assert.True(t, models.IsGenericValidationError(err))
					})
				})

				t.Run("ListWithCostBasis", func(t *testing.T) {
					// given:
					// - a custom currency with an active USD cost basis
					testCases := []struct {
						name             string
						currencyType     *currencies.CurrencyType
						expectedTotal    int
						expectFiatResult bool
					}{
						{
							name:          "custom currencies",
							currencyType:  lo.ToPtr(currencies.CurrencyTypeCustom),
							expectedTotal: 1,
						},
						{
							name:             "custom and fiat currencies",
							expectedTotal:    2,
							expectFiatResult: true,
						},
					}

					for _, testCase := range testCases {
						t.Run(testCase.name, func(t *testing.T) {
							// when:
							// - currencies are listed with cost-basis data expanded
							result, err := env.Service.ListCurrencies(t.Context(), currencies.ListCurrenciesInput{
								Page:         pagination.NewPage(1, 10),
								Namespace:    namespace,
								CurrencyType: testCase.currencyType,
								Code: &filter.FilterString{
									In: lo.ToPtr([]string{"TOKENS", "USD"}),
								},
								CurrencyExpandOptions: currencies.CurrencyExpandOptions{
									CostBasis: true,
								},
							})

							// then:
							// - custom currencies include active cost-basis data while fiat currencies do not
							require.NoError(t, err)
							require.Equal(t, testCase.expectedTotal, result.TotalCount)
							require.Len(t, result.Items, testCase.expectedTotal)

							customCurrencies := lo.Filter(result.Items, func(item currencies.Currency, _ int) bool {
								return item.Type() == currencyx.CurrencyTypeCustom
							})
							require.Len(t, customCurrencies, 1)
							assert.Equal(t, createdCurrency.ID, customCurrencies[0].ID)
							require.NotNil(t, customCurrencies[0].CostBasis)
							require.Len(t, *customCurrencies[0].CostBasis, 1)
							assert.Equal(t, usd.ID, (*customCurrencies[0].CostBasis)[0].ID)

							fiatCurrencies := lo.Filter(result.Items, func(item currencies.Currency, _ int) bool {
								return item.Type() == currencyx.CurrencyTypeFiat
							})
							if testCase.expectFiatResult {
								require.Len(t, fiatCurrencies, 1)
								assert.Equal(t, currencyx.Code("USD"), fiatCurrencies[0].Details().Code)
								assert.Nil(t, fiatCurrencies[0].CostBasis)
							} else {
								assert.Empty(t, fiatCurrencies)
							}
						})
					}
				})

				t.Run("Multiple", func(t *testing.T) {
					// given:
					// - a custom currency with an active USD cost basis
					// when:
					// - an active EUR basis and a future USD replacement are created
					eur, err := env.Service.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
						Namespace:  namespace,
						CurrencyID: createdCurrency.ID,
						FiatCode:   "EUR",
						Rate:       alpacadecimal.RequireFromString("0.009"),
					})
					require.NoError(t, err)
					assert.Equal(t, now, eur.EffectiveFrom)
					assert.Nil(t, eur.EffectiveTo)

					futureEffectiveFrom := now.Add(24 * time.Hour)
					futureUSD, err := env.Service.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
						Namespace:     namespace,
						CurrencyID:    createdCurrency.ID,
						FiatCode:      "USD",
						Rate:          alpacadecimal.RequireFromString("0.012"),
						EffectiveFrom: &futureEffectiveFrom,
					})
					require.NoError(t, err)
					assert.Equal(t, futureEffectiveFrom, futureUSD.EffectiveFrom)
					assert.Nil(t, futureUSD.EffectiveTo)

					t.Run("Get", func(t *testing.T) {
						// when:
						// - the custom currency is retrieved with cost bases expanded
						result, err := env.Service.GetCurrency(t.Context(), currencies.GetCurrencyInput{
							NamespacedID: createdCurrency.NamespacedID,
							CurrencyExpandOptions: currencies.CurrencyExpandOptions{
								CostBasis: true,
							},
						})

						// then:
						// - only the currently active USD and EUR cost bases are returned
						require.NoError(t, err)
						require.NotNil(t, result.CostBasis)
						require.Len(t, *result.CostBasis, 2)

						byFiatCode := lo.SliceToMap(*result.CostBasis, func(item currencies.CostBasis) (currencyx.Code, currencies.CostBasis) {
							return item.FiatCode, item
						})
						require.Contains(t, byFiatCode, currencyx.Code("USD"))
						require.Contains(t, byFiatCode, currencyx.Code("EUR"))
						assert.Equal(t, float64(0.01), byFiatCode["USD"].Rate.InexactFloat64())
						assert.Equal(t, float64(0.009), byFiatCode["EUR"].Rate.InexactFloat64())
						assert.Equal(t, futureEffectiveFrom, lo.FromPtr(byFiatCode["USD"].EffectiveTo))
						assert.Nil(t, byFiatCode["EUR"].EffectiveTo)
					})

					t.Run("List", func(t *testing.T) {
						// when:
						// - the complete cost-basis history is listed
						result, err := env.Service.ListCostBases(t.Context(), currencies.ListCostBasesInput{
							Page:       pagination.NewPage(1, 10),
							Namespace:  namespace,
							CurrencyID: createdCurrency.ID,
						})

						// then:
						// - both fiat currencies and the superseding USD entry are returned
						require.NoError(t, err)
						require.Equal(t, 3, result.TotalCount)
						require.Len(t, result.Items, 3)
						assert.True(t, slices.IsSortedFunc(result.Items, func(a, b currencies.CostBasis) int {
							return b.EffectiveFrom.Compare(a.EffectiveFrom)
						}))

						usdItems := lo.Filter(result.Items, func(item currencies.CostBasis, _ int) bool {
							return item.FiatCode == "USD"
						})
						require.Len(t, usdItems, 2)
						assert.Equal(t, futureUSD.ID, usdItems[0].ID)
						assert.Equal(t, futureEffectiveFrom, usdItems[0].EffectiveFrom)
						assert.Equal(t, usd.ID, usdItems[1].ID)
						assert.Equal(t, now, usdItems[1].EffectiveFrom)
						assert.Equal(t, futureEffectiveFrom, lo.FromPtr(usdItems[1].EffectiveTo))
					})

					t.Run("ListByFiatCode", func(t *testing.T) {
						// when:
						// - the cost-basis history is filtered to EUR
						result, err := env.Service.ListCostBases(t.Context(), currencies.ListCostBasesInput{
							Page:           pagination.NewPage(1, 10),
							Namespace:      namespace,
							CurrencyID:     createdCurrency.ID,
							FilterFiatCode: lo.ToPtr(currencyx.Code("EUR")),
						})

						// then:
						// - only the EUR cost basis is returned
						require.NoError(t, err)
						require.Equal(t, 1, result.TotalCount)
						require.Len(t, result.Items, 1)
						assert.Equal(t, eur.ID, result.Items[0].ID)
						assert.Equal(t, currencyx.Code("EUR"), result.Items[0].FiatCode)
					})
				})

				t.Run("InvalidEffectivePeriod", func(t *testing.T) {
					// given:
					// - an effective period whose end equals its start
					effectiveFrom := now.Add(48 * time.Hour)

					// when:
					// - a cost basis is created with that period
					_, err := env.Service.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
						Namespace:     namespace,
						CurrencyID:    createdCurrency.ID,
						FiatCode:      "GBP",
						Rate:          alpacadecimal.RequireFromString("0.008"),
						EffectiveFrom: &effectiveFrom,
						EffectiveTo:   &effectiveFrom,
					})

					// then:
					// - validation fails before persistence
					require.Error(t, err)
					assert.True(t, models.IsGenericValidationError(err))
					assert.Contains(t, err.Error(), "effective_to")
				})
			})
		})

		t.Run("List", func(t *testing.T) {
			// given:
			// - independently persisted custom currencies
			listNamespace := currenciestestutils.NewTestNamespace(t)
			points, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
				Namespace: listNamespace,
				CurrencyDetails: currencyx.CurrencyDetails{
					Code:               "POINTS",
					Name:               "Points",
					Symbol:             "P",
					Precision:          2,
					DecimalMark:        ".",
					ThousandsSeparator: ",",
				},
			})
			require.NoError(t, err)

			_, err = env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
				Namespace: listNamespace,
				CurrencyDetails: currencyx.CurrencyDetails{
					Code:               "TOKENS",
					Name:               "Tokens",
					Symbol:             "T",
					Precision:          4,
					DecimalMark:        ".",
					ThousandsSeparator: ",",
				},
			})
			require.NoError(t, err)

			// when:
			// - the service lists the custom currency together with selected fiat currencies
			result, err := env.Service.ListCurrencies(t.Context(), currencies.ListCurrenciesInput{
				Namespace: listNamespace,
				Page:      pagination.NewPage(1, 10),
				Code: &filter.FilterString{
					In: lo.ToPtr([]string{"POINTS", "USD", "EUR"}),
				},
			})

			// then:
			// - the custom currency and both fiat currencies are returned
			require.NoError(t, err)
			require.Equal(t, 3, result.TotalCount)

			codes := lo.Map(result.Items, func(item currencies.Currency, _ int) currencyx.Code {
				return item.Details().Code
			})
			assert.ElementsMatch(t, []currencyx.Code{"POINTS", "USD", "EUR"}, codes)

			t.Run("FilterCombination", func(t *testing.T) {
				// given:
				// - ID and code filters that identify different currencies
				testCases := []struct {
					name             string
					filteringOptions currencies.FilteringOptions
					currencyType     *currencies.CurrencyType
					id               *filter.FilterString
					code             *filter.FilterString
					expectedCodes    []currencyx.Code
				}{
					{
						name: "filter by id",
						id: &filter.FilterString{
							In: lo.ToPtr([]string{points.ID}),
						},
						expectedCodes: []currencyx.Code{"POINTS"},
					},
					{
						name:         "filter by custom currency type",
						currencyType: lo.ToPtr(currencies.CurrencyTypeCustom),
						code: &filter.FilterString{
							In: lo.ToPtr([]string{"POINTS", "USD"}),
						},
						expectedCodes: []currencyx.Code{"POINTS"},
					},
					{
						name:         "filter by fiat currency type",
						currencyType: lo.ToPtr(currencies.CurrencyTypeFiat),
						code: &filter.FilterString{
							In: lo.ToPtr([]string{"POINTS", "USD"}),
						},
						expectedCodes: []currencyx.Code{"USD"},
					},
					{
						name: "intersection",
						id: &filter.FilterString{
							In: lo.ToPtr([]string{points.ID}),
						},
						code: &filter.FilterString{
							In: lo.ToPtr([]string{"TOKENS"}),
						},
						expectedCodes: nil,
					},
					{
						name: "union custom currencies",
						filteringOptions: currencies.FilteringOptions{
							Union: true,
						},
						currencyType: lo.ToPtr(currencies.CurrencyTypeCustom),
						id: &filter.FilterString{
							In: lo.ToPtr([]string{points.ID}),
						},
						code: &filter.FilterString{
							In: lo.ToPtr([]string{"TOKENS"}),
						},
						expectedCodes: []currencyx.Code{"POINTS", "TOKENS"},
					},
					{
						name: "union custom and fiat currencies",
						filteringOptions: currencies.FilteringOptions{
							Union: true,
						},
						id: &filter.FilterString{
							In: lo.ToPtr([]string{points.ID}),
						},
						code: &filter.FilterString{
							In: lo.ToPtr([]string{"USD"}),
						},
						expectedCodes: []currencyx.Code{"POINTS", "USD"},
					},
				}

				for _, testCase := range testCases {
					t.Run(testCase.name, func(t *testing.T) {
						// when:
						// - currencies are listed with the selected filter combination mode
						result, err := env.Service.ListCurrencies(t.Context(), currencies.ListCurrenciesInput{
							Page:             pagination.NewPage(1, 10),
							FilteringOptions: testCase.filteringOptions,
							Namespace:        listNamespace,
							CurrencyType:     testCase.currencyType,
							ID:               testCase.id,
							Code:             testCase.code,
						})

						// then:
						// - sibling filters are intersected by default or unioned when requested
						require.NoError(t, err)
						actualCodes := lo.Map(result.Items, func(item currencies.Currency, _ int) currencyx.Code {
							return item.Details().Code
						})
						assert.ElementsMatch(t, testCase.expectedCodes, actualCodes)
					})
				}
			})
		})
	})
}
