package httpdriver

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
)

func convertEvent(e meterevent.Event) (api.IngestedEvent, error) {
	ev := event.New()
	ev.SetID(e.ID)
	ev.SetType(e.Type)
	ev.SetSource(e.Source)
	ev.SetSubject(e.Subject)
	ev.SetTime(e.Time)

	if e.Data != "" {
		var data interface{}
		err := json.Unmarshal([]byte(e.Data), &data)
		if err != nil {
			return api.IngestedEvent{}, fmt.Errorf("parse cloudevents data as json: %w", err)
		}

		err = ev.SetData(event.ApplicationJSON, data)
		if err != nil {
			return api.IngestedEvent{}, fmt.Errorf("set cloudevents data: %w", err)
		}
	}

	var validationError *string
	if len(e.ValidationErrors) > 0 {
		validationError = lo.EmptyableToPtr(errors.Join(e.ValidationErrors...).Error())
	}

	return api.IngestedEvent{
		Event:           ev,
		IngestedAt:      e.IngestedAt,
		StoredAt:        e.StoredAt,
		ValidationError: validationError,
	}, nil
}
