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

func TestEventTypeCompare(t *testing.T) {
	// Sorted by numeric version first, so v10 follows v2, then by name.
	sorted := []EventType{
		{Name: "billing.invoice.created", Version: 1},
		{Name: "entitlement.balance_threshold.triggered", Version: 1},
		{Name: "billing.invoice.created", Version: 2},
		{Name: "billing.invoice.created", Version: 10},
	}

	for i := range sorted {
		require.Zero(t, sorted[i].Compare(sorted[i]), sorted[i].String())

		for j := i + 1; j < len(sorted); j++ {
			require.Negative(t, sorted[i].Compare(sorted[j]), "%s < %s", sorted[i], sorted[j])
			require.Positive(t, sorted[j].Compare(sorted[i]), "%s > %s", sorted[j], sorted[i])
		}
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
