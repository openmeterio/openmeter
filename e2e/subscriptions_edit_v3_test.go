package e2e

import (
	"net/http"
	"slices"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SubscriptionEdit exercises POST /subscriptions/{id}/edit end to end.
// Setup: published plan -> subscription. Then, over the wire:
//   - add_item: the most complex path — the server maps a full RateCardInput
//     through FromAPIBillingRateCard — so it's the one worth pinning at HTTP.
//   - remove_item: confirms the response drops the item again.
//   - not-found: covers the route/decode/error path itself.
func TestV3SubscriptionEdit(t *testing.T) {
	c := newV3Client(t)

	// --- Fixture: customer ---

	customerKey := uniqueKey("sub_edit_customer")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Subscription Edit Test Customer",
		Currency:     lo.ToPtr("USD"),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// --- Fixture: published single-phase plan ---

	plan, err := c.Plans.Create(t.Context(), validPlanRequest("sub_edit_plan"))
	c.requireStatus(http.StatusCreated, err)
	require.NotEmpty(t, plan.Phases)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	// --- Fixture: active subscription on the published plan ---

	sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
		Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, sub)
	require.Equal(t, v3sdk.SubscriptionStatusActive, sub.Status)
	require.NotEmpty(t, sub.Phases, "subscription must expose at least one phase")

	subscriptionID := sub.ID
	phaseKey := sub.Phases[0].Key

	// The new rate card threads both operations: add_item introduces it, remove_item
	// takes it back out by the same key.
	addedItem := validFlatRateCard("sub_edit_item")
	itemKey := addedItem.Key

	immediate := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

	t.Run("Should add an item to the current phase and return it in the response", func(t *testing.T) {
		addItem := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddItem(v3sdk.SubscriptionEditAddItem{
			PhaseKey: phaseKey,
			RateCard: addedItem,
		}))

		updated, err := c.Subscriptions.Edit(t.Context(), subscriptionID, v3sdk.SubscriptionEdit{
			Customizations: []v3sdk.SubscriptionEditOperation{addItem},
			Timing:         &immediate,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)

		assert.True(t, subscriptionHasItem(updated, phaseKey, itemKey),
			"added item %q must be present in phase %q after add_item", itemKey, phaseKey)
	})

	t.Run("Should remove the item and drop it from the response", func(t *testing.T) {
		removeItem := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditRemoveItem(v3sdk.SubscriptionEditRemoveItem{
			PhaseKey: phaseKey,
			ItemKey:  itemKey,
		}))

		updated, err := c.Subscriptions.Edit(t.Context(), subscriptionID, v3sdk.SubscriptionEdit{
			Customizations: []v3sdk.SubscriptionEditOperation{removeItem},
			Timing:         &immediate,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)

		assert.False(t, subscriptionHasItem(updated, phaseKey, itemKey),
			"removed item %q must be gone from phase %q after remove_item", itemKey, phaseKey)
	})

	t.Run("Should return 404 when editing a nonexistent subscription", func(t *testing.T) {
		addItem := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddItem(v3sdk.SubscriptionEditAddItem{
			PhaseKey: phaseKey,
			RateCard: validFlatRateCard("sub_edit_missing"),
		}))

		_, err := c.Subscriptions.Edit(t.Context(), ulid.Make().String(), v3sdk.SubscriptionEdit{
			Customizations: []v3sdk.SubscriptionEditOperation{addItem},
			Timing:         &immediate,
		})
		requireProblem(t, err, http.StatusNotFound)
	})
}

