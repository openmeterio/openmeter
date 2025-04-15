package addondiff_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var now, _ = time.Parse(time.RFC3339, "2025-04-01T00:00:00Z")

func TestApply(t *testing.T) {
	type tcDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	// TODO: we could write purer tests here (without depending on the services) but this is simply more convenient for now
	runWithDeps := func(t *testing.T, fn func(t *testing.T, deps *tcDeps)) {
		clock.SetTime(now)
		defer clock.ResetTime()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)

		fn(t, &tcDeps{
			deps: deps,
		})
	}

	t.Run("Should return a no-op when there's no quantities listed", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t))

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID, subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			}))

			// Let's manually overwrite the quantity, it will be fine now
			subsAdd.Quantities = timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{})

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			view2, err := deps.deps.SubscriptionService.GetView(context.Background(), subView.Subscription.NamespacedID)
			require.NoError(t, err)

			subscriptiontestutils.SpecsEqual(t, spec, view2.Spec)
		})
	})

	t.Run("Should add the new item to the subscription", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t))

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					productcatalog.EffectivePeriod{
						EffectiveFrom: &now,
						EffectiveTo:   nil,
					},
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard2.Clone(),
				),
			)

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// New item should be present in all phases
			for _, p := range spec.GetSortedPhases() {
				b, _ := json.MarshalIndent(p, "", "  ")

				itemHistory, ok := p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard2.Key()]
				require.True(t, ok, "item history missing in phase %s, got %+v, \nfull: %s", p.PhaseKey, lo.Keys(p.ItemsByKey), string(b))

				// It should have a single entry in all phases
				require.Len(t, itemHistory, 1)
				item := itemHistory[0]

				// It should last for the entire phase
				pCad, _ := spec.GetPhaseCadence(p.PhaseKey)
				cad := item.GetCadence(pCad)
				require.True(t, cad.Equal(pCad))

				// It should not have any overrides
				require.Nil(t, item.ActiveFromOverrideRelativeToPhaseStart)
				require.Nil(t, item.ActiveToOverrideRelativeToPhaseStart)

				// It should not have any owner annotations
				require.Empty(t, subscription.AnnotationParser.ListOwnerSubSystems(item.Annotations))

				// It should have the proper RateCard info
				require.True(t, subscriptiontestutils.ExampleAddonRateCard2.Equal(item.RateCard))
			}
		})
	})

	t.Run("Should add multiple instances of new item to the subscription", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t))

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					productcatalog.EffectivePeriod{
						EffectiveFrom: &now,
						EffectiveTo:   nil,
					},
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard3.Clone(),
				),
			)

			// Let's just overwrite the quantity, it will be fine now
			val := subsAdd.Quantities.GetAt(0).GetValue()

			val.Quantity = 3

			subsAdd.Quantities = timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{
				val.AsTimed(),
			})

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// New item should be present in all phases
			for _, p := range spec.GetSortedPhases() {
				b, _ := json.MarshalIndent(p, "", "  ")

				itemHistory, ok := p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard3.Key()]
				require.True(t, ok, "item history missing in phase %s, got %+v, \nfull: %s", p.PhaseKey, lo.Keys(p.ItemsByKey), string(b))

				// It should have a single entry in all phases
				require.Len(t, itemHistory, 1)
				item := itemHistory[0]

				// It should last for the entire phase
				pCad, _ := spec.GetPhaseCadence(p.PhaseKey)
				cad := item.GetCadence(pCad)
				require.True(t, cad.Equal(pCad))

				// It should not have any overrides
				require.Nil(t, item.ActiveFromOverrideRelativeToPhaseStart)
				require.Nil(t, item.ActiveToOverrideRelativeToPhaseStart)

				// It should not have any owner annotations
				require.Empty(t, subscription.AnnotationParser.ListOwnerSubSystems(item.Annotations))

				pr, _ := item.RateCard.AsMeta().Price.AsFlat()
				require.Equal(t, int64(300), pr.Amount.IntPart())

				// It should have the proper RateCard info, which is price * quantity + bool access
				targetMeta := subscriptiontestutils.ExampleAddonRateCard3.AsMeta().Clone()

				targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100 * 3),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})

				target := productcatalog.FlatFeeRateCard{
					RateCardMeta:   targetMeta,
					BillingCadence: subscriptiontestutils.ExampleAddonRateCard3.BillingCadence,
				}

				require.True(t, target.Equal(item.RateCard))
			}
		})
	})

	t.Run("Should add item for defined cadence of the addon", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t))

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.AddDate(0, 0, 3)),
				EffectiveTo:   lo.ToPtr(now.AddDate(0, 1, 8)),
			}

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					effPer,
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard3.Clone(),
				),
			)

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// We should have items in the first 2 phases, then none in later phases
			for pIdx, p := range spec.GetSortedPhases() {
				if pIdx >= 2 {
					require.NotContains(t, p.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard3.Key())

					continue
				}

				b, _ := json.MarshalIndent(p, "", "  ")

				items, ok := p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard3.Key()]
				require.True(t, ok, "item missing in phase %s, got %+v, \nfull: %s", p.PhaseKey, lo.Keys(p.ItemsByKey), b)
				require.Len(t, items, 1)

				item := items[0]

				pCad, err := spec.GetPhaseCadence(p.PhaseKey)
				require.NoError(t, err)
				cad := item.GetCadence(pCad)

				// Items should always fall in the effective period
				require.True(t, effPer.AsPeriod().IsSupersetOf(cad.AsPeriod()), "phase %s: item %s not in effective period %s", p.PhaseKey, cad.AsPeriod(), effPer.AsPeriod())

				// And should always be in the phase
				require.True(t, pCad.AsPeriod().IsSupersetOf(cad.AsPeriod()), "phase %s: item %s not in phase %s", p.PhaseKey, cad.AsPeriod(), pCad.AsPeriod())

				// Now lets be exact on how this should look
				switch pIdx {
				case 0:
					require.True(t, cad.ActiveFrom.Equal(*effPer.EffectiveFrom))
					require.True(t, cad.ActiveTo.Equal(*pCad.ActiveTo))
				case 1:
					require.True(t, cad.ActiveFrom.Equal(pCad.ActiveFrom))
					require.True(t, cad.ActiveTo.Equal(*effPer.EffectiveTo))
				}
			}
		})
	})

	t.Run("Should update an existing Item that fills its entire phase", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(lo.ToPtr(testutils.GetISODuration(t, "P1M")), subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.AddDate(0, 0, 0)),
				EffectiveTo:   nil,
			}

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					effPer,
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard4.Clone(),
				),
			)

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// We should have the updated items in both phases
			for _, p := range spec.GetSortedPhases() {
				items, ok := p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()]
				require.True(t, ok, "item missing in phase %s, got %+v", p.PhaseKey, lo.Keys(p.ItemsByKey))
				require.Len(t, items, 1)

				item := items[0]

				pc, err := spec.GetPhaseCadence(p.PhaseKey)
				require.NoError(t, err)

				cad := item.GetCadence(pc)

				require.True(t, cad.ActiveFrom.Equal(pc.ActiveFrom))

				targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
				targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100 * 2),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})

				target := &productcatalog.FlatFeeRateCard{
					RateCardMeta:   targetMeta,
					BillingCadence: subscriptiontestutils.ExampleAddonRateCard4.BillingCadence,
				}

				b1, _ := json.MarshalIndent(item.RateCard, "", "  ")
				b2, _ := json.MarshalIndent(target, "", "  ")

				require.True(
					t,
					target.Equal(item.RateCard.Clone()),
					"phase %s: item %s not equal to target %s",
					p.PhaseKey,
					b1,
					b2,
				)
			}
		})
	})

	t.Run("Should partially update an existing item in accordance with the cadence", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(lo.ToPtr(testutils.GetISODuration(t, "P1M")), subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.AddDate(0, 0, 5)),
				EffectiveTo:   nil,
			}

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					effPer,
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard4.Clone(),
				),
			)

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// Let's check the first phase looks as we should
			p1 := spec.GetSortedPhases()[0]

			b, _ := json.MarshalIndent(p1, "", "  ")

			require.Contains(t, p1.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard4.Key(), "looked for: %s, full: %s", subscriptiontestutils.ExampleAddonRateCard4.Key(), string(b))
			require.Len(t, p1.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()], 2)

			items := p1.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()]

			pCad, err := spec.GetPhaseCadence(p1.PhaseKey)
			require.NoError(t, err)

			// First item should be from 1 to 5, second from 5 to end of phase
			cad1 := items[0].GetCadence(pCad)
			require.True(t, cad1.ActiveFrom.Equal(pCad.ActiveFrom))
			require.True(t, cad1.ActiveTo.Equal(now.AddDate(0, 0, 5)))

			cad2 := items[1].GetCadence(pCad)
			require.True(t, cad2.ActiveFrom.Equal(now.AddDate(0, 0, 5)))
			require.True(t, cad2.ActiveTo.Equal(*pCad.ActiveTo))

			// First item should have the original rate card
			b1, _ := json.MarshalIndent(items[0].RateCard, "", "  ")
			b2, _ := json.MarshalIndent(subscriptiontestutils.ExampleRateCard3ForAddons, "", "  ")

			require.True(
				t,
				subscriptiontestutils.ExampleRateCard3ForAddons.Equal(items[0].RateCard),
				"phase %s: item %s not equal to target %s",
				p1.PhaseKey,
				b1,
				b2,
			)
			// Second item should have the updated rate card
			targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
			targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100 * 2),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})

			target := &productcatalog.FlatFeeRateCard{
				RateCardMeta:   targetMeta,
				BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
			}

			b1, _ = json.MarshalIndent(items[1].RateCard, "", "  ")
			b2, _ = json.MarshalIndent(target, "", "  ")

			require.True(
				t,
				target.Equal(items[1].RateCard.Clone()),
				"phase %s: item %s not equal to target %s",
				p1.PhaseKey,
				b1,
				b2,
			)
		})
	})

	t.Run("Should partially update an existing item in accordance with the cadence - but for multi instance", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(lo.ToPtr(testutils.GetISODuration(t, "P1M")), subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.AddDate(0, 0, 5)),
				EffectiveTo:   nil,
			}

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					effPer,
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard4.Clone(),
				),
			)

			// Lets just overwrite the quantity, it will be fine now
			val := subsAdd.Quantities.GetAt(0).GetValue()

			val.Quantity = 4

			subsAdd.Quantities = timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{
				val.AsTimed(),
			})

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// Let's check the first phase looks as we should
			{
				p1 := spec.GetSortedPhases()[0]
				require.Contains(t, p1.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard4.Key())
				require.Len(t, p1.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()], 2)

				items := p1.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()]

				pCad, err := spec.GetPhaseCadence(p1.PhaseKey)
				require.NoError(t, err)

				// First item should be from 1 to 5, second from 5 to end of phase
				cad1 := items[0].GetCadence(pCad)
				require.True(t, cad1.ActiveFrom.Equal(pCad.ActiveFrom))
				require.True(t, cad1.ActiveTo.Equal(now.AddDate(0, 0, 5)))

				cad2 := items[1].GetCadence(pCad)
				require.True(t, cad2.ActiveFrom.Equal(now.AddDate(0, 0, 5)))
				require.True(t, cad2.ActiveTo.Equal(*pCad.ActiveTo))

				// First item should have the original rate card
				b1, _ := json.MarshalIndent(items[0].RateCard, "", "  ")
				b2, _ := json.MarshalIndent(subscriptiontestutils.ExampleRateCard3ForAddons, "", "  ")

				require.True(
					t,
					subscriptiontestutils.ExampleRateCard3ForAddons.Equal(items[0].RateCard),
					"phase %s: item %s not equal to target %s",
					p1.PhaseKey,
					b1,
					b2,
				)
				// Second item should have the updated rate card
				targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
				targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100 * (1 + 4)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})

				target := &productcatalog.FlatFeeRateCard{
					RateCardMeta:   targetMeta,
					BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
				}

				b1, _ = json.MarshalIndent(items[1].RateCard, "", "  ")
				b2, _ = json.MarshalIndent(target, "", "  ")

				require.True(
					t,
					target.Equal(items[1].RateCard.Clone()),
					"phase %s: item %s not equal to target %s",
					p1.PhaseKey,
					b1,
					b2,
				)
			}

			// Lets check the second phase
			{
				p2 := spec.GetSortedPhases()[1]
				require.Contains(t, p2.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard4.Key())
				items := p2.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()]

				require.Len(t, items, 1)

				item := items[0]

				pCad, err := spec.GetPhaseCadence(p2.PhaseKey)
				require.NoError(t, err)

				cad := item.GetCadence(pCad)
				require.True(t, cad.ActiveFrom.Equal(pCad.ActiveFrom))
				require.Nil(t, cad.ActiveTo)

				targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
				targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100 * (1 + 4)),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				})

				target := &productcatalog.FlatFeeRateCard{
					RateCardMeta:   targetMeta,
					BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
				}

				b1, _ := json.MarshalIndent(item.RateCard, "", "  ")
				b2, _ := json.MarshalIndent(target, "", "  ")

				require.True(
					t,
					target.Equal(item.RateCard.Clone()),
					"phase %s: item %s not equal to target %s",
					p2.PhaseKey,
					b1,
					b2,
				)
			}
		})
	})

	t.Run("Should guarantee access is continuous across changing items", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			t0 := now
			clock.FreezeTime(t0)

			defer clock.UnFreeze()

			t1 := now.AddDate(0, 0, 5)

			t2 := t1.AddDate(0, 1, 5)

			t3 := t2.AddDate(0, 0, 6)

			t4 := t3.AddDate(0, 0, 9)

			_, err := deps.deps.SubscriptionService.GetView(context.Background(), subView.Subscription.NamespacedID)
			require.NoError(t, err)

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(t1),
				EffectiveTo:   lo.ToPtr(t4),
			}

			_, subsAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID,
				subscriptiontestutils.BuildAddonForTesting(t,
					effPer,
					productcatalog.AddonInstanceTypeSingle,
					subscriptiontestutils.ExampleAddonRateCard4.Clone(),
				),
			)

			diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
			require.NoError(t, err)

			spec := subView.Spec

			// Let's just manipulate the spec object directly
			ogItem := spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleRateCard3ForAddons.Key()][0]
			spec.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleRateCard3ForAddons.Key()] = []*subscription.SubscriptionItemSpec{
				// First, before gap
				{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: spec.Phases["test_phase_1"].PhaseKey,
							ItemKey:  subscriptiontestutils.ExampleRateCard3ForAddons.Key(),
							RateCard: subscriptiontestutils.ExampleRateCard3ForAddons.Clone(),
						},
						CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
							ActiveToOverrideRelativeToPhaseStart: lo.ToPtr(isodate.Between(t0, t2)),
						},
						Annotations: ogItem.Annotations,
					},
				},
				// Second, after gap
				{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: spec.Phases["test_phase_1"].PhaseKey,
							ItemKey:  subscriptiontestutils.ExampleRateCard3ForAddons.Key(),
							RateCard: subscriptiontestutils.ExampleRateCard3ForAddons.Clone(),
						},
						CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
							ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(isodate.Between(t0, t3)),
						},
						Annotations: ogItem.Annotations,
					},
				},
			}

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			// Now let's make sure that we have 5 items:
			key := subscriptiontestutils.ExampleRateCard3ForAddons.Key()

			phase := spec.GetSortedPhases()[0]
			pCad, err := spec.GetPhaseCadence(phase.PhaseKey)
			require.NoError(t, err)

			require.Contains(t, phase.ItemsByKey, key)
			require.Len(t, phase.ItemsByKey[key], 5)
			items := phase.ItemsByKey[key]
			// [t0-t1]: ExampleRateCard3ForAddons
			require.True(t, items[0].GetCadence(pCad).ActiveFrom.Equal(t0))
			require.True(t, items[0].GetCadence(pCad).ActiveTo.Equal(t1))

			// [t1-t2]: ExampleRateCard3ForAddons + ExampleAddonRateCard4
			require.True(t, items[1].GetCadence(pCad).ActiveFrom.Equal(t1))
			require.True(t, items[1].GetCadence(pCad).ActiveTo.Equal(t2))

			// [t2-t3]: ExampleAddonRateCard4
			require.True(t, items[2].GetCadence(pCad).ActiveFrom.Equal(t2))
			require.True(t, items[2].GetCadence(pCad).ActiveTo.Equal(t3))

			// [t3-t4]: ExampleRateCard3ForAddons + ExampleAddonRateCard4
			require.True(t, items[3].GetCadence(pCad).ActiveFrom.Equal(t3))
			require.True(t, items[3].GetCadence(pCad).ActiveTo.Equal(t4))

			// [t4-open]: ExampleRateCard3ForAddons
			require.True(t, items[4].GetCadence(pCad).ActiveFrom.Equal(t4))
			require.Nil(t, items[4].GetCadence(pCad).ActiveTo)
		})
	})

	t.Run("Should create multiple rate cards in the addon", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
				EffectiveTo:   nil,
			}

			_, subAdd := createAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID, subscriptiontestutils.BuildAddonForTesting(t,
				effPer,
				productcatalog.AddonInstanceTypeMultiple,
				subscriptiontestutils.ExampleAddonRateCard4.Clone(),
				subscriptiontestutils.ExampleAddonRateCard3.Clone(),
			))

			diffable, err := addondiff.GetDiffableFromAddon(subView, subAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			p, ok := spec.Phases["test_phase_1"]
			require.True(t, ok)

			pCad, err := spec.GetPhaseCadence(p.PhaseKey)
			require.NoError(t, err)

			// Should update ExampleRateCard3ForAddons with ExampleAddonRateCard4
			require.Contains(t, p.ItemsByKey, subscriptiontestutils.ExampleRateCard3ForAddons.Key())

			items := p.ItemsByKey[subscriptiontestutils.ExampleRateCard3ForAddons.Key()]
			require.Len(t, items, 1)

			item := items[0]

			require.True(t, item.GetCadence(pCad).ActiveFrom.Equal(now))
			require.Nil(t, item.GetCadence(pCad).ActiveTo)

			targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
			targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100 * 2),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})

			target := &productcatalog.FlatFeeRateCard{
				RateCardMeta:   targetMeta,
				BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
			}

			b1, _ := json.MarshalIndent(item.RateCard, "", "  ")
			b2, _ := json.MarshalIndent(target, "", "  ")

			require.True(t, target.Equal(item.RateCard.Clone()), "phase %s: item %s not equal to target %s", p.PhaseKey, b1, b2)

			// Should create ExampleAddonRateCard3
			require.Contains(t, p.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard3.Key())

			items = p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard3.Key()]
			require.Len(t, items, 1)

			item = items[0]

			require.True(t, item.GetCadence(pCad).ActiveFrom.Equal(now))
			require.Nil(t, item.GetCadence(pCad).ActiveTo)

			targetMeta = subscriptiontestutils.ExampleAddonRateCard3.AsMeta().Clone()
			targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})

			target = &productcatalog.FlatFeeRateCard{
				RateCardMeta:   targetMeta,
				BillingCadence: subscriptiontestutils.ExampleAddonRateCard3.BillingCadence,
			}

			b1, _ = json.MarshalIndent(item.RateCard, "", "  ")
			b2, _ = json.MarshalIndent(target, "", "  ")

			require.True(t, target.Equal(item.RateCard.Clone()), "phase %s: item %s not equal to target %s", p.PhaseKey, b1, b2)
		})
	})
}

