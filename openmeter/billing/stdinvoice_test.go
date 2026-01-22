package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSortLines(t *testing.T) {
	lines := StandardLines{
		{
			StandardLineBase: StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=1"),
				}),
				Period: Period{
					Start: time.Now().Add(time.Hour * 24),
				},
			},
			DetailedLines: DetailedLines{
				{
					DetailedLineBase: DetailedLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-2",
							Description: lo.ToPtr("index=1.1"),
						}),
						Index: lo.ToPtr(1),
					},
				},
				{
					DetailedLineBase: DetailedLineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name:        "child-1",
							Description: lo.ToPtr("index=1.0"),
						}),
						Index: lo.ToPtr(0),
					},
				},
			},
		},
		{
			StandardLineBase: StandardLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name:        "usage-based-line",
					Description: lo.ToPtr("index=0"),
				}),
				Period: Period{
					Start: time.Now(),
				},
			},
		},
	}

	lines.Sort()

	require.Equal(t, *lines[0].Description, "index=0")
	require.Equal(t, *lines[1].Description, "index=1")
	children := lines[1].DetailedLines
	require.Equal(t, *children[0].Description, "index=1.0")
	require.Equal(t, *children[1].Description, "index=1.1")
}
