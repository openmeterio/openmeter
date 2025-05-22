package billingadapter

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestSortLines(t *testing.T) {
	lines := []*billing.Line{
		{
			LineBase: billing.LineBase{
				Type: billing.InvoiceLineTypeUsageBased,
				Name: "usage-based-line",
				Period: billing.Period{
					Start: time.Now().Add(time.Hour * 24),
				},
				Description: lo.ToPtr("index=1"),
			},
			Children: billing.NewLineChildren([]*billing.Line{
				{
					LineBase: billing.LineBase{
						ID:          "child-2",
						Type:        billing.InvoiceLineTypeFee,
						Description: lo.ToPtr("index=1.1"),
					},
					FlatFee: &billing.FlatFeeLine{
						Index: lo.ToPtr(1),
					},
				},
				{
					LineBase: billing.LineBase{
						ID:          "child-1",
						Type:        billing.InvoiceLineTypeFee,
						Description: lo.ToPtr("index=1.0"),
					},
					FlatFee: &billing.FlatFeeLine{
						Index: lo.ToPtr(0),
					},
				},
			}),
		},
		{
			LineBase: billing.LineBase{
				Type: billing.InvoiceLineTypeUsageBased,
				Name: "usage-based-line",
				Period: billing.Period{
					Start: time.Now(),
				},
				Description: lo.ToPtr("index=0"),
			},
			Children: billing.NewLineChildren(nil),
		},
	}

	adapter := &adapter{}
	adapter.sortLines(lines)

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].Children.OrEmpty()
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
