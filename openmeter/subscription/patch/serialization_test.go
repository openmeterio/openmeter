package patch_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datex"
)

func assertSame(t *testing.T, in json.Marshaler) {
	out, err := in.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	parsed, err := patch.Deserialize(out)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	assert.Equal(t, in, parsed)
}

func TestShouldSerializeAndDeserialize(t *testing.T) {
	t.Run("Should be the same for PatchAddPhase", func(t *testing.T) {
		p := &patch.PatchAddPhase{
			PhaseKey: "asd",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "asd",
					StartAfter: datex.FromDuration(0),
				},
				CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
			},
		}

		assertSame(t, p)

		p2 := &patch.PatchAddPhase{
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
		p := &patch.PatchRemovePhase{
			PhaseKey: "asd",
			RemoveInput: subscription.RemoveSubscriptionPhaseInput{
				Shift: subscription.RemoveSubscriptionPhaseShiftPrev,
			},
		}

		assertSame(t, p)
	})
	t.Run("Should be the same for PatchStretchPhase", func(t *testing.T) {
		p := &patch.PatchStretchPhase{
			PhaseKey: "asd",
			Duration: datex.FromDuration(time.Hour),
		}

		assertSame(t, p)
	})
	t.Run("Should be the same for PatchAddItem", func(t *testing.T) {
		fp := productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(int64(100)),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}

		pp := productcatalog.Price{}
		pp.FromFlat(fp)

		p := &patch.PatchAddItem{
			PhaseKey: "asd",
			ItemKey:  "asd2",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "asd",
						ItemKey:  "asd2",
						RateCard: subscription.RateCard{
							Name:        "asdname",
							Description: lo.ToPtr("asddesc"),
							TaxConfig: &productcatalog.TaxConfig{
								Stripe: &productcatalog.StripeTaxConfig{
									Code: "stripecode",
								},
							},
							EntitlementTemplate: nil,
							Price:               &pp,
							BillingCadence:      lo.ToPtr(testutils.GetISODuration(t, "P1M")),
						},
					},
				},
			},
		}

		assertSame(t, p)
	})

	t.Run("Should be the same for PatchRemoveItem", func(t *testing.T) {
		p := &patch.PatchRemoveItem{
			PhaseKey: "asd",
			ItemKey:  "asd2",
		}

		assertSame(t, p)
	})
}
