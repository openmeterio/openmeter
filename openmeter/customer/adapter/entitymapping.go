package adapter

import (
	"errors"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func CustomerFromDBEntity(e db.Customer, expands customer.Expands) (*customer.Customer, error) {
	attributions, err := timedCustomerUsageAttributionsFromDBEntity(e)
	if err != nil {
		return nil, err
	}

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
		PrimaryEmail: e.PrimaryEmail,
		Currency:     e.Currency,
		Metadata:     metadata,
		Annotation:   annotations,
	}

	// Only set UsageAttribution if there are subject keys
	if len(attributions) > 0 {
		result.UsageAttribution = &attributions
	}

	if slices.Contains(expands, customer.ExpandSubscriptions) {
		activeSubscriptionIDs, err := resolveActiveSubscriptionIDs(e)
		if err != nil {
			return nil, err
		}

		result.ActiveSubscriptionIDs = mo.Some(activeSubscriptionIDs)
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

func resolveActiveSubscriptionIDs(e db.Customer) ([]string, error) {
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

	return subscriptionIDs, nil
}

func timedCustomerUsageAttributionsFromDBEntity(customerEntity db.Customer) (customer.TimedCustomerUsageAttributions, error) {
	subjectEntities, err := customerEntity.Edges.SubjectsOrErr()
	if err != nil {
		if db.IsNotLoaded(err) {
			return nil, errors.New("subjects must be loaded for customer")
		}

		return nil, err
	}

	timedCustomerUsageAttributions := lo.FilterMap(subjectEntities, func(item *db.CustomerSubjects, _ int) (customer.TimedCustomerUsageAttribution, bool) {
		return customer.TimedCustomerUsageAttribution{
			SubjectKey: item.SubjectKey,
			ActivePeriod: timeutil.OpenPeriod{
				From: lo.ToPtr(item.CreatedAt),
				To:   item.DeletedAt,
			},
		}, true
	})

	// Sort the subject keys to make sure the order is consistent
	slices.SortStableFunc(timedCustomerUsageAttributions, func(a, b customer.TimedCustomerUsageAttribution) int {
		return strings.Compare(a.SubjectKey, b.SubjectKey)
	})

	return timedCustomerUsageAttributions, nil
}
