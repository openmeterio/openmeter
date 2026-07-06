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
