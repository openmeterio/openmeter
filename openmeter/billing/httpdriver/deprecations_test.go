package httpdriver

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestUsageBasedLineParser(t *testing.T) {
	require := require.New(t)

	unitPriceAPI := apiv1models.RateCardUsageBasedPrice{}
	require.NoError(unitPriceAPI.FromUnitPriceWithCommitments(apiv1models.UnitPriceWithCommitments{
		Amount: "100",
	}))

	unitPrice, err := productcataloghttp.AsPrice(unitPriceAPI)
	require.NoError(err)

	unitPrice2API := apiv1models.RateCardUsageBasedPrice{}
	require.NoError(unitPrice2API.FromUnitPriceWithCommitments(apiv1models.UnitPriceWithCommitments{
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
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.NoError(err)
	require.True(unitPrice.Equal(parsed.Price))
	require.Equal(parsed.FeatureKey, "feature-key")

	// Failure case: Missing required fields
	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			FeatureKey: lo.ToPtr("feature-key"),
		},
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
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
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
		Price:      &unitPriceAPI,
		FeatureKey: lo.ToPtr("feature-key-2"),
	})
	require.Error(err)
	require.Nil(parsed)

	parsed, err = mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			Price:      &unitPriceAPI,
			FeatureKey: lo.ToPtr("feature-key"),
		},
		Price:      &unitPrice2API,
		FeatureKey: lo.ToPtr("feature-key"),
	})
	require.Error(err)
	require.Nil(parsed)
}

func TestUsageBasedLineParserRejectsFlatPriceUsageDiscount(t *testing.T) {
	flatPriceAPI := apiv1models.RateCardUsageBasedPrice{}
	require.NoError(t, flatPriceAPI.FromFlatPriceWithPaymentTerm(apiv1models.FlatPriceWithPaymentTerm{
		Amount:      "100",
		PaymentTerm: lo.ToPtr(apiv1models.PricePaymentTermInAdvance),
		Type:        apiv1models.FlatPriceWithPaymentTermTypeFlat,
	}))

	parsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{
		RateCard: &apiv1models.InvoiceUsageBasedRateCard{
			Price: &flatPriceAPI,
			Discounts: &apiv1models.BillingDiscounts{
				Percentage: &apiv1models.BillingDiscountPercentage{
					Percentage: models.NewPercentage(50),
				},
				Usage: &apiv1models.BillingDiscountUsage{
					Quantity: "1",
				},
			},
		},
	})

	require.Nil(t, parsed)
	require.ErrorIs(t, err, productcatalog.ErrUsageDiscountWithFlatPrice)
}
