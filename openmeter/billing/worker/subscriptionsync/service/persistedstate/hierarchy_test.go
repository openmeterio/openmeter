package persistedstate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNormalizePersistedHierarchyPeriodsAndInvoiceAt(t *testing.T) {
	// Given stored service periods and InvoiceAt values with fractional seconds.
	start := time.Date(2026, 10, 1, 0, 0, 0, 123456789, time.UTC)
	end := start.AddDate(0, 1, 0)
	period := timeutil.ClosedPeriod{From: start, To: end}
	hierarchy := splitlinegroup.SplitLineHierarchy{
		Group: splitlinegroup.SplitLineGroup{
			SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{ServicePeriod: period},
		},
		Lines: []splitlinegroup.SplitLineHierarchyLine{
			{
				LineID:        billing.LineID{Namespace: "ns", ID: "line"},
				Invoice:       splitlinegroup.SplitLineHierarchyInvoice{Type: billing.InvoiceTypeGathering},
				ServicePeriod: period,
				InvoiceAt:     end.Add(time.Second),
			},
			{
				LineID:        billing.LineID{Namespace: "ns", ID: "standard"},
				Invoice:       splitlinegroup.SplitLineHierarchyInvoice{Type: billing.InvoiceTypeStandard},
				ServicePeriod: period,
				InvoiceAt:     end.Add(time.Second),
			},
		},
	}

	// When the stored timestamps are normalized for subscription sync.
	normalized, err := normalizePersistedSplitLineHierarchy(hierarchy)
	require.NoError(t, err)

	// Then service periods and gathering InvoiceAt lose fractional seconds; standard InvoiceAt is preserved.
	expectedPeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
	require.Equal(t, expectedPeriod, normalized.Group.ServicePeriod)
	require.Equal(t, expectedPeriod, normalized.Lines[0].ServicePeriod)
	require.Equal(t, time.Date(2026, 11, 1, 0, 0, 1, 0, time.UTC), normalized.Lines[0].InvoiceAt)
	require.Equal(t, expectedPeriod, normalized.Lines[1].ServicePeriod)
	require.Equal(t, end.Add(time.Second), normalized.Lines[1].InvoiceAt)
	require.Equal(t, period, hierarchy.Group.ServicePeriod)
	for _, child := range hierarchy.Lines {
		require.Equal(t, period, child.ServicePeriod)
		require.Equal(t, end.Add(time.Second), child.InvoiceAt)
	}
}

func TestHierarchyOwnershipUsesFirstLiveChildAtGroupEnd(t *testing.T) {
	// Given three children ending on November 1: a deleted line, a subscription-managed line, and a manual line.
	end := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	hierarchy := &splitlinegroup.SplitLineHierarchy{
		Group: splitlinegroup.SplitLineGroup{
			SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{ServicePeriod: timeutil.ClosedPeriod{To: end}},
		},
		Lines: []splitlinegroup.SplitLineHierarchyLine{
			{LineID: billing.LineID{ID: "deleted"}, ServicePeriod: timeutil.ClosedPeriod{To: end}, DeletedAt: &end},
			{
				LineID:        billing.LineID{ID: "first"},
				Invoice:       splitlinegroup.SplitLineHierarchyInvoice{Type: billing.InvoiceTypeStandard, Status: billing.StandardInvoiceStatusDeleted},
				ServicePeriod: timeutil.ClosedPeriod{To: end},
				ManagedBy:     billing.SubscriptionManagedLine,
				Annotations:   models.Annotations{billing.AnnotationSubscriptionSyncIgnore: true},
			},
			{
				LineID:        billing.LineID{ID: "second"},
				ServicePeriod: timeutil.ClosedPeriod{To: end},
				ManagedBy:     billing.ManuallyManagedLine,
			},
		},
	}
	item, err := newPersistedSplitLineHierarchy(hierarchy)
	require.NoError(t, err)

	// When sync reads ownership and annotations, it uses the first undeleted line, even on a deleted invoice.
	require.Equal(t, "first", item.getLastLineForAnnotations().ID)
	require.True(t, item.IsSubscriptionManaged())
	require.True(t, item.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncIgnore))

	// Then putting the manual line first changes the group's ownership and ignore flag.
	hierarchy.Lines[1], hierarchy.Lines[2] = hierarchy.Lines[2], hierarchy.Lines[1]
	require.Equal(t, "second", item.getLastLineForAnnotations().ID)
	require.False(t, item.IsSubscriptionManaged())
	require.False(t, item.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncIgnore))
	hierarchy.Lines = nil
	require.False(t, item.IsSubscriptionManaged())
	require.False(t, item.HasLastLineAnnotation(billing.AnnotationSubscriptionSyncIgnore))
}
