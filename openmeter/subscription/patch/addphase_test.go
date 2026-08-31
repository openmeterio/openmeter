package patch_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestAddPhase(t *testing.T) {
	// The example spec has three phases: test_phase_1 (start 0, P1M),
	// test_phase_2 (start P1M, P2M), test_phase_3 (start P3M, open-ended).
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	tests := testsuite[patch.PatchAddPhase]{
		SystemTime: now,
		TT: []testcase[patch.PatchAddPhase]{
			{
				// Regression: a nil duration on a phase inserted before an existing
				// one used to be dereferenced during phase shifting and panic. It must
				// surface as a forbidden error instead. P2W lands after now (so it
				// clears the not-in-the-past check) but before test_phase_2 at P1M.
				Name: "Should reject a phase without duration inserted before an existing phase",
				Patch: patch.PatchAddPhase{
					PhaseKey: "inserted_phase",
					CreateInput: subscription.CreateSubscriptionPhaseInput{
						Duration: nil,
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   "inserted_phase",
							StartAfter: datetime.MustParseDuration(t, "P2W"),
							Name:       "Inserted",
						},
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)
					require.GreaterOrEqual(t, len(s.Phases), 2)
					return s
				},
				Ctx:           subscription.ApplyContext{CurrentTime: now},
				ExpectedError: &subscription.PatchValidationError{Msg: "cannot add a phase without a duration before an existing phase"},
			},
			{
				// A nil duration is still valid for a genuine last phase: P6M sits
				// after every existing phase, so no shifting is needed and the guard
				// above must not trigger.
				Name: "Should allow a last phase without duration",
				Patch: patch.PatchAddPhase{
					PhaseKey: "appended_phase",
					CreateInput: subscription.CreateSubscriptionPhaseInput{
						Duration: nil,
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   "appended_phase",
							StartAfter: datetime.MustParseDuration(t, "P6M"),
							Name:       "Appended",
						},
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)
					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)
					s.Phases["appended_phase"] = &subscription.SubscriptionPhaseSpec{
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   "appended_phase",
							StartAfter: datetime.MustParseDuration(t, "P6M"),
							Name:       "Appended",
						},
						ItemsByKey: make(map[string][]*subscription.SubscriptionItemSpec),
					}
					return *s
				},
			},
		},
	}

	tests.Run(t)
}
