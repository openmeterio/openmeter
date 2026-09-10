package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGenerateDetailedLinesRejectsMissingMeteredQuantities(t *testing.T) {
	quantity := alpacadecimal.NewFromInt(1)

	for _, tc := range []struct {
		name                         string
		meteredQuantity              *alpacadecimal.Decimal
		meteredPreLinePeriodQuantity *alpacadecimal.Decimal
		expectedError                string
	}{
		{
			name:                         "current quantity",
			meteredPreLinePeriodQuantity: &quantity,
			expectedError:                "metered quantity is required",
		},
		{
			name:            "pre-line period quantity",
			meteredQuantity: &quantity,
			expectedError:   "pre-line period metered quantity is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a metered line whose quantity snapshot is incomplete
			line := &billing.StandardLine{
				StandardLineBase: billing.StandardLineBase{
					Currency: "USD",
					Period: timeutil.ClosedPeriod{
						From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						To:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
				UsageBased: &billing.UsageBasedLine{
					Price:                        productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: quantity}),
					MeteredQuantity:              tc.meteredQuantity,
					MeteredPreLinePeriodQuantity: tc.meteredPreLinePeriodQuantity,
				},
			}

			// when: detailed-line calculation is invoked despite the incomplete snapshot
			_, err := New(Config{}).GenerateDetailedLines(line)

			// then: rating rejects the input instead of dereferencing a nil quantity
			require.EqualError(t, err, tc.expectedError)
		})
	}
}
