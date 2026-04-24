package events

import (
	"encoding/json"
	"fmt"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
)

const cloudEventsSpecVersion = "1.0"

// toAPIMeteringIngestedEvent converts a meterevent.Event to its API wire form.
func toAPIMeteringIngestedEvent(e meterevent.Event) (api.MeteringIngestedEvent, error) {
	event := api.MeteringEvent{
		Id:          e.ID,
		Source:      e.Source,
		Specversion: cloudEventsSpecVersion,
		Subject:     e.Subject,
		Type:        e.Type,
		Time:        nullable.NewNullableWithValue[api.DateTime](e.Time),
	}

	if e.Data != "" {
		var data map[string]any
		if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
			return api.MeteringIngestedEvent{}, fmt.Errorf("parse event data as json: %w", err)
		}
		event.Data = nullable.NewNullableWithValue(data)
		event.Datacontenttype = nullable.NewNullableWithValue[api.MeteringEventDatacontenttype](api.MeteringEventDatacontenttype("application/json"))
	}

	return api.MeteringIngestedEvent{
		Event:            event,
		Customer:         toAPICustomerReference(e.CustomerID),
		IngestedAt:       e.IngestedAt,
		StoredAt:         e.StoredAt,
		ValidationErrors: toAPIMeteringIngestedEventValidationErrors(e.ValidationErrors),
	}, nil
}

func toAPICustomerReference(id *string) *api.CustomerReference {
	if id == nil || *id == "" {
		return nil
	}
	return &api.CustomerReference{Id: *id}
}

func toAPIMeteringIngestedEventValidationErrors(errs []error) *[]api.MeteringIngestedEventValidationError {
	if len(errs) == 0 {
		return nil
	}

	result := lo.FilterMap(errs, func(err error, _ int) (api.MeteringIngestedEventValidationError, bool) {
		if err == nil {
			return api.MeteringIngestedEventValidationError{}, false
		}

		return api.MeteringIngestedEventValidationError{
			Code:    "validation_error",
			Message: err.Error(),
		}, true
	})

	if len(result) == 0 {
		return nil
	}

	return lo.ToPtr(result)
}
