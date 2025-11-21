package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	modelshttp "github.com/openmeterio/openmeter/pkg/models/http"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const limit = 1000

type (
	ListMetersParams   = api.ListMetersParams
	ListMetersResponse = []api.Meter
	ListMetersHandler  httptransport.HandlerWithArgs[ListMetersRequest, ListMetersResponse, ListMetersParams]
)

type ListMetersRequest struct {
	namespace      string
	page           pagination.Page
	orderBy        meter.OrderBy
	order          sortx.Order
	includeDeleted bool
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
				page:           pagination.NewPage(1, limit),
				includeDeleted: lo.FromPtrOr(params.IncludeDeleted, false),
				orderBy:        meter.OrderBy(lo.FromPtrOr(params.OrderBy, api.MeterOrderByKey)),
				order:          sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderASC)),
			}, nil
		},
		func(ctx context.Context, request ListMetersRequest) (ListMetersResponse, error) {
			if err := request.page.Validate(); err != nil {
				return ListMetersResponse{}, models.NewGenericValidationError(fmt.Errorf("invalid pagination: %w", err))
			}

			result, err := h.meterService.ListMeters(ctx, meter.ListMetersParams{
				Namespace:      request.namespace,
				Page:           request.page,
				IncludeDeleted: request.includeDeleted,
				OrderBy:        request.orderBy,
				Order:          request.order,
			})
			if err != nil {
				return ListMetersResponse{}, fmt.Errorf("failed to list meters: %w", err)
			}

			// Response
			resp := pagination.MapResult(result, ToAPIMeter)

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
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return GetMeterResponse{}, fmt.Errorf("failed to get meter: %w", err)
			}

			return ToAPIMeter(m), nil
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
			err := validateJSONPaths(ctx, h.streaming, request.MeterCreate.ValueProperty, request.MeterCreate.GroupBy)
			if err != nil {
				return CreateMeterResponse{}, err
			}

			// Create meter
			input := meter.CreateMeterInput{
				Namespace: request.Namespace,
				Key:       request.MeterCreate.Slug,
				// Default the name to slug if not provided
				Name:          lo.FromPtrOr(request.MeterCreate.Name, request.MeterCreate.Slug),
				EventType:     request.MeterCreate.EventType,
				EventFrom:     request.MeterCreate.EventFrom,
				Aggregation:   meter.MeterAggregation(request.MeterCreate.Aggregation),
				Description:   request.MeterCreate.Description,
				ValueProperty: request.MeterCreate.ValueProperty,
				GroupBy:       lo.FromPtrOr(request.MeterCreate.GroupBy, map[string]string{}),
				Metadata:      modelshttp.AsMetadata(request.MeterCreate.Metadata),
			}

			m, err := h.meterService.CreateMeter(ctx, input)
			if err != nil {
				return CreateMeterResponse{}, fmt.Errorf("failed to create meter: %w", err)
			}

			return ToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createMeter"),
		)...,
	)
}

type (
	UpdateMeterRequest = struct {
		namespace string
		idOrKey   string
		api.MeterUpdate
	}
	UpdateMeterResponse = api.Meter
)

type UpdateMeterHandler = httptransport.HandlerWithArgs[UpdateMeterRequest, UpdateMeterResponse, string]

func (h handler) UpdateMeter() UpdateMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterIdOrKey string) (UpdateMeterRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateMeterRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := api.MeterUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateMeterRequest{}, fmt.Errorf("failed to decode update meter request: %w", err)
			}

			return UpdateMeterRequest{
				namespace:   namespace,
				idOrKey:     meterIdOrKey,
				MeterUpdate: body,
			}, nil
		},
		func(ctx context.Context, request UpdateMeterRequest) (UpdateMeterResponse, error) {
			// Get current meter
			currentMeter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrKey,
			})
			if err != nil {
				return UpdateMeterResponse{}, fmt.Errorf("failed to get meter: %w", err)
			}

			// Validate JSON paths via ClickHouse
			err = validateGroupByJSONPaths(ctx, h.streaming, request.MeterUpdate.GroupBy)
			if err != nil {
				return UpdateMeterResponse{}, err
			}

			// Update meter
			input := meter.UpdateMeterInput{
				ID: models.NamespacedID{
					Namespace: currentMeter.Namespace,
					ID:        currentMeter.ID,
				},
				Name:        lo.FromPtrOr(request.MeterUpdate.Name, currentMeter.Key),
				Description: request.MeterUpdate.Description,
				GroupBy:     lo.FromPtrOr(request.MeterUpdate.GroupBy, map[string]string{}),
				Metadata:    modelshttp.AsMetadata(request.MeterUpdate.Metadata),
			}

			m, err := h.meterService.UpdateMeter(ctx, input)
			if err != nil {
				return UpdateMeterResponse{}, fmt.Errorf("failed to update meter: %w", err)
			}

			return ToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateMeter"),
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

// Validate JSON paths via ClickHouse
// This is necessary because ClickHouse is more strict than JSONPath libraries in Go.
// Ideally this would be a RegExp
func validateJSONPaths(ctx context.Context, streaming streaming.Connector, valueProperty *string, groupBy *map[string]string) error {
	err := validateValuePropertyJSONPath(ctx, streaming, valueProperty)
	if err != nil {
		return err
	}

	err = validateGroupByJSONPaths(ctx, streaming, groupBy)
	if err != nil {
		return err
	}

	return nil
}

// Validate Value Property JSON path via ClickHouse
func validateValuePropertyJSONPath(ctx context.Context, streaming streaming.Connector, valueProperty *string) error {
	if valueProperty == nil {
		return nil
	}

	isValid, err := streaming.ValidateJSONPath(ctx, *valueProperty)
	if err != nil {
		return fmt.Errorf("validate json path in clickhouse: %w", err)
	}

	if !isValid {
		return models.NewGenericValidationError(fmt.Errorf("invalid JSONPath: %w", err))
	}

	return nil
}

// Validate GroupBy JSON paths via ClickHouse
func validateGroupByJSONPaths(ctx context.Context, streaming streaming.Connector, groupBy *map[string]string) error {
	if groupBy == nil {
		return nil
	}

	for groupByKey, jsonPath := range *groupBy {
		isValid, err := streaming.ValidateJSONPath(ctx, jsonPath)
		if err != nil {
			return fmt.Errorf("validate json path in clickhouse: %w", err)
		}

		if !isValid {
			return models.NewGenericValidationError(fmt.Errorf("invalid JSONPath for %s group by: %w", groupByKey, err))
		}
	}

	return nil
}
