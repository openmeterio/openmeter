package meta_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestPlanFiltersMatchRecordedChargeAttribution(t *testing.T) {
	// given: a grant restricted to a feature and one concrete plan version.
	grant := ledger.CreditFilters{
		Version:  ledger.CreditFiltersVersion2,
		Features: []string{"input_tokens"},
		Plans:    []ledger.PlanFilter{{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}},
	}
	for _, tc := range []struct {
		name    string
		plan    *meta.SubscriptionPlan
		feature string
		matches bool
	}{
		{"matching snapshot", &meta.SubscriptionPlan{Key: "pro", Version: 2}, "input_tokens", true},
		{"earlier version", &meta.SubscriptionPlan{Key: "pro", Version: 1}, "input_tokens", false},
		{"another plan", &meta.SubscriptionPlan{Key: "starter", Version: 2}, "input_tokens", false},
		{"another feature", &meta.SubscriptionPlan{Key: "pro", Version: 2}, "output_tokens", false},
		{"unattributed charge", nil, "input_tokens", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when: the charge's immutable intent is mapped to ledger route dimensions.
			intent := meta.Intent{SubscriptionPlan: tc.plan}
			route := ledger.Route{Filters: intent.GetCreditFilters(tc.feature)}
			require.NoError(t, route.Filters.Validate())

			// then: only the recorded plan and feature can match the grant.
			require.Equal(t, tc.matches, grant.Matches(route))
		})
	}
}
