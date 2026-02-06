package billing

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestGatheringInvoiceLineInvoiceAtAccessor(t *testing.T) {
	line := GatheringLine{
		GatheringLineBase: GatheringLineBase{
			InvoiceAt: lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")),
		},
	}

	wrapper := &gatheringInvoiceLineGenericWrapper{GatheringLine: line}

	target := lo.Must(time.Parse(time.RFC3339, "2026-01-01T01:00:00Z"))
	wrapper.SetInvoiceAt(target)
	require.Equal(t, target, wrapper.GetInvoiceAt())

	accessor, ok := (GenericInvoiceLine)(wrapper).(InvoiceAtAccessor)
	if !ok {
		t.Fatalf("wrapper is not an InvoiceAtAccessor")
	}

	target = lo.Must(time.Parse(time.RFC3339, "2026-01-01T02:00:00Z"))
	accessor.SetInvoiceAt(target)
	require.Equal(t, target, accessor.GetInvoiceAt())
}
