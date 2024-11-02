package subscription

import (
	"reflect"

	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionPrice price.Price

func (s SubscriptionPrice) AsSpec() *CreatePriceSpec {
	return &CreatePriceSpec{
		CreateInput: price.CreateInput{
			Spec: price.Spec{
				PhaseKey: s.PhaseKey,
				ItemKey:  s.ItemKey,
				Value:    s.Value,
				Key:      s.Key,
			},
			CadencedModel: s.CadencedModel,
			SubscriptionId: models.NamespacedID{
				Namespace: s.Namespace,
				ID:        s.SubscriptionId,
			},
		},
		SubscriptionItemRef: SubscriptionItemRef{
			SubscriptionId: s.SubscriptionId,
			PhaseKey:       s.PhaseKey,
			ItemKey:        s.ItemKey,
		},
		Cadence: s.CadencedModel,
	}
}

type CreatePriceSpec struct {
	CreateInput         price.CreateInput
	SubscriptionItemRef SubscriptionItemRef
	Cadence             models.CadencedModel
}

func (s CreatePriceSpec) Self() CreatePriceSpec {
	return s
}

func (s CreatePriceSpec) Equal(other CreatePriceSpec) bool {
	return reflect.DeepEqual(s, other)
}

type CreatePriceInput price.Spec

func (s *CreatePriceInput) ToCreatePriceSpec(
	namespace string,
	subscriptionId string,
	cadence models.CadencedModel,
) CreatePriceSpec {
	return CreatePriceSpec{
		CreateInput: price.CreateInput{
			Spec:          price.Spec(*s),
			CadencedModel: cadence,
			SubscriptionId: models.NamespacedID{
				Namespace: namespace,
				ID:        subscriptionId,
			},
		},
		SubscriptionItemRef: SubscriptionItemRef{
			SubscriptionId: subscriptionId,
			PhaseKey:       s.PhaseKey,
			ItemKey:        s.ItemKey,
		},
		Cadence: cadence,
	}
}
