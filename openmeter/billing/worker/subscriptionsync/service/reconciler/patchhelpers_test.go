package reconciler

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetPatchesForUpdateUsageBasedLineEmitsOnlyIntendedFields(t *testing.T) {
	t.Parallel()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	deletedAt := period.From.Add(time.Hour)
	existing := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-1",
				Name:      "line",
			}),
			ManagedBy:     billing.SubscriptionManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			InvoiceID:     "invoice-1",
			Currency:      currencyx.FiatCode("USD"),
			ServicePeriod: period,
			InvoiceAt:     period.To,
			FeatureKey:    "feature-key",
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
	existing.DeletedAt = &deletedAt
	expected := existing
	expected.DeletedAt = nil
	expected.ServicePeriod.To = expected.ServicePeriod.To.AddDate(0, 1, 0)
	expected.InvoiceAt = expected.ServicePeriod.To
	invoices := persistedstate.Invoices{
		"invoice-1": billing.GatheringInvoice{
			GatheringInvoiceBase: billing.GatheringInvoiceBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: "ns",
					ID:        "invoice-1",
					Name:      "invoice",
				}),
			},
		}.AsInvoice(),
	}

	patch, err := getPatchesForUpdateUsageBasedLine(existing.AsGenericLine(), expected, invoices)
	require.NoError(t, err)
	require.NotNil(t, patch)
	update, err := patch.AsUpdateLinePatch()
	require.NoError(t, err)

	servicePeriod, ok := update.ServicePeriod.Get()
	require.True(t, ok)
	require.Equal(t, expected.ServicePeriod, servicePeriod)
	invoiceAt, ok := update.InvoiceAt.Get()
	require.True(t, ok)
	require.Equal(t, expected.InvoiceAt, invoiceAt)
	deletedAtUpdate, ok := update.DeletedAt.Get()
	require.True(t, ok)
	require.Nil(t, deletedAtUpdate)
	require.True(t, update.FlatFeePerUnitAmount.IsAbsent())
	require.NotNil(t, existing.DeletedAt)
	require.Equal(t, period, existing.ServicePeriod)
}
