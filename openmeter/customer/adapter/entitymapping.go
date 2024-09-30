package adapter

import (
	"github.com/samber/lo"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CustomerFromDBEntity(e db.Customer) *customerentity.Customer {
	var subjectKeys []string

	if e.Edges.Subjects != nil {
		subjectKeys = lo.Map(
			e.Edges.Subjects,
			func(item *db.CustomerSubjects, _ int) string {
				return item.SubjectKey
			},
		)
	}

	result := &customerentity.Customer{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        e.ID,
			Namespace: e.Namespace,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: e.DeletedAt,
		}),
		Name: e.Name,
		UsageAttribution: customerentity.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
		PrimaryEmail: e.PrimaryEmail,
		Currency:     e.Currency,
		Timezone:     e.Timezone,
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

	return result
}
