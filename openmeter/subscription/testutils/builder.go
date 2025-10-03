package subscriptiontestutils

import (
	"fmt"
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// First let's add utils for building plans
type testPlanbuilder struct {
	p plan.CreatePlanInput
}

func (b *testPlanbuilder) AddPhase(dur *datetime.ISODuration, rcs ...productcatalog.RateCard) *testPlanbuilder {
	idx := len(b.p.Plan.Phases) + 1

	b.p.Plan.Phases = append(b.p.Plan.Phases, productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         fmt.Sprintf("test_phase_%d", idx),
			Name:        fmt.Sprintf("Test Phase %d", idx),
			Description: lo.ToPtr(fmt.Sprintf("Test Phase %d Description", idx)),
			Duration:    dur,
		},
		RateCards: rcs,
	})

	return b
}

func (b *testPlanbuilder) SetMeta(meta productcatalog.PlanMeta) *testPlanbuilder {
	b.p.Plan.PlanMeta = meta
	return b
}

func (b *testPlanbuilder) Build() plan.CreatePlanInput {
	return b.p
}

func BuildTestPlanInput(t *testing.T) *testPlanbuilder {
	b := &testPlanbuilder{
		p: plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: ExampleNamespace,
			},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Test Plan",
					Key:            "test_plan",
					Version:        1,
					Currency:       currency.USD,
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{},
			},
		},
	}

	return b
}

type testSubscriptionSpecBuilder struct {
	s subscription.SubscriptionSpec
	t *testing.T
}

func (b *testSubscriptionSpecBuilder) AddPhase(dur *datetime.ISODuration, rcs ...productcatalog.RateCard) *testSubscriptionSpecBuilder {
	idx := len(b.s.Phases) + 1
	startAfter := datetime.ISODurationBetween(b.s.ActiveFrom, b.s.ActiveFrom)

	if idx > 1 {
		phases := b.s.GetSortedPhases()
		cad, err := b.s.GetPhaseCadence(phases[idx-1].PhaseKey)
		require.NoError(b.t, err)
		if cad.ActiveTo == nil {
			b.t.Fatalf("phase %s has no active to, cannot add new phase without specifying duration for previous", phases[idx-1].PhaseKey)
		}
		startAfter = datetime.ISODurationBetween(b.s.ActiveFrom, *cad.ActiveTo)
	}

	// Let's build the new phase
	newPhase := subscription.SubscriptionPhaseSpec{
		CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
			PhaseKey:    fmt.Sprintf("test_phase_%d", idx),
			Name:        fmt.Sprintf("Test Phase %d", idx),
			Description: lo.ToPtr(fmt.Sprintf("Test Phase %d Description", idx)),
			StartAfter:  startAfter,
			SortHint:    lo.ToPtr(uint8(idx)),
		},
		CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
		ItemsByKey:                           make(map[string][]*subscription.SubscriptionItemSpec),
	}

	// Let's add the RateCards as Items
	for _, rc := range rcs {
		newPhase.ItemsByKey[rc.Key()] = append(newPhase.ItemsByKey[rc.Key()], &subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: newPhase.PhaseKey,
					ItemKey:  rc.Key(),
					RateCard: rc,
				},
				CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
			},
		})
	}

	for key, items := range newPhase.ItemsByKey {
		if len(items) > 1 {
			b.t.Fatalf("multiple ratecards specified with same key %s", key)
		}
	}

	b.s.Phases[newPhase.PhaseKey] = &newPhase

	return b
}

func (b *testSubscriptionSpecBuilder) Build() (subscription.SubscriptionSpec, error) {
	spec := b.s

	if err := spec.SyncAnnotations(); err != nil {
		return spec, fmt.Errorf("failed to sync annotations: %w", err)
	}

	if err := spec.Validate(); err != nil {
		return spec, fmt.Errorf("failed to validate spec: %w", err)
	}

	return spec, nil
}

func BuildTestSubscriptionSpec(t *testing.T) *testSubscriptionSpecBuilder {
	now := clock.Now()

	return &testSubscriptionSpecBuilder{
		s: subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				Plan: &subscription.PlanRef{
					Key:     "test_plan",
					Version: 1,
				},
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				CustomerId:    "01K6JCPG631MH1EKEQB2YMDBJW",
				ActiveFrom:    now,
				ActiveTo:      nil,
				Name:          "test_subscription",
				BillingAnchor: now,
				Currency:      currencyx.Code(currency.USD),
			},
			Phases: make(map[string]*subscription.SubscriptionPhaseSpec),
		},
		t: t,
	}
}
