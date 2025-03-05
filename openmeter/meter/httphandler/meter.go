package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const limit = 1000

type (
	// TODO: update when meter pagination is implemented
	ListMetersParams   = interface{}
	ListMetersResponse = []api.Meter
	ListMetersHandler  httptransport.HandlerWithArgs[ListMetersRequest, ListMetersResponse, ListMetersParams]
)

type ListMetersRequest struct {
	namespace string
	page      pagination.Page
}

// ListMeters returns a handler for listing meters.
func (h *handler) ListMeters() ListMetersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListMetersParams) (ListMetersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListMetersRequest{}, err
			}

			return ListMetersRequest{
				namespace: ns,
				// TODO: update when meter pagination is implemented
				page: pagination.NewPage(1, limit),
			}, nil
		},
		func(ctx context.Context, request ListMetersRequest) (ListMetersResponse, error) {
			result, err := h.meterService.ListMeters(ctx, meter.ListMetersParams{
				Namespace: request.namespace,
				Page:      request.page,
			})
			if err != nil {
				return ListMetersResponse{}, fmt.Errorf("failed to list meters: %w", err)
			}

			// Response
			resp := pagination.MapPagedResponse(result, ToAPIMeter)

			return resp.Items, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMetersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listMeters"),
		)...,
	)
}

type (
	GetMeterParams   = string
	GetMeterResponse = api.Meter
	GetMeterHandler  httptransport.HandlerWithArgs[GetMeterRequest, GetMeterResponse, GetMeterParams]
)

type GetMeterRequest struct {
	namespace string
	idOrSlug  string
}

// GetMeter returns a handler for listing meters.
func (h *handler) GetMeter() GetMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, idOrSlug GetMeterParams) (GetMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetMeterRequest{}, err
			}

			return GetMeterRequest{
				namespace: ns,
				idOrSlug:  idOrSlug,
			}, nil
		},
		func(ctx context.Context, request GetMeterRequest) (GetMeterResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return GetMeterResponse{}, fmt.Errorf("failed to get meter: %w", err)
			}

			return ToAPIMeter(meter), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getMeter"),
		)...,
	)
}

type (
	CreateMeterRequest = struct {
		Namespace string
		api.MeterCreate
	}
	CreateMeterResponse = api.Meter
)

type CreateMeterHandler = httptransport.Handler[CreateMeterRequest, CreateMeterResponse]

func (h handler) CreateMeter() CreateMeterHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateMeterRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateMeterRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := api.MeterCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateMeterRequest{}, fmt.Errorf("failed to decode create meter request: %w", err)
			}

			return CreateMeterRequest{
				Namespace:   namespace,
				MeterCreate: body,
			}, nil
		},
		func(ctx context.Context, request CreateMeterRequest) (CreateMeterResponse, error) {
			// Validate JSON paths via ClickHouse
			// This is necessary because ClickHouse is more strict than JSONPath libraries in Go.
			// Ideally this would be a RegExp
			if request.MeterCreate.ValueProperty != nil {
				isValid, err := h.streaming.ValidateJSONPath(ctx, *request.MeterCreate.ValueProperty)
				if err != nil {
					return CreateMeterResponse{}, fmt.Errorf("validate json path in clickhouse: %w", err)
				}

				if !isValid {
					return CreateMeterResponse{}, models.NewGenericValidationError(fmt.Errorf("invalid JSONPath: %w", err))
				}
			}

			if request.MeterCreate.GroupBy != nil {
				for groupByKey, jsonPath := range *request.MeterCreate.GroupBy {
					isValid, err := h.streaming.ValidateJSONPath(ctx, jsonPath)
					if err != nil {
						return CreateMeterResponse{}, fmt.Errorf("validate json path in clickhouse: %w", err)
					}

					if !isValid {
						return CreateMeterResponse{}, models.NewGenericValidationError(fmt.Errorf("invalid JSONPath for %s group by: %w", groupByKey, err))
					}
				}
			}

			// Create meter
			input := meter.CreateMeterInput{
				Namespace:     request.Namespace,
				Key:           request.MeterCreate.Slug,
				Name:          request.MeterCreate.Name,
				EventType:     request.MeterCreate.EventType,
				Aggregation:   meter.MeterAggregation(request.MeterCreate.Aggregation),
				Description:   request.MeterCreate.Description,
				ValueProperty: request.MeterCreate.ValueProperty,
			}

			if request.MeterCreate.GroupBy != nil {
				input.GroupBy = *request.MeterCreate.GroupBy
			} else {
				input.GroupBy = map[string]string{}
			}

			meter, err := h.meterService.CreateMeter(ctx, input)
			if err != nil {
				return CreateMeterResponse{}, fmt.Errorf("failed to create meter: %w", err)
			}

			return ToAPIMeter(meter), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createMeter"),
		)...,
	)
}

type (
	DeleteMeterRequest  = models.NamespacedID
	DeleteMeterResponse = any
	DeleteMeterParams   = string
)

type DeleteMeterHandler = httptransport.HandlerWithArgs[DeleteMeterRequest, DeleteMeterResponse, DeleteMeterParams]

func (h handler) DeleteMeter() DeleteMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID DeleteMeterParams) (DeleteMeterRequest, error) {
			id := models.NamespacedID{
				ID: meterID,
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return id, err
			}

			id.Namespace = namespace

			return id, nil
		},
		func(ctx context.Context, request DeleteMeterRequest) (DeleteMeterResponse, error) {
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.Namespace,
				IDOrSlug:  request.ID,
			})
			if err != nil {
				return nil, err
			}

			err = h.meterService.DeleteMeter(ctx, meter.DeleteMeterInput{
				Namespace: request.Namespace,
				IDOrSlug:  m.ID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to delete meter: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteMeterResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteMeter"),
		)...,
	)
}
