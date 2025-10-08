package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListSubjectsResponse = *[]string
	ListSubjectsHandler  httptransport.HandlerWithArgs[ListSubjectsRequest, ListSubjectsResponse, ListSubjectsParams]
)

type ListSubjectsParams struct {
	IdOrSlug string
	From     *time.Time
	To       *time.Time
}

type ListSubjectsRequest struct {
	namespace string
	idOrSlug  string
	from      *time.Time
	to        *time.Time
}

// ListSubjects returns a handler for query meter.
func (h *handler) ListSubjects() ListSubjectsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubjectsParams) (ListSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubjectsRequest{}, err
			}

			return ListSubjectsRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
				from:      params.From,
				to:        params.To,
			}, nil
		},
		func(ctx context.Context, request ListSubjectsRequest) (ListSubjectsResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			subjectKeys, err := h.streaming.ListSubjects(ctx, streaming.ListSubjectsParams{
				Namespace: request.namespace,
				Meter:     &meter,
				From:      request.from,
				To:        request.to,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list subjects: %w", err)
			}

			return &subjectKeys, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubjectsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
		)...,
	)
}

type (
	QueryMeterResponse = *api.MeterQueryResult
	QueryMeterHandler  httptransport.HandlerWithArgs[QueryMeterRequest, QueryMeterResponse, QueryMeterParams]
)

type QueryMeterParams struct {
	api.QueryMeterParams
	IdOrSlug string
}

type QueryMeterRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterParams
}

// QueryMeter returns a handler for query meter.
func (h *handler) QueryMeter() QueryMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryMeterParams) (QueryMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterRequest{}, err
			}

			return QueryMeterRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
				params:    params.QueryMeterParams,
			}, nil
		},
		func(ctx context.Context, request QueryMeterRequest) (QueryMeterResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := h.toQueryParamsFromRequest(
				ctx,
				meter,
				// Convert the POST request body to a GET request params
				ToRequestFromQueryParamsPOSTBody(request.params),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
		)...,
	)
}

type QueryMeterPostRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterPostJSONRequestBody
}

type (
	QueryMeterPostParams   = string // meterIdOrSlug
	QueryMeterPostResponse = QueryMeterResponse
	QueryMeterPostHandler  httptransport.HandlerWithArgs[QueryMeterPostRequest, QueryMeterPostResponse, QueryMeterPostParams]
)

// QueryMeterPost returns a handler for query meter via POST.
func (h *handler) QueryMeterPost() QueryMeterPostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterIdOrSlug QueryMeterPostParams) (QueryMeterPostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterPostRequest{}, err
			}

			var request api.QueryMeterPostJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &request); err != nil {
				return QueryMeterPostRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			return QueryMeterPostRequest{
				namespace: ns,
				idOrSlug:  meterIdOrSlug,
				params:    request,
			}, nil
		},
		func(ctx context.Context, request QueryMeterPostRequest) (QueryMeterPostResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := h.toQueryParamsFromRequest(ctx, meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterPostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterPost"),
		)...,
	)
}

type (
	ListGroupByValuesResponse = *[]string
	ListGroupByValuesHandler  httptransport.HandlerWithArgs[ListGroupByValuesRequest, ListGroupByValuesResponse, ListGroupByValuesParams]
)

type ListGroupByValuesParams struct {
	IdOrSlug   string
	GroupByKey string
	From       *time.Time
	To         *time.Time
	Search     *string
}

type ListGroupByValuesRequest struct {
	namespace  string
	idOrSlug   string
	groupByKey string
	from       *time.Time
	to         *time.Time
	search     *string
}

// ListGroupByValues returns a handler for list group by values.
func (h *handler) ListGroupByValues() ListGroupByValuesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListGroupByValuesParams) (ListGroupByValuesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListGroupByValuesRequest{}, err
			}

			// Set default to last 24 hours
			if params.From == nil && params.To == nil {
				params.From = lo.ToPtr(time.Now().Add(-time.Hour * 24))
			}

			return ListGroupByValuesRequest{
				namespace:  ns,
				idOrSlug:   params.IdOrSlug,
				groupByKey: params.GroupByKey,
				from:       params.From,
				to:         params.To,
				search:     params.Search,
			}, nil
		},
		func(ctx context.Context, request ListGroupByValuesRequest) (ListGroupByValuesResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			groupByValues, err := h.streaming.ListGroupByValues(ctx, streaming.ListGroupByValuesParams{
				Namespace:  request.namespace,
				Meter:      meter,
				GroupByKey: request.groupByKey,
				From:       request.from,
				To:         request.to,
				Search:     request.search,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list group by values: %w", err)
			}

			return &groupByValues, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListGroupByValuesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listGroupByValues"),
		)...,
	)
}
