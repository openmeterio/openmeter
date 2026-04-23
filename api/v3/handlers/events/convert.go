package events

import (
	"encoding/json"
	"fmt"

	"github.com/oapi-codegen/nullable"

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
		Event:      event,
		Customer:   toAPICustomerReference(e.CustomerID),
		IngestedAt: e.IngestedAt,
		StoredAt:   e.StoredAt,
	}, nil
}

func toAPICustomerReference(id *string) *api.CustomerReference {
	if id == nil || *id == "" {
		return nil
	}
	return &api.CustomerReference{Id: *id}
}
