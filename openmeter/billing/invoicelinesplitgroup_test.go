package billing

import (
	"encoding/json"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLineWithInvoiceHeadersMarshaling(t *testing.T) {
	lines := LinesWithInvoiceHeaders{
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
	}
	jsonBytes, err := json.Marshal(lines)
	require.NoError(t, err)
	var unmarshalled LinesWithInvoiceHeaders
	require.NoError(t, json.Unmarshal(jsonBytes, &unmarshalled))
	require.Equal(t, lines, unmarshalled)
}
