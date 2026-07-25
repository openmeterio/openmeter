package httpdriver

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMergeStandardLineFromInvoiceLineReplaceUpdateOverwritesTaxConfig(t *testing.T) {
	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: "1",
	}))

	productCatalogPrice, err := productcataloghttp.AsPrice(price)
	require.NoError(t, err)

	taxCodeID := "tax-code-id"
	taxBehavior := api.TaxBehaviorExclusive
	productCatalogTaxBehavior := productcatalog.ExclusiveTaxBehavior
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	featureKey := "feature-key"

	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-id",
				Name:      "line",
				CreatedAt: period.From,
				UpdatedAt: period.From,
			}),
			Period: period,
			TaxConfig: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Behavior:  &productCatalogTaxBehavior,
					TaxCodeID: &taxCodeID,
				},
				TaxCode: &taxcode.TaxCode{
					NamespacedID: models.NamespacedID{
						Namespace: "ns",
						ID:        taxCodeID,
					},
					Name: "Tax Code",
				},
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      productCatalogPrice,
			FeatureKey: featureKey,
		},
	}

	mergedLine, err := mergeStandardLineFromInvoiceLineReplaceUpdate(line, api.InvoiceLineReplaceUpdate{
		Name:      line.Name,
		Period:    api.Period(period),
		InvoiceAt: period.To,
		Price:     &price,
		RateCard: &api.InvoiceUsageBasedRateCard{
			FeatureKey: &featureKey,
			Price:      &price,
			TaxConfig: &api.TaxConfig{
				Behavior:  &taxBehavior,
				TaxCodeId: &taxCodeID,
			},
		},
		TaxConfig: &api.TaxConfig{
			Behavior:  &taxBehavior,
			TaxCodeId: &taxCodeID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, mergedLine.TaxConfig)
	require.Equal(t, taxCodeID, *mergedLine.TaxConfig.TaxCodeID)
	require.Nil(t, mergedLine.TaxConfig.TaxCode)
}

func TestMergeStandardLineFromInvoiceLineReplaceUpdateLeavesProviderDefaultOmissionForInvoiceNormalization(t *testing.T) {
	period := invoiceLineTestPeriod()
	line := standardInvoiceLineForMergeTest(t, period)

	defaultTaxCodeID := "default-tax-code-id"
	productCatalogTaxBehavior := productcatalog.ExclusiveTaxBehavior
	line.TaxConfig = &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &productCatalogTaxBehavior,
			TaxCodeID: &defaultTaxCodeID,
		},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{
				Namespace: "ns",
				ID:        defaultTaxCodeID,
			},
			Name: "Default Tax Code",
		},
	}

	taxBehavior := api.TaxBehaviorExclusive
	mergedLine, err := mergeStandardLineFromInvoiceLineReplaceUpdate(line, invoiceLineReplaceUpdateForMergeTest(t, period, line.UsageBased.FeatureKey, "1", func(update *api.InvoiceLineReplaceUpdate) {
		update.RateCard.TaxConfig = &api.TaxConfig{
			Behavior: &taxBehavior,
		}
		update.TaxConfig = update.RateCard.TaxConfig
	}))
	require.NoError(t, err)
	require.NotNil(t, mergedLine.TaxConfig)
	require.Nil(t, mergedLine.TaxConfig.TaxCodeID)
	require.Nil(t, mergedLine.TaxConfig.TaxCode)
}

func TestMergeStandardLineFromInvoiceLineReplaceUpdateDoesNotPreserveExplicitTaxCodeWhenPayloadOmitsTaxCodeIdentity(t *testing.T) {
	period := invoiceLineTestPeriod()
	line := standardInvoiceLineForMergeTest(t, period)

	explicitTaxCodeID := "explicit-tax-code-id"
	productCatalogTaxBehavior := productcatalog.ExclusiveTaxBehavior
	line.TaxConfig = &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &productCatalogTaxBehavior,
			TaxCodeID: &explicitTaxCodeID,
		},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{
				Namespace: "ns",
				ID:        explicitTaxCodeID,
			},
			Name: "Explicit Tax Code",
		},
	}

	taxBehavior := api.TaxBehaviorExclusive
	mergedLine, err := mergeStandardLineFromInvoiceLineReplaceUpdate(line, invoiceLineReplaceUpdateForMergeTest(t, period, line.UsageBased.FeatureKey, "1", func(update *api.InvoiceLineReplaceUpdate) {
		update.RateCard.TaxConfig = &api.TaxConfig{
			Behavior: &taxBehavior,
		}
		update.TaxConfig = update.RateCard.TaxConfig
	}))
	require.NoError(t, err)
	require.NotNil(t, mergedLine.TaxConfig)
	require.Nil(t, mergedLine.TaxConfig.TaxCodeID)
	require.Nil(t, mergedLine.TaxConfig.TaxCode)
}

