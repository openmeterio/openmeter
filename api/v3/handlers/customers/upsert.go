package customers

import (
	"context"
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpsertCustomerRequest struct {
		Namespace      string
		CustomerID     string
		CustomerMutate customer.CustomerMutate
	}
	UpsertCustomerParams   = string
	UpsertCustomerResponse = api.BillingCustomer
	UpsertCustomerHandler  httptransport.HandlerWithArgs[UpsertCustomerRequest, UpsertCustomerResponse, UpsertCustomerParams]
)

// UpdateCustomer returns a handler for updating a customer.
func (h *handler) UpsertCustomer() UpsertCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID UpsertCustomerParams) (UpsertCustomerRequest, error) {
			body := api.UpsertCustomerRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpsertCustomerRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertCustomerRequest{}, err
			}

			req := UpsertCustomerRequest{
				Namespace:  ns,
				CustomerID: customerID,
				// Key cannot be updated according to api.UpsertCustomerRequest.
				// Therefore, at this point we don't have a key yet. It is ignored in this conversion.
				CustomerMutate: ConvertUpsertCustomerRequestToCustomerMutate(body),
			}

			return req, nil
		},
		func(ctx context.Context, request UpsertCustomerRequest) (UpsertCustomerResponse, error) {
			// Get the customer
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					ID:        request.CustomerID,
					Namespace: request.Namespace,
				},
			})
			if err != nil {
				return UpsertCustomerResponse{}, err
			}

			if cus == nil {
				return UpsertCustomerResponse{}, apierrors.NewNotFoundError(ctx, errors.New("customer not found"), "customer")
			}

			if cus.IsDeleted() {
				return UpsertCustomerResponse{},
					apierrors.NewGoneError(
						ctx,
						errors.New("customer is deleted"),
					)
			}

			// Use the key from the just retrieved customer, as it is required by the UpdateCustomer service method.
			request.CustomerMutate.Key = cus.Key

			updatedCustomer, err := h.service.UpdateCustomer(ctx, customer.UpdateCustomerInput{
				CustomerID:     cus.GetID(),
				CustomerMutate: request.CustomerMutate,
			})
			if err != nil {
				return UpsertCustomerResponse{}, err
			}

			if updatedCustomer == nil {
				return UpsertCustomerResponse{}, errors.New("failed to update customer")
			}

			return ConvertCustomerRequestToBillingCustomer(*updatedCustomer), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpsertCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsert-customer"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
