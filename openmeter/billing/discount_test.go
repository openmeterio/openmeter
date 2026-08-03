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
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(-1)}},
			},
			price:    unitPrice,
			wantErrs: []error{productcatalog.ErrUsageDiscountNegativeQuantity},
		},
		{
			name: "valid percentage and positive usage rejects flat price",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}},
			},
			price:    flatPrice,
			wantErrs: []error{productcatalog.ErrUsageDiscountWithFlatPrice},
		},
		{
			name: "negative usage on flat price returns both usage errors",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(-1)}},
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
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}},
				Usage:      &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}},
			},
			price: unitPrice,
		},
		{
			name: "percentage only",
			discounts: Discounts{
				Percentage: &PercentageDiscount{PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)}},
			},
			price: flatPrice,
		},
		{
			name: "usage only for unit price",
			discounts: Discounts{
				Usage: &UsageDiscount{UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(1)}},
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
