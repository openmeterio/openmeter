package addondiff_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRestore(t *testing.T) {
	type tcDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	withDeps := func(fn func(t *testing.T, deps *tcDeps)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)

			fn(t, &tcDeps{
				deps: deps,
			})
		}
	}

	t.Run("Should do nothing if RateCard of addon is not present in the spec", withDeps(func(t *testing.T, deps *tcDeps) {
		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, &subscriptiontestutils.ExampleRateCard2).Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard1),
			models.CadencedModel{
				ActiveFrom: now,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Now let's NOT apply the diff, only restore the diff
		// err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		// require.NoError(t, err)

		restore := env.diffable.GetRestores()
		err := spec.Apply(restore, subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should delete single RateCard that was created by the addon", withDeps(func(t *testing.T, deps *tcDeps) {
		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, &subscriptiontestutils.ExampleRateCard2).Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard1),
			models.CadencedModel{
				ActiveFrom: now,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Now let's apply the diff, then restore the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should delete a single RateCard that was created by the addon and leave in place the RateCard with the same key as that was part of the subscription", withDeps(func(t *testing.T, deps *tcDeps) {
		oneMonth := datetime.MustParseDuration(t, "P1M")

		oneMonthLater, _ := oneMonth.AddTo(now)

		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, &subscriptiontestutils.ExampleRateCard1, &subscriptiontestutils.ExampleRateCard2).Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard1),
			models.CadencedModel{
				ActiveFrom: oneMonthLater,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Let's add a manual EndDate to ExampleRateCard2
		spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth
		ogSpec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth

		// Now let's apply the diff, then restore the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should delete a single RateCard from a phase where it was created by the addon but leave RateCard with same key in another phase where it was already present", withDeps(func(t *testing.T, deps *tcDeps) {
		oneMonth := datetime.MustParseDuration(t, "P1M")

		oneMonthLater, _ := oneMonth.AddTo(now)

		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(&oneMonth, &subscriptiontestutils.ExampleRateCard1, &subscriptiontestutils.ExampleRateCard2).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard2).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard1),
			models.CadencedModel{
				ActiveFrom: oneMonthLater, // will only affect the second phase
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Now let's apply the diff, then restore the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should restore a single RateCard by undoing the effects of the addon RateCard restoring original entitlement template", withDeps(func(t *testing.T, deps *tcDeps) {
		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard5), // will add 50 usage
			models.CadencedModel{
				ActiveFrom: now,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Now let's apply the diff, then restore the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should restore a single RateCard and delete subsequent created RateCards", withDeps(func(t *testing.T, deps *tcDeps) {
		oneMonth := datetime.MustParseDuration(t, "P1M")

		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard5), // will add 50 usage
			models.CadencedModel{
				ActiveFrom: now,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Let's change the spec so the one item ends a bit early
		spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth
		ogSpec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth

		// Now let's apply the diff, then restore the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should restore a single RateCard and then delete it in the later step", withDeps(func(t *testing.T, deps *tcDeps) {
		// First we add apply an AddonRateCard once,
		// then twice,
		// then remove it once, (which should leave the first addition in place)
		// then again (which should remove the added items)
		oneMonth := datetime.MustParseDuration(t, "P1M")

		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard5), // will add 50 usage
			models.CadencedModel{
				ActiveFrom: now,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// We'll need an extra copy of it for the comparisons
		sView, err := deps.deps.SubscriptionService.GetView(context.Background(), env.subView.Subscription.NamespacedID)
		require.NoError(t, err)

		intermediateSpec := sView.AsSpec()

		// Let's change the spec so the one item ends a bit early
		spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth
		ogSpec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth
		intermediateSpec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &oneMonth

		// Now let's apply the diff twice
		err = spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Now let's restore it once
		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = intermediateSpec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		subscriptiontestutils.SpecsEqual(t, intermediateSpec, spec)

		// Now let's restore it once more
		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should re-combine single split item when restoring", withDeps(func(t *testing.T, deps *tcDeps) {
		oneMonth := datetime.MustParseDuration(t, "P1M")

		oneMonthLater, _ := oneMonth.AddTo(now)

		env := buildSubAndAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard5), // will add 50 usage
			models.CadencedModel{
				ActiveFrom: oneMonthLater,
				ActiveTo:   nil,
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Let's apply the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Now let's restore it
		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		// Let's assert that nothing changed
		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should not recombine different original items into one", withDeps(func(t *testing.T, deps *tcDeps) {
		twoMonths := datetime.MustParseDuration(t, "P2M")

		twoMonthsLater, _ := twoMonths.AddTo(now)

		// We need to create the resources separately as we will do an in-between edit
		p, a := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				SetMeta(productcatalog.PlanMeta{
					Name:           "Test Plan",
					Key:            "test_plan",
					Version:        1,
					Currency:       currency.USD,
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				}).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeSingle, &subscriptiontestutils.ExampleAddonRateCard5),
		)

		sView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		ogItem := sView.Spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()][0]

		require.NoError(t, ogItem.RateCard.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			m.Name = "Different Name for This one"
			return m, nil
		}))

		// Let's advance some time
		clock.SetTime(clock.Now().AddDate(0, 0, 1))

		// Let's make an edit to the sub effective at 1 month
		sView, err := deps.deps.WorkflowService.EditRunning(
			context.Background(),
			sView.Subscription.NamespacedID,
			[]subscription.Patch{
				patch.PatchRemoveItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleRateCard1.Key(),
				},
				patch.PatchAddItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleRateCard1.Key(),
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							Annotations: ogItem.Annotations,
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: "test_phase_1",
								ItemKey:  subscriptiontestutils.ExampleRateCard1.Key(),
								RateCard: ogItem.RateCard,
							},
						},
					},
				},
			},
			subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			},
		)
		require.NoError(t, err)

		sViewCpy, err := deps.deps.SubscriptionService.GetView(context.Background(), sView.Subscription.NamespacedID)
		require.NoError(t, err)

		// Let's add the addon
		subsAdd := subscriptiontestutils.CreateAddonForSubscription(t, &deps.deps, sView.Subscription.NamespacedID, a.NamespacedID, models.CadencedModel{
			ActiveFrom: twoMonthsLater,
			ActiveTo:   nil,
		})

		diffable, err := addondiff.GetDiffableFromAddon(sView, subsAdd)
		require.NoError(t, err)

		spec := sView.AsSpec()
		ogSpec := sViewCpy.AsSpec()

		// Now, let's apply the diff
		err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		err = spec.Apply(diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))

	t.Run("Should successfully restore boolean entitlement value", func(t *testing.T) {
		t.Run("When original item HAS boolean entitlement", withDeps(func(t *testing.T, deps *tcDeps) {
			env := buildSubAndAddon(
				t,
				&deps.deps,
				subscriptiontestutils.BuildTestPlanInput(t).
					AddPhase(nil, &subscriptiontestutils.ExampleRateCard4ForAddons).
					Build(),
				subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
				}, productcatalog.AddonInstanceTypeMultiple, &subscriptiontestutils.ExampleAddonRateCard6),
				models.CadencedModel{
					ActiveFrom: now,
					ActiveTo:   nil,
				},
			)

			spec := env.subView.Spec
			ogSpec := env.subViewCopy.Spec

			// Let's apply the diff twice
			err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			err = spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// Now let's restore it twice as well
			err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
		}))

		t.Run("When original item DOES NOT have boolean entitlement", withDeps(func(t *testing.T, deps *tcDeps) {
			env := buildSubAndAddon(
				t,
				&deps.deps,
				subscriptiontestutils.BuildTestPlanInput(t).
					AddPhase(nil, &subscriptiontestutils.ExampleRateCard5ForAddons).
					Build(),
				subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
				}, productcatalog.AddonInstanceTypeMultiple, &subscriptiontestutils.ExampleAddonRateCard6),
				models.CadencedModel{
					ActiveFrom: now,
					ActiveTo:   nil,
				},
			)

			spec := env.subView.Spec
			ogSpec := env.subViewCopy.Spec

			// Let's apply the diff twice
			err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			err = spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// Now let's restore it twice as well
			err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
		}))
	})

	t.Run("Should respect addon quantity when restoring", withDeps(func(t *testing.T, deps *tcDeps) {
		env := buildSubAndMultiInstanceAddon(
			t,
			&deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
			}, productcatalog.AddonInstanceTypeMultiple, &subscriptiontestutils.ExampleAddonRateCard5),
			[]subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				{
					ActiveFrom: now,
					Quantity:   3,
				},
			},
		)

		spec := env.subView.Spec
		ogSpec := env.subViewCopy.Spec

		// Let's apply the diff
		err := spec.Apply(env.diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		item := spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleAddonRateCard5.Key()][0]
		mt, err := item.RateCard.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)

		// Let's check it was properly applied
		require.Equal(t, 250.0, *mt.IssueAfterReset)

		// Now let's restore it
		err = spec.Apply(env.diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now})
		require.NoError(t, err)

		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)

		item = spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleAddonRateCard5.Key()][0]
		mt, err = item.RateCard.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)

		// Let's check it was properly removed
		require.Equal(t, 100.0, *mt.IssueAfterReset)
	}))

	t.Run("Should apply and remove same single instance addon 4 times in a row with slightly progressive times", withDeps(func(t *testing.T, deps *tcDeps) {
		// We will set up a subscriptionAddon which has a series of quantities set:
		// 1st, single quantity with 1 at t1
		// 2nd, two quantities, second with 0 at t2
		// 3rd, single quantity with 1 at t3
		// 4th, two quantities, second with 0 at t4
		// 5th, single quantity with 1 at t5
		// 6th, two quantities, second with 0 at t6
		// 7th, single quantity with 1 at t7
		// 8th, two quantities, second with 0 at t8

		// Each new quantity will only be added after a round of apply and restore

		// Let's start by creating a sub with some addon
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil,
					&subscriptiontestutils.ExampleRateCard1, // Flat price
					&productcatalog.UsageBasedRateCard{ // Dynamic price
						RateCardMeta: productcatalog.RateCardMeta{
							Key:  "dynamic_rc_1",
							Name: "Dynamic Rate Card 1",
							Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
								Mode: productcatalog.VolumeTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
										FlatPrice: &productcatalog.PriceTierFlatPrice{
											Amount: alpacadecimal.NewFromInt(10),
										},
									},
									{
										// UpToAmount: nil for infinity
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromInt(1),
										},
									},
								},
							}),
						},
						BillingCadence: subscriptiontestutils.ISOMonth,
					},
				).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				&productcatalog.UsageBasedRateCard{ // Dynamic price for addon
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "addon_dynamic_rc",
						Name: "Addon Dynamic Rate Card",
						Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
							Mode: productcatalog.VolumeTieredPrice,
							Tiers: []productcatalog.PriceTier{
								{
									// UpToAmount: nil for infinity
									UnitPrice: &productcatalog.PriceTierUnitPrice{
										Amount: alpacadecimal.NewFromFloat(0.5),
									},
								},
							},
						}),
					},
					BillingCadence: subscriptiontestutils.ISOMonth,
				},
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		subViewCopy, err := deps.deps.SubscriptionService.GetView(context.Background(), subView.Subscription.NamespacedID)
		require.NoError(t, err)

		ogSpec := subViewCopy.AsSpec()

		startTime := clock.Now()

		subsAdd := subscriptiontestutils.CreateMultiInstanceAddonForSubscription(
			t,
			&deps.deps,
			subView.Subscription.NamespacedID,
			add.NamespacedID,
			[]subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				{
					ActiveFrom: startTime,
					Quantity:   1,
				},
			},
		)

		spec := subView.AsSpec() // We can reuse this in all iterations as if everything works as intended it always gets restored

		for idx := range 8 {
			// Lets pass time
			clock.SetTime(clock.Now().Add(time.Minute))
			diff, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err, "failed to get diffable for iteration %d", idx)

			err = spec.Apply(diff.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err, "failed to apply for iteration %d", idx)

			err = spec.Apply(diff.GetRestores(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err, "failed to restore for iteration %d", idx)

			clock.SetTime(clock.Now().Add(time.Minute))

			// Finally lets toggle the quantity
			sAdd, err := deps.deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subsAdd.NamespacedID, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: clock.Now(),
				Quantity:   lo.Ternary(idx%2 == 0, 1, 0),
			})
			require.NoError(t, err, "failed to change quantity for iteration %d", idx)

			subsAdd = *sAdd
		}

		subscriptiontestutils.SpecsEqual(t, ogSpec, spec)
	}))
}

