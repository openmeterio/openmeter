package upstream

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSourceFactories(t *testing.T) {
	for name := range sourceEvents {
		first, ok := NewEvent(name)
		require.True(t, ok)
		second, ok := NewEvent(name)
		require.True(t, ok)
		require.NotSame(t, first, second)
		require.Equal(t, name, first.EventName())
	}

	event, ok := NewEvent("unrelated")
	require.False(t, ok)
	require.Nil(t, event)
}

func TestResolveSourceEvent(t *testing.T) {
	event := charges.CustomerChargesChangedEvent{Namespace: "ns", CustomerID: "customer"}
	expected := []rte.DemandSelector{{Namespace: "ns", Kind: rte.CalculationKindCreditBalance, OwnerIDs: []string{"customer"}}}
	r := &resolver{}

	for _, payload := range []marshaler.Event{event, &event} {
		selectors, err := r.Resolve(t.Context(), payload)
		require.NoError(t, err)
		require.Equal(t, expected, selectors)
	}

	var missing *charges.CustomerChargesChangedEvent
	_, err := r.Resolve(t.Context(), missing)
	require.True(t, models.IsGenericValidationError(err))
}
