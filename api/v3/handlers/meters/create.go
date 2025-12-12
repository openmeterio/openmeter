package meters

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	meter "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateMeterRequest  = meter.CreateMeterInput
	CreateMeterResponse = api.Meter
	CreateMeterHandler  httptransport.Handler[CreateMeterRequest, CreateMeterResponse]
)

// CreateMeter returns a new httptransport.Handler for creating a meter.
func (h *handler) CreateMeter() CreateMeterHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateMeterRequest, error) {
			body := api.CreateMeterRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateMeterRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateMeterRequest{}, err
			}

			req := ConvertFromCreateMeterRequestToCreateMeterInput(ns, body)

			return req, nil
		},
		func(ctx context.Context, request CreateMeterRequest) (CreateMeterResponse, error) {
			m, err := h.service.CreateMeter(ctx, request)
			if err != nil {
				return CreateMeterResponse{}, err
			}

			return ConvertMeterToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateMeterResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
