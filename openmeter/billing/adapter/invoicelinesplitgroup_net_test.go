package billingadapter

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

func TestSplitLineHistoryMappingPreservesDiscounts(t *testing.T) {
	line := &db.BillingInvoiceLine{
		ID:             "line-1",
		Namespace:      "ns",
		Type:           billing.InvoiceLineAdapterTypeUsageBased,
		Amount:         alpacadecimal.NewFromInt(100),
		DiscountsTotal: alpacadecimal.NewFromInt(40),
		Edges: db.BillingInvoiceLineEdges{
			BillingInvoice: &db.BillingInvoice{
				ID:        "invoice-1",
				Namespace: "ns",
				Status:    billing.StandardInvoiceStatusPaid,
			},
		},
	}

	history, err := (&adapter{}).mapSplitLineHierarchyLinesFromDB([]*db.BillingInvoiceLine{line})
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "40", history[0].DiscountsTotal.String())

	hierarchy := billing.SplitLineHierarchy{Lines: history}
	net, err := hierarchy.SumNetAmount(billing.SumNetAmountInput{})
	require.NoError(t, err)
	require.Equal(t, "60", net.String())
}
