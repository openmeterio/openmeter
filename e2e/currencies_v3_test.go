package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CurrenciesListExpandCostBasis(t *testing.T) {
	c := newV3Client(t)

	// Custom currency codes must be 4-24 characters; uniqueKey stays within that range.
	code := uniqueKey("cbc")

	custom, err := c.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Name:              "Cost Basis Currency " + code,
		Code:              code,
		Precision:         2,
		DecimalMark:       ".",
		ThousandSeparator: ",",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, custom)

	costBasis, err := c.Currencies.CreateCostBasis(t.Context(), custom.ID, v3sdk.CreateCostBasisRequest{
		FiatCode: "USD",
		Rate:     "1.5",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, costBasis)

	// Filtering by the unique code isolates this test from other currencies in the shared namespace.
	findByCode := func(t *testing.T, params v3sdk.CurrencyListParams) *v3sdk.CurrencyCustom {
		t.Helper()

		params.Filter = &v3sdk.CurrencyFilter{Code: &v3sdk.StringFilter{Eq: lo.ToPtr(code)}}

		page, err := c.Currencies.List(t.Context(), params)
		c.requireStatus(http.StatusOK, err)
		require.Len(t, page.Data, 1)

		found, err := page.Data[0].AsCurrencyCustom()
		require.NoError(t, err)
		require.Equal(t, code, found.Code)

		return found
	}

	t.Run("expand=cost_basis populates the active cost basis", func(t *testing.T) {
		// given:
		// - a custom currency with one cost basis
		// when:
		// - listing with expand=cost_basis
		found := findByCode(t, v3sdk.CurrencyListParams{
			Expand: []v3sdk.CurrencyExpand{v3sdk.CurrencyExpandCostBasis},
		})

		// then:
		// - the response carries the currency's active cost basis
		require.Len(t, found.CostBasis, 1)
		assert.Equal(t, costBasis.ID, found.CostBasis[0].ID)
		assert.Equal(t, costBasis.FiatCode, found.CostBasis[0].FiatCode)
		assert.Equal(t, costBasis.Rate, found.CostBasis[0].Rate)
	})

	t.Run("cost basis is omitted without expand", func(t *testing.T) {
		// given:
		// - the same custom currency with one cost basis
		// when:
		// - listing without the expand parameter
		found := findByCode(t, v3sdk.CurrencyListParams{})

		// then:
		// - the cost basis is not expanded, proving the parameter changes the response
		assert.Empty(t, found.CostBasis)
	})
}

func TestV3CustomCurrencyUpdate(t *testing.T) {
	c := newV3Client(t)

	// Custom currency codes must be 4-24 characters; uniqueKey stays within that range.
	code := uniqueKey("upd")

	custom, err := c.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Name:              "Update Currency " + code,
		Code:              code,
		Symbol:            lo.ToPtr("U"),
		Precision:         3,
		DecimalMark:       ".",
		ThousandSeparator: ",",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, custom)

	t.Run("presentational attributes are replaced", func(t *testing.T) {
		// given:
		// - a custom currency with the default formatting
		// when:
		// - name, symbol, decimal mark and thousand separator are all supplied
		updated, err := c.Currencies.UpdateCustomCurrency(t.Context(), custom.ID, v3sdk.CurrencyCustomUpdate{
			Name:              "Update Currency Renamed",
			Symbol:            lo.ToPtr("€"),
			DecimalMark:       ",",
			ThousandSeparator: ".",
		})

		// then:
		// - the new values are returned while the immutable code and precision stay put
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Update Currency Renamed", updated.Name)
		assert.Equal(t, lo.ToPtr("€"), updated.Symbol)
		assert.Equal(t, ",", updated.DecimalMark)
		assert.Equal(t, ".", updated.ThousandSeparator)
		assert.Equal(t, code, updated.Code)
		assert.EqualValues(t, 3, updated.Precision)
	})

	t.Run("an omitted symbol is cleared", func(t *testing.T) {
		// given:
		// - the currency currently carries a symbol
		// when:
		// - a replacement without a symbol is submitted
		updated, err := c.Currencies.UpdateCustomCurrency(t.Context(), custom.ID, v3sdk.CurrencyCustomUpdate{
			Name:              "Update Currency Again",
			DecimalMark:       ",",
			ThousandSeparator: ".",
		})

		// then:
		// - the symbol is removed, since the request replaces the whole representation
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Update Currency Again", updated.Name)
		assert.Nil(t, updated.Symbol)
	})

	t.Run("a decimal mark colliding with the thousand separator is rejected", func(t *testing.T) {
		// when:
		// - both separators are the same character
		_, err := c.Currencies.UpdateCustomCurrency(t.Context(), custom.ID, v3sdk.CurrencyCustomUpdate{
			Name:              "Update Currency",
			DecimalMark:       ".",
			ThousandSeparator: ".",
		})

		// then:
		// - the request fails because formatted amounts would be ambiguous
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertProblemDetail(t, problem, "decimal_mark and thousands_separator must differ")
	})

	t.Run("a missing name is rejected", func(t *testing.T) {
		// when:
		// - the required name is omitted
		_, err := c.Currencies.UpdateCustomCurrency(t.Context(), custom.ID, v3sdk.CurrencyCustomUpdate{
			DecimalMark:       ",",
			ThousandSeparator: ".",
		})

		// then:
		// - the request is rejected, since PUT replaces the whole representation
		// The TypeSpec minLength on name rejects this before the domain
		// validator's own "name is required" check is reached.
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertInvalidParameterRule(t, problem, "min_length")
		assertProblemDetail(t, problem, "name [min_length]")
	})
}

func TestV3CurrenciesListFilterByType(t *testing.T) {
	c := newV3Client(t)

	// Custom currency codes must be 4-24 characters; uniqueKey stays within that range.
	code := uniqueKey("ftc")

	custom, err := c.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Name:              "Filter Type Currency " + code,
		Code:              code,
		Precision:         2,
		DecimalMark:       ".",
		ThousandSeparator: ",",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, custom)

	// Pinning the list to the code created above leaves the type filter as the only
	// thing that can change whether the currency is returned.
	listByCodeAndType := func(t *testing.T, currencyType v3sdk.CurrencyType) []v3sdk.Currency {
		t.Helper()

		page, err := c.Currencies.List(t.Context(), v3sdk.CurrencyListParams{
			Filter: &v3sdk.CurrencyFilter{
				Type: lo.ToPtr(currencyType),
				Code: &v3sdk.StringFilter{Eq: lo.ToPtr(code)},
			},
		})
		c.requireStatus(http.StatusOK, err)

		return page.Data
	}

	t.Run("filter[type]=custom returns the custom currency", func(t *testing.T) {
		// given:
		// - a custom currency
		// when:
		// - listing with filter[type]=custom
		data := listByCodeAndType(t, v3sdk.CurrencyTypeCustom)

		// then:
		// - the custom currency is returned
		require.Len(t, data, 1)
		found, err := data[0].AsCurrencyCustom()
		require.NoError(t, err)
		assert.Equal(t, code, found.Code)
	})

	t.Run("filter[type]=fiat excludes the custom currency", func(t *testing.T) {
		// given:
		// - the same custom currency
		// when:
		// - listing with filter[type]=fiat
		// then:
		// - the custom currency is excluded, proving the type filter is applied
		assert.Empty(t, listByCodeAndType(t, v3sdk.CurrencyTypeFiat))
	})
}
