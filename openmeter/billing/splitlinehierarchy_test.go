package billing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSplitLineHierarchyPreviouslyBilledAmountExcludesGatheringLines(t *testing.T) {
	// Given an issued line and a gathering line that both have stored amounts.
	hierarchy := SplitLineHierarchy{
		Lines: []SplitLineHierarchyLine{
			{
				Invoice: SplitLineHierarchyInvoice{
					Type:   InvoiceTypeStandard,
					Status: StandardInvoiceStatusPaid,
				},
				Amount: alpacadecimal.NewFromInt(40),
			},
			{
				Invoice: SplitLineHierarchyInvoice{
					Type:   InvoiceTypeGathering,
					Status: StandardInvoiceStatusGathering,
				},
				Amount: alpacadecimal.NewFromInt(60),
			},
		},
	}

	// When previously billed amounts are summed, only the issued line contributes.
	amount, err := hierarchy.SumNetAmount(SumNetAmountInput{})
	require.NoError(t, err)
	require.Equal(t, float64(40), amount.InexactFloat64())
}

func TestSplitLineHierarchyPreviouslyBilledAmountNetsDiscounts(t *testing.T) {
	hierarchy := SplitLineHierarchy{
		Lines: []SplitLineHierarchyLine{{
			Invoice:        SplitLineHierarchyInvoice{Type: InvoiceTypeStandard, Status: StandardInvoiceStatusPaid},
			Amount:         alpacadecimal.NewFromInt(100),
			DiscountsTotal: alpacadecimal.NewFromInt(40),
		}},
	}

	amount, err := hierarchy.SumNetAmount(SumNetAmountInput{})
	require.NoError(t, err)
	require.Equal(t, float64(60), amount.InexactFloat64())
}

func TestSplitLineHierarchyCloneDoesNotShareMutableFields(t *testing.T) {
	// Given a hierarchy with subscriptions, deletion timestamps, and nested annotations.
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	source := SplitLineHierarchy{
		Group: SplitLineGroup{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "group"},
			SplitLineGroupMutableFields: SplitLineGroupMutableFields{
				Description: lo.ToPtr("group description"),
				Metadata:    models.Metadata{"key": "value"},
			},
			Subscription: &SubscriptionReference{ItemID: "group item"},
		},
		Lines: []SplitLineHierarchyLine{
			{
				LineID: LineID{Namespace: "ns", ID: "line"},
				Invoice: SplitLineHierarchyInvoice{
					InvoiceID: InvoiceID{Namespace: "ns", ID: "invoice"},
					Type:      InvoiceTypeStandard,
					Status:    StandardInvoiceStatusDeleted,
					DeletedAt: &now,
				},
				DeletedAt:    &now,
				Subscription: &SubscriptionReference{ItemID: "child item"},
				Annotations:  models.Annotations{"nested": map[string]interface{}{"value": "original"}},
			},
		},
	}
	before, err := json.Marshal(source)
	require.NoError(t, err)

	// When subscriptions, timestamps, metadata, and annotations on the clone are changed.
	clone, err := source.Clone()
	require.NoError(t, err)
	clone.Group.Subscription.ItemID = "changed"
	clone.Group.Metadata["key"] = "changed"
	*clone.Group.Description = "changed"
	clone.Lines[0].Subscription.ItemID = "changed"
	*clone.Lines[0].DeletedAt = now.Add(time.Hour)
	*clone.Lines[0].Invoice.DeletedAt = now.Add(2 * time.Hour)
	clone.Lines[0].Annotations["nested"].(map[string]interface{})["value"] = "changed"

	// Then the original hierarchy remains unchanged.
	after, err := json.Marshal(source)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
}
