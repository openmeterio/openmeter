package apiconvert

import (
	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
)

func ToAPISubject(s subject.Subject) apiv1models.Subject {
	var metadata *map[string]interface{}

	if s.Metadata != nil {
		m := map[string]interface{}{}

		for k, v := range s.Metadata {
			m[k] = v
		}

		metadata = &m
	}

	return apiv1models.Subject{
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		DeletedAt:        s.DeletedAt,
		Id:               s.Id,
		Key:              s.Key,
		DisplayName:      s.DisplayName,
		Metadata:         metadata,
		StripeCustomerId: s.StripeCustomerId,
	}
}
