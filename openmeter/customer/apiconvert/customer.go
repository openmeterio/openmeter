package apiconvert

import (
	"maps"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ToAPICustomer(c customer.Customer) apiv1models.Customer {
	apiCustomer := apiv1models.Customer{
		Id:           c.ManagedResource.ID,
		Key:          c.Key,
		Name:         c.Name,
		PrimaryEmail: c.PrimaryEmail,
		Description:  c.Description,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		DeletedAt:    c.DeletedAt,
		Metadata:     ToAPIMetadata(lo.FromPtr(c.Metadata)),
		Annotations:  ToAPIAnnotations(lo.FromPtr(c.Annotation)),
	}

	if c.UsageAttribution != nil {
		apiCustomer.UsageAttribution = &apiv1models.CustomerUsageAttribution{SubjectKeys: c.UsageAttribution.SubjectKeys}
	}

	if c.BillingAddress != nil {
		address := apiv1models.Address{
			City:        c.BillingAddress.City,
			State:       c.BillingAddress.State,
			PostalCode:  c.BillingAddress.PostalCode,
			Line1:       c.BillingAddress.Line1,
			Line2:       c.BillingAddress.Line2,
			PhoneNumber: c.BillingAddress.PhoneNumber,
		}

		if c.BillingAddress.Country != nil {
			address.Country = lo.ToPtr(string(*c.BillingAddress.Country))
		}

		apiCustomer.BillingAddress = &address
	}

	if c.Currency != nil {
		apiCustomer.Currency = lo.ToPtr(string(*c.Currency))
	}

	return apiCustomer
}

func ToAPIMetadata(metadata models.Metadata) *apiv1models.Metadata {
	if len(metadata) == 0 {
		return nil
	}

	return lo.ToPtr(apiv1models.Metadata(maps.Clone(metadata)))
}

func ToAPIAnnotations(annotations models.Annotations) *apiv1models.Annotations {
	if len(annotations) == 0 {
		return nil
	}

	return lo.ToPtr(apiv1models.Annotations(maps.Clone(annotations)))
}
