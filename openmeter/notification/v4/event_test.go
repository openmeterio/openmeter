package v4

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
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

func TestParseEventType(t *testing.T) {
	et, err := ParseEventType("v12.billing.invoice.created")
	require.NoError(t, err)
	require.Equal(t, EventType{Name: "billing.invoice.created", Version: 12}, et)
	require.Equal(t, "v12.billing.invoice.created", et.String())

	for _, invalid := range []string{
		"",
		"billing.invoice.created",
		"v0.billing.invoice.created",
		"v01.billing.invoice.created",
		"v+1.billing.invoice.created",
		"V1.billing.invoice.created",
		"v1",
		"v1.invoice",
		"v1.billing.Invoice.created",
		"v1.billing.*",
		"v99999999999999999999.billing.invoice.created",
	} {
		_, err := ParseEventType(invalid)
		require.True(t, models.IsGenericValidationError(err), "%q: %v", invalid, err)
	}
}
