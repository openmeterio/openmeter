package billing

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestSortLines(t *testing.T) {
	input := NewLineChildren([]*Line{
		{
			LineBase: LineBase{
				Type: InvoiceLineTypeUsageBased,
				Name: "usage-based-line",
				Period: Period{
					Start: time.Now().Add(time.Hour * 24),
				},
				Description: lo.ToPtr("index=1"),
			},
			DetailedLines: DetailedLines{
				{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:          "child-2",
						Description: lo.ToPtr("index=1.1"),
					}),
					Index: lo.ToPtr(1),
				},
				{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:          "child-1",
						Description: lo.ToPtr("index=1.0"),
					}),
					Index: lo.ToPtr(0),
				},
			},
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
		},
	})

	lines := input.Sorted().OrEmpty()

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].DetailedLines
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
