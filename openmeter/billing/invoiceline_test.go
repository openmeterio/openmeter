package billing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInvoiceLineConversion tests the conversion between invoice lines and generic invoice lines.
// The AsInvoiceLine method on the GenericInvoiceLine provides a typesafe way to convert the line to the appropriate line type,
// without having to know exactly the type (ptr, wrapper, etc.) of the line.
func TestInvoiceLineConversion(t *testing.T) {
	t.Run("gathering line to generic line to gathering line", func(t *testing.T) {
		gLine := GatheringLine{
			GatheringLineBase: GatheringLineBase{
				InvoiceID: "test1234",
			},
		}

		var genericLine GenericInvoiceLine = &gatheringInvoiceLineGenericWrapper{GatheringLine: gLine}

		convertedGLine, err := genericLine.AsInvoiceLine().AsGatheringLine()
		require.NoError(t, err)
		require.Equal(t, gLine, convertedGLine)

		_, err = genericLine.AsInvoiceLine().AsStandardLine()
		require.Error(t, err)
	})

	t.Run("standard line to generic line to standard line", func(t *testing.T) {
		sLine := StandardLine{
			StandardLineBase: StandardLineBase{
				InvoiceID: "test1234",
			},
		}

		var genericLine GenericInvoiceLine = &standardInvoiceLineGenericWrapper{StandardLine: &sLine}

		convertedSLine, err := genericLine.AsInvoiceLine().AsStandardLine()
		require.NoError(t, err)
		require.Equal(t, sLine, convertedSLine)

		_, err = genericLine.AsInvoiceLine().AsGatheringLine()
		require.Error(t, err)
	})
}
