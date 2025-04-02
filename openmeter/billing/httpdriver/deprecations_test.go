package httpdriver

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
)

func TestFlatFeeLineParser(t *testing.T) {
	require := require.New(t)

	// Success case: All required deprecated fields are provided
	parsed, err := mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		PerUnitAmount: lo.ToPtr("100"),
		Quantity:      lo.ToPtr("1"),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(defaultFlatFeePaymentTerm)),
	},
	)
	require.NoError(err)
	require.Equal(parsed.PerUnitAmount.InexactFloat64(), float64(100))
	require.Equal(parsed.Quantity.InexactFloat64(), float64(1))
	require.Equal(parsed.PaymentTerm, defaultFlatFeePaymentTerm)

	// Success case: All required ratecartd fields are provided
	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Price: &api.FlatPriceWithPaymentTerm{
				Amount: "100",
				Type:   api.FlatPriceWithPaymentTermTypeFlat,
			},
			Quantity: lo.ToPtr("1"),
		},
	})
	require.NoError(err)
	require.Equal(parsed.PerUnitAmount.InexactFloat64(), float64(100))
	require.Equal(parsed.Quantity.InexactFloat64(), float64(1))
	require.Equal(parsed.PaymentTerm, defaultFlatFeePaymentTerm)

	// Success case: All deprecated and ratecard fields are provided with same values
	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Price: &api.FlatPriceWithPaymentTerm{
				Amount: "100",
				Type:   api.FlatPriceWithPaymentTermTypeFlat,
			},
			Quantity: lo.ToPtr("1"),
		},
		PerUnitAmount: lo.ToPtr("100"),
		Quantity:      lo.ToPtr("1"),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(defaultFlatFeePaymentTerm)),
	})
	require.NoError(err)
	require.Equal(parsed.PerUnitAmount.InexactFloat64(), float64(100))
	require.Equal(parsed.Quantity.InexactFloat64(), float64(1))
	require.Equal(parsed.PaymentTerm, defaultFlatFeePaymentTerm)

	// Failure case: Missing required fields
	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Price: &api.FlatPriceWithPaymentTerm{
				Amount: "100",
			},
			Quantity: lo.ToPtr("1"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Price: &api.FlatPriceWithPaymentTerm{
				Type: api.FlatPriceWithPaymentTermTypeFlat,
			},
			Quantity: lo.ToPtr("1"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Quantity: lo.ToPtr("1"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Quantity: lo.ToPtr("1"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	// Failure case #2: Value mismatch
	parsed, err = mapAndValidateFlatFeeRateCardDeprecatedFields(flatFeeRateCardItems{
		RateCard: &api.InvoiceFlatFeeRateCard{
			Price: &api.FlatPriceWithPaymentTerm{
				Type: api.FlatPriceWithPaymentTermTypeFlat,
			},
			Quantity: lo.ToPtr("1"),
		},
		PerUnitAmount: lo.ToPtr("100"),
		Quantity:      lo.ToPtr("1"),
		PaymentTerm:   lo.ToPtr(api.PricePaymentTerm(defaultFlatFeePaymentTerm)),
	})
	require.Error(err)
	require.Nil(parsed)
}

func TestUsageBasedLineParser(t *testing.T) {
	require := require.New(t)

	unitPriceAPI := api.RateCardUsageBasedPrice{}
	require.NoError(unitPriceAPI.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "100",
	}))

	unitPrice, err := planhttpdriver.AsPrice(unitPriceAPI)
	require.NoError(err)

	unitPrice2API := api.RateCardUsageBasedPrice{}
	require.NoError(unitPrice2API.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "200",
	}))

	// Success case: All required deprecated fields are provided
	parsed, err := mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		Price:      &unitPriceAPI,
		FeatureKey: lo.ToPtr("feature-key"),
	})
	require.NoError(err)
	require.True(unitPrice.Equal(parsed.Price))
	require.Equal(parsed.FeatureKey, "feature-key")

	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.NoError(err)
	require.True(unitPrice.Equal(parsed.Price))
	require.Equal(parsed.FeatureKey, "feature-key")

	// Failure case: Missing required fields
	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price: &unitPriceAPI,
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{})
	require.Error(err)
	require.Nil(parsed)

	// Failure case #2: Value mismatch
	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
		Price:      &unitPriceAPI,
		FeatureKey: lo.ToPtr("feature-key-2"),
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateUsageBasedRateCardDeprecatedFields(usageBasedRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
		Price:      &unitPrice2API,
		FeatureKey: lo.ToPtr("feature-key"),
	})
	require.Error(err)
	require.Nil(parsed)
}
