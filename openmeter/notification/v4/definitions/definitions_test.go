package definitions_test

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/definitions"
)

var update = flag.Bool("update", false, "rewrite definitions.sum from the current definitions")

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

// definitions.sum records the fingerprint of every definition, so changing one
// in place shows up in review. Only a definition that was never released may
// change; rewrite the file with -update after deciding that it is safe.
func TestDefinitionsSum(t *testing.T) {
	defs := definitions.All()
	slices.SortFunc(defs, func(a, b v4.EventTypeDefinition) int {
		return a.EventType.Compare(b.EventType)
	})

	var want strings.Builder
	for _, def := range defs {
		fingerprint, err := def.Fingerprint()
		require.NoError(t, err, def.EventType.String())

		fmt.Fprintf(&want, "%s %s\n", def.EventType, fingerprint)
	}

	if *update {
		require.NoError(t, os.WriteFile("definitions.sum", []byte(want.String()), 0o644))

		return
	}

	got, err := os.ReadFile("definitions.sum")
	require.NoError(t, err)
	require.Equal(t, want.String(), string(got),
		"An Event type definition changed. A released definition must not change: add a new version instead. "+
			"If the changed definitions were never released, run: "+
			"go test ./openmeter/notification/v4/definitions -run TestDefinitionsSum -update")
}
