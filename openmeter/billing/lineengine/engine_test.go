package lineengine

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	featuremeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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

type quantitySnapshotFeatureServiceStub struct {
	features []feature.Feature
	err      error
}

func (s quantitySnapshotFeatureServiceStub) ListFeatures(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return pagination.Result[feature.Feature]{Items: s.features}, s.err
}

type quantitySnapshotMeterServiceStub struct {
	meters []meter.Meter
	err    error
}

func (s quantitySnapshotMeterServiceStub) ListMeters(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{Items: s.meters}, s.err
}

func TestSnapshotLineQuantitiesContinuesWithPartialFeatureMeters(t *testing.T) {
	period := lineEngineOverrideTestPeriod()
	meterID := "valid-meter-id"
	validMeter := meter.Meter{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace: "namespace",
			ID:        meterID,
			Name:      "Valid meter",
			CreatedAt: period.From,
			UpdatedAt: period.From,
		}),
		Key:         "valid-meter",
		Aggregation: meter.MeterAggregationSum,
	}
	engine, streamingConnector := newQuantitySnapshotTestEngine(t,
		[]feature.Feature{
			{Namespace: "namespace", ID: "valid-feature-id", Key: "valid-feature", MeterID: &meterID},
			{Namespace: "namespace", ID: "meterless-feature-id", Key: "meterless-feature"},
		},
		[]meter.Meter{validMeter},
		nil,
	)
	streamingConnector.AddRow(validMeter.Key, meter.MeterQueryRow{
		Value:       7,
		WindowStart: period.From,
		WindowEnd:   period.To,
	})

	validLine := quantitySnapshotTestLine("line-valid", "valid-feature", productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
	}))
	meterlessLine := quantitySnapshotTestLine("line-meterless", "meterless-feature", productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
	}))
	missingFlatLine := quantitySnapshotTestLine("line-missing-flat", "missing-feature", productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(1),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	}))
	featurelessLine := quantitySnapshotTestLine("line-featureless", "", productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(1),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	}))

	// Given valid, meterless, missing-feature, and featureless lines.
	lines := billing.StandardLines{validLine, meterlessLine, missingFlatLine, featurelessLine}

	// When their quantities are snapshotted from a partial feature-meter result.
	err := engine.SnapshotLineQuantities(t.Context(), quantitySnapshotTestInvoice(), lines)

	// Then metered lines retain feature validation, while flat lines snapshot without resolving their feature.
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.ElementsMatch(t, billing.ValidationIssues{
		{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:  "feature[meterless-feature]: usage based invoice line: feature has no meters",
			Path:     "/lines/line-meterless",
		},
	}, issues)
	require.Equal(t, 7.0, validLine.UsageBased.Quantity.InexactFloat64())
	require.Nil(t, meterlessLine.UsageBased.Quantity)
	require.Equal(t, 1.0, missingFlatLine.UsageBased.Quantity.InexactFloat64())
	require.Equal(t, 1.0, featurelessLine.UsageBased.Quantity.InexactFloat64())
}

func TestSnapshotLineQuantitiesRejectsResolverSystemErrors(t *testing.T) {
	resolverErr := errors.New("feature service unavailable")
	engine, _ := newQuantitySnapshotTestEngine(t, nil, nil, resolverErr)
	line := quantitySnapshotTestLine("line-valid", "valid-feature", productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
	}))

	// Given a feature service that cannot resolve any requested features.
	// When quantity snapshotting attempts to resolve the line's feature meter.
	err := engine.SnapshotLineQuantities(t.Context(), quantitySnapshotTestInvoice(), billing.StandardLines{line})

	// Then the operational failure aborts snapshotting instead of becoming a validation issue.
	issues, systemErr := billing.ToValidationIssues(err)
	require.Nil(t, issues)
	require.ErrorIs(t, systemErr, resolverErr)
	require.Nil(t, line.UsageBased.Quantity)
}

func TestLineEngineValidationErrorOwnsValidationIssues(t *testing.T) {
	// Given a line-scoped validation issue without a component owner.
	err := billing.ValidationError{
		Err: billing.ValidationWithFieldPrefix("lines/line-id", billing.ErrInvoiceLineFeatureHasNoMeters),
	}

	// When it crosses the line-engine boundary.
	wrappedErr := billing.NewLineEngineValidationError(&Engine{}, err)
	issues, systemErr := billing.ToValidationIssues(wrappedErr)

	// Then the line engine becomes the owning component without losing the original error chain or issue details.
	require.ErrorAs(t, wrappedErr, &billing.ValidationError{})
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{
		{
			Severity:  billing.ValidationIssueSeverityCritical,
			Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:   billing.ErrInvoiceLineFeatureHasNoMeters.Message,
			Component: billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice),
			Path:      "/lines/line-id",
		},
	}, issues)
}

func newQuantitySnapshotTestEngine(t *testing.T, features []feature.Feature, meters []meter.Meter, featureServiceErr error) (*Engine, *streamingtestutils.MockStreamingConnector) {
	t.Helper()

	featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: quantitySnapshotFeatureServiceStub{features: features, err: featureServiceErr},
		MeterService:   quantitySnapshotMeterServiceStub{meters: meters},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	return &Engine{
		featureMeterResolver:         featureMeterResolver,
		streamingConnector:           streamingConnector,
		maxParallelQuantitySnapshots: 4,
	}, streamingConnector
}

func quantitySnapshotTestInvoice() billing.StandardInvoice {
	return billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "namespace",
			Customer: billing.InvoiceCustomer{
				CustomerID: "customer-id",
				Name:       "Customer",
			},
		},
	}
}

func quantitySnapshotTestLine(id, featureKey string, price *productcatalog.Price) *billing.StandardLine {
	period := lineEngineOverrideTestPeriod()

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "namespace",
				ID:        id,
				Name:      id,
				CreatedAt: period.From,
				UpdatedAt: period.From,
			}),
			Period: period,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      price,
			FeatureKey: featureKey,
		},
	}
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
