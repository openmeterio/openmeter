package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func TestFromDBRunBaseMapsSchemaLevelPriorRunSemantics(t *testing.T) {
	t.Run("legacy lineage is unknown", func(t *testing.T) {
		mapped, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-1",
			Namespace:   "namespace",
			SchemaLevel: usagebased.RealizationRunSchemaLevelLegacy,
		})
		require.NoError(t, err)
		require.True(t, mapped.PriorRunID.IsAbsent())
	})

	t.Run("level two first run has known empty lineage", func(t *testing.T) {
		mapped, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-1",
			Namespace:   "namespace",
			SchemaLevel: usagebased.RealizationRunSchemaLevelPriorRun,
		})
		require.NoError(t, err)
		require.True(t, mapped.PriorRunID.IsPresent())
		require.Nil(t, mapped.PriorRunID.OrEmpty())
	})

	t.Run("level two maps referenced prior run", func(t *testing.T) {
		priorRunID := "run-1"
		mapped, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-2",
			Namespace:   "namespace",
			SchemaLevel: usagebased.RealizationRunSchemaLevelPriorRun,
			PriorRunID:  &priorRunID,
		})
		require.NoError(t, err)
		require.True(t, mapped.PriorRunID.IsPresent())
		require.Equal(t, &usagebased.RealizationRunID{
			Namespace: "namespace",
			ID:        priorRunID,
		}, mapped.PriorRunID.OrEmpty())
	})
}

func TestFromDBRunBaseRejectsInvalidSchemaLevelState(t *testing.T) {
	t.Run("unknown schema level", func(t *testing.T) {
		_, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-1",
			Namespace:   "namespace",
			SchemaLevel: 3,
		})
		require.ErrorContains(t, err, "unsupported usage-based realization run schema level: 3")
	})

	t.Run("legacy run with prior run id", func(t *testing.T) {
		priorRunID := "prior-run"
		_, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-1",
			Namespace:   "namespace",
			SchemaLevel: usagebased.RealizationRunSchemaLevelLegacy,
			PriorRunID:  &priorRunID,
		})
		require.ErrorContains(t, err, "legacy usage-based realization run has prior run id")
	})

	t.Run("self-referencing prior run", func(t *testing.T) {
		priorRunID := "run-1"
		_, err := fromDBRunBase(&entdb.ChargeUsageBasedRuns{
			ID:          "run-1",
			Namespace:   "namespace",
			SchemaLevel: usagebased.RealizationRunSchemaLevelPriorRun,
			PriorRunID:  &priorRunID,
		})
		require.ErrorContains(t, err, "cannot reference itself as prior run")
	})
}
