package usagebased

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

func TestValidateExpands(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateExpands(meta.Expands{meta.ExpandRealizations}))
	require.NoError(t, validateExpands(meta.Expands{meta.ExpandRealizations, meta.ExpandDetailedLines}))
	require.Error(t, validateExpands(meta.Expands{meta.ExpandDetailedLines}))
}

func TestRatingEngineValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, RatingEngineDelta.Validate())
	require.NoError(t, RatingEnginePeriodPreserving.Validate())
	require.Error(t, RatingEngine("").Validate())
	require.Error(t, RatingEngine("unknown").Validate())
}
