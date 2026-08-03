package adapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	currenciestestenvutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/env"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customcurrencydb "github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListCustomCurrenciesFiltersCurrencyType(t *testing.T) {
	env := currenciestestenvutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)
	created, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		name          string
		currencyType  currencyx.CurrencyType
		expectedCodes []currencyx.Code
	}{
		{
			name:          "custom",
			currencyType:  currencyx.CurrencyTypeCustom,
			expectedCodes: []currencyx.Code{created.Details().Code},
		},
		{
			name:          "fiat",
			currencyType:  currencyx.CurrencyTypeFiat,
			expectedCodes: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// given:
			// - a persisted custom currency and a requested currency type
			// when:
			// - custom currencies are listed directly through the repository
			result, err := env.Repository.ListCustomCurrencies(t.Context(), currencies.ListCurrenciesInput{
				Page:         pagination.NewPage(1, 10),
				Namespace:    namespace,
				CurrencyType: &testCase.currencyType,
			})

			// then:
			// - the adapter returns custom records only for the custom type
			require.NoError(t, err)
			actualCodes := lo.Map(result.Items, func(item currencies.Currency, _ int) currencyx.Code {
				return item.Details().Code
			})
			assert.ElementsMatch(t, testCase.expectedCodes, actualCodes)
			assert.Equal(t, len(testCase.expectedCodes), result.TotalCount)
		})
	}
}

