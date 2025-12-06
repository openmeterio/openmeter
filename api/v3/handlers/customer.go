package handlers

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/samber/lo"
)

type CustomerHandler interface {
	ListCustomers() ListCustomersHandler
	CreateCustomer() CreateCustomerHandler
	// DeleteCustomer() DeleteCustomerHandler
	GetCustomer() GetCustomerHandler
	// UpdateCustomer() UpdateCustomerHandler
}

type customerHandler struct {
	service          customer.Service
	resolveNamespace func(ctx context.Context) (string, error)
	options          []httptransport.HandlerOption
}

func NewCustomerHandler(
	resolveNamespace func(ctx context.Context) (string, error),
	service customer.Service,
	options ...httptransport.HandlerOption,
) CustomerHandler {
	return &customerHandler{
		service:          service,
		resolveNamespace: resolveNamespace,
		options:          options,
	}
}

type (
	ListCustomersParams   = v3.ListCustomersParams
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersResponse = response.CursorPaginationResponse[Customer]
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)

func (h *customerHandler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomersParams) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			req := ListCustomersRequest{
				Namespace: ns,

				// TODO cursor pagination
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomersRequest) (ListCustomersResponse, error) {
			resp, err := h.service.ListCustomers(ctx, request)
			if err != nil {
				return ListCustomersResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			customers := lo.Map(resp.Items, func(item customer.Customer, _ int) Customer {
				return Customer{
					BillingCustomer: ConvertCustomerRequestToBillingCustomer(item),
				}
			})

			// Map the customers to the API
			r := response.NewCursorPaginationResponse(customers)
			// TODO: set the size of the page from the request params
			// r.Meta.Page.Size = request.Page.Size
			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customers"),
		)...,
	)
}

type (
	CreateCustomerRequest  = customer.CreateCustomerInput
	CreateCustomerResponse = api.BillingCustomer
	CreateCustomerHandler  httptransport.Handler[CreateCustomerRequest, CreateCustomerResponse]
)

// CreateCustomer returns a new httptransport.Handler for creating a customer.
func (h *customerHandler) CreateCustomer() CreateCustomerHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCustomerRequest, error) {
			body := api.CreateCustomerRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerRequest{}, err
			}

			req := ConvertFromCreateCustomerRequestToCreateCustomerInput(ns, body)

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

			return ConvertCustomerRequestToBillingCustomer(*customer), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer"),
		)...,
	)
}

type (
	GetCustomerRequest  = customer.GetCustomerInput
	GetCustomerResponse = api.BillingCustomer
	GetCustomerParams   = string
	GetCustomerHandler  httptransport.HandlerWithArgs[GetCustomerRequest, GetCustomerResponse, GetCustomerParams]
)

// GetCustomer returns a handler for getting a customer.
func (h *customerHandler) GetCustomer() GetCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIDOrKey GetCustomerParams) (GetCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerRequest{}, err
			}

			return GetCustomerRequest{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   customerIDOrKey,
				},
			}, nil
		},
		func(ctx context.Context, request GetCustomerRequest) (GetCustomerResponse, error) {
			// Get the customer
			cus, err := h.service.GetCustomer(ctx, request)
			if err != nil {
				return GetCustomerResponse{}, err
			}

			if cus == nil {
				return GetCustomerResponse{}, fmt.Errorf("failed to get customer")
			}

			return ConvertCustomerRequestToBillingCustomer(*cus), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer"),
		)...,
	)
}