type buildRes struct {
	subView     *subscription.SubscriptionView
	subViewCopy *subscription.SubscriptionView
	diffable    addondiff.Diffable
}

// buildSubAndAddon is a test setup helper
func buildSubAndAddon(
	t *testing.T,
	deps *subscriptiontestutils.SubscriptionDependencies,
	planInp plan.CreatePlanInput,
	addonInp addon.CreateAddonInput,
	subsAddCadence models.CadencedModel,
) *buildRes {
	t.Helper()

	p, a := subscriptiontestutils.CreatePlanWithAddon(
		t,
		*deps,
		planInp,
		addonInp,
	)

	subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, deps, p, now)

	subView2, err := deps.SubscriptionService.GetView(context.Background(), subView.Subscription.NamespacedID)
	require.NoError(t, err)

	subsAdd := subscriptiontestutils.CreateAddonForSubscription(t, deps, subView.Subscription.NamespacedID, a.NamespacedID, subsAddCadence)

	diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
	require.NoError(t, err)

	return &buildRes{
		subView:     &subView,
		subViewCopy: &subView2,
		diffable:    diffable,
	}
}

func buildSubAndMultiInstanceAddon(
	t *testing.T,
	deps *subscriptiontestutils.SubscriptionDependencies,
	planInp plan.CreatePlanInput,
	addonInp addon.CreateAddonInput,
	quants []subscriptionaddon.CreateSubscriptionAddonQuantityInput,
) *buildRes {
	t.Helper()

	p, a := subscriptiontestutils.CreatePlanWithAddon(
		t,
		*deps,
		planInp,
		addonInp,
	)

	subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, deps, p, now)

	subView2, err := deps.SubscriptionService.GetView(context.Background(), subView.Subscription.NamespacedID)
	require.NoError(t, err)

	subsAdd := subscriptiontestutils.CreateMultiInstanceAddonForSubscription(t, deps, subView.Subscription.NamespacedID, a.NamespacedID, quants)

	diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
	require.NoError(t, err)

	return &buildRes{
		subView:     &subView,
		subViewCopy: &subView2,
		diffable:    diffable,
	}
}
