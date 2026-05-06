package testutils

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ExpectedDetailedLine struct {
	ChildUniqueReferenceID string
	Category               stddetailedline.Category
	ServicePeriod          *timeutil.ClosedPeriod
	CorrectsRunID          *string
	PerUnitAmount          float64
	Quantity               float64
	Totals                 ExpectedTotals
}

type ExpectedTotals struct {
	Amount              float64
	ChargesTotal        float64
	DiscountsTotal      float64
	TaxesInclusiveTotal float64
	TaxesExclusiveTotal float64
	TaxesTotal          float64
	CreditsTotal        float64
	Total               float64
}

func ToExpectedDetailedLinesWithServicePeriod(lines usagebased.DetailedLines) []ExpectedDetailedLine {
	return lo.Map(lines, func(line usagebased.DetailedLine, _ int) ExpectedDetailedLine {
		return ExpectedDetailedLine{
			ChildUniqueReferenceID: line.ChildUniqueReferenceID,
			Category:               line.Category,
			ServicePeriod:          lo.ToPtr(line.ServicePeriod),
			CorrectsRunID:          line.CorrectsRunID,
			PerUnitAmount:          line.PerUnitAmount.InexactFloat64(),
			Quantity:               line.Quantity.InexactFloat64(),
			Totals:                 ToExpectedTotals(line.Totals),
		}
	})
}

func ToExpectedTotals(in totals.Totals) ExpectedTotals {
	return ExpectedTotals{
		Amount:              in.Amount.InexactFloat64(),
		ChargesTotal:        in.ChargesTotal.InexactFloat64(),
		DiscountsTotal:      in.DiscountsTotal.InexactFloat64(),
		TaxesInclusiveTotal: in.TaxesInclusiveTotal.InexactFloat64(),
		TaxesExclusiveTotal: in.TaxesExclusiveTotal.InexactFloat64(),
		TaxesTotal:          in.TaxesTotal.InexactFloat64(),
		CreditsTotal:        in.CreditsTotal.InexactFloat64(),
		Total:               in.Total.InexactFloat64(),
	}
}

func FormatDetailedLineChildUniqueReferenceID(id string, servicePeriod timeutil.ClosedPeriod) string {
	return fmt.Sprintf(
		"%s@[%s..%s]",
		id,
		servicePeriod.From.UTC().Format(time.RFC3339),
		servicePeriod.To.UTC().Format(time.RFC3339),
	)
}

func NewIntentForTest(t testing.TB, servicePeriod timeutil.ClosedPeriod, price productcatalog.Price, discounts productcatalog.Discounts) usagebased.Intent {
	t.Helper()

	intent := usagebased.Intent{
		Intent: chargesmeta.Intent{
			Name:              "usage-charge",
			ManagedBy:         billing.SubscriptionManagedLine,
			CustomerID:        "customer-1",
			Currency:          currencyx.Code("USD"),
			ServicePeriod:     servicePeriod,
			FullServicePeriod: servicePeriod,
			BillingPeriod:     servicePeriod,
		},
		InvoiceAt:      servicePeriod.To,
		SettlementMode: productcatalog.InvoiceOnlySettlementMode,
		FeatureKey:     "feature-1",
		Price:          price,
		Discounts:      discounts,
	}

	require.NoError(t, intent.Validate())

	return intent
}

func NewUnitPriceIntentForTest(t testing.TB, servicePeriod timeutil.ClosedPeriod, amount alpacadecimal.Decimal) usagebased.Intent {
	t.Helper()

	return NewIntentForTest(
		t,
		servicePeriod,
		*productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: amount,
		}),
		productcatalog.Discounts{},
	)
}
