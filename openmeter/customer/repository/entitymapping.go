package repository

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CustomerFromDBEntity(e db.Customer) *customer.Customer {
	var subjectKeys []string

	if e.Edges.Subjects != nil {
		subjectKeys = lo.Map(
			e.Edges.Subjects,
			func(item *db.CustomerSubjects, _ int) string {
				return item.SubjectKey
			},
		)
	}

	result := &customer.Customer{
		// TODO: create common function to convert managed resource entity to model
		ManagedResource: models.ManagedResource{
			ID: e.ID,
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: e.CreatedAt.UTC(),
				UpdatedAt: e.UpdatedAt.UTC(),
				DeletedAt: func() *time.Time {
					if e.DeletedAt == nil {
						return nil
					}

					deletedAt := e.DeletedAt.UTC()

					return &deletedAt
				}(),
			},
		},
		Name: e.Name,
		UsageAttribution: customer.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
		PrimaryEmail:      e.PrimaryEmail,
		Currency:          e.Currency,
		TaxProvider:       e.TaxProvider,
		InvoicingProvider: e.InvoicingProvider,
		PaymentProvider:   e.PaymentProvider,
	}

	if e.ExternalMappingStripeCustomerID != nil {
		result.External = &customer.CustomerExternalMapping{
			StripeCustomerID: e.ExternalMappingStripeCustomerID,
		}
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
