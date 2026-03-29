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
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCalculateAmountAfterProration(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	fullMonth := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(30 * 24 * time.Hour),
	}

	halfMonth := timeutil.ClosedPeriod{
		From: now,
		To:   now.Add(15 * 24 * time.Hour),
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
			InvoiceAt:             now,
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

	t.Run("half period returns half amount", func(t *testing.T) {
		intent := baseIntent()

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromInt(50)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("zero length service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("zero length full service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.FullServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("rounds to currency precision", func(t *testing.T) {
		intent := baseIntent()
		// Use a period ratio that produces a non-terminating decimal
		// 10 days out of 30 = 1/3, so 100 * 1/3 = 33.333... rounded to 33.33 for USD
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now.Add(10 * 24 * time.Hour),
		}
		intent.FullServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now.Add(30 * 24 * time.Hour),
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromFloat(33.33)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("JPY rounds to zero decimal places", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currencyx.Code("JPY")
		intent.AmountBeforeProration = alpacadecimal.NewFromInt(1000)
		// 10 days out of 30 = 1/3, so 1000 * 1/3 = 333.333... rounded to 333 for JPY
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now.Add(10 * 24 * time.Hour),
		}
		intent.FullServicePeriod = timeutil.ClosedPeriod{
			From: now,
			To:   now.Add(30 * 24 * time.Hour),
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromInt(333)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("invalid currency returns error", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currencyx.Code("INVALID")

		_, err := intent.CalculateAmountAfterProration()
		require.Error(t, err)
	})
}
