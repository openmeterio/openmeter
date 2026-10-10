package ledger_test

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestFiltersMatchRoutes(t *testing.T) {
	filters := ledger.CreditFilters{
		Features: []string{"input", "output"},
		Plans: []ledger.PlanFilter{
			{Key: "pro", Version: &ledger.VersionFilter{Gte: lo.ToPtr(2)}},
			{Key: "enterprise"},
		},
	}

	for _, tc := range []struct {
		name, feature, plan string
		version             int
		matches             bool
	}{
		{"both dimensions", "input", "pro", 2, true},
		{"future version", "output", "pro", 20, true},
		{"alternative plan", "input", "enterprise", 1, true},
		{"older version", "input", "pro", 1, false},
		{"wrong plan", "input", "starter", 2, false},
		{"wrong feature", "storage", "pro", 2, false},
		{"missing plan", "input", "", 0, false},
		{"missing feature", "", "pro", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			route := ledger.Route{}

			if tc.feature != "" {
				route.Filters.Features = []string{tc.feature}
			}

			if tc.plan != "" {
				route.Filters.Plans = []ledger.PlanFilter{{Key: tc.plan, Version: &ledger.VersionFilter{Eq: lo.ToPtr(tc.version)}}}
			}

			require.Equal(t, tc.matches, filters.Matches(route))
			require.True(t, (ledger.CreditFilters{}).Matches(route))
		})
	}
}

func TestVersionComparisons(t *testing.T) {
	for _, tc := range []struct {
		constraint      ledger.VersionFilter
		matching, other int
	}{
		{ledger.VersionFilter{Eq: lo.ToPtr(2)}, 2, 3},
		{ledger.VersionFilter{In: []int{1, 3}}, 3, 2},
		{ledger.VersionFilter{Gte: lo.ToPtr(2)}, 2, 1},
		{ledger.VersionFilter{Lte: lo.ToPtr(2)}, 2, 3},
	} {
		filters := ledger.CreditFilters{
			Plans: []ledger.PlanFilter{{Key: "pro", Version: &tc.constraint}},
		}
		require.NoError(t, filters.Validate())

		for _, v := range []int{tc.matching, tc.other} {
			route := ledger.Route{
				Filters: ledger.CreditFilters{
					Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(v)}}},
				},
			}
			require.Equal(t, v == tc.matching, filters.Matches(route))
		}
	}
}

func TestFilterStorage(t *testing.T) {
	for _, features := range [][]string{nil, {}, {"output", "input"}} {
		input := ledger.CreditFilters{Version: ledger.CreditFiltersVersion1, Features: features}
		encoded, err := json.Marshal(input.Normalize())
		require.NoError(t, err)

		var decoded ledger.CreditFilters
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
		var filters ledger.CreditFilters
		require.Error(t, json.Unmarshal([]byte(input), &filters), input)
	}
}

