package meters

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/query"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	QueryMeterRequest struct {
		models.NamespacedID
		Body api.MeterQueryRequest
	}
	QueryMeterResponse = api.MeterQueryResult
	QueryMeterParams   = string
	QueryMeterHandler  httptransport.HandlerWithArgs[QueryMeterRequest, QueryMeterResponse, QueryMeterParams]
)

func (h *handler) QueryMeter() QueryMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID QueryMeterParams) (QueryMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterRequest{}, err
			}

			var body api.MeterQueryRequest
			if err := request.ParseBody(r, &body); err != nil {
				return QueryMeterRequest{}, err
			}

			return QueryMeterRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        meterID,
				},
				Body: body,
			}, nil
		},
		func(ctx context.Context, req QueryMeterRequest) (QueryMeterResponse, error) {
			m, err := h.service.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: req.Namespace,
				IDOrSlug:  req.ID,
			})
			if err != nil {
				return QueryMeterResponse{}, err
			}

			params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))
			if err != nil {
				return QueryMeterResponse{}, err
			}

			rows, err := h.streaming.QueryMeter(ctx, req.Namespace, m, params)
			if err != nil {
				return QueryMeterResponse{}, err
			}

			return ConvertMeterQueryResultToAPI(req.Body.From, req.Body.To, rows), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
