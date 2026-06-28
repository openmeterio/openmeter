package lineengine

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestValidateLegacyLineOverrideRejectsSplitLinePeriodChange(t *testing.T) {
	period := lineEngineOverrideTestPeriod()
	line := standardLineForLineEngineOverrideTest(t, period)
	line.SplitLineGroupID = lo.ToPtr("split-line-group-id")

	err := validateLegacyLineOverride(billing.InvoiceLineOverride{
		ExistingLine: line.AsGenericLine(),
		ChangesToApply: billing.ExistingLineOverride{
			Period: mo.Some(timeutil.ClosedPeriod{
				From: period.From,
				To:   period.To.AddDate(0, 1, 0),
			}),
		},
	})
	require.ErrorIs(t, err, billing.ErrInvoiceLineNoPeriodChangeForSplitLine)
}

func TestValidateLegacyLineOverrideValidatesSplitLineUsageDiscountChanges(t *testing.T) {
	period := lineEngineOverrideTestPeriod()

	t.Run("unchanged usage discount succeeds", func(t *testing.T) {
		line := standardLineForLineEngineOverrideTest(t, period)
		line.SplitLineGroupID = lo.ToPtr("split-line-group-id")
		line.RateCardDiscounts = usageDiscountForLineEngineOverrideTest("10")

		err := validateLegacyLineOverride(billing.InvoiceLineOverride{
			ExistingLine: line.AsGenericLine(),
			ChangesToApply: billing.ExistingLineOverride{
				Discounts: mo.Some(usageDiscountForLineEngineOverrideTest("10")),
			},
		})
		require.NoError(t, err)
	})

	t.Run("changed usage discount fails", func(t *testing.T) {
		line := standardLineForLineEngineOverrideTest(t, period)
		line.SplitLineGroupID = lo.ToPtr("split-line-group-id")
		line.RateCardDiscounts = usageDiscountForLineEngineOverrideTest("10")

		err := validateLegacyLineOverride(billing.InvoiceLineOverride{
			ExistingLine: line.AsGenericLine(),
			ChangesToApply: billing.ExistingLineOverride{
				Discounts: mo.Some(usageDiscountForLineEngineOverrideTest("11")),
			},
		})
		require.ErrorIs(t, err, billing.ErrInvoiceLineProgressiveBillingUsageDiscountUpdateForbidden)
	})
}

func TestValidateLegacyLineOverrideRejectsSubscriptionManagedPeriodChange(t *testing.T) {
	period := lineEngineOverrideTestPeriod()
	line := standardLineForLineEngineOverrideTest(t, period)
	line.Subscription = &billing.SubscriptionReference{
		SubscriptionID: "subscription-id",
		PhaseID:        "phase-id",
		ItemID:         "item-id",
		BillingPeriod:  period,
	}

	err := validateLegacyLineOverride(billing.InvoiceLineOverride{
		ExistingLine: line.AsGenericLine(),
		ChangesToApply: billing.ExistingLineOverride{
			Period: mo.Some(timeutil.ClosedPeriod{
				From: period.From,
				To:   period.To.AddDate(0, 1, 0),
			}),
		},
	})
	require.ErrorIs(t, err, billing.ErrInvoiceLineNoPeriodChangeForSubscriptionManagedLine)
}

func lineEngineOverrideTestPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func standardLineForLineEngineOverrideTest(t *testing.T, period timeutil.ClosedPeriod) *billing.StandardLine {
	t.Helper()

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-id",
				Name:      "line",
				CreatedAt: period.From,
				UpdatedAt: period.From,
			}),
			ManagedBy: billing.ManuallyManagedLine,
			Engine:    billing.LineEngineTypeInvoice,
			InvoiceID: "invoice-id",
			Currency:  "USD",
			Period:    period,
			InvoiceAt: period.To,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.RequireFromString("1")}),
			FeatureKey: "feature-key",
		},
	}
}

func usageDiscountForLineEngineOverrideTest(quantity string) billing.Discounts {
	return billing.Discounts{
		Usage: &billing.UsageDiscount{
			UsageDiscount: productcatalog.UsageDiscount{
				Quantity: alpacadecimal.RequireFromString(quantity),
			},
		},
	}
}
