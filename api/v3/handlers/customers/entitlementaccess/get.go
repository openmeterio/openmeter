package customersentitlement

import (
	"context"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementValueParams struct {
		CustomerID    apiv3models.ULID
		EntitlementID string
		Expand        []apiv3models.BillingEntitlementAccessExpand
		At            *time.Time
	}
	GetCustomerEntitlementValueResponse = apiv3models.BillingEntitlementValueResult
	GetCustomerEntitlementValueHandler  httptransport.HandlerWithArgs[GetCustomerEntitlementValueRequest, GetCustomerEntitlementValueResponse, GetCustomerEntitlementValueParams]

	GetCustomerEntitlementValueByFeatureKeyParams struct {
		CustomerID apiv3models.ULID
		FeatureKey string
		Expand     []apiv3models.BillingEntitlementAccessExpand
		At         *time.Time
	}
	GetCustomerEntitlementValueByFeatureKeyResponse = apiv3models.BillingEntitlementFeatureValueResult
	GetCustomerEntitlementValueByFeatureKeyHandler  httptransport.HandlerWithArgs[GetCustomerEntitlementValueRequest, GetCustomerEntitlementValueByFeatureKeyResponse, GetCustomerEntitlementValueByFeatureKeyParams]
)

type GetCustomerEntitlementValueRequest struct {
	entitlement.GetCustomerEntitlementAccessInput
	Expands []apiv3models.BillingEntitlementAccessExpand
	current bool
}

func (h *handler) GetCustomerEntitlementValue() GetCustomerEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementValueParams) (GetCustomerEntitlementValueRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementValueRequest{}, err
			}

			return GetCustomerEntitlementValueRequest{
				GetCustomerEntitlementAccessInput: entitlement.GetCustomerEntitlementAccessInput{
					CustomerID:    customer.CustomerID{Namespace: ns, ID: params.CustomerID},
					EntitlementID: params.EntitlementID,
					At:            lo.FromPtrOr(params.At, clock.Now()),
				},
				current: params.At == nil,
				Expands: params.Expand,
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementValueRequest) (GetCustomerEntitlementValueResponse, error) {
			if request.current {
				ctx = entitlement.CurrentValueOperationRead.WithContext(ctx)
			}

			access, err := h.entitlementService.GetCustomerEntitlementAccess(ctx, request.GetCustomerEntitlementAccessInput)
			if err != nil {
				return GetCustomerEntitlementValueResponse{}, err
			}

			return mapEntitlementAccessToAPI(access, request.Expands...)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementValueResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-value"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func (h *handler) GetCustomerEntitlementValueByFeatureKey() GetCustomerEntitlementValueByFeatureKeyHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementValueByFeatureKeyParams) (GetCustomerEntitlementValueRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementValueRequest{}, err
			}

			return GetCustomerEntitlementValueRequest{
				GetCustomerEntitlementAccessInput: entitlement.GetCustomerEntitlementAccessInput{
					CustomerID: customer.CustomerID{Namespace: ns, ID: params.CustomerID},
					FeatureKey: params.FeatureKey,
					At:         lo.FromPtrOr(params.At, clock.Now()),
				},
				current: params.At == nil,
				Expands: params.Expand,
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementValueRequest) (GetCustomerEntitlementValueByFeatureKeyResponse, error) {
			if request.current {
				ctx = entitlement.CurrentValueOperationRead.WithContext(ctx)
			}

			access, err := h.entitlementService.GetCustomerEntitlementAccess(ctx, request.GetCustomerEntitlementAccessInput)
			if err != nil {
				return GetCustomerEntitlementValueByFeatureKeyResponse{}, err
			}

			if access.Type == "" {
				return GetCustomerEntitlementValueByFeatureKeyResponse{
					FeatureKey: request.FeatureKey,
					HasAccess:  false,
				}, nil
			}

			value, err := mapEntitlementAccessToAPI(access, request.Expands...)
			if err != nil {
				return GetCustomerEntitlementValueByFeatureKeyResponse{}, err
			}

			return GetCustomerEntitlementValueByFeatureKeyResponse{
				FeatureKey: value.FeatureKey,
				Type:       lo.ToPtr(value.Type),
				HasAccess:  value.HasAccess,
				Config:     value.Config,
				Value:      value.Value,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementValueByFeatureKeyResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-value-by-feature-key"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