func TestMergeStandardLineFromInvoiceLineReplaceUpdateDropsResolvedTaxCodeWhenTaxConfigChanges(t *testing.T) {
	period := invoiceLineTestPeriod()
	line := standardInvoiceLineForMergeTest(t, period)

	oldTaxCodeID := "old-tax-code-id"
	oldTaxBehavior := productcatalog.ExclusiveTaxBehavior
	line.TaxConfig = &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &oldTaxBehavior,
			TaxCodeID: &oldTaxCodeID,
		},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{
				Namespace: "ns",
				ID:        oldTaxCodeID,
			},
			Name: "Old Tax Code",
		},
	}

	newTaxCodeID := "new-tax-code-id"
	newTaxBehavior := api.TaxBehaviorInclusive
	mergedLine, err := mergeStandardLineFromInvoiceLineReplaceUpdate(line, invoiceLineReplaceUpdateForMergeTest(t, period, line.UsageBased.FeatureKey, "1", func(update *api.InvoiceLineReplaceUpdate) {
		update.RateCard.TaxConfig = &api.TaxConfig{
			Behavior:  &newTaxBehavior,
			TaxCodeId: &newTaxCodeID,
		}
		update.TaxConfig = update.RateCard.TaxConfig
	}))
	require.NoError(t, err)
	require.NotNil(t, mergedLine.TaxConfig)
	require.Nil(t, mergedLine.TaxConfig.TaxCode)
	require.Equal(t, newTaxCodeID, *mergedLine.TaxConfig.TaxCodeID)
}

func TestMergeGatheringLineFromInvoiceLineReplaceUpdateAcceptsRateCardOnlyPrice(t *testing.T) {
	period := invoiceLineTestPeriod()
	line := gatheringInvoiceLineForMergeTest(t, period)

	mergedLine, err := mergeGatheringLineFromInvoiceLineReplaceUpdate(line, invoiceLineReplaceUpdateForMergeTest(t, period, line.FeatureKey, "2", func(update *api.InvoiceLineReplaceUpdate) {
		update.Price = nil
		update.FeatureKey = nil
		update.TaxConfig = nil
	}))
	require.NoError(t, err)
	require.True(t, gatheringLinePrice(t, mergedLine).Equal(productCatalogPriceForTest(t, "2")))
	require.Equal(t, line.FeatureKey, mergedLine.FeatureKey)
}

func TestMergeStandardInvoiceLinesFromAPITombstonesOmittedLines(t *testing.T) {
	period := invoiceLineTestPeriod()
	keptLine := standardInvoiceLineForMergeTest(t, period)
	keptLine.ID = "kept-line-id"
	deletedLine := standardInvoiceLineForMergeTest(t, period)
	deletedLine.ID = "deleted-line-id"
	invoice := &billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-id",
			Currency:  "USD",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{keptLine, deletedLine}),
	}

	mergedLines, err := (&handler{}).mergeStandardInvoiceLinesFromAPI(t.Context(), invoice, []api.InvoiceLineReplaceUpdate{
		invoiceLineReplaceUpdateForMergeTest(t, period, keptLine.UsageBased.FeatureKey, "1", func(update *api.InvoiceLineReplaceUpdate) {
			update.Id = lo.ToPtr(keptLine.ID)
		}),
	})
	require.NoError(t, err)

	lines := mergedLines.OrEmpty()
	require.Len(t, lines, 2)

	kept := requireStandardLineByID(t, lines, keptLine.ID)
	require.Nil(t, kept.DeletedAt)

	deleted := requireStandardLineByID(t, lines, deletedLine.ID)
	require.NotNil(t, deleted.DeletedAt)
}

func invoiceLineTestPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func standardInvoiceLineForMergeTest(t *testing.T, period timeutil.ClosedPeriod) *billing.StandardLine {
	t.Helper()

	featureKey := "feature-key"
	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-id",
				Name:      "line",
				CreatedAt: period.From,
				UpdatedAt: period.From,
			}),
			InvoiceID: "invoice-id",
			Currency:  "USD",
			Period:    period,
			InvoiceAt: period.To,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      productCatalogPriceForTest(t, "1"),
			FeatureKey: featureKey,
		},
	}
}

func gatheringInvoiceLineForMergeTest(t *testing.T, period timeutil.ClosedPeriod) billing.GatheringLine {
	t.Helper()

	featureKey := "feature-key"
	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-id",
				Name:      "line",
				CreatedAt: period.From,
				UpdatedAt: period.From,
			}),
			InvoiceID:     "invoice-id",
			Currency:      "USD",
			ServicePeriod: period,
			InvoiceAt:     period.To,
			Price:         *productCatalogPriceForTest(t, "1"),
			FeatureKey:    featureKey,
		},
	}
}

func invoiceLineReplaceUpdateForMergeTest(t *testing.T, period timeutil.ClosedPeriod, featureKey string, amount string, edits ...func(*api.InvoiceLineReplaceUpdate)) api.InvoiceLineReplaceUpdate {
	t.Helper()

	price := apiPriceForMergeTest(t, amount)
	out := api.InvoiceLineReplaceUpdate{
		Name:       "line",
		Period:     api.Period(period),
		InvoiceAt:  period.To,
		FeatureKey: &featureKey,
		Price:      &price,
		RateCard: &api.InvoiceUsageBasedRateCard{
			FeatureKey: &featureKey,
			Price:      &price,
		},
	}

	for _, edit := range edits {
		edit(&out)
	}

	return out
}

func apiPriceForMergeTest(t *testing.T, amount string) api.RateCardUsageBasedPrice {
	t.Helper()

	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
		Amount: amount,
	}))

	return price
}

func productCatalogPriceForTest(t *testing.T, amount string) *productcatalog.Price {
	t.Helper()

	price, err := productcataloghttp.AsPrice(apiPriceForMergeTest(t, amount))
	require.NoError(t, err)

	return price
}

func requireStandardLineByID(t *testing.T, lines billing.StandardLines, id string) *billing.StandardLine {
	t.Helper()

	line, ok := lo.Find(lines, func(line *billing.StandardLine) bool {
		return line.ID == id
	})
	require.True(t, ok, "line %q", id)

	return line
}

func gatheringLinePrice(t *testing.T, line billing.GatheringLine) *productcatalog.Price {
	t.Helper()

	return line.Price.Clone()
}
