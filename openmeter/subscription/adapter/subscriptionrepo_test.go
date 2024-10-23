package adapter_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPatchParsing(t *testing.T) {
	t.Run("Should create and retrieve same patches", func(t *testing.T) {
		now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(now)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup()

		customerRepo := subscriptiontestutils.NewCustomerAdapter(t, dbDeps)
		repo := subscriptiontestutils.NewRepo(t, dbDeps)

		cus := customerRepo.CreateExampleCustomer(t)
		sub := repo.CreateExampleSubscription(t, cus.ID)

		uPDur, _ := datex.ISOString("P1M").Parse()

		startAfter, _ := datex.ISOString("P1M").Parse()
		extendBy, _ := datex.ISOString("P1D").Parse()

		createdPatches, err := repo.CreateSubscriptionPatches(ctx, models.NamespacedID{
			Namespace: subscriptiontestutils.ExampleNamespace,
			ID:        sub.ID,
		}, []subscription.CreateSubscriptionPatchInput{
			// AddItemPatch
			// With metered entitlement
			{
				AppliedAt:  now,
				BatchIndex: 0,
				Patch: subscription.PatchAddItem{
					PhaseKey: "test",
					ItemKey:  "test",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey:   "test",
							ItemKey:    "test",
							FeatureKey: lo.ToPtr("feature-1"),
							CreateEntitlementInput: &subscription.CreateSubscriptionEntitlementSpec{
								EntitlementType:         entitlement.EntitlementTypeMetered,
								IssueAfterReset:         lo.ToPtr(100.0),
								PreserveOverageAtReset:  lo.ToPtr(true),
								IssueAfterResetPriority: lo.ToPtr(uint8(1)),
								// MeasureUsageFrom:        mFrom,
								UsagePeriodISODuration: &uPDur,
								IsSoftLimit:            lo.ToPtr(true),
							},
						},
					},
				},
			},
			// With static entitlement
			{
				AppliedAt:  now,
				BatchIndex: 1,
				Patch: subscription.PatchAddItem{
					PhaseKey: "test",
					ItemKey:  "test",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey:   "test",
							ItemKey:    "test",
							FeatureKey: lo.ToPtr("feature-1"),
							CreateEntitlementInput: &subscription.CreateSubscriptionEntitlementSpec{
								EntitlementType: entitlement.EntitlementTypeStatic,
								Config:          []byte(`{"key": "value"}`),
							},
						},
					},
				},
			},
			// With boolean entitlement
			{
				AppliedAt:  now,
				BatchIndex: 2,
				Patch: subscription.PatchAddItem{
					PhaseKey: "test",
					ItemKey:  "test",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey:   "test",
							ItemKey:    "test",
							FeatureKey: lo.ToPtr("feature-1"),
							CreateEntitlementInput: &subscription.CreateSubscriptionEntitlementSpec{
								EntitlementType: entitlement.EntitlementTypeBoolean,
							},
						},
					},
				},
			},
			// With price
			{
				AppliedAt:  now,
				BatchIndex: 3,
				Patch: subscription.PatchAddItem{
					PhaseKey: "test",
					ItemKey:  "test",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey:   "test",
							ItemKey:    "test",
							FeatureKey: lo.ToPtr("feature-1"),
							CreatePriceInput: &price.Spec{
								PhaseKey: "test",
								ItemKey:  "test",
								Value:    "100.0",
								Key:      "test",
							},
						},
					},
				},
			},
			// AddPhasePatch
			{
				AppliedAt:  now,
				BatchIndex: 4,
				Patch: subscription.PatchAddPhase{
					PhaseKey: "test",
					CreateInput: subscription.CreateSubscriptionPhaseInput{
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   "test2",
							StartAfter: startAfter,
						},
						CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{
							CreateDiscountInput: nil, // TODO: Add discounts
						},
					},
				},
			},
			// ExtenPhasePatch
			{
				AppliedAt:  now,
				BatchIndex: 5,
				Patch: subscription.PatchExtendPhase{
					PhaseKey: "test",
					Duration: extendBy,
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to create patches: %v", err)
		}

		patches, err := repo.GetSubscriptionPatches(ctx, models.NamespacedID{
			Namespace: sub.Namespace,
			ID:        sub.ID,
		})
		if err != nil {
			t.Fatalf("failed to get patches: %v", err)
		}

		assert.Equal(t, len(createdPatches), len(patches))
		assert.Equal(t, 6, len(patches))

		for i := range patches {
			assert.Equal(t, createdPatches[i].ID, patches[i].ID, "failed for patch %d", i)
			assert.Equal(t, createdPatches[i].Value, patches[i].Value, "failed for patch %d", i)
		}
	})
}
