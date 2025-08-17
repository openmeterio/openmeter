package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSortLines(t *testing.T) {
	lines := NewInvoiceLines([]*Line{
		{
			LineBase: LineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=1"),
				}),
				Period: Period{
					Start: time.Now().Add(time.Hour * 24),
				},
			},
			Children: NewLineChildren([]DetailedLine{
				{
					LineBase: LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-2",
							Description: lo.ToPtr("index=1.1"),
						}),
					},
					FlatFee: FlatFeeLine{
						Index: lo.ToPtr(1),
					},
				},
				{
					LineBase: LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-1",
							Description: lo.ToPtr("index=1.0"),
						}),
					},
					FlatFee: FlatFeeLine{
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
				Period: Period{
					Start: time.Now(),
				},
			},
			Children: NewLineChildren(nil),
		},
	})

	lines.Sort()

	require.Equal(t, *lines.OrEmpty()[0].Description, "index=0")
	require.Equal(t, *lines.OrEmpty()[1].Description, "index=1")
	children := lines.OrEmpty()[1].Children
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
