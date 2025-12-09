package handlers

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	response "github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type MeterHandler interface {
	ListMeters() ListMetersHandler
	// GetMeter() GetMeterHandler
	// CreateMeter() CreateMeterHandler
}

type meterHandler struct {
	service          meter.Service
	resolveNamespace func(ctx context.Context) (string, error)
	options          []httptransport.HandlerOption
}

func NewMeterHandler(
	resolveNamespace func(ctx context.Context) (string, error),
	service meter.Service,
	options ...httptransport.HandlerOption,
) MeterHandler {
	return &meterHandler{
		service:          service,
		resolveNamespace: resolveNamespace,
		options:          options,
	}
}

type (
	ListMetersRequest  = meter.ListMetersParams
	ListMetersResponse = response.CursorPaginationResponse[Meter]
	ListMetersHandler  httptransport.Handler[ListMetersRequest, ListMetersResponse]
)

// ListMeters returns a handler for listing meters.
func (h *meterHandler) ListMeters() ListMetersHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListMetersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListMetersRequest{}, err
			}

			return ListMetersRequest{
				Namespace: ns,

				// TODO: pagination
			}, nil
		},
		func(ctx context.Context, request ListMetersRequest) (ListMetersResponse, error) {
			result, err := h.service.ListMeters(ctx, request)
			if err != nil {
				return ListMetersResponse{}, err
			}

			meters, err := slicesx.MapWithErr(result.Items, func(item meter.Meter) (Meter, error) {
				m, err := ConvertMeter(item)
				return Meter{
					Meter: m,
				}, err
			})
			if err != nil {
				return ListMetersResponse{}, apierrors.NewInternalError(ctx, err)
			}

			// Response
			resp := response.NewCursorPaginationResponse(meters)

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMetersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listMeters"),
		)...,
	)
}
