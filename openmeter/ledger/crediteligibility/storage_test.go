package crediteligibility_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
)

func TestFilterStorage(t *testing.T) {
	for _, features := range [][]string{nil, {}, {"output", "input"}} {
		input := crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: features}
		encoded, err := json.Marshal(input.Normalize())
		require.NoError(t, err)
		var decoded crediteligibility.Filters
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, input.Version, decoded.Version)
		roundTrip, err := json.Marshal(decoded)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(roundTrip))
	}
	for _, input := range []string{
		`{"schema_version":2,"features":["input"]}`,
		`{"schema_version":1,"regions":["eu"]}`,
		`{"schema_version":1,"features":[""]}`,
		`{"schema_version":1,"features":["input","input"]}`,
		`{"schema_version":1} {}`,
	} {
		var filters crediteligibility.Filters
		require.Error(t, json.Unmarshal([]byte(input), &filters), input)
	}
}

func TestFiltersVersion1(t *testing.T) {
	// Given a v1 row, decoding retains its format and normalizes its dimensions.
	var stored crediteligibility.Filters
	require.NoError(t, json.Unmarshal([]byte(`{"schema_version":1,"features":["output","input"]}`), &stored))
	require.Equal(t, crediteligibility.FiltersVersion1, stored.Version)
	require.Equal(t, []string{"input", "output"}, stored.Features)

	// When re-encoded, the persisted v1 representation is preserved.
	encoded, err := json.Marshal(stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"schema_version":1,"features":["input","output"]}`, string(encoded))
}

func TestFiltersRejectUnsupportedVersions(t *testing.T) {
	for _, input := range []string{`{}`, `null`, `{"schema_version":0}`, `{"schema_version":99}`, `{"schema_version":1,"plans":[]}`} {
		var stored crediteligibility.Filters
		require.Error(t, json.Unmarshal([]byte(input), &stored), input)
	}
	for _, version := range []crediteligibility.FiltersVersion{0, 99} {
		filters := crediteligibility.Filters{Version: version}
		require.Equal(t, version, filters.Normalize().Version)
		require.Error(t, filters.Validate())
		_, err := json.Marshal(filters)
		require.Error(t, err)
	}
}
