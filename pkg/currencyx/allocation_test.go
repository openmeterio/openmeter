package currencyx_test

import (
	"cmp"
	"strings"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestAllocateByWeight(t *testing.T) {
	usd := testCurrencyFiat(t, "USD")

	t.Run("allocates exact proportional shares", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("6.00"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("10")},
				{Key: "B", Weight: dec("5")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("4.00")},
			{Key: "B", Amount: dec("2.00")},
		}, allocations)
	})

	t.Run("distributes rounded remainder by largest remainder", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("0.05"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("1")},
				{Key: "B", Weight: dec("1")},
				{Key: "C", Weight: dec("1")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("0.02")},
			{Key: "B", Amount: dec("0.02")},
			{Key: "C", Amount: dec("0.01")},
		}, allocations)
	})

	t.Run("uses key comparator as deterministic remainder tie-breaker", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("0.05"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "C", Weight: dec("1")},
				{Key: "A", Weight: dec("1")},
				{Key: "B", Weight: dec("1")},
			},
			CompareKey: cmp.Compare[string],
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "C", Amount: dec("0.01")},
			{Key: "A", Amount: dec("0.02")},
			{Key: "B", Amount: dec("0.02")},
		}, allocations)
	})

	t.Run("uses currency precision", func(t *testing.T) {
		jpy := testCurrencyFiat(t, "JPY")

		allocations, err := currencyx.AllocateByWeight(jpy, currencyx.WeightedAllocationInput[string]{
			Amount: dec("5"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("1")},
				{Key: "B", Weight: dec("1")},
				{Key: "C", Weight: dec("1")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("2")},
			{Key: "B", Amount: dec("2")},
			{Key: "C", Amount: dec("1")},
		}, allocations)
	})

	t.Run("uses custom currency precision", func(t *testing.T) {
		custom := testCurrencyCustom(t, "CREDITS", 4)

		allocations, err := currencyx.AllocateByWeight(custom, currencyx.WeightedAllocationInput[string]{
			Amount: dec("0.0005"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("1")},
				{Key: "B", Weight: dec("1")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("0.0003")},
			{Key: "B", Amount: dec("0.0002")},
		}, allocations)
	})

	t.Run("omits zero allocations", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("0.01"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("100")},
				{Key: "B", Weight: dec("1")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("0.01")},
		}, allocations)
	})

	t.Run("supports dimensionless weights", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("100.00"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("1")},
				{Key: "B", Weight: dec("2")},
				{Key: "C", Weight: dec("3")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("16.67")},
			{Key: "B", Amount: dec("33.33")},
			{Key: "C", Amount: dec("50.00")},
		}, allocations)
	})
}

func TestAllocateByWeightValidation(t *testing.T) {
	usd := testCurrencyFiat(t, "USD")

	cases := []struct {
		name     string
		input    currencyx.WeightedAllocationInput[string]
		contains string
	}{
		{
			name: "amount must be rounded",
			input: currencyx.WeightedAllocationInput[string]{
				Amount: dec("1.001"),
				Items: []currencyx.WeightedAllocationItem[string]{
					{Key: "A", Weight: dec("2.00")},
				},
			},
			contains: "amount must be rounded to currency precision",
		},
		{
			name: "weight must be positive",
			input: currencyx.WeightedAllocationInput[string]{
				Amount: dec("1.00"),
				Items: []currencyx.WeightedAllocationItem[string]{
					{Key: "A", Weight: dec("0.00")},
				},
			},
			contains: "items[0].weight must be positive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := currencyx.AllocateByWeight(usd, tc.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.contains)
		})
	}

	t.Run("zero amount does not require items", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("0.00"),
		})
		require.NoError(t, err)
		require.Empty(t, allocations)
	})

	t.Run("weight does not need currency precision", func(t *testing.T) {
		allocations, err := currencyx.AllocateByWeight(usd, currencyx.WeightedAllocationInput[string]{
			Amount: dec("1.00"),
			Items: []currencyx.WeightedAllocationItem[string]{
				{Key: "A", Weight: dec("0.001")},
				{Key: "B", Weight: dec("0.002")},
			},
		})
		require.NoError(t, err)

		requireAllocationsEqual(t, []currencyx.WeightedAllocation[string]{
			{Key: "A", Amount: dec("0.33")},
			{Key: "B", Amount: dec("0.67")},
		}, allocations)
	})
}

