package meters

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	meter "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteMeterRequest struct {
		Namespace string
		IDOrSlug  string
	}
	DeleteMeterResponse = interface{}
	DeleteMeterParams   = string
	DeleteMeterHandler  httptransport.HandlerWithArgs[DeleteMeterRequest, DeleteMeterResponse, DeleteMeterParams]
)

// DeleteMeter returns a handler for deleting a meter.
func (h *handler) DeleteMeter() DeleteMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID DeleteMeterParams) (DeleteMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteMeterRequest{}, err
			}

			return DeleteMeterRequest{
				Namespace: ns,
				IDOrSlug:  meterID,
			}, nil
		},
		func(ctx context.Context, request DeleteMeterRequest) (DeleteMeterResponse, error) {
			// FIXME: make delete idempotent, return 204 for repeated deletion
			err := h.service.DeleteMeter(ctx, meter.DeleteMeterInput{
				Namespace: request.Namespace,
				IDOrSlug:  request.IDOrSlug,
			})
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteMeterResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
