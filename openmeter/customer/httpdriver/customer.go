package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomersRequest  = customerentity.ListCustomersInput
	ListCustomersResponse = api.CustomerPaginatedResponse
	ListCustomersParams   = api.ListCustomersParams
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)

// ListCustomers returns a handler for listing customers.
func (h *handler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomersParams) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			req := ListCustomersRequest{
				Namespace: ns,

				// Pagination
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, customer.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, customer.DefaultPageNumber),
				},

				// Order
				OrderBy: defaultx.WithDefault(params.OrderBy, api.CustomerOrderByName),
				Order:   sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderASC)),

				// Filters
				Name:         params.Name,
				PrimaryEmail: params.PrimaryEmail,
				Subject:      params.Subject,

				// Modifiers
				IncludeDeleted: lo.FromPtrOr(params.IncludeDeleted, customer.IncludeDeleted),
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

				item, err = customerToAPI(customer)
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
	CreateCustomerRequest  = customerentity.CreateCustomerInput
	CreateCustomerResponse = api.Customer
	CreateCustomerHandler  httptransport.Handler[CreateCustomerRequest, CreateCustomerResponse]
)

// CreateCustomer returns a new httptransport.Handler for creating a customer.
func (h *handler) CreateCustomer() CreateCustomerHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCustomerRequest, error) {
			body := api.CustomerCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateCustomerRequest{}, fmt.Errorf("field to decode create customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerRequest{}, err
			}

			req := CreateCustomerRequest{
				Namespace: ns,
				CustomerMutate: customerentity.CustomerMutate{
					Name:             body.Name,
					Description:      body.Description,
					UsageAttribution: customerentity.CustomerUsageAttribution(body.UsageAttribution),
					PrimaryEmail:     body.PrimaryEmail,
					BillingAddress:   mapAddress(body.BillingAddress),
					Currency:         mapCurrency(body.Currency),
					Timezone:         mapTimezone(body.Timezone),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request CreateCustomerRequest) (CreateCustomerResponse, error) {
			customer, err := h.service.CreateCustomer(ctx, request)
			if err != nil {
				return CreateCustomerResponse{}, err
			}

			if customer == nil {
				return CreateCustomerResponse{}, fmt.Errorf("failed to create customer")
			}

			return customerToAPI(*customer)
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
	UpdateCustomerRequest  = customerentity.UpdateCustomerInput
	UpdateCustomerResponse = api.Customer
	UpdateCustomerHandler  httptransport.HandlerWithArgs[UpdateCustomerRequest, UpdateCustomerResponse, string]
)

// UpdateCustomer returns a handler for updating a customer.
func (h *handler) UpdateCustomer() UpdateCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID string) (UpdateCustomerRequest, error) {
			body := api.CustomerReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateCustomerRequest{}, fmt.Errorf("field to decode update customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerRequest{}, err
			}

			req := UpdateCustomerRequest{
				CustomerID: customerentity.CustomerID{
					Namespace: ns,
					ID:        customerID,
				},
				CustomerMutate: customerentity.CustomerMutate{
					Name:             body.Name,
					Description:      body.Description,
					UsageAttribution: customerentity.CustomerUsageAttribution(body.UsageAttribution),
					PrimaryEmail:     body.PrimaryEmail,
					BillingAddress:   mapAddress(body.BillingAddress),
					Currency:         mapCurrency(body.Currency),
					Timezone:         mapTimezone(body.Timezone),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateCustomerRequest) (UpdateCustomerResponse, error) {
			customer, err := h.service.UpdateCustomer(ctx, request)
			if err != nil {
				return UpdateCustomerResponse{}, err
			}

			if customer == nil {
				return UpdateCustomerResponse{}, fmt.Errorf("failed to update customer")
			}

			return customerToAPI(*customer)
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
	DeleteCustomerRequest  = customerentity.DeleteCustomerInput
	DeleteCustomerResponse = interface{}
	DeleteCustomerHandler  httptransport.HandlerWithArgs[DeleteCustomerRequest, DeleteCustomerResponse, string]
)

// DeleteCustomer returns a handler for deleting a customer.
func (h *handler) DeleteCustomer() DeleteCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID string) (DeleteCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerRequest{}, err
			}

			return DeleteCustomerRequest{
				Namespace: ns,
				ID:        customerID,
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerRequest) (DeleteCustomerResponse, error) {
			err := h.service.DeleteCustomer(ctx, request)
			if err != nil {
				return nil, err
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
	GetCustomerRequest  = customerentity.GetCustomerInput
	GetCustomerResponse = api.Customer
	GetCustomerHandler  httptransport.HandlerWithArgs[GetCustomerRequest, GetCustomerResponse, string]
)

// GetCustomer returns a handler for getting a customer.
func (h *handler) GetCustomer() GetCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID string) (GetCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerRequest{}, err
			}

			return GetCustomerRequest{
				Namespace: ns,
				ID:        customerID,
			}, nil
		},
		func(ctx context.Context, request GetCustomerRequest) (GetCustomerResponse, error) {
			customer, err := h.service.GetCustomer(ctx, request)
			if err != nil {
				return GetCustomerResponse{}, err
			}

			if customer == nil {
				return GetCustomerResponse{}, fmt.Errorf("failed to get customer")
			}

			return customerToAPI(*customer)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomer"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
