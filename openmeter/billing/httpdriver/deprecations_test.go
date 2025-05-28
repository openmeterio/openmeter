package httpdriver

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

func TestUsageBasedLineParser(t *testing.T) {
	require := require.New(t)

	unitPriceAPI := api.RateCardUsageBasedPrice{}
	require.NoError(unitPriceAPI.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "100",
	}))

	unitPrice, err := productcataloghttp.AsPrice(unitPriceAPI)
	require.NoError(err)

	unitPrice2API := api.RateCardUsageBasedPrice{}
	require.NoError(unitPrice2API.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "200",
	}))

	// Success case: All required deprecated fields are provided
	parsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		Price:      &unitPriceAPI,
		FeatureKey: lo.ToPtr("feature-key"),
	})
	require.NoError(err)
	require.True(unitPrice.Equal(parsed.Price))
	require.Equal(parsed.FeatureKey, "feature-key")

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.NoError(err)
	require.True(unitPrice.Equal(parsed.Price))
	require.Equal(parsed.FeatureKey, "feature-key")

	// Failure case: Missing required fields
	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price: &unitPriceAPI,
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{})
	require.Error(err)
	require.Nil(parsed)

	// Failure case #2: Value mismatch
	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
		Price:      &unitPriceAPI,
		FeatureKey: lo.ToPtr("feature-key-2"),
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
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
