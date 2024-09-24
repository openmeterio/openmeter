package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersResponse = api.CustomerList
	ListCustomersParams   = api.ListCustomersParams
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)

// ListCustomers returns a handler for listing customers.
func (h *handler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomersParams) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListCustomersRequest{
				Namespace:      ns,
				IncludeDeleted: lo.FromPtrOr(params.IncludeDeleted, customer.IncludeDeleted),
				// OrderBy:        defaultx.WithDefault(params.OrderBy, api.ListCustomersParamsOrderById),
				// Order:          sortx.Order(defaultx.WithDefault(params.Order, api.ListCustomersParamsOrderSortOrderASC)),
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, customer.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, customer.DefaultPageNumber),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomersRequest) (ListCustomersResponse, error) {
			resp, err := h.service.ListCustomers(ctx, request)
			if err != nil {
				return ListCustomersResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			items := make([]api.Customer, 0, len(resp.Items))

			for _, customer := range resp.Items {
				var item api.Customer

				item, err = customer.AsAPICustomer()
				if err != nil {
					return ListCustomersResponse{}, fmt.Errorf("failed to cast customer customer: %w", err)
				}

				items = append(items, item)
			}

			return ListCustomersResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomers"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreateCustomerRequest  = customer.CreateCustomerInput
	CreateCustomerResponse = api.Customer
	CreateCustomerHandler  httptransport.Handler[CreateCustomerRequest, CreateCustomerResponse]
)

// CreateCustomer returns a new httptransport.Handler for creating a customer.
func (h *handler) CreateCustomer() CreateCustomerHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCustomerRequest, error) {
			body := api.Customer{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateCustomerRequest{}, fmt.Errorf("field to decode create customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := newCreateCustomerInput(ns, body)

			return req, nil
		},
		func(ctx context.Context, request CreateCustomerRequest) (CreateCustomerResponse, error) {
			customer, err := h.service.CreateCustomer(ctx, request)
			if err != nil {
				return CreateCustomerResponse{}, fmt.Errorf("failed to create customer: %w", err)
			}

			return customer.AsAPICustomer()
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateCustomerRequest  = customer.UpdateCustomerInput
	UpdateCustomerResponse = api.Customer
	UpdateCustomerHandler  httptransport.HandlerWithArgs[UpdateCustomerRequest, UpdateCustomerResponse, api.ULID]
)

// UpdateCustomer returns a handler for updating a customer.
func (h *handler) UpdateCustomer() UpdateCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID api.ULID) (UpdateCustomerRequest, error) {
			body := api.Customer{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateCustomerRequest{}, fmt.Errorf("field to decode update customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := newUpdateCustomerInput(ns, body)
			req.ID = customerID

			return req, nil
		},
		func(ctx context.Context, request UpdateCustomerRequest) (UpdateCustomerResponse, error) {
			customer, err := h.service.UpdateCustomer(ctx, request)
			if err != nil {
				return UpdateCustomerResponse{}, fmt.Errorf("failed to update customer: %w", err)
			}

			return customer.AsAPICustomer()
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteCustomerRequest  = customer.DeleteCustomerInput
	DeleteCustomerResponse = interface{}
	DeleteCustomerHandler  httptransport.HandlerWithArgs[DeleteCustomerRequest, DeleteCustomerResponse, api.CustomerIdentifier]
)

// DeleteCustomer returns a handler for deleting a customer.
func (h *handler) DeleteCustomer() DeleteCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID api.CustomerIdentifier) (DeleteCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeleteCustomerRequest{
				Namespace: ns,
				ID:        customerID,
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerRequest) (DeleteCustomerResponse, error) {
			err := h.service.DeleteCustomer(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete customer: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteCustomerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetCustomerRequest  = customer.GetCustomerInput
	GetCustomerResponse = api.Customer
	GetCustomerHandler  httptransport.HandlerWithArgs[GetCustomerRequest, GetCustomerResponse, api.CustomerIdentifier]
)

// GetCustomer returns a handler for getting a customer.
func (h *handler) GetCustomer() GetCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID api.CustomerIdentifier) (GetCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetCustomerRequest{
				Namespace: ns,
				ID:        customerID,
			}, nil
		},
		func(ctx context.Context, request GetCustomerRequest) (GetCustomerResponse, error) {
			customer, err := h.service.GetCustomer(ctx, request)
			if err != nil {
				return GetCustomerResponse{}, fmt.Errorf("failed to get customer: %w", err)
			}

			return customer.AsAPICustomer()
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