func TestApplyWithMultiInstance(t *testing.T) {
	type tcDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	runWithDeps := func(t *testing.T, fn func(t *testing.T, deps *tcDeps)) {
		clock.SetTime(now)
		defer clock.ResetTime()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)

		fn(t, &tcDeps{
			deps: deps,
		})
	}

	t.Run("Should create items in phase according to quantities", func(t *testing.T) {
		runWithDeps(t, func(t *testing.T, deps *tcDeps) {
			_, subView := createSubFromPlan(t, &deps.deps, subscriptiontestutils.BuildTestPlan(t).
				AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).
				Build())

			t0 := now

			t1 := t0.AddDate(0, 0, 1)

			t2 := t1.AddDate(0, 0, 1)

			effPer := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
				EffectiveTo:   nil,
			}

			_, subAdd := createMultiInstanceAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID, subscriptiontestutils.BuildAddonForTesting(t,
				effPer,
				productcatalog.AddonInstanceTypeMultiple,
				subscriptiontestutils.ExampleAddonRateCard4.Clone(),
			), []subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				{
					ActiveFrom: t1,
					Quantity:   1,
				},
				{
					ActiveFrom: t2,
					Quantity:   2,
				},
			})

			diffable, err := addondiff.GetDiffableFromAddon(subView, subAdd)
			require.NoError(t, err)

			spec := subView.Spec

			err = spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})
			require.NoError(t, err)

			p, ok := spec.Phases["test_phase_1"]
			require.True(t, ok)

			pCad, err := spec.GetPhaseCadence(p.PhaseKey)
			require.NoError(t, err)

			require.Contains(t, p.ItemsByKey, subscriptiontestutils.ExampleAddonRateCard4.Key())
			require.Len(t, p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()], 3)

			items := p.ItemsByKey[subscriptiontestutils.ExampleAddonRateCard4.Key()]

			// First item: from t0 to t1, ExampleRateCard3ForAddons
			require.True(t, items[0].GetCadence(pCad).ActiveFrom.Equal(t0))
			require.True(t, items[0].GetCadence(pCad).ActiveTo.Equal(t1))
			require.True(t, items[0].RateCard.Equal(subscriptiontestutils.ExampleRateCard3ForAddons.Clone()))

			// Second item: from t1 to t2, ExampleRateCard3ForAddons + ExampleAddonRateCard4
			require.True(t, items[1].GetCadence(pCad).ActiveFrom.Equal(t1))
			require.True(t, items[1].GetCadence(pCad).ActiveTo.Equal(t2))

			targetMeta := subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()
			targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100 * 2),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})

			target := &productcatalog.FlatFeeRateCard{
				RateCardMeta:   targetMeta,
				BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
			}

			b1, _ := json.MarshalIndent(items[1].RateCard, "", "  ")
			b2, _ := json.MarshalIndent(target, "", "  ")

			require.True(t, target.Equal(items[1].RateCard.Clone()), "phase %s: item %s not equal to target %s", p.PhaseKey, b1, b2)

			// Third item: from t2 to open, ExampleRateCard3ForAddons + 2 x ExampleAddonRateCard4
			require.True(t, items[2].GetCadence(pCad).ActiveFrom.Equal(t2))
			require.Nil(t, items[2].GetCadence(pCad).ActiveTo)

			targetMeta = subscriptiontestutils.ExampleRateCard3ForAddons.AsMeta().Clone()

			targetMeta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100 * 3),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})

			target = &productcatalog.FlatFeeRateCard{
				RateCardMeta:   targetMeta,
				BillingCadence: subscriptiontestutils.ExampleRateCard3ForAddons.BillingCadence,
			}

			b1, _ = json.MarshalIndent(items[2].RateCard, "", "  ")
			b2, _ = json.MarshalIndent(target, "", "  ")

			require.True(t, target.Equal(items[2].RateCard.Clone()), "phase %s: item %s not equal to target %s", p.PhaseKey, b1, b2)
		})
	})
}