func TestAllocateByAmount(t *testing.T) {
	usd := testCurrencyFiat(t, "USD")

	t.Run("allocates exact proportional shares", func(t *testing.T) {
		allocations, err := currencyx.AllocateByAmount(usd, currencyx.AmountAllocationInput[string]{
			Amount: dec("6.00"),
			Items: []currencyx.AmountAllocationItem[string]{
				{Key: "A", Amount: dec("10.00")},
				{Key: "B", Amount: dec("5.00")},
			},
		})
		require.NoError(t, err)

		requireAmountAllocationsEqual(t, []currencyx.AmountAllocation[string]{
			{Key: "A", Amount: dec("4.00")},
			{Key: "B", Amount: dec("2.00")},
		}, allocations)
	})

	t.Run("allocates rounded shares by largest remainder", func(t *testing.T) {
		allocations, err := currencyx.AllocateByAmount(usd, currencyx.AmountAllocationInput[string]{
			Amount: dec("5.00"),
			Items: []currencyx.AmountAllocationItem[string]{
				{Key: "A", Amount: dec("10.00")},
				{Key: "B", Amount: dec("5.00")},
			},
		})
		require.NoError(t, err)

		requireAmountAllocationsEqual(t, []currencyx.AmountAllocation[string]{
			{Key: "A", Amount: dec("3.33")},
			{Key: "B", Amount: dec("1.67")},
		}, allocations)
	})

	t.Run("uses key comparator as deterministic remainder tie-breaker", func(t *testing.T) {
		allocations, err := currencyx.AllocateByAmount(usd, currencyx.AmountAllocationInput[string]{
			Amount: dec("0.05"),
			Items: []currencyx.AmountAllocationItem[string]{
				{Key: "C", Amount: dec("1.00")},
				{Key: "A", Amount: dec("1.00")},
				{Key: "B", Amount: dec("1.00")},
			},
			CompareKey: cmp.Compare[string],
		})
		require.NoError(t, err)

		requireAmountAllocationsEqual(t, []currencyx.AmountAllocation[string]{
			{Key: "C", Amount: dec("0.01")},
			{Key: "A", Amount: dec("0.02")},
			{Key: "B", Amount: dec("0.02")},
		}, allocations)
	})

	t.Run("uses currency precision", func(t *testing.T) {
		jpy := testCurrencyFiat(t, "JPY")

		allocations, err := currencyx.AllocateByAmount(jpy, currencyx.AmountAllocationInput[string]{
			Amount: dec("5"),
			Items: []currencyx.AmountAllocationItem[string]{
				{Key: "A", Amount: dec("10")},
				{Key: "B", Amount: dec("10")},
				{Key: "C", Amount: dec("10")},
			},
		})
		require.NoError(t, err)

		requireAmountAllocationsEqual(t, []currencyx.AmountAllocation[string]{
			{Key: "A", Amount: dec("2")},
			{Key: "B", Amount: dec("2")},
			{Key: "C", Amount: dec("1")},
		}, allocations)
	})

	t.Run("omits zero allocations", func(t *testing.T) {
		allocations, err := currencyx.AllocateByAmount(usd, currencyx.AmountAllocationInput[string]{
			Amount: dec("0.01"),
			Items: []currencyx.AmountAllocationItem[string]{
				{Key: "A", Amount: dec("100.00")},
				{Key: "B", Amount: dec("1.00")},
			},
		})
		require.NoError(t, err)

		requireAmountAllocationsEqual(t, []currencyx.AmountAllocation[string]{
			{Key: "A", Amount: dec("0.01")},
		}, allocations)
	})
}

func TestAllocateByAmountValidation(t *testing.T) {
	usd := testCurrencyFiat(t, "USD")

	cases := []struct {
		name     string
		input    currencyx.AmountAllocationInput[string]
		contains string
	}{
		{
			name: "amount must be rounded",
			input: currencyx.AmountAllocationInput[string]{
				Amount: dec("1.001"),
				Items: []currencyx.AmountAllocationItem[string]{
					{Key: "A", Amount: dec("2.00")},
				},
			},
			contains: "amount must be rounded to currency precision",
		},
		{
			name: "item amount must be positive",
			input: currencyx.AmountAllocationInput[string]{
				Amount: dec("1.00"),
				Items: []currencyx.AmountAllocationItem[string]{
					{Key: "A", Amount: dec("0.00")},
				},
			},
			contains: "items[0].amount must be positive",
		},
		{
			name: "item amount must be rounded",
			input: currencyx.AmountAllocationInput[string]{
				Amount: dec("1.00"),
				Items: []currencyx.AmountAllocationItem[string]{
					{Key: "A", Amount: dec("2.001")},
				},
			},
			contains: "items[0].amount must be rounded to currency precision",
		},
		{
			name: "amount must not exceed total item amount",
			input: currencyx.AmountAllocationInput[string]{
				Amount: dec("3.00"),
				Items: []currencyx.AmountAllocationItem[string]{
					{Key: "A", Amount: dec("2.00")},
				},
			},
			contains: "amount must not exceed total item amount",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := currencyx.AllocateByAmount(usd, tc.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.contains)
		})
	}

	t.Run("zero amount does not require items", func(t *testing.T) {
		allocations, err := currencyx.AllocateByAmount(usd, currencyx.AmountAllocationInput[string]{
			Amount: dec("0.00"),
		})
		require.NoError(t, err)
		require.Empty(t, allocations)
	})
}

func testCurrencyFiat(t *testing.T, code string) currencyx.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code(code)).
		Build()
	require.NoError(t, err)

	return currency
}

func testCurrencyCustom(t *testing.T, code string, precision uint32) currencyx.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code(code)).
		WithName(code).
		WithPrecision(precision).
		Build()
	require.NoError(t, err)

	return currency
}

func requireAllocationsEqual[T comparable](t *testing.T, expected, actual []currencyx.WeightedAllocation[T]) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for i, expectedAllocation := range expected {
		require.Equal(t, expectedAllocation.Key, actual[i].Key)
		require.Truef(
			t,
			expectedAllocation.Amount.Equal(actual[i].Amount),
			"expected allocation %d amount %s, got %s",
			i,
			expectedAllocation.Amount.String(),
			actual[i].Amount.String(),
		)
	}
}

func requireAmountAllocationsEqual[T comparable](t *testing.T, expected, actual []currencyx.AmountAllocation[T]) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for i, expectedAllocation := range expected {
		require.Equal(t, expectedAllocation.Key, actual[i].Key)
		require.Truef(
			t,
			expectedAllocation.Amount.Equal(actual[i].Amount),
			"expected allocation %d amount %s, got %s",
			i,
			expectedAllocation.Amount.String(),
			actual[i].Amount.String(),
		)
	}
}

func dec(value string) alpacadecimal.Decimal {
	return alpacadecimal.RequireFromString(strings.TrimSpace(value))
}
