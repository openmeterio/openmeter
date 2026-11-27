package customersentitlements

import (
	"context"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementHistoryRequest  = entitlement.GetCustomerEntitlementHistoryInput
	GetCustomerEntitlementHistoryResponse = apiv3models.BillingEntitlementHistory
	GetCustomerEntitlementHistoryParams   struct {
		CustomerID    apiv3models.ULID
		EntitlementID apiv3models.ULID
		Params        apiv3models.GetCustomerEntitlementHistoryParams
	}
	GetCustomerEntitlementHistoryHandler = httptransport.HandlerWithArgs[GetCustomerEntitlementHistoryRequest, GetCustomerEntitlementHistoryResponse, GetCustomerEntitlementHistoryParams]
)

func (h *handler) GetCustomerEntitlementHistory() GetCustomerEntitlementHistoryHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementHistoryParams) (GetCustomerEntitlementHistoryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementHistoryRequest{}, err
			}

			windowSize, err := mapHistoryWindowSize(params.Params.WindowSize)
			if err != nil {
				return GetCustomerEntitlementHistoryRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "window_size", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			timeZone := time.UTC
			if params.Params.TimeZone != nil {
				timeZone, err = time.LoadLocation(*params.Params.TimeZone)
				if err != nil {
					return GetCustomerEntitlementHistoryRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "time_zone", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
			}

			return GetCustomerEntitlementHistoryRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID: params.EntitlementID,
				From:          params.Params.From,
				To:            params.Params.To,
				WindowSize:    windowSize,
				TimeZone:      timeZone,
			}, nil
		},
		func(ctx context.Context, req GetCustomerEntitlementHistoryRequest) (GetCustomerEntitlementHistoryResponse, error) {
			history, err := h.service.GetCustomerEntitlementHistory(ctx, req)
			if err != nil {
				return GetCustomerEntitlementHistoryResponse{}, err
			}

			return mapHistoryToAPI(history), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementHistoryResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-history"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			// Reading a metered balance saves a snapshot, so this GET cannot be read-only.
			httptransport.WithPrimaryDatabase(),
		)...,
	)
}
