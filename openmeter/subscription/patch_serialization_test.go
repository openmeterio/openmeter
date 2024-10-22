package subscription_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/applieddiscount"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func assertSame(t *testing.T, in json.Marshaler) {
	out, err := in.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	parsed, err := subscription.Deserialize(out)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	assert.Equal(t, in, parsed)
}

func TestShouldSerializeAndDeserialize(t *testing.T) {
	t.Run("Should be the same for PatchAddPhase", func(t *testing.T) {
		p := &subscription.PatchAddPhase{
			PhaseKey: "asd",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "asd",
					StartAfter: datex.FromDuration(0),
				},
				CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{
					CreateDiscountInput: &applieddiscount.Spec{
						PhaseKey:  "asd",
						AppliesTo: []string{"asd"},
					},
				},
			},
		}

		assertSame(t, p)

		p2 := &subscription.PatchAddPhase{
			PhaseKey: "asd",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "asd",
					StartAfter: datex.FromDuration(0),
				},
			},
		}

		assertSame(t, p2)
	})
	t.Run("Should be the same for PatchRemovePhase", func(t *testing.T) {
		p := &subscription.PatchRemovePhase{
			PhaseKey: "asd",
		}

		assertSame(t, p)
	})
	t.Run("Should be the same for PatchExtendPhase", func(t *testing.T) {
		p := &subscription.PatchExtendPhase{
			PhaseKey: "asd",
			Duration: datex.FromDuration(time.Hour),
		}

		assertSame(t, p)
	})
	t.Run("Should be the same for PatchAddItem", func(t *testing.T) {
		p := &subscription.PatchAddItem{
			PhaseKey: "asd",
			ItemKey:  "asd2",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey:   "asd",
					ItemKey:    "asd2",
					FeatureKey: lo.ToPtr("feature-1"),
					CreatePriceInput: &price.Spec{
						PhaseKey: "asd",
						ItemKey:  "asd2",
						Value:    "1.0",
					},
				},
			},
		}

		assertSame(t, p)

		p2 := &subscription.PatchAddItem{
			PhaseKey: "asd",
			ItemKey:  "asd2",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: "asd",
					ItemKey:  "asd2",
				},
			},
		}

		assertSame(t, p2)
	})

	t.Run("Should be the same for PatchRemoveItem", func(t *testing.T) {
		p := &subscription.PatchRemoveItem{
			PhaseKey: "asd",
			ItemKey:  "asd2",
		}

		assertSame(t, p)
	})
}
