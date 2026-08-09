package billing

import (
	"errors"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDiscountsValidateForPrice(t *testing.T) {
	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(100),
	})
	flatPrice := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(100),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})

	tests := []struct {
		name      string
		discounts Discounts
		price     *productcatalog.Price
		wantErrs  []error
	}{
		{
			name: "valid percentage and negative usage validates usage",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}, CorrelationID: "percentage"},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(-1)}, CorrelationID: "usage"},
			},
			price:    unitPrice,
			wantErrs: []error{productcatalog.ErrUsageDiscountNegativeQuantity},
		},
		{
			name: "valid percentage and positive usage rejects flat price",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}, CorrelationID: "percentage"},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}, CorrelationID: "usage"},
			},
			price:    flatPrice,
			wantErrs: []error{productcatalog.ErrUsageDiscountWithFlatPrice},
		},
		{
			name: "negative usage on flat price returns both usage errors",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}, CorrelationID: "percentage"},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(-1)}, CorrelationID: "usage"},
			},
			price: flatPrice,
			wantErrs: []error{
				productcatalog.ErrUsageDiscountNegativeQuantity,
				productcatalog.ErrUsageDiscountWithFlatPrice,
			},
		},
		{
			name: "valid percentage and usage for unit price",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}, CorrelationID: "percentage"},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}, CorrelationID: "usage"},
			},
			price: unitPrice,
		},
		{
			name: "percentage only",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}, CorrelationID: "percentage"},
			},
			price: flatPrice,
		},
		{
			name: "usage only for unit price",
			discounts: Discounts{
				Usage: &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}, CorrelationID: "usage"},
			},
			price: unitPrice,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.discounts.ValidateForPrice(test.price)
			if len(test.wantErrs) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, wantErr := range test.wantErrs {
				require.Truef(t, errors.Is(err, wantErr), "expected error %q in %v", wantErr, err)
			}
		})
	}
}

func TestDiscountsRequireCorrelationIDs(t *testing.T) {
	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(100),
	})

	tests := []struct {
		name      string
		discounts Discounts
	}{
		{
			name: "percentage",
			discounts: Discounts{
				Percentage: &PercentageDiscount{
					PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
				},
			},
		},
		{
			name: "usage",
			discounts: Discounts{
				Usage: &UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorContains(t, test.discounts.Validate(), "correlation ID is required")
			require.ErrorContains(t, test.discounts.ValidateForPrice(unitPrice), "correlation ID is required")
		})
	}

	require.NoError(t, (Discounts{}).Validate())
	require.NoError(t, (Discounts{}).ValidateForPrice(nil))
	require.NoError(t, (Discounts{
		Percentage: &PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
			CorrelationID:      "any non-empty string is valid",
		},
	}).Validate())
}

func TestDiscountsFromProductCatalog(t *testing.T) {
	discounts := productcatalog.Discounts{
		Percentage: &productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
		Usage:      &productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
	}

	billingDiscounts := DiscountsFromProductCatalog(discounts)

	require.NoError(t, billingDiscounts.Validate())
	require.NotEmpty(t, billingDiscounts.Percentage.CorrelationID)
	require.NotEmpty(t, billingDiscounts.Usage.CorrelationID)
	require.Equal(t, discounts, billingDiscounts.ToProductCatalog())
}

func TestDiscountFromProductCatalog(t *testing.T) {
	percentage := &productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}
	usage := &productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}

	billingPercentage := PercentageDiscountFromProductCatalog(percentage)
	billingUsage := UsageDiscountFromProductCatalog(usage)

	require.NoError(t, billingPercentage.Validate())
	require.NoError(t, billingUsage.Validate())
	require.NotEmpty(t, billingPercentage.CorrelationID)
	require.NotEmpty(t, billingUsage.CorrelationID)
	require.Equal(t, *percentage, billingPercentage.PercentageDiscount)
	require.Equal(t, *usage, billingUsage.UsageDiscount)
	require.Nil(t, PercentageDiscountFromProductCatalog(nil))
	require.Nil(t, UsageDiscountFromProductCatalog(nil))
}

func TestDiscountsReplaceFromProductCatalog(t *testing.T) {
	existing := Discounts{
		Percentage: &PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
			CorrelationID:      "existing-percentage",
		},
	}

	replaced := existing.ReplaceFromProductCatalog(productcatalog.Discounts{
		Percentage: &productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
		Usage:      &productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
	})

	require.NoError(t, replaced.Validate())
	require.Equal(t, "existing-percentage", replaced.Percentage.CorrelationID)
	require.Equal(t, models.NewPercentage(50), replaced.Percentage.Percentage)
	require.NotEmpty(t, replaced.Usage.CorrelationID)

	retried := replaced.ReplaceFromProductCatalog(replaced.ToProductCatalog())
	require.Equal(t, replaced, retried)

	removed := replaced.ReplaceFromProductCatalog(productcatalog.Discounts{})
	require.True(t, removed.IsEmpty())

	readded := removed.ReplaceFromProductCatalog(productcatalog.Discounts{
		Percentage: &productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
	})
	require.NotEqual(t, "existing-percentage", readded.Percentage.CorrelationID)
}

func TestDiscountsUpsertCorrelationIDs(t *testing.T) {
	discounts := Discounts{
		Percentage: &PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
			CorrelationID:      "existing-percentage-discount",
		},
		Usage: &UsageDiscount{
			UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)},
		},
	}

	first := discounts.UpsertCorrelationIDs()
	retried := first.UpsertCorrelationIDs()

	require.Equal(t, "existing-percentage-discount", first.Percentage.CorrelationID)
	require.NotEmpty(t, first.Usage.CorrelationID)
	require.Equal(t, first, retried)
	require.Empty(t, discounts.Usage.CorrelationID)
}
