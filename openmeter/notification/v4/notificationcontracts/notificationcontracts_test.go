package notificationcontracts_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/notificationcontracts"
)

// Every generated contract must be accepted by the Event Registry, so an
// invalid schema fails here instead of at startup.
func TestDefinitionsRegister(t *testing.T) {
	definitions := notificationcontracts.Definitions()
	require.NotEmpty(t, definitions)

	registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
		Logger:      slog.New(slog.DiscardHandler),
		Definitions: definitions,
	})
	require.NoError(t, err)
	require.Len(t, registry.EventTypes(nil), len(definitions))
}
