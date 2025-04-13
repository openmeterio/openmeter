package negcache

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfDecimalJSON(t *testing.T) {
	d := NewInfDecimal(10)
	require.Equal(t, d, serializeDeserialize(t, d))

	require.Equal(t, infinite, serializeDeserialize(t, infinite))
}

func serializeDeserialize(t *testing.T, d InfDecimal) InfDecimal {
	jsonStr, err := json.Marshal(d)
	require.NoError(t, err)

	var d2 InfDecimal
	require.NoError(t, json.Unmarshal(jsonStr, &d2))
	return d2
}
