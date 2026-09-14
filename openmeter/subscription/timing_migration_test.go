package subscription

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestMigrationTimingRejectsSubscriptionEnd(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// given a subscription ending at the next billing boundary
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	spec := SubscriptionSpec{
		CreateSubscriptionPlanInput:     CreateSubscriptionPlanInput{BillingCadence: datetime.MustParseDuration(t, "P1M")},
		CreateSubscriptionCustomerInput: CreateSubscriptionCustomerInput{ActiveFrom: now, ActiveTo: &end, BillingAnchor: now},
		Phases: map[string]*SubscriptionPhaseSpec{
			"phase": {CreateSubscriptionPhasePlanInput: CreateSubscriptionPhasePlanInput{PhaseKey: "phase"}},
		},
	}

	// when the migration would take effect exactly at cancellation
	err := (Timing{Enum: lo.ToPtr(TimingNextBillingCycle)}).ValidateForMigration(spec)

	// then the half-open subscription lifetime excludes that boundary
	require.ErrorContains(t, err, "must be active at migration time")
}

func TestMigrationImmediateTimingUsesCurrentTime(t *testing.T) {
	// given a running subscription and a clock that advances normally
	spec := SubscriptionSpec{
		CreateSubscriptionCustomerInput: CreateSubscriptionCustomerInput{ActiveFrom: clock.Now().Add(-time.Hour)},
	}

	// when resolving and validating immediate timing
	err := (Timing{Enum: lo.ToPtr(TimingImmediate)}).ValidateForMigration(spec)

	// then elapsed time between those operations does not make the request historical
	require.NoError(t, err)
}
