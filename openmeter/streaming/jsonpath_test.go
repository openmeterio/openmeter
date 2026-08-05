package streaming_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// jsonPathConnector controls the validity verdict and error; other Connector methods
// come from the embedded mock and are unused here.
type jsonPathConnector struct {
	*testutils.MockStreamingConnector
	invalid map[string]bool
	err     error
}

func (c jsonPathConnector) ValidateJSONPath(_ context.Context, jsonPath string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}

	return !c.invalid[jsonPath], nil
}

func newJSONPathConnector(t *testing.T, invalid map[string]bool, err error) streaming.Connector {
	t.Helper()

	return jsonPathConnector{
		MockStreamingConnector: testutils.NewMockStreamingConnector(t),
		invalid:                invalid,
		err:                    err,
	}
}

func TestValidateJSONPaths(t *testing.T) {
	t.Run("accepts valid value property and group by paths", func(t *testing.T) {
		connector := newJSONPathConnector(t, nil, nil)

		err := streaming.ValidateJSONPaths(t.Context(), connector, lo.ToPtr("$.value"), map[string]string{
			"region": "$.region",
		})

		require.NoError(t, err)
	})

	t.Run("accepts a nil value property and a nil group by", func(t *testing.T) {
		connector := newJSONPathConnector(t, nil, nil)

		err := streaming.ValidateJSONPaths(t.Context(), connector, nil, nil)

		require.NoError(t, err)
	})

	t.Run("rejects an invalid value property with a clean message", func(t *testing.T) {
		connector := newJSONPathConnector(t, map[string]bool{"$.bad[": true}, nil)

		err := streaming.ValidateJSONPaths(t.Context(), connector, lo.ToPtr("$.bad["), nil)

		// The message must name the path and not leak the %!w(<nil>) nil-wrap artifact.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "$.bad[")
		assert.NotContains(t, err.Error(), "%!w")
		assert.NotContains(t, err.Error(), "<nil>")
		assert.True(t, models.IsGenericValidationError(err), "expected a generic validation error, got %T", err)
	})

	t.Run("rejects an invalid group by path naming the key and path", func(t *testing.T) {
		connector := newJSONPathConnector(t, map[string]bool{"$.bad[": true}, nil)

		err := streaming.ValidateJSONPaths(t.Context(), connector, lo.ToPtr("$.value"), map[string]string{
			"region": "$.bad[",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "$.bad[")
		assert.Contains(t, err.Error(), "region")
		assert.NotContains(t, err.Error(), "%!w")
		assert.NotContains(t, err.Error(), "<nil>")
		assert.True(t, models.IsGenericValidationError(err))
	})

	t.Run("propagates a connector error as a non-validation error", func(t *testing.T) {
		connectorErr := errors.New("clickhouse unavailable")
		connector := newJSONPathConnector(t, nil, connectorErr)

		err := streaming.ValidateJSONPaths(t.Context(), connector, lo.ToPtr("$.value"), nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, connectorErr)
		assert.False(t, models.IsGenericValidationError(err), "a backend outage is not a user validation error")
		assert.True(t, strings.Contains(err.Error(), "validate json path in clickhouse"))
	})
}
