package subscription_test

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTransformation(t *testing.T) {
	p1 := subscription.PatchAddItem{
		PhaseKey: "phase1",
		ItemKey:  "item1",
		CreateInput: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
				PhaseKey: "phase1",
				ItemKey:  "item1",
			},
		},
	}

	p2 := subscription.PatchRemoveItem{
		PhaseKey: "phase1",
		ItemKey:  "item2",
	}
	t.Run("Happy path", func(t *testing.T) {
		currTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		out, err := subscription.TransformPatchesForRepository([]subscription.Patch{
			p1,
			p2,
		}, currTime)
		assert.NoError(t, err)
		assert.Len(t, out, 2)
		assert.Equal(t, currTime, out[0].AppliedAt)
		assert.Equal(t, currTime, out[1].AppliedAt)
		assert.Equal(t, 0, out[0].BatchIndex)
		assert.Equal(t, 1, out[1].BatchIndex)
		assert.Equal(t, "/phases/phase1/items/item1", string(out[0].Path()))
		assert.Equal(t, "/phases/phase1/items/item2", string(out[1].Path()))
	})
}