func createSubFromPlan(t *testing.T, deps *subscriptiontestutils.SubscriptionDependencies, planInp plan.CreatePlanInput) (subscription.Plan, subscription.SubscriptionView) {
	deps.FeatureConnector.CreateExampleFeatures(t)
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	plan := deps.PlanHelper.CreatePlan(t, planInp)
	subView, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  cust.Namespace,
		CustomerID: cust.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name: "test",
			Timing: subscription.Timing{
				Custom: &now,
			},
		},
	}, plan)
	require.NoError(t, err)

	return plan, subView
}

func createMultiInstanceAddonForSub(t *testing.T, deps *subscriptiontestutils.SubscriptionDependencies, subID models.NamespacedID, addonInp addon.CreateAddonInput, quants []subscriptionaddon.CreateSubscriptionAddonQuantityInput) (addon.Addon, subscriptionaddon.SubscriptionAddon) {
	t.Helper()

	add := deps.AddonService.CreateTestAddon(t, addonInp)

	subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
		AddonID:        add.ID,
		SubscriptionID: subID.ID,
		RateCards: lo.Map(add.RateCards, func(rc addon.RateCard, _ int) subscriptionaddon.CreateSubscriptionAddonRateCardInput {
			return subscriptionaddon.CreateSubscriptionAddonRateCardInput{
				AddonRateCardID: rc.ID,
			}
		}),
		InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: quants[0].ActiveFrom,
			Quantity:   quants[0].Quantity,
		},
	})
	require.NoError(t, err)

	if len(quants) == 1 {
		return add, *subAdd
	}

	for _, quant := range quants[1:] {
		subAdd, err = deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, quant)
		require.NoError(t, err)
	}

	return add, *subAdd
}

// this is a bit hacky, we reuse the addon's effective period as cadence for the subscriptionaddon
func createAddonForSub(t *testing.T, deps *subscriptiontestutils.SubscriptionDependencies, subID models.NamespacedID, addonInp addon.CreateAddonInput) (addon.Addon, subscriptionaddon.SubscriptionAddon) {
	t.Helper()

	quants := []subscriptionaddon.CreateSubscriptionAddonQuantityInput{
		{
			ActiveFrom: *addonInp.EffectivePeriod.EffectiveFrom,
			Quantity:   1,
		},
	}

	if addonInp.EffectiveTo != nil {
		quants = append(quants, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: *addonInp.EffectiveTo,
			Quantity:   0,
		})
	}

	return createMultiInstanceAddonForSub(t, deps, subID, addonInp, quants)
}
