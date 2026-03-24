package billing

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type UnitConfigTestSuite struct {
	BaseSuite
}

func TestUnitConfig(t *testing.T) {
	suite.Run(t, new(UnitConfigTestSuite))
}

// TestUnitConfigDivideWithCeil tests that UnitConfig with divide+ceil converts
// raw metered quantities (e.g. bytes) into billing units (e.g. GB) and rounds up.
// This is the "package pricing" replacement pattern.
func (s *UnitConfigTestSuite) TestUnitConfigDivideWithCeil() {
	namespace := s.GetUniqueNamespace("ns-unitconfig-divide-ceil")
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	// Set up a meter
	meterSlug := "tokens-total"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Tokens Total",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))
	defer func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
		s.MockStreamingConnector.Reset()
	}()

	// Create the feature
	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	periodEnd := time.Now().Add(-time.Second)
	periodStart := periodEnd.Add(-time.Hour)
	invoiceAt := periodEnd.Add(-time.Millisecond)

	// Register 1,500,001 raw tokens (should round up to 2 million-packs with divide 1e6 + ceil)
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 1500001, periodStart.Add(time.Minute))

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.ConversionOperationDivide,
		ConversionFactor: alpacadecimal.NewFromFloat(1e6),
		Rounding:         productcatalog.RoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("M"),
	}

	s.Run("UnitConfig is persisted and returned on gathering lines", func() {
		res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.Code(currency.USD),
				Lines: []billing.GatheringLine{
					{
						GatheringLineBase: billing.GatheringLineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Namespace: namespace,
								Name:      "Tokens",
							}),
							ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
							InvoiceAt:     invoiceAt,
							ManagedBy:     billing.ManuallyManagedLine,
							Currency:      currencyx.Code(currency.USD),
							FeatureKey:    feat.Key,
							Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount: alpacadecimal.NewFromFloat(10),
							})),
							UnitConfig: unitConfig,
						},
					},
				},
			})
		s.NoError(err)
		s.Len(res.Lines, 1)

		// Verify UnitConfig is stored and returned
		line := res.Lines[0]
		s.NotNil(line.UnitConfig, "UnitConfig should be persisted on the gathering line")
		s.Equal(productcatalog.ConversionOperationDivide, line.UnitConfig.Operation)
		s.True(line.UnitConfig.ConversionFactor.Equal(alpacadecimal.NewFromFloat(1e6)))
		s.Equal(productcatalog.RoundingModeCeiling, line.UnitConfig.Rounding)
		s.Equal(lo.ToPtr("M"), line.UnitConfig.DisplayUnit)
	})

	s.Run("UnitConfig converts quantities on invoice lines", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		lines := invoices[0].Lines.OrEmpty()
		s.NotEmpty(lines)

		// Find the usage-based line
		var usageLine *billing.StandardLine
		for _, l := range lines {
			if l.UsageBased != nil && l.UsageBased.FeatureKey == feat.Key {
				usageLine = l
				break
			}
		}
		s.NotNil(usageLine, "should have a usage-based line")

		// UnitConfig should be snapshotted on the invoice line
		s.NotNil(usageLine.UsageBased.UnitConfig, "UnitConfig should be snapshotted on invoice line")
		s.Equal(productcatalog.ConversionOperationDivide, usageLine.UsageBased.UnitConfig.Operation)

		// MeteredQuantity should be the raw value (1500001)
		s.NotNil(usageLine.UsageBased.MeteredQuantity)
		s.True(usageLine.UsageBased.MeteredQuantity.Equal(alpacadecimal.NewFromFloat(1500001)),
			"MeteredQuantity should be raw meter value, got %s", usageLine.UsageBased.MeteredQuantity.String())

		// After conversion (1500001 / 1e6 = 1.500001) and ceil rounding → 2
		// The detailed line should reflect the converted+rounded quantity
		s.NotEmpty(usageLine.DetailedLines, "should have detailed lines")

		// The total should be 2 * $10 = $20 (2 million-packs at $10 each)
		s.True(usageLine.Totals.Total.Equal(alpacadecimal.NewFromFloat(20)),
			"Total should be 20 (2 packages * $10), got %s", usageLine.Totals.Total.String())
	})
}

