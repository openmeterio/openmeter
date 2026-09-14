package patch_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestDiffItems(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := start.Add(10 * 24 * time.Hour)
	key := subscriptiontestutils.ExampleFeatureKey

	t.Run("unchanged feature with a hydrated reference has no patches", func(t *testing.T) {
		// given the same catalog item, represented by a key on the subscription
		current, _ := getDefaultSpec(t, start)
		target, _ := getDefaultSpec(t, start)
		rc := target.Phases["test_phase_1"].ItemsByKey[key][0].RateCard
		require.NoError(t, rc.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Feature.ID = lo.ToPtr("resolved-feature-id")
			return meta, nil
		}))
		// when comparing from the middle of the first phase
		patches, err := patch.DiffItems(patch.DiffItemsInput{Current: *current, Target: *target, At: at})
		// then reference expansion alone does not interrupt service
		require.NoError(t, err)
		require.Empty(t, patches)
	})

	t.Run("changed current item retains history and starts one new version", func(t *testing.T) {
		// given a current item and a target with different commercial content
		current, _ := getDefaultSpec(t, start)
		target, _ := getDefaultSpec(t, start)
		rc := target.Phases["test_phase_1"].ItemsByKey[key][0].RateCard
		require.NoError(t, rc.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Name = "amended"
			return meta, nil
		}))
		// when applying the generated patches
		patches, err := patch.DiffItems(patch.DiffItemsInput{Current: *current, Target: *target, At: at})
		require.NoError(t, err)
		require.Len(t, patches, 1)
		require.NoError(t, current.ApplyMany(lo.Map(patches, subscription.ToApplies), subscription.ApplyContext{CurrentTime: at}))
		// then history stays at index zero and the amendment starts at the effective time
		items := current.Phases["test_phase_1"].ItemsByKey[key]
		require.Len(t, items, 2)
		cadence, err := current.GetPhaseCadence("test_phase_1")
		require.NoError(t, err)
		require.Equal(t, start, items[0].GetCadence(cadence).ActiveFrom)
		require.Equal(t, at, *items[0].GetCadence(cadence).ActiveTo)
		require.Equal(t, at, items[1].GetCadence(cadence).ActiveFrom)
		require.Equal(t, "amended", items[1].RateCard.AsMeta().Name)
	})

	t.Run("unchanged active prefix is retained when only a future version differs", func(t *testing.T) {
		// given a future quantity-like change on the current item
		current, _ := getDefaultSpec(t, start)
		target, _ := getDefaultSpec(t, start)
		boundary := start.Add(20 * 24 * time.Hour)
		offset := datetime.ISODurationBetween(start, boundary)
		original := current.Phases["test_phase_1"].ItemsByKey[key][0]
		for _, spec := range []*subscription.SubscriptionSpec{current, target} {
			first := spec.Phases["test_phase_1"].ItemsByKey[key][0]
			next := *first
			next.RateCard = first.RateCard.Clone()
			first.ActiveToOverrideRelativeToPhaseStart = &offset
			next.ActiveFromOverrideRelativeToPhaseStart = &offset
			spec.Phases["test_phase_1"].ItemsByKey[key] = append(spec.Phases["test_phase_1"].ItemsByKey[key], &next)
		}
		require.NoError(t, target.Phases["test_phase_1"].ItemsByKey[key][1].RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Name = "future amendment"
			return meta, nil
		}))
		// when migrating before the future change
		patches, err := patch.DiffItems(patch.DiffItemsInput{Current: *current, Target: *target, At: at})
		require.NoError(t, err)
		require.NoError(t, current.ApplyMany(lo.Map(patches, subscription.ToApplies), subscription.ApplyContext{CurrentTime: at}))
		// then no extra boundary is introduced at migration time
		items := current.Phases["test_phase_1"].ItemsByKey[key]
		require.Len(t, items, 2)
		require.Equal(t, original, items[0])
		require.Equal(t, "future amendment", items[1].RateCard.AsMeta().Name)
	})
}
