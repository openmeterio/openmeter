package subscriptionitem

import (
	"fmt"

	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

// Item is whats included in the subscription.
// This can entail
// - a price the customer is paying (before modifiers)
// - A feature they get access to (via entitlement)
//
// The reason this is different from a Plan's RateCard is that it's a continous representation through the subscription.
// RateCards are phase specific and they encode instructions on what the contents are. Contents are what the customer is actually getting.
type Item struct {
	// The key that identifies the content.
	ContentKey ItemKey
}

type ItemInstance struct {
	Item
}

// ItemKeyType determines by what the SubscriptionContent can be referenced.
// This can be a feature, or if a feature is not present, a price.
type ItemKeyType string

const (
	// If a feature is present in the SubscriptionContent, we reference by the feature.
	ContentKeyFeature ItemKeyType = "feature"
	// If a feature is not present in the SubscriptionContent, we reference by the price.
	ContentKeyPrice ItemKeyType = "price"
)

// ItemKey is a key that identifies a given subscription content.
type ItemKey struct {
	typ ItemKeyType

	// Prices have to be identifiable for modifiers to work...
	priceId string
	// Features are identified by their ref (either Id or Key). If Key is chosen then the active feature by that key is used.
	featureRef modelref.IdOrKeyRef
}

func (k ItemKey) ByType() (string, ItemKeyType) {
	if k.typ == ContentKeyFeature {
		return k.featureRef.Value(), k.typ
	}

	return k.priceId, k.typ
}

func (k ItemKey) Value() string {
	typ, val := k.ByType()
	return fmt.Sprintf("%s:%s", typ, val)
}

func (k ItemKey) MarshalJSON() ([]byte, error) {
	return []byte(`"` + k.Value() + `"`), nil
}

func NewItemKeyFromValue(value string) (ItemKey, error) {
	typ, val, err := modelref.SeparatedKeyParser(value)
	if err != nil {
		return ItemKey{}, err
	}

	if typ == string(ContentKeyFeature) {
		ref, err := modelref.NewIdOrKeyRefFromValue(val)
		if err != nil {
			return ItemKey{}, err
		}

		return ItemKey{
			typ:        ContentKeyFeature,
			featureRef: ref,
		}, nil
	} else if typ == string(ContentKeyPrice) {
		return ItemKey{
			typ:     ContentKeyPrice,
			priceId: val,
		}, nil
	}

	return ItemKey{}, fmt.Errorf("unknown type: %s", typ)
}