// TestUnitConfigMultiply tests UnitConfig with multiply operation (dynamic/margin pricing pattern).
func (s *UnitConfigTestSuite) TestUnitConfigMultiply() {
	namespace := s.GetUniqueNamespace("ns-unitconfig-multiply")
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "api-calls"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "API Calls",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))
	defer func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
		s.MockStreamingConnector.Reset()
	}()

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	periodEnd := time.Now().Add(-time.Second)
	periodStart := periodEnd.Add(-time.Hour)
	invoiceAt := periodEnd.Add(-time.Millisecond)

	// Register 100 API calls, with 1.2x margin multiplier → effective quantity = 120
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 100, periodStart.Add(time.Minute))

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.ConversionOperationMultiply,
		ConversionFactor: alpacadecimal.NewFromFloat(1.2),
		Rounding:         productcatalog.RoundingModeNone,
	}

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "API Calls",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						FeatureKey:    feat.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(1),
						})),
						UnitConfig: unitConfig,
					},
				},
			},
		})
	s.NoError(err)
	s.Len(res.Lines, 1)

	// Invoice and check
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.NotEmpty(lines)

	var usageLine *billing.StandardLine
	for _, l := range lines {
		if l.UsageBased != nil && l.UsageBased.FeatureKey == feat.Key {
			usageLine = l
			break
		}
	}
	s.NotNil(usageLine, "should have a usage-based line")

	// Raw metered = 100, after 1.2x multiply = 120, at $1/unit = $120
	s.NotNil(usageLine.UsageBased.MeteredQuantity)
	s.True(usageLine.UsageBased.MeteredQuantity.Equal(alpacadecimal.NewFromFloat(100)),
		"MeteredQuantity should be 100, got %s", usageLine.UsageBased.MeteredQuantity.String())

	s.True(usageLine.Totals.Total.Equal(alpacadecimal.NewFromFloat(120)),
		"Total should be 120 (100 * 1.2 * $1), got %s", usageLine.Totals.Total.String())
}

// TestUnitConfigWithTieredPrice tests the key use case: UnitConfig with graduated tiered pricing.
// This is the "tiered package pricing" pattern from the proposal.
func (s *UnitConfigTestSuite) TestUnitConfigWithTieredPrice() {
	namespace := s.GetUniqueNamespace("ns-unitconfig-tiered")
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "tokens-tiered"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Tokens Tiered",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))
	defer func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
		s.MockStreamingConnector.Reset()
	}()

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	periodEnd := time.Now().Add(-time.Second)
	periodStart := periodEnd.Add(-time.Hour)
	invoiceAt := periodEnd.Add(-time.Millisecond)

	// Register 7,500,001 raw tokens
	// With divide by 1e6 + ceil → 8 million-packs
	// Graduated tiered: 0-5 at $10/unit, 5+ at $8/unit
	// Tier 1: 5 units × $10 = $50
	// Tier 2: 3 units × $8 = $24
	// Total: $74
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 7500001, periodStart.Add(time.Minute))

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.ConversionOperationDivide,
		ConversionFactor: alpacadecimal.NewFromFloat(1e6),
		Rounding:         productcatalog.RoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("M"),
	}

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Tokens Tiered",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						FeatureKey:    feat.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
							Mode: productcatalog.GraduatedTieredPrice,
							Tiers: []productcatalog.PriceTier{
								{
									UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
									UnitPrice: &productcatalog.PriceTierUnitPrice{
										Amount: alpacadecimal.NewFromFloat(10),
									},
								},
								{
									UnitPrice: &productcatalog.PriceTierUnitPrice{
										Amount: alpacadecimal.NewFromFloat(8),
									},
								},
							},
						})),
						UnitConfig: unitConfig,
					},
				},
			},
		})
	s.NoError(err)
	s.Len(res.Lines, 1)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.NotEmpty(lines)

	var usageLine *billing.StandardLine
	for _, l := range lines {
		if l.UsageBased != nil && l.UsageBased.FeatureKey == feat.Key {
			usageLine = l
			break
		}
	}
	s.NotNil(usageLine, "should have a usage-based line")

	// MeteredQuantity should be the raw value (7500001)
	s.NotNil(usageLine.UsageBased.MeteredQuantity)
	s.True(usageLine.UsageBased.MeteredQuantity.Equal(alpacadecimal.NewFromFloat(7500001)),
		"MeteredQuantity should be 7500001, got %s", usageLine.UsageBased.MeteredQuantity.String())

	// Total should be $74 (5 × $10 + 3 × $8)
	s.True(usageLine.Totals.Total.Equal(alpacadecimal.NewFromFloat(74)),
		"Total should be 74 (5*$10 + 3*$8), got %s", usageLine.Totals.Total.String())
}

