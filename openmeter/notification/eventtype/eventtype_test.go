package eventtype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventType(t *testing.T) {
	et := EventType{Name: "billing.invoice.created", Version: 1}

	require.Equal(t, "v1.billing.invoice.created", et.String())
	require.NoError(t, et.Validate())

	for _, invalid := range []EventType{
		{Name: "invoice", Version: 1},
		{Name: "Billing.invoice.created", Version: 1},
		{Name: "billing.invoice.created.", Version: 1},
		{Name: "billing.invoice.created", Version: 0},
	} {
		require.Error(t, invalid.Validate(), invalid.String())
	}
}
