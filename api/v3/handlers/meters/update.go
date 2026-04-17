package meters

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateMeterRequest  = meter.UpdateMeterInput
	UpdateMeterResponse = api.Meter
	UpdateMeterParams   = string
	UpdateMeterHandler  httptransport.HandlerWithArgs[UpdateMeterRequest, UpdateMeterResponse, UpdateMeterParams]
)

// UpdateMeter returns a new httptransport.Handler for updating a meter.
func (h *handler) UpdateMeter() UpdateMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID UpdateMeterParams) (UpdateMeterRequest, error) {
			body := api.UpdateMeterRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateMeterRequest{}, err
			}

			if body.Dimensions != nil {
				if err := validateDimensionsWithoutReserved(*body.Dimensions); err != nil {
					return UpdateMeterRequest{}, err
				}
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateMeterRequest{}, err
			}

			return FromAPIUpdateMeterRequest(ns, meterID, body)
		},
		func(ctx context.Context, request UpdateMeterRequest) (UpdateMeterResponse, error) {
			m, err := h.service.UpdateMeter(ctx, request)
			if err != nil {
				return UpdateMeterResponse{}, err
			}

			return ToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
