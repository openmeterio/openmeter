package adapter

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CustomerFromDBEntity(e db.Customer) (*customer.Customer, error) {
	subjects, err := e.Edges.SubjectsOrErr()
	if err != nil {
		if db.IsNotLoaded(err) {
			return nil, errors.New("subjects must be loaded for customer")
		}

		return nil, err
	}

	subjectKeys := lo.FilterMap(subjects, func(item *db.CustomerSubjects, _ int) (string, bool) {
		if item == nil {
			return "", false
		}

		return item.SubjectKey, true
	})

	subscriptions, err := e.Edges.SubscriptionOrErr()
	if err != nil {
		if db.IsNotLoaded(err) {
			return nil, errors.New("subscriptions must be loaded for customer")
		}

		return nil, err
	}

	subscriptionIDs := lo.FilterMap(subscriptions, func(item *db.Subscription, _ int) (string, bool) {
		if item == nil {
			return "", false
		}

		return item.ID, true
	})

	var metadata *models.Metadata

	if len(e.Metadata) > 0 {
		metadata = lo.ToPtr(models.NewMetadata(e.Metadata))
	}

	var annotations *models.Annotations

	if len(e.Annotations) > 0 {
		annotations = &e.Annotations
	}

	result := &customer.Customer{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:          e.ID,
			Namespace:   e.Namespace,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
			DeletedAt:   e.DeletedAt,
			Name:        e.Name,
			Description: e.Description,
		}),
		UsageAttribution: customer.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
		PrimaryEmail: e.PrimaryEmail,
		Currency:     e.Currency,
		Metadata:     metadata,
		Annotation:   annotations,

		ActiveSubscriptionIDs: subscriptionIDs,
	}

	if e.Key != "" {
		result.Key = &e.Key
	}

	if e.BillingAddressCity != nil || e.BillingAddressCountry != nil || e.BillingAddressLine1 != nil || e.BillingAddressLine2 != nil || e.BillingAddressPhoneNumber != nil || e.BillingAddressPostalCode != nil || e.BillingAddressState != nil {
		result.BillingAddress = &models.Address{
			City:        e.BillingAddressCity,
			Country:     e.BillingAddressCountry,
			Line1:       e.BillingAddressLine1,
			Line2:       e.BillingAddressLine2,
			PhoneNumber: e.BillingAddressPhoneNumber,
			PostalCode:  e.BillingAddressPostalCode,
			State:       e.BillingAddressState,
		}
	}

	return result, nil
}
