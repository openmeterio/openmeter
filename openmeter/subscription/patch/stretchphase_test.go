package patch_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datex"
)

func TestStretchPhase(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	s, p := getDefaultSpec(t, now)

	getSpec := func(_ *testing.T) subscription.SubscriptionSpec {
		phases := make(map[string]*subscription.SubscriptionPhaseSpec)

		for k, v := range s.Phases {
			vCopy := *v
			phases[k] = &vCopy
		}

		s2 := *s
		s2.Phases = phases

		return s2
	}

	tests := testsuite[patch.PatchStretchPhase]{
		SystemTime: now,
		TT: []testcase[patch.PatchStretchPhase]{
			{
				Name: "Should extend first phase by 1 Month",
				Patch: patch.PatchStretchPhase{
					PhaseKey: "test_phase_1",
					Duration: testutils.GetISODuration(t, "P1M"),
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's validate the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					require.Equal(t, "test_phase_1", p.Phases[0].Key)

					_, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					// The second phase should start after 1 month
					require.Equal(t, p2.StartAfter, datex.NewPeriod(0, 1, 0, 0, 0, 0, 0))

					return &s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					for k := range s.Phases {
						if k != "test_phase_1" {
							nSA, err := s.Phases[k].StartAfter.Add(testutils.GetISODuration(t, "P1M"))
							require.NoError(t, err)

							s.Phases[k].StartAfter = nSA
						}
					}

					return s
				},
			},
			{
				Name: "Should shrink first phase by 2 weeks",
				Patch: patch.PatchStretchPhase{
					PhaseKey: "test_phase_1",
					Duration: testutils.GetISODuration(t, "-P2W"),
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's validate the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					require.Equal(t, "test_phase_1", p.Phases[0].Key)

					_, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					// The second phase should start after 1 month
					require.Equal(t, p2.StartAfter, datex.NewPeriod(0, 1, 0, 0, 0, 0, 0))

					return &s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					for k := range s.Phases {
						if k != "test_phase_1" {
							nSA, err := s.Phases[k].StartAfter.Subtract(testutils.GetISODuration(t, "P2W"))
							require.NoError(t, err)

							s.Phases[k].StartAfter = nSA
						}
					}

					return s
				},
			},
			{
				Name: "Should not allow stretching if there's a single phase",
				Patch: patch.PatchStretchPhase{
					PhaseKey: "test_phase_1",
					Duration: testutils.GetISODuration(t, "P1M"),
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's validate the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					require.Equal(t, "test_phase_1", p.Phases[0].Key)

					_, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					delete(s.Phases, "test_phase_2")
					delete(s.Phases, "test_phase_3")

					return &s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "cannot stretch a single phase"},
			},
			{
				Name: "Should not allow stretching more than phase length",
				Patch: patch.PatchStretchPhase{
					PhaseKey: "test_phase_1",
					Duration: testutils.GetISODuration(t, "-P1M"),
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's validate the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					require.Equal(t, "test_phase_1", p.Phases[0].Key)

					_, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					return &s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "phase test_phase_1 would disappear due to stretching"},
			},
			{
				Name: "Should work when stretching past next phase",
				Patch: patch.PatchStretchPhase{
					PhaseKey: "test_phase_1",
					Duration: testutils.GetISODuration(t, "P5M"),
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's validate the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					require.Equal(t, "test_phase_1", p.Phases[0].Key)

					_, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					return &s
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					for k := range s.Phases {
						if k != "test_phase_1" {
							nSA, err := s.Phases[k].StartAfter.Add(testutils.GetISODuration(t, "P5M"))
							require.NoError(t, err)

							s.Phases[k].StartAfter = nSA
						}
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
			},
		},
	}

	tests.Run(t)
}

func TestASD(t *testing.T) {
	p1 := testutils.GetISODuration(t, "P1M")
	p2 := testutils.GetISODuration(t, "P2W")
	p3 := testutils.GetISODuration(t, "-P2W")

	r1, err := p1.Subtract(p2)
	require.Nil(t, err)

	r2, err := p1.Add(p3)
	require.Nil(t, err)

	require.Equal(t, r1, r2)
}
