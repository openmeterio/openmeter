// Code generated by github.com/jmattheis/goverter, DO NOT EDIT.
//go:build !goverter

package httphandler

import (
	api "github.com/openmeterio/openmeter/api"
	apiconverter "github.com/openmeterio/openmeter/openmeter/apiconverter"
	meterevent "github.com/openmeterio/openmeter/openmeter/meterevent"
)

func init() {
	convertListEventsV2Params = func(source api.ListEventsV2Params, context string) (meterevent.ListEventsV2Params, error) {
		var metereventListEventsV2Params meterevent.ListEventsV2Params
		metereventListEventsV2Params.Namespace = convertNamespace(context)
		metereventListEventsV2Params.ClientID = source.ClientId
		pPaginationCursor, err := apiconverter.ConvertCursorPtr(source.Cursor)
		if err != nil {
			return metereventListEventsV2Params, err
		}
		metereventListEventsV2Params.Cursor = pPaginationCursor
		if source.Limit != nil {
			xint := *source.Limit
			metereventListEventsV2Params.Limit = &xint
		}
		metereventListEventsV2Params.ID = apiconverter.ConvertStringPtr(source.Id)
		metereventListEventsV2Params.Source = apiconverter.ConvertStringPtr(source.Source)
		metereventListEventsV2Params.Subject = apiconverter.ConvertStringPtr(source.Subject)
		metereventListEventsV2Params.Type = apiconverter.ConvertStringPtr(source.Type)
		metereventListEventsV2Params.Time = apiconverter.ConvertTimePtr(source.Time)
		metereventListEventsV2Params.IngestedAt = apiconverter.ConvertTimePtr(source.IngestedAt)
		return metereventListEventsV2Params, nil
	}
}