func TestCostBasisEagerLoaders(t *testing.T) {
	env := currenciestestenvutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	at := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)

	t.Run("active currency", func(t *testing.T) {
		// given:
		// - a live currency with cost bases spanning every effective-period and deletion boundary
		namespace := currenciestestutils.NewTestNamespace(t)
		currency, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
			Namespace: namespace,
			CurrencyDetails: currencyx.CurrencyDetails{
				Code:               "TOKENS",
				Name:               "Tokens",
				Symbol:             "T",
				Precision:          2,
				DecimalMark:        ".",
				ThousandsSeparator: ",",
			},
		})
		require.NoError(t, err)

		fixtures := []struct {
			name          string
			fiatCode      currencyx.Code
			effectiveFrom time.Time
			effectiveTo   *time.Time
			deletedAt     *time.Time
		}{
			{
				name:          "expired",
				fiatCode:      "USD",
				effectiveFrom: at.Add(-3 * time.Hour),
				effectiveTo:   lo.ToPtr(at.Add(-2 * time.Hour)),
			},
			{
				name:          "active",
				fiatCode:      "EUR",
				effectiveFrom: at.Add(-time.Hour),
				effectiveTo:   lo.ToPtr(at.Add(time.Hour)),
			},
			{
				name:          "effective from boundary",
				fiatCode:      "GBP",
				effectiveFrom: at,
			},
			{
				name:          "effective to boundary",
				fiatCode:      "CAD",
				effectiveFrom: at.Add(-time.Hour),
				effectiveTo:   lo.ToPtr(at),
			},
			{
				name:          "scheduled",
				fiatCode:      "JPY",
				effectiveFrom: at.Add(time.Hour),
			},
			{
				name:          "deleted before",
				fiatCode:      "CHF",
				effectiveFrom: at.Add(-time.Hour),
				deletedAt:     lo.ToPtr(at.Add(-time.Second)),
			},
			{
				name:          "deleted at boundary",
				fiatCode:      "NZD",
				effectiveFrom: at.Add(-time.Hour),
				deletedAt:     lo.ToPtr(at),
			},
			{
				name:          "deleted after",
				fiatCode:      "AUD",
				effectiveFrom: at.Add(-time.Hour),
				deletedAt:     lo.ToPtr(at.Add(time.Second)),
			},
		}

		ids := make(map[string]string, len(fixtures))
		for _, fixture := range fixtures {
			row, err := env.Client.CurrencyCostBasis.Create().
				SetNamespace(namespace).
				SetCurrencyID(currency.ID).
				SetFiatCode(fixture.fiatCode).
				SetRate(alpacadecimal.RequireFromString("1")).
				SetEffectiveFrom(fixture.effectiveFrom).
				SetNillableEffectiveTo(fixture.effectiveTo).
				SetNillableDeletedAt(fixture.deletedAt).
				Save(t.Context())
			require.NoError(t, err)
			ids[fixture.name] = row.ID
		}

		testCases := []struct {
			name        string
			eagerLoad   func(*entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery
			expectedIDs []string
		}{
			{
				name: "all non-deleted history",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithCostBasis(query)
				},
				expectedIDs: []string{
					ids["expired"],
					ids["active"],
					ids["effective from boundary"],
					ids["effective to boundary"],
					ids["scheduled"],
				},
			},
			{
				name: "active",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithActiveCostBasis(query, at)
				},
				expectedIDs: []string{
					ids["active"],
					ids["effective from boundary"],
					ids["deleted after"],
				},
			},
			{
				name: "active and scheduled",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithActiveAndScheduledCostBasis(query, at)
				},
				expectedIDs: []string{
					ids["active"],
					ids["effective from boundary"],
					ids["scheduled"],
					ids["deleted after"],
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				// given:
				// - the live currency and its complete cost-basis fixture set
				// when:
				// - the currency is queried through the selected cost-basis loader
				actualIDs := loadCostBasisIDs(t, env, currency.ID, testCase.eagerLoad)

				// then:
				// - lifecycle and effective-period boundaries match that loader's contract
				assert.ElementsMatch(t, testCase.expectedIDs, actualIDs)
			})
		}
	})

	t.Run("deleted currency", func(t *testing.T) {
		// given:
		// - a deleted currency with cost bases spanning its deletion-time snapshot boundaries
		namespace := currenciestestutils.NewTestNamespace(t)
		currency, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
			Namespace: namespace,
			CurrencyDetails: currencyx.CurrencyDetails{
				Code:               "CREDITS",
				Name:               "Credits",
				Symbol:             "C",
				Precision:          2,
				DecimalMark:        ".",
				ThousandsSeparator: ",",
			},
		})
		require.NoError(t, err)

		deletedAt := at.Add(-24 * time.Hour)
		fixtures := []struct {
			name          string
			fiatCode      currencyx.Code
			effectiveFrom time.Time
			effectiveTo   *time.Time
			deletedAt     *time.Time
		}{
			{
				name:          "expired at deletion",
				fiatCode:      "USD",
				effectiveFrom: deletedAt.Add(-3 * time.Hour),
				effectiveTo:   lo.ToPtr(deletedAt.Add(-time.Hour)),
				deletedAt:     &deletedAt,
			},
			{
				name:          "active at deletion",
				fiatCode:      "EUR",
				effectiveFrom: deletedAt.Add(-time.Hour),
				effectiveTo:   lo.ToPtr(deletedAt.Add(time.Hour)),
				deletedAt:     &deletedAt,
			},
			{
				name:          "effective from at deletion",
				fiatCode:      "GBP",
				effectiveFrom: deletedAt,
				deletedAt:     &deletedAt,
			},
			{
				name:          "effective to at deletion",
				fiatCode:      "CAD",
				effectiveFrom: deletedAt.Add(-time.Hour),
				effectiveTo:   &deletedAt,
				deletedAt:     &deletedAt,
			},
			{
				name:          "scheduled after deletion",
				fiatCode:      "JPY",
				effectiveFrom: deletedAt.Add(time.Hour),
				deletedAt:     &deletedAt,
			},
			{
				name:          "different deletion timestamp",
				fiatCode:      "CHF",
				effectiveFrom: deletedAt.Add(-time.Hour),
				deletedAt:     lo.ToPtr(deletedAt.Add(time.Second)),
			},
			{
				name:          "not deleted",
				fiatCode:      "AUD",
				effectiveFrom: deletedAt.Add(-time.Hour),
			},
		}

		ids := make(map[string]string, len(fixtures))
		for _, fixture := range fixtures {
			row, err := env.Client.CurrencyCostBasis.Create().
				SetNamespace(namespace).
				SetCurrencyID(currency.ID).
				SetFiatCode(fixture.fiatCode).
				SetRate(alpacadecimal.RequireFromString("1")).
				SetEffectiveFrom(fixture.effectiveFrom).
				SetNillableEffectiveTo(fixture.effectiveTo).
				SetNillableDeletedAt(fixture.deletedAt).
				Save(t.Context())
			require.NoError(t, err)
			ids[fixture.name] = row.ID
		}

		_, err = env.Client.CustomCurrency.UpdateOneID(currency.ID).
			SetDeletedAt(deletedAt).
			Save(t.Context())
		require.NoError(t, err)

		testCases := []struct {
			name        string
			eagerLoad   func(*entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery
			expectedIDs []string
		}{
			{
				name: "all non-deleted history",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithCostBasis(query)
				},
				expectedIDs: []string{ids["not deleted"]},
			},
			{
				name: "active",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithActiveCostBasis(query, at)
				},
				expectedIDs: []string{
					ids["active at deletion"],
					ids["effective from at deletion"],
				},
			},
			{
				name: "active and scheduled",
				eagerLoad: func(query *entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery {
					return currencyadapter.WithActiveAndScheduledCostBasis(query, at)
				},
				expectedIDs: []string{
					ids["active at deletion"],
					ids["effective from at deletion"],
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				// given:
				// - the deleted currency and its complete cost-basis fixture set
				// when:
				// - the deleted currency is queried through the selected cost-basis loader
				actualIDs := loadCostBasisIDs(t, env, currency.ID, testCase.eagerLoad)

				// then:
				// - snapshot loaders use the currency deletion instant while history ignores deleted rows
				assert.ElementsMatch(t, testCase.expectedIDs, actualIDs)
			})
		}
	})
}

