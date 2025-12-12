package customers

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	customer "github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteCustomerRequest struct {
		Namespace  string
		CustomerID string
	}
	DeleteCustomerResponse = interface{}
	DeleteCustomerParams   = string
	DeleteCustomerHandler  httptransport.HandlerWithArgs[DeleteCustomerRequest, DeleteCustomerResponse, DeleteCustomerParams]
)

// DeleteCustomer returns a handler for deleting a customer.
func (h *handler) DeleteCustomer() DeleteCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID DeleteCustomerParams) (DeleteCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerRequest{}, err
			}

			return DeleteCustomerRequest{
				Namespace:  ns,
				CustomerID: customerID,
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerRequest) (DeleteCustomerResponse, error) {
			err := h.service.DeleteCustomer(ctx, customer.DeleteCustomerInput{
				Namespace: request.Namespace,
				ID:        request.CustomerID,
			})
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteCustomerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-customer"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
