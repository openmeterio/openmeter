package repository

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CustomerFromDBEntity(e db.Customer) *customer.Customer {
	var subjectKeys []string

	if e.Edges.Subjects != nil {
		subjectKeys = make([]string, 0, len(e.Edges.Subjects))
		for _, subject := range e.Edges.Subjects {
			subjectKeys = append(subjectKeys, subject.SubjectKey)
		}
	}

	result := &customer.Customer{
		// TODO: create common function to convert managed resource entity to model
		ManagedResource: models.ManagedResource{
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
		UsageAttribution: customer.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
		PrimaryEmail:      e.PrimaryEmail,
		Currency:          e.Currency,
		TaxProvider:       e.TaxProvider,
		InvoicingProvider: e.InvoicingProvider,
		PaymentProvider:   e.PaymentProvider,
	}

	if e.AddressCity != nil || e.AddressCountry != nil || e.AddressLine1 != nil || e.AddressLine2 != nil || e.AddressPhoneNumber != nil || e.AddressPostalCode != nil || e.AddressState != nil {
		result.Address = &models.Address{
			City:        e.AddressCity,
			Country:     e.AddressCountry,
			Line1:       e.AddressLine1,
			Line2:       e.AddressLine2,
			PhoneNumber: e.AddressPhoneNumber,
			PostalCode:  e.AddressPostalCode,
			State:       e.AddressState,
		}
	}

	return result
}
