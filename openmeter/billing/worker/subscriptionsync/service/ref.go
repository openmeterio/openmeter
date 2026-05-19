package service

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionReferenceType string

const (
	SubscriptionReferenceTypeID   SubscriptionReferenceType = "id"
	SubscriptionReferenceTypeView SubscriptionReferenceType = "view"
)

var SubscriptionReferenceTypes = []SubscriptionReferenceType{
	SubscriptionReferenceTypeID,
	SubscriptionReferenceTypeView,
}

func (t SubscriptionReferenceType) Validate() error {
	if slices.Contains(SubscriptionReferenceTypes, t) {
		return nil
	}

	return fmt.Errorf("invalid subscription reference type: %s", t)
}

type subscriptionReferenceOrView struct {
	t    SubscriptionReferenceType
	id   *models.NamespacedID
	view *subscription.SubscriptionView
}

func newSubscriptionReferenceOrView[T models.NamespacedID | subscription.SubscriptionView | subscription.Subscription](refOrView T) subscriptionReferenceOrView {
	switch v := any(refOrView).(type) {
	case models.NamespacedID:
		return subscriptionReferenceOrView{
			t:  SubscriptionReferenceTypeID,
			id: &v,
		}
	case subscription.SubscriptionView:
		return subscriptionReferenceOrView{
			t:    SubscriptionReferenceTypeView,
			view: &v,
		}
	case subscription.Subscription:
		return subscriptionReferenceOrView{
			t:  SubscriptionReferenceTypeID,
			id: &v.NamespacedID,
		}
	default:
		return subscriptionReferenceOrView{}
	}
}

func (r subscriptionReferenceOrView) Validate() error {
	switch r.t {
	case SubscriptionReferenceTypeID:
		if r.id == nil {
			return fmt.Errorf("subscription ID is required")
		}

		return nil
	case SubscriptionReferenceTypeView:
		if r.view == nil {
			return fmt.Errorf("subscription view is required")
		}

		return r.view.Validate(true)
	default:
		return fmt.Errorf("invalid subscription reference type: %s", r.t)
	}
}

func (r subscriptionReferenceOrView) Type() SubscriptionReferenceType {
	return r.t
}

func (r subscriptionReferenceOrView) AsNamespacedID() (models.NamespacedID, error) {
	if r.t != SubscriptionReferenceTypeID {
		return models.NamespacedID{}, fmt.Errorf("subscription reference type is not ID: %s", r.t)
	}

	if r.id == nil {
		return models.NamespacedID{}, fmt.Errorf("subscription ID is required")
	}

	return *r.id, nil
}

func (r subscriptionReferenceOrView) AsSubscriptionView() (subscription.SubscriptionView, error) {
	if r.t != SubscriptionReferenceTypeView {
		return subscription.SubscriptionView{}, fmt.Errorf("subscription reference type is not view: %s", r.t)
	}

	if r.view == nil {
		return subscription.SubscriptionView{}, fmt.Errorf("subscription view is required")
	}

	return *r.view, nil
}

func (r subscriptionReferenceOrView) GetID() models.NamespacedID {
	switch r.t {
	case SubscriptionReferenceTypeID:
		if r.id == nil {
			return models.NamespacedID{}
		}

		return *r.id
	case SubscriptionReferenceTypeView:
		if r.view == nil {
			return models.NamespacedID{}
		}

		return r.view.Subscription.NamespacedID
	default:
		return models.NamespacedID{}
	}
}
