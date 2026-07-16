package billingadapter

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGatheringLineMappings(t *testing.T) {
	now := time.Date(2026, 7, 16, 6, 0, 0, 0, time.UTC)
	periodEnd := now.Add(time.Hour)
	featureKey := "feature"
	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	legacyRow := &db.BillingInvoiceLine{
		ID:          "line-id",
		Namespace:   "namespace",
		CreatedAt:   now,
		UpdatedAt:   now,
		Name:        "line",
		Currency:    currencyx.Code("USD"),
		PeriodStart: now,
		PeriodEnd:   periodEnd,
		InvoiceID:   "invoice-id",
		InvoiceAt:   periodEnd,
		ManagedBy:   billing.SystemManagedLine,
		Engine:      billing.LineEngineTypeInvoice,
		Type:        billing.InvoiceLineAdapterTypeUsageBased,
		Edges: db.BillingInvoiceLineEdges{
			UsageBasedLine: &db.BillingInvoiceUsageBasedLineConfig{
				ID:         "usage-config-id",
				FeatureKey: &featureKey,
				Price:      price,
			},
		},
	}

	dedicatedRow := &db.BillingGatheringInvoiceLine{
		ID:                 legacyRow.ID,
		Namespace:          legacyRow.Namespace,
		CreatedAt:          legacyRow.CreatedAt,
		UpdatedAt:          legacyRow.UpdatedAt,
		Name:               legacyRow.Name,
		Currency:           legacyRow.Currency,
		ServicePeriodStart: legacyRow.PeriodStart,
		ServicePeriodEnd:   legacyRow.PeriodEnd,
		PriceType:          price.Type(),
		FeatureKey:         &featureKey,
		Price:              price,
		InvoiceID:          legacyRow.InvoiceID,
		InvoiceAt:          legacyRow.InvoiceAt,
		ManagedBy:          legacyRow.ManagedBy,
		Engine:             legacyRow.Engine,
	}

	a := &adapter{}
	legacyLine, err := a.fromDBBillingInvoiceLine(legacyRow)
	require.NoError(t, err)
	require.Equal(t, billing.GatheringLineTableInvoiceLines, legacyLine.DBState.Source)

	dedicatedLine, err := a.fromDBBillingGatheringInvoiceLine(dedicatedRow)
	require.NoError(t, err)
	require.Equal(t, billing.GatheringLineTableGatheringInvoiceLines, dedicatedLine.DBState.Source)

	legacyLineWithoutDBState, err := legacyLine.WithoutDBState()
	require.NoError(t, err)
	legacyLineWithoutDBState.UBPConfigID = ""

	dedicatedLineWithoutDBState, err := dedicatedLine.WithoutDBState()
	require.NoError(t, err)

	require.Equal(t, legacyLineWithoutDBState, dedicatedLineWithoutDBState)
}

func TestMergeGatheringLines(t *testing.T) {
	legacyLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{ID: "legacy-line"}),
		},
	}
	require.NoError(t, legacyLine.SaveDBSnapshot(billing.GatheringLineTableInvoiceLines))

	dedicatedLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{ID: "dedicated-line"}),
		},
	}
	require.NoError(t, dedicatedLine.SaveDBSnapshot(billing.GatheringLineTableGatheringInvoiceLines))

	merged, err := mergeGatheringLines(
		billing.GatheringLines{legacyLine},
		billing.GatheringLines{dedicatedLine},
	)
	require.NoError(t, err)
	require.Len(t, merged, 2)

	duplicate := dedicatedLine
	duplicate.ID = legacyLine.ID
	duplicate.DBState.Line.ID = legacyLine.ID

	_, err = mergeGatheringLines(
		billing.GatheringLines{legacyLine},
		billing.GatheringLines{duplicate},
	)
	require.ErrorContains(t, err, "gathering line exists in multiple tables")
}

func TestDiffGatheringInvoiceLinesRejectsDedicatedTableRows(t *testing.T) {
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{ID: "line-id"}),
		},
	}
	require.NoError(t, line.SaveDBSnapshot(billing.GatheringLineTableGatheringInvoiceLines))

	_, err := diffGatheringInvoiceLines(billing.GatheringLines{line})
	require.ErrorContains(t, err, "writing gathering lines from table billing_gathering_invoice_lines is not supported")
}
