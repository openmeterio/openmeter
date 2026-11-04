package definitions_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/definitions"
)

// Every generated contract must be accepted by the Event Registry, so an
// invalid schema fails here instead of at startup.
func TestDefinitionsRegister(t *testing.T) {
	defs := definitions.All()
	require.NotEmpty(t, defs)

	registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
		Logger:      slog.New(slog.DiscardHandler),
		Definitions: defs,
	})
	require.NoError(t, err)
	require.Len(t, registry.EventTypes(nil), len(defs))
}

// definitions.sum records the fingerprint and replacement of every
// definition, so changing one in place shows up in review.
func TestDefinitionsSum(t *testing.T) {
	require.NoError(t, definitions.Verify(),
		"An Event type definition or its deprecation changed. A released definition must not change: add a new version instead. "+
			"If the changed definitions were never released, run: make update-notification-definitions-sum")
}
