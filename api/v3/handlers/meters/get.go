package meters

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetMeterRequest  = meter.GetMeterInput
	GetMeterResponse = api.Meter
	GetMeterParams   = string
	GetMeterHandler  httptransport.HandlerWithArgs[GetMeterRequest, GetMeterResponse, GetMeterParams]
)

// GetMeter returns a handler for getting a meter.
func (h *handler) GetMeter() GetMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID GetMeterParams) (GetMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetMeterRequest{}, err
			}

			return GetMeterRequest{
				Namespace: ns,
				IDOrSlug:  meterID,
			}, nil
		},
		func(ctx context.Context, request GetMeterRequest) (GetMeterResponse, error) {
			// Get the meter
			m, err := h.service.GetMeterByIDOrSlug(ctx, request)
			if err != nil {
				return GetMeterResponse{}, err
			}

			return ConvertMeterToAPIMeter(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
