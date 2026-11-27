package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestStandardLineFeatureMeterReference(t *testing.T) {
	// given:
	// - persisted standard lines with metered, flat, and absent feature dependencies
	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})
	flatPrice := productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(1)})
	lines := StandardLines{
		{
			StandardLineBase: StandardLineBase{ManagedResource: models.ManagedResource{ID: "metered-line"}},
			UsageBased:       &UsageBasedLine{FeatureKey: "metered-feature", Price: unitPrice},
		},
		{
			StandardLineBase: StandardLineBase{ManagedResource: models.ManagedResource{ID: "flat-line"}},
			UsageBased:       &UsageBasedLine{FeatureKey: "flat-feature", Price: flatPrice},
		},
		{
			StandardLineBase: StandardLineBase{ManagedResource: models.ManagedResource{ID: "featureless-line"}},
		},
	}

	// when:
	// - their feature references and identities are read
	meteredRef := lines[0].GetFeatureMeterRef()
	flatRef := lines[1].GetFeatureMeterRef()
	featurelessRef := lines[2].GetFeatureMeterRef()

	// then:
	// - meter requirements follow the price and every persisted line retains its identity
	require.Equal(t, "metered-feature", meteredRef.IDOrKey.Key)
	require.True(t, meteredRef.RequireMeter)
	require.Equal(t, "flat-feature", flatRef.IDOrKey.Key)
	require.False(t, flatRef.RequireMeter)
	require.Nil(t, featurelessRef)
	require.Equal(t, featuremeter.FeatureReferenceIdentity{
		Kind: featuremeter.FeatureReferenceKindLines,
		ID:   "metered-line",
	}, lines[0].GetFeatureMeterOwner())
	require.Equal(t, featuremeter.FeatureReferenceIdentity{
		Kind: featuremeter.FeatureReferenceKindLines,
		ID:   "flat-line",
	}, lines[1].GetFeatureMeterOwner())
	require.Equal(t, featuremeter.FeatureReferenceIdentity{
		Kind: featuremeter.FeatureReferenceKindLines,
		ID:   "featureless-line",
	}, lines[2].GetFeatureMeterOwner())
}

func TestStandardLineValidateAllowsNonNegativeTotals(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.Zero

	require.NoError(t, line.Validate())

	line.Totals.Total = alpacadecimal.NewFromInt(1)

	require.NoError(t, line.Validate())
}

func TestStandardLineValidateIgnoresInvoiceAt(t *testing.T) {
	t.Parallel()

	line := validStandardLineForValidation()
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
	})
	line.UsageBased.FeatureKey = "feature-key"

	for name, invoiceAt := range map[string]time.Time{
		"zero":                {},
		"before period start": line.Period.From.Add(-time.Hour),
		"inside period":       line.Period.From.Add(30 * time.Minute),
		"at period end":       line.Period.To,
		"after period end":    line.Period.To.Add(time.Hour),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			line := line
			line.InvoiceAt = invoiceAt

			require.NoError(t, line.Validate())
		})
	}
}

func TestStandardLineValidateRejectsNegativeTotals(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.NewFromInt(-1)

	require.ErrorContains(t, line.Validate(), "totals: total is negative")
}

func TestStandardLineValidateAllowsNegativeDetailedLineQuantityWithPositiveTotal(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.NewFromInt(1)
	line.DetailedLines = DetailedLines{
		{
			DetailedLineBase: DetailedLineBase{
				InvoiceID: line.InvoiceID,
				Base: stddetailedline.Base{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{
							Namespace: line.Namespace,
						},
						ID:   "detail_123",
						Name: "usage correction",
					},
					Category:               stddetailedline.CategoryRegular,
					ChildUniqueReferenceID: "detail_123",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					ServicePeriod:          line.Period,
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(-1),
				},
			},
		},
	}

	require.NoError(t, line.Validate())
}

func TestExistingLineOverrideApplyStandardLineDoesNotMutateOriginalUsageBasedPrice(t *testing.T) {
	line := validStandardLineForValidation()
	originalPrice := line.UsageBased.Price.Clone()
	overridePrice := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	updatedLine, err := ExistingLineOverride{
		Price: mo.Some(overridePrice),
	}.Apply(line.AsGenericLine())

	require.NoError(t, err)
	require.True(t, originalPrice.Equal(line.UsageBased.Price))

	updatedStandardLine, err := updatedLine.AsInvoiceLine().AsStandardLine()
	require.NoError(t, err)
	require.True(t, overridePrice.Equal(updatedStandardLine.UsageBased.Price))
	require.NotSame(t, overridePrice, updatedStandardLine.UsageBased.Price)
}

