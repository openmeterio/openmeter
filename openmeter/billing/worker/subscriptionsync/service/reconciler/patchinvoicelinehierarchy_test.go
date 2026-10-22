package reconciler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type hierarchyPatchTestItem struct {
	persistedstate.Item
	hierarchy *billing.SplitLineHierarchy
}

func (i hierarchyPatchTestItem) GetSplitLineHierarchy() *billing.SplitLineHierarchy {
	return i.hierarchy
}

func newHierarchyPatchTestHistory() *billing.SplitLineHierarchy {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	boundary := from.AddDate(0, 0, 14)
	to := from.AddDate(0, 1, 0)

	return &billing.SplitLineHierarchy{
		Group: billing.SplitLineGroup{
			NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "group"},
			SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
				Name:          "group",
				ServicePeriod: timeutil.ClosedPeriod{From: from, To: to},
			},
		},
		Lines: []billing.SplitLineHierarchyLine{
			{
				LineID: billing.LineID{Namespace: "namespace", ID: "issued"},
				Invoice: billing.SplitLineHierarchyInvoice{
					InvoiceID: billing.InvoiceID{Namespace: "namespace", ID: "issued-invoice"},
					Type:      billing.InvoiceTypeStandard,
					Status:    billing.StandardInvoiceStatusPaid,
				},
				ManagedBy:     billing.SubscriptionManagedLine,
				ServicePeriod: timeutil.ClosedPeriod{From: from, To: boundary},
			},
			{
				LineID: billing.LineID{Namespace: "namespace", ID: "gathering"},
				Invoice: billing.SplitLineHierarchyInvoice{
					InvoiceID: billing.InvoiceID{Namespace: "namespace", ID: "gathering-invoice"},
					Type:      billing.InvoiceTypeGathering,
					Status:    billing.StandardInvoiceStatusGathering,
				},
				ManagedBy:     billing.SubscriptionManagedLine,
				ServicePeriod: timeutil.ClosedPeriod{From: boundary, To: to},
				InvoiceAt:     to,
			},
		},
	}
}

func TestHierarchyShrinkRestoresSubscriptionDeletedChild(t *testing.T) {
	// Given an issued October 1-15 line and a deleted, subscription-managed gathering line through November 1.
	hierarchy := newHierarchyPatchTestHistory()
	deletedAt := hierarchy.Group.ServicePeriod.From
	hierarchy.Lines[1].DeletedAt = &deletedAt
	before, err := json.Marshal(hierarchy)
	require.NoError(t, err)
	target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	target.ServicePeriod = hierarchy.Group.ServicePeriod
	target.ServicePeriod.To = time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	target.FullServicePeriod = target.ServicePeriod
	target.BillingPeriod = target.ServicePeriod

	// When cancellation moves the service-period end to October 20.
	collection := newLineHierarchyPatchCollection(0)
	require.NoError(t, collection.AddShrink(target.UniqueID, hierarchyPatchTestItem{hierarchy: hierarchy}, target))
	require.Len(t, collection.Patches(), 1)
	patches, err := collection.Patches()[0].GetInvoicePatches()
	require.NoError(t, err)

	// Then the patch shortens and restores the gathering line, leaving the issued line alone.
	require.Len(t, patches, 2)
	update, err := patches[0].AsUpdateLinePatch()
	require.NoError(t, err)
	require.Equal(t, hierarchy.Lines[1].LineID, update.Line)
	require.Equal(t, hierarchy.Lines[1].Invoice.ID, update.InvoiceID)
	require.Equal(t, timeutil.ClosedPeriod{From: hierarchy.Lines[1].ServicePeriod.From, To: target.ServicePeriod.To}, update.ServicePeriod.MustGet())
	require.Equal(t, target.ServicePeriod.To, update.InvoiceAt.MustGet())
	require.True(t, update.FlatFeePerUnitAmount.IsAbsent())
	require.True(t, update.DeletedAt.IsPresent())
	require.Nil(t, update.DeletedAt.MustGet())
	groupUpdate, err := patches[1].AsUpdateSplitLineGroupPatch()
	require.NoError(t, err)
	require.Equal(t, target.ServicePeriod, groupUpdate.TargetState.ServicePeriod)
	after, err := json.Marshal(hierarchy)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
}

