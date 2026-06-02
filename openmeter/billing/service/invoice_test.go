package billingservice

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCollectNewlyDeletedStandardLines(t *testing.T) {
	before := billing.NewStandardInvoiceLines([]*billing.StandardLine{
		newStandardLineForLineEngineTest("newly-deleted", billing.LineEngineTypeInvoice, false),
		newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, true),
		newStandardLineForLineEngineTest("unchanged", billing.LineEngineTypeInvoice, false),
	})

	after := billing.NewStandardInvoiceLines([]*billing.StandardLine{
		newStandardLineForLineEngineTest("newly-deleted", billing.LineEngineTypeInvoice, true),
		newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, true),
		newStandardLineForLineEngineTest("unchanged", billing.LineEngineTypeInvoice, false),
		newStandardLineForLineEngineTest("new-deleted-line", billing.LineEngineTypeInvoice, true),
	})

	deletedLines, err := collectNewlyDeletedStandardLines(before, after)
	require.NoError(t, err)
	require.Len(t, deletedLines, 1)
	require.Equal(t, "newly-deleted", deletedLines[0].ID)
	require.NotNil(t, deletedLines[0].DeletedAt)
}

func newStandardLineForLineEngineTest(id string, engine billing.LineEngineType, deleted bool) *billing.StandardLine {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var deletedAt *time.Time
	if deleted {
		deletedAt = lo.ToPtr(now.Add(time.Hour))
	}

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel: models.ManagedModel{
					DeletedAt: deletedAt,
				},
				ID:   id,
				Name: id,
			},
			Engine:    engine,
			InvoiceID: "invoice-1",
			Currency:  currencyx.Code("USD"),
			ManagedBy: billing.ManuallyManagedLine,
			Period: timeutil.ClosedPeriod{
				From: now,
				To:   now.Add(time.Hour),
			},
			InvoiceAt: now.Add(time.Hour),
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(100),
			}),
		},
	}
}