func TestFiltersVersion1(t *testing.T) {
	// Given a v1 row, decoding retains its format and normalizes its dimensions.
	var stored ledger.CreditFilters
	require.NoError(t, json.Unmarshal([]byte(`{"schema_version":1,"features":["output","input"]}`), &stored))
	require.Equal(t, ledger.CreditFiltersVersion1, stored.Version)
	require.Equal(t, []string{"input", "output"}, stored.Features)

	// When re-encoded, the persisted v1 representation is preserved.
	encoded, err := json.Marshal(stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"schema_version":1,"features":["input","output"]}`, string(encoded))
}

func TestFiltersRejectUnsupportedVersions(t *testing.T) {
	for _, input := range []string{`{}`, `null`, `{"schema_version":0}`, `{"schema_version":99}`, `{"schema_version":1,"plans":[]}`} {
		var stored ledger.CreditFilters
		require.Error(t, json.Unmarshal([]byte(input), &stored), input)
	}

	for _, version := range []ledger.CreditFiltersVersion{-1, 99} {
		filters := ledger.CreditFilters{Version: version}
		require.Equal(t, version, filters.Normalize().Version)
		require.Error(t, filters.Validate())
		_, err := json.Marshal(filters)
		require.Error(t, err)
	}
}

func TestFiltersPreserveSelectedVersion(t *testing.T) {
	for _, version := range []ledger.CreditFiltersVersion{ledger.CreditFiltersVersion1, ledger.CreditFiltersVersion2} {
		// Given an explicitly selected format, even feature-only v2 stays v2.
		stored := ledger.CreditFilters{Version: version, Features: []string{"input"}}
		encoded, err := json.Marshal(stored)
		require.NoError(t, err)

		var decoded ledger.CreditFilters
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, version, decoded.Version)
		require.True(t, stored.Equal(decoded))
		roundTrip, err := json.Marshal(decoded)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(roundTrip))

		// When used as matching dimensions, storage version has no effect.
		v1 := ledger.CreditFilters{Version: ledger.CreditFiltersVersion1, Features: []string{"input"}}
		require.True(t, stored.Equal(v1))
		require.Equal(t, stored.String(), v1.String())

		var filters ledger.CreditFilters
		require.NoError(t, json.Unmarshal(encoded, &filters))
		require.True(t, stored.Equal(filters))
		canonical, err := json.Marshal(filters)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(canonical))
	}
}

func TestPlanFiltersRequireVersion2(t *testing.T) {
	// Given a new plan restriction, its creator can omit the storage version.
	filters := ledger.CreditFilters{Features: []string{"input"}, Plans: []ledger.PlanFilter{{Key: "pro"}}}
	encoded, err := json.Marshal(filters)
	require.NoError(t, err)
	require.JSONEq(t, `{"schema_version":2,"features":["input"],"plans":[{"key":"pro"}]}`, string(encoded))

	var stored ledger.CreditFilters
	require.NoError(t, json.Unmarshal(encoded, &stored))
	require.Equal(t, ledger.CreditFiltersVersion2, stored.Version)
	require.True(t, filters.Equal(stored))

	// Neither the v1 writer nor the v1 reader may discard plan restrictions.
	filters.Version = ledger.CreditFiltersVersion1
	_, err = json.Marshal(filters)
	require.ErrorContains(t, err, "cannot represent plans")
	require.Error(t, json.Unmarshal([]byte(`{"schema_version":1,"plans":[{"key":"pro"}]}`), &stored))
}

func TestFiltersDefaultVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		filters ledger.CreditFilters
		version ledger.CreditFiltersVersion
	}{
		{"unrestricted", ledger.CreditFilters{}, ledger.CreditFiltersVersion1},
		{"features", ledger.CreditFilters{Features: []string{"input"}}, ledger.CreditFiltersVersion1},
		{"plans", ledger.CreditFilters{Plans: []ledger.PlanFilter{{Key: "pro"}}}, ledger.CreditFiltersVersion2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: application filters omit the storage version.
			require.NoError(t, tc.filters.Validate())
			normalized := tc.filters.Normalize()
			require.Equal(t, tc.version, normalized.Version)
			require.Equal(t, normalized, normalized.Normalize())

			// when: encoded directly, the writer supplies the same default.
			encoded, err := json.Marshal(tc.filters)
			require.NoError(t, err)

			var decoded ledger.CreditFilters
			require.NoError(t, json.Unmarshal(encoded, &decoded))

			// then: persisted data is explicitly versioned without mutating the caller.
			require.Equal(t, normalized, decoded)
			require.Zero(t, tc.filters.Version)
		})
	}
}

func TestFiltersDefaultVersionStillValidatesDimensions(t *testing.T) {
	for _, filters := range []ledger.CreditFilters{
		{Features: []string{""}},
		{Features: []string{"input", "input"}},
		{Plans: []ledger.PlanFilter{{Key: ""}}},
		{Plans: []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{}}}},
	} {
		require.Error(t, filters.Validate())
		_, err := json.Marshal(filters)
		require.Error(t, err)
	}
}

func TestFilterStorageRejectsUnknownRestrictions(t *testing.T) {
	for _, input := range []string{
		`{"schema_version":99,"features":["input"]}`,
		`{"schema_version":1,"regions":["eu"]}`,
		`{"schema_version":2,"regions":["eu"]}`,
		`{"schema_version":2,"plans":[{"key":"pro","version":{"gt":2}}]}`,
		`{"schema_version":2,"plans":[{"key":"pro","version":{}}]}`,
		`{"schema_version":2,"plans":[{"key":"pro","version":{"eq":1,"gte":2}}]}`,
		`{"schema_version":2,"plans":[{"key":"pro","version":{"in":[]}}]}`,
		`{"schema_version":1} {}`,
	} {
		var filters ledger.CreditFilters
		require.Error(t, json.Unmarshal([]byte(input), &filters), input)
	}
}

func TestFilterCanonicalRoundTrip(t *testing.T) {
	input := ledger.CreditFilters{
		Version:  ledger.CreditFiltersVersion2,
		Features: []string{"output", "input"},
		Plans: []ledger.PlanFilter{
			{Key: "pro", Version: &ledger.VersionFilter{In: []int{3, 2, 3}}},
			{Key: "enterprise"},
		},
	}

	encoded, err := json.Marshal(input)
	require.NoError(t, err)

	var decoded ledger.CreditFilters
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.True(t, input.Equal(decoded))
	require.Equal(t, []string{"input", "output"}, decoded.Features)
	clone := input.Normalize()
	clone.Features[0] = "changed"
	require.Equal(t, "output", input.Features[0])
	clone.Plans[1].Version.In[0] = 99
	require.Equal(t, []int{3, 2, 3}, input.Plans[0].Version.In)
}