func TestHierarchyExtensionSelectsLastChildWhenPeriodsEndTogether(t *testing.T) {
	for _, tc := range []struct {
		name           string
		setup          func(*billing.SplitLineHierarchy)
		expectedLineID string
		restoresLine   bool
	}{
		{
			name:           "deleted child last",
			setup:          func(*billing.SplitLineHierarchy) {},
			expectedLineID: "historical",
			restoresLine:   true,
		},
		{
			name: "live child last",
			setup: func(hierarchy *billing.SplitLineHierarchy) {
				hierarchy.Lines[1], hierarchy.Lines[2] = hierarchy.Lines[2], hierarchy.Lines[1]
			},
			expectedLineID: "gathering",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given live and deleted gathering lines that both end on November 1.
			hierarchy := newHierarchyPatchTestHistory()
			historical, err := hierarchy.Lines[1].Clone()
			require.NoError(t, err)
			historical.ID = "historical"
			historical.DeletedAt = &hierarchy.Group.ServicePeriod.From
			hierarchy.Lines = append(hierarchy.Lines, historical)
			tc.setup(hierarchy)

			target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
			target.ServicePeriod = hierarchy.Group.ServicePeriod
			target.ServicePeriod.To = target.ServicePeriod.To.AddDate(0, 0, 10)
			target.FullServicePeriod = target.ServicePeriod
			target.BillingPeriod = target.ServicePeriod
			before, err := json.Marshal(hierarchy)
			require.NoError(t, err)

			// When service continues through November 11.
			collection := newLineHierarchyPatchCollection(0)
			require.NoError(t, collection.AddExtend(hierarchyPatchTestItem{hierarchy: hierarchy}, target))
			patches, err := collection.Patches()[0].GetInvoicePatches()
			require.NoError(t, err)

			// Then the last line in stored order is extended and restored if it was deleted.
			require.Len(t, patches, 2)
			update, err := patches[0].AsUpdateLinePatch()
			require.NoError(t, err)
			require.Equal(t, billing.LineID{Namespace: "namespace", ID: tc.expectedLineID}, update.Line)
			require.Equal(t, target.ServicePeriod.To, update.ServicePeriod.MustGet().To)
			require.Equal(t, target.ServicePeriod.To, update.InvoiceAt.MustGet())
			require.Equal(t, tc.restoresLine, update.DeletedAt.IsPresent())
			if update.DeletedAt.IsPresent() {
				require.Nil(t, update.DeletedAt.MustGet())
			}

			after, err := json.Marshal(hierarchy)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}

func TestHierarchyIgnoreOnDeletedChildSuppressesPatches(t *testing.T) {
	// Given a deleted gathering line marked to be ignored by subscription sync.
	hierarchy := newHierarchyPatchTestHistory()
	historical, err := hierarchy.Lines[1].Clone()
	require.NoError(t, err)
	historical.ID = "historical"
	historical.DeletedAt = &hierarchy.Group.ServicePeriod.From
	historical.Annotations = models.Annotations{billing.AnnotationSubscriptionSyncIgnore: true}
	hierarchy.Lines = append(hierarchy.Lines, historical)
	item := hierarchyPatchTestItem{hierarchy: hierarchy}

	// When sync attempts to delete, shorten, or extend the group, the annotation blocks all three operations.
	ignored := newLineHierarchyPatchCollection(0)
	require.NoError(t, ignored.AddDelete("unique", item))
	target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())
	target.ServicePeriod = hierarchy.Group.ServicePeriod
	target.ServicePeriod.To = time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	target.FullServicePeriod = target.ServicePeriod
	target.BillingPeriod = target.ServicePeriod
	require.NoError(t, ignored.AddShrink("unique", item, target))
	target.ServicePeriod.To = time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	target.FullServicePeriod = target.ServicePeriod
	target.BillingPeriod = target.ServicePeriod
	require.NoError(t, ignored.AddExtend(item, target))
	require.True(t, ignored.IsEmpty())
}
