package httphandler

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	apiconverter "github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func convertListEventsV2Params(params api.ListEventsV2Params, namespace string) (meterevent.ListEventsV2Params, error) {
	cursor, err := apiconverter.ConvertCursorPtr(params.Cursor)
	if err != nil {
		return meterevent.ListEventsV2Params{}, err
	}

	p := meterevent.ListEventsV2Params{
		Namespace: namespace,
		ClientID:  params.ClientId,
		Cursor:    cursor,
		Limit:     params.Limit,
	}

	if params.Filter != nil {
		p.ID = apiconverter.ConvertStringPtr(params.Filter.Id)
		p.Source = apiconverter.ConvertStringPtr(params.Filter.Source)
		p.Subject = apiconverter.ConvertStringPtr(params.Filter.Subject)
		p.CustomerID = apiconverter.ConvertIDExactPtr(params.Filter.CustomerId)
		p.Type = apiconverter.ConvertStringPtr(params.Filter.Type)
		p.Time = apiconverter.ConvertTimePtr(params.Filter.Time)
		p.IngestedAt = apiconverter.ConvertTimePtr(params.Filter.IngestedAt)
	}

	return p, nil
}

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
		CustomerId:      e.CustomerID,
		IngestedAt:      e.IngestedAt,
		StoredAt:        e.StoredAt,
		ValidationError: validationError,
	}, nil
}

func convertListEventsV2Response(events pagination.Result[meterevent.Event]) (api.IngestedEventCursorPaginatedResponse, error) {
	items := make([]api.IngestedEvent, len(events.Items))
	for i, e := range events.Items {
		ev, err := convertEvent(e)
		if err != nil {
			return api.IngestedEventCursorPaginatedResponse{}, err
		}
		items[i] = ev
	}

	return api.IngestedEventCursorPaginatedResponse{
		Items:      items,
		NextCursor: events.NextCursor.EncodePtr(),
	}, nil
}
