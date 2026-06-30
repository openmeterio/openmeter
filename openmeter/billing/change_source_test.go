package billing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeSourceRequire(t *testing.T) {
	require.NoError(t, ChangeSourceSystem.Require(ChangeSourceSystem))

	require.ErrorContains(t, ChangeSourceAPIRequest.Require(ChangeSourceSystem), "must be system")
	require.ErrorContains(t, ChangeSource("invalid").Require(ChangeSourceSystem), "invalid change source")
}