// TestUnitConfigWithFloorRounding tests UnitConfig with floor rounding — "bill only for complete units consumed".
func (s *UnitConfigTestSuite) TestUnitConfigWithFloorRounding() {
	namespace := s.GetUniqueNamespace("ns-unitconfig-floor")
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")
	s.NotNil(customerEntity)

	meterSlug := "bytes-floor"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Bytes Floor",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))
	defer func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
		s.MockStreamingConnector.Reset()
	}()

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	periodEnd := time.Now().Add(-time.Second)
	periodStart := periodEnd.Add(-time.Hour)
	invoiceAt := periodEnd.Add(-time.Millisecond)

	// Register 1,999 bytes — with divide by 1000 + floor → 1 KB
	// At $5/KB = $5
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 1999, periodStart.Add(time.Minute))

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.ConversionOperationDivide,
		ConversionFactor: alpacadecimal.NewFromFloat(1000),
		Rounding:         productcatalog.RoundingModeFloor,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("KB"),
	}

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Bytes",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						FeatureKey:    feat.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						})),
						UnitConfig: unitConfig,
					},
				},
			},
		})
	s.NoError(err)
	s.Len(res.Lines, 1)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	var usageLine *billing.StandardLine
	for _, l := range lines {
		if l.UsageBased != nil && l.UsageBased.FeatureKey == feat.Key {
			usageLine = l
			break
		}
	}
	s.NotNil(usageLine, "should have a usage-based line")

	// MeteredQuantity should be the raw value (1999)
	s.NotNil(usageLine.UsageBased.MeteredQuantity)
	s.True(usageLine.UsageBased.MeteredQuantity.Equal(alpacadecimal.NewFromFloat(1999)),
		"MeteredQuantity should be 1999, got %s", usageLine.UsageBased.MeteredQuantity.String())

	// 1999 / 1000 = 1.999, floor → 1, at $5/unit = $5
	s.True(usageLine.Totals.Total.Equal(alpacadecimal.NewFromFloat(5)),
		"Total should be 5 (1 KB * $5), got %s", usageLine.Totals.Total.String())
}

// TestUnitConfigNil tests that billing works normally when UnitConfig is not set.
func (s *UnitConfigTestSuite) TestUnitConfigNil() {
	namespace := s.GetUniqueNamespace("ns-unitconfig-nil")
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	customerEntity := s.CreateTestCustomer(namespace, "test-customer")

	meterSlug := "events"
	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: meterID,
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Events",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	}))
	defer func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
		s.MockStreamingConnector.Reset()
	}()

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      meterSlug,
		Key:       meterSlug,
		MeterID:   lo.ToPtr(meterID),
	}))

	periodEnd := time.Now().Add(-time.Second)
	periodStart := periodEnd.Add(-time.Hour)
	invoiceAt := periodEnd.Add(-time.Millisecond)

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 50, periodStart.Add(time.Minute))

	// No UnitConfig — raw quantity should flow through unmodified
	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []billing.GatheringLine{
				{
					GatheringLineBase: billing.GatheringLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: namespace,
							Name:      "Events",
						}),
						ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Currency:      currencyx.Code(currency.USD),
						FeatureKey:    feat.Key,
						Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(2),
						})),
						// No UnitConfig
					},
				},
			},
		})
	s.NoError(err)
	s.Len(res.Lines, 1)
	s.Nil(res.Lines[0].UnitConfig, "UnitConfig should be nil")

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	var usageLine *billing.StandardLine
	for _, l := range lines {
		if l.UsageBased != nil && l.UsageBased.FeatureKey == feat.Key {
			usageLine = l
			break
		}
	}
	s.NotNil(usageLine)
	s.Nil(usageLine.UsageBased.UnitConfig, "UnitConfig should remain nil")

	// 50 events * $2 = $100
	s.True(usageLine.Totals.Total.Equal(alpacadecimal.NewFromFloat(100)),
		"Total should be 100 (50 * $2), got %s", usageLine.Totals.Total.String())
}
