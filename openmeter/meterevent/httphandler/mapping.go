//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package httphandler

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./mapping.gen.go
// goverter:extend github.com/openmeterio/openmeter/openmeter/apiconverter:ConvertStringPtr
// goverter:extend github.com/openmeterio/openmeter/openmeter/apiconverter:ConvertTimePtr
// goverter:extend github.com/openmeterio/openmeter/openmeter/apiconverter:ConvertCursorPtr
// goverter:extend convertEvent
var (
	// goverter:matchIgnoreCase
	// goverter:context namespace
	// goverter:map namespace Namespace | convertNamespace
	convertListEventsV2Params func(params api.ListEventsV2Params, namespace string) (meterevent.ListEventsV2Params, error)
)

// goverter:context namespace
func convertNamespace(namespace string) string {
	return namespace
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
		NextCursor: events.NextCursor,
	}, nil
}
