package billing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSplitLineGroupMarshaling(t *testing.T) {
	now := time.Now().In(time.UTC)

	testGroup := SplitLineHierarchy{
		Group: SplitLineGroup{
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			NamespacedID: models.NamespacedID{
				Namespace: "test",
				ID:        "123",
			},
		},
		Lines: LinesWithInvoiceHeaders{
			{
				Line: &standardInvoiceLineGenericWrapper{StandardLine: &StandardLine{
					StandardLineBase: StandardLineBase{
						ManagedResource: models.ManagedResource{
							ID: "123",
						},
					},
					UsageBased: &UsageBasedLine{
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						}),
					},
				}},
				Invoice: &StandardInvoice{
					StandardInvoiceBase: StandardInvoiceBase{
						ID:     "123",
						Number: "invoice-123",
					},
				},
			},
			{
				Line: &gatheringInvoiceLineGenericWrapper{GatheringLine: GatheringLine{
					GatheringLineBase: GatheringLineBase{
						ManagedResource: models.ManagedResource{
							ID: "123",
						},
						Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(100),
						}),
					},
				}},
				Invoice: &GatheringInvoice{
					GatheringInvoiceBase: GatheringInvoiceBase{
						ManagedResource: models.ManagedResource{
							ID: "123",
						},
						Number: "invoice-123",
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(testGroup)
	if err != nil {
		t.Fatalf("failed to marshal test group: %v", err)
	}

	t.Logf("test group: %s", string(jsonBytes))

	var unmarshalled SplitLineHierarchy
	err = json.Unmarshal(jsonBytes, &unmarshalled)
	if err != nil {
		t.Fatalf("failed to unmarshal test group: %v", err)
	}

	t.Logf("unmarshalled test group: %+v", unmarshalled)

	require.Equal(t, testGroup, unmarshalled)
}
