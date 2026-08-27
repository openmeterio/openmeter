package billinginvoices

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mergeTestPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func standardLineForMergeTest(t *testing.T, id string, period timeutil.ClosedPeriod) *billing.StandardLine {
	t.Helper()

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        id,
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
			Price:      productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(1)}),
			FeatureKey: "feature-key",
		},
	}
}

func apiLineForMergeTest(t *testing.T, period timeutil.ClosedPeriod, id *string) api.UpdateInvoiceLine {
	t.Helper()

	price := api.UpdatePrice{}
	require.NoError(t, price.FromUpdatePriceFlat(api.UpdatePriceFlat{
		Amount: "1",
		Type:   api.UpdatePriceFlatTypeFlat,
	}))

	standardLine := api.UpdateInvoiceStandardLine{
		Id:   id,
		Name: "line",
		ServicePeriod: api.UpdateClosedPeriod{
			From: period.From,
			To:   period.To,
		},
		RateCard: api.UpdateInvoiceLineRateCard{
			Price:      price,
			FeatureKey: lo.ToPtr("feature-key"),
		},
	}

	var out api.UpdateInvoiceLine
	require.NoError(t, out.FromUpdateInvoiceStandardLine(standardLine))

	return out
}

func TestMapRateCardSurfacesUnitConfigSnapshot(t *testing.T) {
	period := mergeTestPeriod()

	// given: a standard line whose usage-based config carries a unit_config snapshot
	// (a divide-by-1000, ceiling-rounded, "GB"-labeled conversion).
	line := standardLineForMergeTest(t, "line-id", period)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(5)})
	line.UsageBased.UnitConfig = &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	// when: mapping the line's rate card to the v3 API type.
	rc, err := mapRateCard(line)
	require.NoError(t, err)

	// then: the snapshot surfaces field-for-field on the API rate card.
	require.NotNil(t, rc.UnitConfig)
	require.Equal(t, api.BillingUnitConfigOperationDivide, rc.UnitConfig.Operation)
	require.Equal(t, "1000", rc.UnitConfig.ConversionFactor)
	require.Equal(t, lo.ToPtr(api.BillingUnitConfigRoundingModeCeiling), rc.UnitConfig.Rounding)
	require.Equal(t, lo.ToPtr(0), rc.UnitConfig.Precision)
	require.Equal(t, lo.ToPtr("GB"), rc.UnitConfig.DisplayUnit)
}

func TestMapRateCardOmitsUnitConfigWhenAbsent(t *testing.T) {
	period := mergeTestPeriod()

	// given: a standard line with no unit_config snapshot (the common case).
	line := standardLineForMergeTest(t, "line-id", period)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(5)})

	// when: mapping the rate card.
	rc, err := mapRateCard(line)
	require.NoError(t, err)

	// then: the API field is nil (omitted) — identity with today's output.
	require.Nil(t, rc.UnitConfig)
}

func TestMergeStandardLineFromAPIPreservesDiscountCorrelationID(t *testing.T) {
	period := mergeTestPeriod()
	line := standardLineForMergeTest(t, "line-id", period)
	line.RateCardDiscounts = billing.Discounts{
		Percentage: &billing.PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
			CorrelationID:      "existing-percentage",
		},
	}

	apiLine, err := apiLineForMergeTest(t, period, lo.ToPtr(line.ID)).AsUpdateInvoiceStandardLine()
	require.NoError(t, err)
	apiLine.RateCard.Discounts = &api.UpdateDiscounts{Percentage: lo.ToPtr(float32(20))}

	mergedLine, err := mergeStandardLineFromAPI(line, apiLine)
	require.NoError(t, err)
	require.Equal(t, models.NewPercentage(20), mergedLine.RateCardDiscounts.Percentage.Percentage)
	require.Equal(t, "existing-percentage", mergedLine.RateCardDiscounts.Percentage.CorrelationID)
}

func TestMapUsageQuantityDetailSurfacesStoredQuantities(t *testing.T) {
	period := mergeTestPeriod()

	// given: a converted line — raw metered 1300 bytes, billed 2 GB after a
	// divide-by-1000, ceiling conversion (the engine already stored both).
	line := standardLineForMergeTest(t, "line-id", period)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(5)})
	line.UsageBased.MeteredQuantity = lo.ToPtr(decimal.NewFromInt(1300))
	line.UsageBased.Quantity = lo.ToPtr(decimal.NewFromInt(2))
	line.UsageBased.UnitConfig = &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	// when: building the audit trail.
	detail := toAPIUsageQuantityDetail(line)

	// then: raw and invoiced surface the stored values verbatim (raw != invoiced,
	// so the conversion is visible), with the display unit.
	require.NotNil(t, detail)
	require.Equal(t, "1300", detail.RawQuantity)
	require.Equal(t, "2", detail.InvoicedQuantity)
	require.Equal(t, lo.ToPtr("GB"), detail.DisplayUnit)
}

