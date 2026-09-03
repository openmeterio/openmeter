package lineengine

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestConfigValidateReturnsAllErrors(t *testing.T) {
	err := (Config{}).Validate()

	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, "split line group adapter is required")
	require.ErrorContains(t, err, "rating service is required")
	require.ErrorContains(t, err, "feature meter resolver is required")
	require.ErrorContains(t, err, "streaming connector is required")
	require.ErrorContains(t, err, "max parallel quantity snapshots must be greater than 0")
}

func TestSnapshotValidationIssueClassification(t *testing.T) {
	firstErr := &billing.ErrSnapshotFeatureHasNoMeter{
		LineID:     "line-1",
		FeatureKey: "first",
	}
	secondErr := &billing.ErrSnapshotFeatureHasNoMeter{
		LineID:     "line-2",
		FeatureKey: "second",
	}

	t.Run("converts a validation-only error tree", func(t *testing.T) {
		firstValidationErr := firstErr.AsValidationIssue()
		require.ErrorIs(t, firstValidationErr, billing.ErrInvoiceLineFeatureHasNoMeters)

		issues, systemErr := billing.ToValidationIssues(errors.Join(
			fmt.Errorf("snapshot first: %w", firstValidationErr),
			secondErr.AsValidationIssue(),
		))

		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{
			{
				Severity:  billing.ValidationIssueSeverityCritical,
				Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
				Message:   "feature[first]: usage based invoice line: feature has no meters",
				Component: billing.ValidationComponentOpenMeterMetering,
				Path:      "/lines/line-1",
			},
			{
				Severity:  billing.ValidationIssueSeverityCritical,
				Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
				Message:   "feature[second]: usage based invoice line: feature has no meters",
				Component: billing.ValidationComponentOpenMeterMetering,
				Path:      "/lines/line-2",
			},
		}, issues)
	})

	t.Run("keeps mixed operational failures fatal", func(t *testing.T) {
		issues, systemErr := billing.ToValidationIssues(errors.Join(
			firstErr.AsValidationIssue(),
			errors.New("querying usage backend"),
		))

		require.Nil(t, issues)
		require.ErrorContains(t, systemErr, "querying usage backend")
	})
}

func TestFeatureMetersErrorWrapperClassifiesMissingMeterAssociation(t *testing.T) {
	wrapped := featureMetersErrorWrapper{FeatureMeters: billingfeaturemeter.FeatureMeterCollection{
		ByKey: map[string]billingfeaturemeter.FeatureMeter{
			"meterless": {
				Feature: feature.Feature{Key: "meterless"},
			},
		},
	}}

	t.Run("missing meter association is snapshot validation", func(t *testing.T) {
		_, err := wrapped.GetByKey("meterless", true)

		var snapshotErr *billing.ErrSnapshotFeatureHasNoMeter
		require.ErrorAs(t, err, &snapshotErr)
		require.Equal(t, "meterless", snapshotErr.FeatureKey)
	})

	t.Run("missing feature preserves not found error", func(t *testing.T) {
		_, err := wrapped.GetByKey("missing", true)

		var snapshotErr *billing.ErrSnapshotFeatureHasNoMeter
		require.False(t, errors.As(err, &snapshotErr))
		require.True(t, models.IsGenericNotFoundError(err))
	})
}

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

func TestValidateLegacyLineOverrideValidatesSubscriptionManagedPeriodChange(t *testing.T) {
	period := lineEngineOverrideTestPeriod()
	subscription := &billing.SubscriptionReference{
		SubscriptionID: "subscription-id",
		PhaseID:        "phase-id",
		ItemID:         "item-id",
		BillingPeriod:  period,
	}

	t.Run("usage-based line period change fails", func(t *testing.T) {
		line := standardLineForLineEngineOverrideTest(t, period)
		line.Subscription = subscription

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
	})

	t.Run("flat-fee line period change succeeds", func(t *testing.T) {
		line := standardLineForLineEngineOverrideTest(t, period)
		line.Subscription = subscription
		line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.RequireFromString("1"),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		err := validateLegacyLineOverride(billing.InvoiceLineOverride{
			ExistingLine: line.AsGenericLine(),
			ChangesToApply: billing.ExistingLineOverride{
				Period: mo.Some(timeutil.ClosedPeriod{
					From: period.From,
					To:   period.To.AddDate(0, 1, 0),
				}),
			},
		})
		require.NoError(t, err)
	})

	t.Run("usage-based to flat-fee period change fails", func(t *testing.T) {
		line := standardLineForLineEngineOverrideTest(t, period)
		line.Subscription = subscription

		err := validateLegacyLineOverride(billing.InvoiceLineOverride{
			ExistingLine: line.AsGenericLine(),
			ChangesToApply: billing.ExistingLineOverride{
				Period: mo.Some(timeutil.ClosedPeriod{
					From: period.From,
					To:   period.To.AddDate(0, 1, 0),
				}),
				Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.RequireFromString("1"),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})),
			},
		})
		require.ErrorIs(t, err, billing.ErrInvoiceLineNoPeriodChangeForSubscriptionManagedLine)
	})
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
