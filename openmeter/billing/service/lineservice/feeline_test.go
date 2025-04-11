package lineservice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFeeLine(t *testing.T) {
	t.Run("should apply discount percentage", func(t *testing.T) {
		line := generateFeeLine(t, generateFeeLineInput{
			Quantity: 1,
			Amount:   100,
			RateCardDiscount: billing.PercentageDiscount{
				PercentageDiscount: productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(50),
				},
				CorrelationID: "example-correlation-id",
			},
		})

		require.NoError(t, line.CalculateDetailedLines())

		ExpectJSONEqual(t, billing.LineDiscounts{
			Amount: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(50),
						LineDiscountBase: billing.LineDiscountBase{
							Reason: billing.NewDiscountReasonFrom(billing.PercentageDiscount{
								PercentageDiscount: productcatalog.PercentageDiscount{
									Percentage: models.NewPercentage(50),
								},
								CorrelationID: "example-correlation-id",
							}),
							ChildUniqueReferenceID: lo.ToPtr("rateCardDiscount/correlationID=example-correlation-id"),
						},
					},
				},
			},
		}, line.line.Discounts)
	})
}

type generateFeeLineInput struct {
	Quantity         float64
	Amount           float64
	RateCardDiscount billing.PercentageDiscount
}

func generateFeeLine(t *testing.T, in generateFeeLineInput) *feeLine {
	return &feeLine{
		lineBase: lineBase{
			line: &billing.Line{
				LineBase: billing.LineBase{
					Currency: "USD",
					Period: billing.Period{
						Start: time.Now(),
						End:   time.Now().Add(time.Hour * 24),
					},
					RateCardDiscounts: billing.Discounts{
						Percentage: lo.ToPtr(in.RateCardDiscount.Clone()),
					},
				},
				FlatFee: &billing.FlatFeeLine{
					PerUnitAmount: alpacadecimal.NewFromFloat(in.Amount),
					Quantity:      alpacadecimal.NewFromFloat(in.Quantity),
				},
			},
		},
	}
}

func ExpectJSONEqual(t *testing.T, exp, actual any) {
	t.Helper()

	aJSON, err := json.Marshal(exp)
	require.NoError(t, err)

	bJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	require.JSONEq(t, string(aJSON), string(bJSON))
}
