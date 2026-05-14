package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestDetailedLineValidateAllowsNegativeQuantity(t *testing.T) {
	line := validDetailedLineForValidation()
	line.Quantity = alpacadecimal.NewFromInt(-1)

	require.NoError(t, line.Validate())
}

func TestDetailedLineValidateRejectsNegativePerUnitAmount(t *testing.T) {
	line := validDetailedLineForValidation()
	line.PerUnitAmount = alpacadecimal.NewFromInt(-1)

	require.ErrorContains(t, line.Validate(), "price should be positive or zero")
}

func validDetailedLineForValidation() DetailedLine {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return DetailedLine{
		DetailedLineBase: DetailedLineBase{
			InvoiceID: "inv_123",
			Base: stddetailedline.Base{
				Category:               stddetailedline.CategoryRegular,
				ChildUniqueReferenceID: "child_123",
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				ServicePeriod: timeutil.ClosedPeriod{
					From: start,
					To:   start.Add(time.Hour),
				},
				Currency:      currencyx.Code("USD"),
				PerUnitAmount: alpacadecimal.NewFromInt(10),
				Quantity:      alpacadecimal.NewFromInt(1),
			},
		},
	}
}
