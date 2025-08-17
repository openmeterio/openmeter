package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSortLines(t *testing.T) {
	lines := []*Line{
		{
			LineBase: LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=1"),
				}),
				Type: InvoiceLineTypeUsageBased,
				Period: Period{
					Start: time.Now().Add(time.Hour * 24),
				},
			},
			Children: NewLineChildren([]*Line{
				{
					LineBase: LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-2",
							Description: lo.ToPtr("index=1.1"),
						}),
						Type: InvoiceLineTypeFee,
					},
					FlatFee: &FlatFeeLine{
						Index: lo.ToPtr(1),
					},
				},
				{
					LineBase: LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-1",
							Description: lo.ToPtr("index=1.0"),
						}),
						Type: InvoiceLineTypeFee,
					},
					FlatFee: &FlatFeeLine{
						Index: lo.ToPtr(0),
					},
				},
			}),
		},
		{
			LineBase: LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=0"),
				}),
				Type: InvoiceLineTypeUsageBased,
				Period: Period{
					Start: time.Now(),
				},
			},
			Children: NewLineChildren(nil),
		},
	}

	sortLines(lines)

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].Children
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
