package adapter

import (
	"fmt"

	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CustomerFromDBEntity(e db.Customer) (*customerentity.Customer, error) {
	var subjectKeys []string

	if e.Edges.Subjects != nil {
		subjectKeys = lo.Map(
			e.Edges.Subjects,
			func(item *db.CustomerSubjects, _ int) string {
				return item.SubjectKey
			},
		)
	}

	var currentSubID *string

	if len(e.Edges.Subscription) > 0 {
		var subs []subscription.Subscription
		for _, s := range e.Edges.Subscription {
			sub, err := subscriptionrepo.MapDBSubscription(s)
			if err != nil {
				return nil, fmt.Errorf("failed to map subscription with id %s: %w", s.ID, err)
			}
			subs = append(subs, sub)
		}

		// Let's find the active one
		if active, found := lo.Find(subs, func(s subscription.Subscription) bool {
			return s.CadencedModel.IsActiveAt(clock.Now())
		}); found {
			currentSubID = &active.ID
		}
	}

	result := &customerentity.Customer{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:          e.ID,
			Namespace:   e.Namespace,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
			DeletedAt:   e.DeletedAt,
			Name:        e.Name,
			Description: e.Description,
		}),
		UsageAttribution: customerentity.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
		PrimaryEmail:          e.PrimaryEmail,
		Currency:              e.Currency,
		Timezone:              e.Timezone,
		CurrentSubscriptionID: currentSubID,
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