func TestStandardLineWithTargetStateKeepsInvoicingReferences(t *testing.T) {
	// given: a synced line with a detailed line, and a target state that knows nothing about the invoicing app
	existing := validStandardLineForValidation()
	existing.ExternalIDs.Invoicing = "il_line"
	existing.InvoicingAppTaxCode = lo.ToPtr("txcd_10000000")
	existing.LegacyStripeTaxCode = "txcd_99999999"
	existing.DetailedLines = DetailedLines{
		{
			DetailedLineBase: DetailedLineBase{Base: stddetailedline.Base{
				ManagedResource:        models.ManagedResource{ID: "detailed_123"},
				ChildUniqueReferenceID: "flat-price",
				ExternalIDs:            externalid.LineExternalIDs{Invoicing: "il_detailed"},
			}},
		},
	}

	target := validStandardLineForValidation()
	target.DetailedLines = DetailedLines{
		{
			DetailedLineBase: DetailedLineBase{Base: stddetailedline.Base{ChildUniqueReferenceID: "flat-price"}},
		},
	}

	// when: the target state is applied to the synced line
	merged, err := existing.AsGenericLine().WithTargetState(target.AsGenericLine())
	require.NoError(t, err)

	// then: the line and the reused detailed line still point at their objects in the invoicing app
	mergedLine, err := merged.AsInvoiceLine().AsStandardLine()
	require.NoError(t, err)
	require.Equal(t, "il_line", mergedLine.ExternalIDs.Invoicing)
	require.Equal(t, lo.ToPtr("txcd_10000000"), mergedLine.InvoicingAppTaxCode)
	require.Equal(t, "txcd_99999999", mergedLine.LegacyStripeTaxCode)
	require.Len(t, mergedLine.DetailedLines, 1)
	require.Equal(t, "detailed_123", mergedLine.DetailedLines[0].ID)
	require.Equal(t, "il_detailed", mergedLine.DetailedLines[0].ExternalIDs.Invoicing)
}

func TestStandardLineDoesNotExposeInvoiceAtAccessor(t *testing.T) {
	type invoiceAtReader interface {
		GetInvoiceAt() time.Time
	}

	line := validStandardLineForValidation()

	// StandardLine.InvoiceAt is retained only to display the original invoice-at
	// timestamp when a gathering line is rendered into a standard invoice line.
	// Standard-line business logic must not discover it through an accessor and
	// treat it as scheduling state.
	_, valueImplements := any(line).(invoiceAtReader)
	_, pointerImplements := any(&line).(invoiceAtReader)
	require.False(t, valueImplements, "StandardLine must not expose InvoiceAt through an accessor")
	require.False(t, pointerImplements, "*StandardLine must not expose InvoiceAt through an accessor")
}

func validStandardLineForValidation() StandardLine {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return StandardLine{
		StandardLineBase: StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "test-namespace",
				},
				ID:   "line_123",
				Name: "usage",
			},
			ManagedBy: SystemManagedLine,
			InvoiceID: "inv_123",
			Currency:  currencyx.FiatCode("USD"),
			Period: timeutil.ClosedPeriod{
				From: start,
				To:   start.Add(time.Hour),
			},
			InvoiceAt: start.Add(time.Hour),
			Totals: totals.Totals{
				Total: alpacadecimal.NewFromInt(1),
			},
		},
		UsageBased: &UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
}

func TestStandardLineBaseBilledTaxCode(t *testing.T) {
	tests := []struct {
		name                string
		invoicingAppTaxCode *string
		legacyStripeTaxCode string
		invoicingAppType    app.AppType
		want                string
	}{
		{
			name:                "stamped value wins",
			invoicingAppTaxCode: lo.ToPtr("txcd_20000000"),
			invoicingAppType:    app.AppTypeStripe,
			want:                "txcd_20000000",
		},
		{
			name:                "stamped value is used on non-Stripe invoices too",
			invoicingAppTaxCode: lo.ToPtr("custom-code"),
			invoicingAppType:    app.AppTypeCustomInvoicing,
			want:                "custom-code",
		},
		{
			name:                "stamped empty means provider default",
			invoicingAppTaxCode: lo.ToPtr(""),
			invoicingAppType:    app.AppTypeStripe,
			want:                "",
		},
		{
			name:                "stamped empty wins over the legacy code",
			invoicingAppTaxCode: lo.ToPtr(""),
			legacyStripeTaxCode: "txcd_10000000",
			invoicingAppType:    app.AppTypeStripe,
			want:                "",
		},
		{
			name:                "unstamped Stripe line falls back to the legacy code",
			legacyStripeTaxCode: "txcd_10000000",
			invoicingAppType:    app.AppTypeStripe,
			want:                "txcd_10000000",
		},
		{
			name:             "unstamped Stripe line without a legacy code uses the provider default",
			invoicingAppType: app.AppTypeStripe,
			want:             "",
		},
		{
			name:                "unstamped non-Stripe line ignores the legacy code",
			legacyStripeTaxCode: "txcd_10000000",
			invoicingAppType:    app.AppTypeCustomInvoicing,
			want:                "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a line with the stamp and legacy code of the test case
			line := StandardLineBase{
				InvoicingAppTaxCode: tt.invoicingAppTaxCode,
				LegacyStripeTaxCode: tt.legacyStripeTaxCode,
			}

			// when: the billed tax code is resolved for the invoicing app
			got := line.BilledTaxCode(tt.invoicingAppType)

			// then: it follows the stamp, then the Stripe legacy code
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStandardLineBaseCloneCopiesInvoicingAppTaxCode(t *testing.T) {
	// given: a stamped line
	line := StandardLineBase{InvoicingAppTaxCode: lo.ToPtr("txcd_10000000")}

	// when: it is cloned and the original stamp changes
	cloned := line.Clone()
	*line.InvoicingAppTaxCode = "changed"

	// then: the clone keeps its own value, and Equal sees the difference
	require.Equal(t, "txcd_10000000", *cloned.InvoicingAppTaxCode)
	require.False(t, line.Equal(cloned))
}
