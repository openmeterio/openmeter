package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
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
