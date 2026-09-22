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
		`{"schema_version":99,"features":["input"]}`,
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

func TestFiltersPreserveSelectedVersion(t *testing.T) {
	for _, version := range []crediteligibility.FiltersVersion{crediteligibility.FiltersVersion1, crediteligibility.FiltersVersion2} {
		// Given an explicitly selected format, even feature-only v2 stays v2.
		stored := crediteligibility.Filters{Version: version, Features: []string{"input"}}
		encoded, err := json.Marshal(stored)
		require.NoError(t, err)
		var decoded crediteligibility.Filters
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, version, decoded.Version)
		require.True(t, stored.Equal(decoded))
		roundTrip, err := json.Marshal(decoded)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(roundTrip))

		// When used as matching dimensions, storage version has no effect.
		v1 := crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: []string{"input"}}
		require.True(t, stored.Equal(v1))
		require.Equal(t, stored.String(), v1.String())
		var filters crediteligibility.Filters
		require.NoError(t, json.Unmarshal(encoded, &filters))
		require.True(t, stored.Equal(filters))
		canonical, err := json.Marshal(filters)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(canonical))
	}
}

func TestPlanFiltersRequireVersion2(t *testing.T) {
	// Given a plan restriction, its creator explicitly selects v2.
	filters := crediteligibility.Filters{Version: crediteligibility.FiltersVersion2, Features: []string{"input"}, Plans: []crediteligibility.PlanFilter{{Key: "pro"}}}
	encoded, err := json.Marshal(filters)
	require.NoError(t, err)
	require.JSONEq(t, `{"schema_version":2,"features":["input"],"plans":[{"key":"pro"}]}`, string(encoded))
	var stored crediteligibility.Filters
	require.NoError(t, json.Unmarshal(encoded, &stored))
	require.Equal(t, crediteligibility.FiltersVersion2, stored.Version)
	require.True(t, filters.Equal(stored))

	// Neither the v1 writer nor the v1 reader may discard plan restrictions.
	filters.Version = crediteligibility.FiltersVersion1
	_, err = json.Marshal(filters)
	require.ErrorContains(t, err, "cannot represent plans")
	require.Error(t, json.Unmarshal([]byte(`{"schema_version":1,"plans":[{"key":"pro"}]}`), &stored))
}
