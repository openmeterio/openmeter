package flatfee

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCalculateAmountAfterProration(t *testing.T) {
	// 2026-01-01 to 2026-02-01 (full month)
	fullMonthStart := datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime()
	fullMonthEnd := datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime()
	// 2026-01-01 to 2026-01-16 (half month, 15 out of 31 days)
	halfMonthEnd := datetime.MustParseTimeInLocation(t, "2026-01-16T00:00:00Z", time.UTC).AsTime()

	fullMonth := timeutil.ClosedPeriod{
		From: fullMonthStart,
		To:   fullMonthEnd,
	}

	halfMonth := timeutil.ClosedPeriod{
		From: fullMonthStart,
		To:   halfMonthEnd,
	}

	amount100 := alpacadecimal.NewFromInt(100)

	baseIntent := func() Intent {
		return Intent{
			Intent: meta.Intent{
				Name:              "test",
				CustomerID:        "cust-1",
				Currency:          currencyx.Code("USD"),
				ManagedBy:         "system",
				ServicePeriod:     halfMonth,
				FullServicePeriod: fullMonth,
				BillingPeriod:     fullMonth,
			},
			InvoiceAt:             fullMonthStart,
			SettlementMode:        productcatalog.InvoiceOnlySettlementMode,
			PaymentTerm:           productcatalog.InAdvancePaymentTerm,
			AmountBeforeProration: amount100,
			ProRating: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
		}
	}

	t.Run("proration disabled returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ProRating = productcatalog.ProRatingConfig{
			Enabled: false,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("equal periods returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ServicePeriod = fullMonth
		intent.FullServicePeriod = fullMonth

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("half period returns prorated amount", func(t *testing.T) {
		intent := baseIntent()

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		// 15 days out of 31 days = 100 * 15/31 = 48.387... rounded to 48.39 for USD
		expected := alpacadecimal.NewFromFloat(48.39)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("zero length service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   fullMonthStart,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("zero length full service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.FullServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   fullMonthStart,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("rounds to currency precision", func(t *testing.T) {
		intent := baseIntent()
		// 10 days out of 31 = 100 * 10/31 = 32.258... rounded to 32.26 for USD
		tenDaysEnd := datetime.MustParseTimeInLocation(t, "2026-01-11T00:00:00Z", time.UTC).AsTime()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   tenDaysEnd,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromFloat(32.26)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("JPY rounds to zero decimal places", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currencyx.Code("JPY")
		intent.AmountBeforeProration = alpacadecimal.NewFromInt(1000)
		// 10 days out of 31 = 1000 * 10/31 = 322.580... rounded to 323 for JPY
		tenDaysEnd := datetime.MustParseTimeInLocation(t, "2026-01-11T00:00:00Z", time.UTC).AsTime()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   tenDaysEnd,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromInt(323)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("invalid currency returns error", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currencyx.Code("INVALID")

		_, err := intent.CalculateAmountAfterProration()
		require.Error(t, err)
	})
}
