package apiconvert

import (
	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
)

func ToAPISubject(s subject.Subject) apimodels.Subject {
	var metadata *map[string]interface{}

	if s.Metadata != nil {
		m := map[string]interface{}{}

		for k, v := range s.Metadata {
			m[k] = v
		}

		metadata = &m
	}

	return apimodels.Subject{
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