func loadCostBasisIDs(
	t *testing.T,
	env *currenciestestenvutils.TestEnv,
	currencyID string,
	eagerLoad func(*entdb.CustomCurrencyQuery) *entdb.CustomCurrencyQuery,
) []string {
	t.Helper()

	row, err := eagerLoad(
		env.Client.CustomCurrency.Query(),
	).Where(customcurrencydb.ID(currencyID)).Only(t.Context())
	require.NoError(t, err)

	result, err := currencyadapter.FromDBCustomCurrency(row)
	require.NoError(t, err)
	require.NotNil(t, result.CostBasis)

	return lo.Map(*result.CostBasis, func(item currencies.CostBasis, _ int) string {
		return item.ID
	})
}

func TestGetCostBasisAt(t *testing.T) {
	env := currenciestestenvutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)
	currency, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	firstEffectiveFrom := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	secondEffectiveFrom := firstEffectiveFrom.Add(24 * time.Hour)

	// given:
	// - consecutive USD cost bases and a EUR cost basis ending at the same boundary
	firstUSD, err := env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("USD").
		SetRate(alpacadecimal.RequireFromString("0.01")).
		SetEffectiveFrom(firstEffectiveFrom).
		SetEffectiveTo(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	secondUSD, err := env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("USD").
		SetRate(alpacadecimal.RequireFromString("0.02")).
		SetEffectiveFrom(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	_, err = env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("EUR").
		SetRate(alpacadecimal.RequireFromString("0.009")).
		SetEffectiveFrom(firstEffectiveFrom).
		SetEffectiveTo(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	testCases := []struct {
		name       string
		fiatCode   currencyx.Code
		at         time.Time
		expectedID string
		notFound   bool
	}{
		{
			name:       "effective from is inclusive",
			fiatCode:   "USD",
			at:         firstEffectiveFrom,
			expectedID: firstUSD.ID,
		},
		{
			name:       "newer cost basis wins at its effective start",
			fiatCode:   "USD",
			at:         secondEffectiveFrom,
			expectedID: secondUSD.ID,
		},
		{
			name:       "open interval remains effective",
			fiatCode:   "USD",
			at:         secondEffectiveFrom.Add(30 * 24 * time.Hour),
			expectedID: secondUSD.ID,
		},
		{
			name:     "before first interval",
			fiatCode: "USD",
			at:       firstEffectiveFrom.Add(-time.Nanosecond),
			notFound: true,
		},
		{
			name:     "effective to is exclusive",
			fiatCode: "EUR",
			at:       secondEffectiveFrom,
			notFound: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// when:
			// - the cost basis effective at the requested instant is queried
			result, err := env.Repository.GetCostBasisAt(t.Context(), currencies.GetCostBasisAtInput{
				Namespace:  namespace,
				CurrencyID: currency.ID,
				FiatCode:   testCase.fiatCode,
				At:         testCase.at,
			})

			// then:
			// - interval boundaries select the matching row or return a typed not-found error
			if testCase.notFound {
				require.Error(t, err)
				assert.True(t, models.IsGenericNotFoundError(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedID, result.ID)
		})
	}
}