// TestV3SubscriptionEditAddPhase covers the add_phase edit over HTTP, which the
// item-focused test above never reaches. It uses a three-phase subscription
// (phase 1 current, phases 2 and 3 in the future) so a new phase can be inserted
// between the two future phases without touching the current one:
//   - guard: inserting a phase without a duration before an existing phase must
//     surface as a 4xx (not a 5xx) — regression for a nil-duration panic in the
//     phase-shifting path.
//   - happy: a bounded phase batched with an add_item that populates it (a lone
//     add_phase would create an empty, invalid phase) is accepted and shows up in
//     the response.
func TestV3SubscriptionEditAddPhase(t *testing.T) {
	c := newV3Client(t)

	// --- Fixture: customer ---

	customerKey := uniqueKey("sub_addphase_customer")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Subscription AddPhase Test Customer",
		Currency:     lo.ToPtr("USD"),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// --- Fixture: published three-phase plan. Phase starts land at 0, P1M, P3M, so
	//     there is a gap between the two future phases to insert into. ---

	phase2 := validPlanPhase("phase_2", false)
	phase2.Duration = lo.ToPtr("P2M") // 0 + P1M + P2M => phase 3 starts at P3M
	planBody := validPlanRequest("sub_addphase_plan")
	planBody.Phases = []v3sdk.PlanPhaseInput{
		validPlanPhase("phase_1", false /* bounded, P1M */),
		phase2,
		validPlanPhase("phase_3", true /* last, open-ended */),
	}
	plan, err := c.Plans.Create(t.Context(), planBody)
	c.requireStatus(http.StatusCreated, err)
	require.Len(t, plan.Phases, 3)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	// --- Fixture: active subscription; phase 1 current, phases 2 and 3 in the future ---

	sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
		Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.Len(t, sub.Phases, 3)
	subscriptionID := sub.ID

	immediate := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

	// P2M lands between the two future phases (P1M and P3M): after "now", before
	// phase 3, and clear of the current phase. Inserting there drives the
	// phase-shifting branch — the one that used to panic on a nil duration.
	const insertStartAfter = "P2M"

	// Guard first: it fails before applying anything, so the subscription is
	// untouched for the happy case that follows.
	t.Run("Should reject add_phase without a duration before an existing phase with a 4xx", func(t *testing.T) {
		addPhase := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddPhase(v3sdk.SubscriptionEditAddPhase{
			Phase: v3sdk.SubscriptionPhaseCreate{
				Key:        uniqueKey("guard_phase"),
				Name:       "Guard Phase",
				StartAfter: v3sdk.NullableValue(insertStartAfter),
				// Duration omitted on purpose — the regression under test.
			},
		}))

		_, err := c.Subscriptions.Edit(t.Context(), subscriptionID, v3sdk.SubscriptionEdit{
			Customizations: []v3sdk.SubscriptionEditOperation{addPhase},
			Timing:         &immediate,
		})
		// The domain guard must map to 400 Bad Request carrying its message — not just
		// any 4xx, and definitely not the pre-fix 5xx panic. The 400 (vs a generic
		// 500) is the payoff of the v3 edit error encoder wiring the patch errors up.
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertProblemDetail(t, problem, "duration")
	})

	t.Run("Should add a bounded phase populated with an item and return it", func(t *testing.T) {
		newPhaseKey := uniqueKey("added_phase")
		item := validFlatRateCard("added_phase_item")

		addPhase := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddPhase(v3sdk.SubscriptionEditAddPhase{
			Phase: v3sdk.SubscriptionPhaseCreate{
				Key:        newPhaseKey,
				Name:       "Added Phase",
				StartAfter: v3sdk.NullableValue(insertStartAfter),
				Duration:   lo.ToPtr("P1M"),
			},
		}))
		addItem := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddItem(v3sdk.SubscriptionEditAddItem{
			PhaseKey: newPhaseKey,
			RateCard: item,
		}))

		updated, err := c.Subscriptions.Edit(t.Context(), subscriptionID, v3sdk.SubscriptionEdit{
			// add_phase then add_item: the phase must exist before it can be populated.
			Customizations: []v3sdk.SubscriptionEditOperation{addPhase, addItem},
			Timing:         &immediate,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)

		assert.True(t, subscriptionHasItem(updated, newPhaseKey, item.Key),
			"added phase %q with item %q must be present after add_phase", newPhaseKey, item.Key)

		// Position, not just presence: the phase was inserted at P2M into a
		// 0/P1M/P3M layout, so ordered by start time it must be the third of four
		// phases — after the phases at 0 and P1M, before the phase at P3M.
		require.Len(t, updated.Phases, 4, "add_phase should yield a fourth phase")
		ordered := slices.Clone(updated.Phases)
		slices.SortFunc(ordered, func(a, b v3sdk.SubscriptionPhase) int {
			return a.ActiveFrom.Compare(b.ActiveFrom)
		})
		assert.Equal(t, newPhaseKey, ordered[2].Key,
			"inserted phase must sort third by start time (between the P1M and P3M phases)")
	})

	t.Run("Should reject add_phase with null start_after on a running subscription as 403", func(t *testing.T) {
		// null start_after is the v1-parity wire form of "at subscription start"; on a
		// running subscription that point is in the past, so the domain rejects it with
		// 403 "cannot add phase in the past" rather than a client-side 400.
		addPhase := lo.Must(v3sdk.SubscriptionEditOperationFromSubscriptionEditAddPhase(v3sdk.SubscriptionEditAddPhase{
			Phase: v3sdk.SubscriptionPhaseCreate{
				Key:        uniqueKey("null_start_phase"),
				Name:       "Null Start Phase",
				StartAfter: v3sdk.Null[string](),
				Duration:   lo.ToPtr("P1M"),
			},
		}))

		_, err := c.Subscriptions.Edit(t.Context(), subscriptionID, v3sdk.SubscriptionEdit{
			Customizations: []v3sdk.SubscriptionEditOperation{addPhase},
			Timing:         &immediate,
		})
		problem := requireProblem(t, err, http.StatusForbidden)
		assertProblemDetail(t, problem, "past")
	})
}

// subscriptionHasItem reports whether the phase keyed by phaseKey carries a
// currently-listed item whose rate card key is itemKey. The edit response
// resolves each phase to the item version active at query time, so this reflects
// the effect of the last applied customization.
func subscriptionHasItem(sub *v3sdk.BillingSubscription, phaseKey, itemKey string) bool {
	for _, phase := range sub.Phases {
		if phase.Key != phaseKey {
			continue
		}
		for _, item := range phase.Items {
			if item.RateCard.Key == itemKey {
				return true
			}
		}
	}
	return false
}
