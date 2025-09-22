package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
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

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*app.AppCustomerPreConditionError](ctx, http.StatusConflict, err, w)
	}
}