func TestMapUsageQuantityDetailOmitsWhenNoUnitConfig(t *testing.T) {
	period := mergeTestPeriod()

	// given: a usage-based line with quantities but no unit_config (the common case).
	line := standardLineForMergeTest(t, "line-id", period)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(5)})
	line.UsageBased.MeteredQuantity = lo.ToPtr(decimal.NewFromInt(10))
	line.UsageBased.Quantity = lo.ToPtr(decimal.NewFromInt(10))

	// when/then: no conversion happened, so nothing is surfaced — identity with
	// today's output for non-unit_config invoices.
	require.Nil(t, toAPIUsageQuantityDetail(line))
}

func TestMapUsageQuantityDetailOmitsWhenQuantitiesNil(t *testing.T) {
	period := mergeTestPeriod()

	// given: a unit_config line that never rated, so its quantities are still nil.
	line := standardLineForMergeTest(t, "line-id", period)
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(5)})
	line.UsageBased.UnitConfig = &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	// when/then: missing either endpoint yields no trail (and never panics).
	require.Nil(t, toAPIUsageQuantityDetail(line))
}

func TestMergeStandardInvoiceLinesFromAPITombstonesOmittedLines(t *testing.T) {
	period := mergeTestPeriod()
	keptLine := standardLineForMergeTest(t, "kept-line-id", period)
	deletedLine := standardLineForMergeTest(t, "deleted-line-id", period)

	inv := &billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-id",
			Currency:  "USD",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{keptLine, deletedLine}),
	}

	lines := []api.UpdateInvoiceLine{
		apiLineForMergeTest(t, period, lo.ToPtr("kept-line-id")),
		apiLineForMergeTest(t, period, nil),
	}

	merged, err := mergeStandardInvoiceLinesFromAPI(inv, &lines)
	require.NoError(t, err)

	all := merged.OrEmpty()
	require.Len(t, all, 3)

	kept, ok := lo.Find(all, func(l *billing.StandardLine) bool { return l.ID == "kept-line-id" })
	require.True(t, ok)
	require.Nil(t, kept.DeletedAt)

	deleted, ok := lo.Find(all, func(l *billing.StandardLine) bool { return l.ID == "deleted-line-id" })
	require.True(t, ok)
	require.NotNil(t, deleted.DeletedAt)

	newLines := lo.Filter(all, func(l *billing.StandardLine, _ int) bool {
		return l.ID != "kept-line-id" && l.ID != "deleted-line-id"
	})
	require.Len(t, newLines, 1)
}

func TestMergeStandardInvoiceLinesFromAPINilLeavesLinesUnchanged(t *testing.T) {
	period := mergeTestPeriod()
	existing := standardLineForMergeTest(t, "line-id", period)

	inv := &billing.StandardInvoice{
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{existing}),
	}

	merged, err := mergeStandardInvoiceLinesFromAPI(inv, nil)
	require.NoError(t, err)
	require.Equal(t, inv.Lines, merged)
}

func TestMergeInvoiceCustomerFromAPIPreservesImmutableFields(t *testing.T) {
	existing := billing.InvoiceCustomer{
		CustomerID: "cust-id",
		Key:        lo.ToPtr("cust-key"),
		Name:       "Old Name",
	}

	updated := api.UpdateInvoiceCustomer{
		Id:   "attacker-id",
		Key:  lo.ToPtr("attacker-key"),
		Name: "New Name",
		BillingAddress: &api.UpdateAddress{
			City: lo.ToPtr("Ghent"),
		},
	}

	merged := mergeInvoiceCustomerFromAPI(existing, updated)

	require.Equal(t, "cust-id", merged.CustomerID)
	require.Equal(t, lo.ToPtr("cust-key"), merged.Key)
	require.Equal(t, "New Name", merged.Name)
	require.NotNil(t, merged.BillingAddress)
	require.Equal(t, "Ghent", lo.FromPtr(merged.BillingAddress.City))
}

func TestMergeInvoiceSupplierFromAPI(t *testing.T) {
	existing := billing.SupplierContact{
		ID:   "supplier-id",
		Name: "Old Supplier",
	}

	updated := api.UpdateSupplier{
		Id:   lo.ToPtr("attacker-id"),
		Name: lo.ToPtr("New Supplier"),
		TaxId: &api.UpdateBillingPartyTaxIdentity{
			Code: lo.ToPtr("TAX-1"),
		},
	}

	merged := mergeInvoiceSupplierFromAPI(existing, updated)

	require.Equal(t, "supplier-id", merged.ID)
	require.Equal(t, "New Supplier", merged.Name)
	require.Equal(t, lo.ToPtr("TAX-1"), merged.TaxCode)
}
