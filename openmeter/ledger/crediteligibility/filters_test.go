package crediteligibility_test

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
)

func TestFiltersMatchRoutes(t *testing.T) {
	filters := crediteligibility.Filters{
		Version:  crediteligibility.FiltersVersion2,
		Features: []string{"input", "output"},
		Plans: []crediteligibility.PlanFilter{
			{Key: "pro", Version: &crediteligibility.VersionFilter{Gte: lo.ToPtr(2)}},
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
			route := ledger.Route{Filters: crediteligibility.Filters{Version: crediteligibility.FiltersVersion1}}
			if tc.feature != "" {
				route.Filters.Features = []string{tc.feature}
			}
			if tc.plan != "" {
				route.Filters.Version = crediteligibility.FiltersVersion2
				route.Filters.Plans = []crediteligibility.PlanFilter{{Key: tc.plan, Version: &crediteligibility.VersionFilter{Eq: lo.ToPtr(tc.version)}}}
			}
			require.Equal(t, tc.matches, filters.Matches(route))
			require.True(t, (crediteligibility.Filters{Version: crediteligibility.FiltersVersion1}).Matches(route))
		})
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
		var filters crediteligibility.Filters
		require.Error(t, json.Unmarshal([]byte(input), &filters), input)
	}
}

func TestFilterCanonicalRoundTrip(t *testing.T) {
	input := crediteligibility.Filters{
		Version:  crediteligibility.FiltersVersion2,
		Features: []string{"output", "input"},
		Plans: []crediteligibility.PlanFilter{
			{Key: "pro", Version: &crediteligibility.VersionFilter{In: []int{3, 2, 3}}},
			{Key: "enterprise"},
		},
	}
	encoded, err := json.Marshal(input)
	require.NoError(t, err)
	var decoded crediteligibility.Filters
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.True(t, input.Equal(decoded))
	require.Equal(t, []string{"input", "output"}, decoded.Features)
	clone := input.Normalize()
	clone.Features[0] = "changed"
	require.Equal(t, "output", input.Features[0])
	clone.Plans[1].Version.In[0] = 99
	require.Equal(t, []int{3, 2, 3}, input.Plans[0].Version.In)
}

func TestVersionComparisons(t *testing.T) {
	for _, tc := range []struct {
		constraint      crediteligibility.VersionFilter
		matching, other int
	}{
		{crediteligibility.VersionFilter{Eq: lo.ToPtr(2)}, 2, 3},
		{crediteligibility.VersionFilter{In: []int{1, 3}}, 3, 2},
		{crediteligibility.VersionFilter{Gte: lo.ToPtr(2)}, 2, 1},
		{crediteligibility.VersionFilter{Lte: lo.ToPtr(2)}, 2, 3},
	} {
		filters := crediteligibility.Filters{Version: crediteligibility.FiltersVersion2, Plans: []crediteligibility.PlanFilter{{Key: "pro", Version: &tc.constraint}}}
		require.NoError(t, filters.Validate())
		for _, v := range []int{tc.matching, tc.other} {
			route := ledger.Route{Filters: crediteligibility.Filters{Version: crediteligibility.FiltersVersion2, Plans: []crediteligibility.PlanFilter{{Key: "pro", Version: &crediteligibility.VersionFilter{Eq: lo.ToPtr(v)}}}}}
			require.Equal(t, v == tc.matching, filters.Matches(route))
		}
	}
}
