package customerbalance

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
)

func TestLiveBalanceConsumesOnlyMatchingPlanSources(t *testing.T) {
	// given: the same feature has credits for two distinct plan versions.
	plans := []crediteligibility.Filters{
		{Version: crediteligibility.FiltersVersion2, Features: []string{"api-calls"}, Plans: []crediteligibility.PlanFilter{{Key: "pro", Version: &crediteligibility.VersionFilter{Eq: lo.ToPtr(1)}}}},
		{Version: crediteligibility.FiltersVersion2, Features: []string{"api-calls"}, Plans: []crediteligibility.PlanFilter{{Key: "pro", Version: &crediteligibility.VersionFilter{Eq: lo.ToPtr(2)}}}},
	}
	sources := []liveBalanceSource{
		{route: ledger.Route{Filters: plans[0]}, amount: alpacadecimal.NewFromInt(40)},
		{route: ledger.Route{Filters: plans[1]}, amount: alpacadecimal.NewFromInt(30)},
		{route: ledger.Route{}, amount: alpacadecimal.NewFromInt(10)},
	}
	// when: a v2 impact exceeds all matching sources.
	consumed := consumeLiveBalanceSources(sources, ledger.Route{Filters: plans[1]}, alpacadecimal.NewFromInt(100))
	// then: live allocation leaves v1 untouched, matching booked collection behavior.
	require.Equal(t, float64(40), consumed.InexactFloat64())
	require.Equal(t, float64(40), sources[0].amount.InexactFloat64())
	require.Equal(t, float64(0), sources[1].amount.InexactFloat64())
	require.Equal(t, float64(0), sources[2].amount.InexactFloat64())
}
