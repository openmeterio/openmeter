package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subject"
)

func FromSubject(s subject.Subject) api.Subject {
	var metadata *map[string]interface{}

	if s.Metadata != nil {
		m := map[string]interface{}{}

		for k, v := range s.Metadata {
			m[k] = v
		}

		metadata = &m
	}

	return api.Subject{
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
