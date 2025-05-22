package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestSortLines(t *testing.T) {
	lines := []*Line{
		{
			LineBase: LineBase{
				Type: InvoiceLineTypeUsageBased,
				Name: "usage-based-line",
				Period: Period{
					Start: time.Now().Add(time.Hour * 24),
				},
				Description: lo.ToPtr("index=1"),
			},
			Children: NewLineChildren([]*Line{
				{
					LineBase: LineBase{
						ID:          "child-2",
						Type:        InvoiceLineTypeFee,
						Description: lo.ToPtr("index=1.1"),
					},
					FlatFee: &FlatFeeLine{
						Index: lo.ToPtr(1),
					},
				},
				{
					LineBase: LineBase{
						ID:          "child-1",
						Type:        InvoiceLineTypeFee,
						Description: lo.ToPtr("index=1.0"),
					},
					FlatFee: &FlatFeeLine{
						Index: lo.ToPtr(0),
					},
				},
			}),
		},
		{
			LineBase: LineBase{
				Type: InvoiceLineTypeUsageBased,
				Name: "usage-based-line",
				Period: Period{
					Start: time.Now(),
				},
				Description: lo.ToPtr("index=0"),
			},
			Children: NewLineChildren(nil),
		},
	}

	sortLines(lines)

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].Children.OrEmpty()
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
