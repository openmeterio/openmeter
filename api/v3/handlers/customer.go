package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type CustomerHandler interface {
	ListCustomers() ListCustomersHandler
	CreateCustomer() CreateCustomerHandler
	DeleteCustomer() DeleteCustomerHandler
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
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersResponse = response.OffsetPaginationResponse[Customer]
	ListCustomersHandler  httptransport.Handler[ListCustomersRequest, ListCustomersResponse]
)

func (h *customerHandler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			attributes, err := request.GetAttributes(r,
				request.WithOffsetPagination(),
				request.WithDefaultSort(&request.SortBy{Field: "name", Order: request.SortOrderAsc}),
			)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			req := ListCustomersRequest{
				Namespace: ns,
				Page:      pagination.NewPage(attributes.Pagination.Number, attributes.Pagination.Size),
			}

			// Pick the first sort if there are multiple
			if len(attributes.Sorts) > 0 {
				req.OrderBy = attributes.Sorts[0].Field
				req.Order = attributes.Sorts[0].Order.ToSortxOrder()
			}

			// Filters
			if attributes.Filters != nil {
				for field, f := range attributes.Filters {
					switch field {
					case "key":
						req.Key = f.ToFilterString()
					case "name":
						req.Name = f.ToFilterString()
					case "primary_email":
						req.PrimaryEmail = f.ToFilterString()
					case "subject":
						req.Subject = f.ToFilterString()
					case "customer_ids":
						req.CustomerIDs = f.ToFilterString()
					}
				}
			}

			slog.Info("request", "request", req)

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
			r := response.NewOffsetPaginationResponse(customers, response.OffsetMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

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

type (
	DeleteCustomerRequest struct {
		Namespace       string
		CustomerIDOrKey string
	}
	DeleteCustomerResponse = interface{}
	DeleteCustomerParams   = string
	DeleteCustomerHandler  httptransport.HandlerWithArgs[DeleteCustomerRequest, DeleteCustomerResponse, DeleteCustomerParams]
)

// DeleteCustomer returns a handler for deleting a customer.
func (h *customerHandler) DeleteCustomer() DeleteCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIDOrKey DeleteCustomerParams) (DeleteCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerRequest{}, err
			}

			return DeleteCustomerRequest{
				Namespace:       ns,
				CustomerIDOrKey: customerIDOrKey,
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerRequest) (DeleteCustomerResponse, error) {
			// TODO: we should not allow key identifier for mutable operations
			// Get the customer
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   request.CustomerIDOrKey,
					Namespace: request.Namespace,
				},
			})
			if err != nil {
				return DeleteCustomerRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return DeleteCustomerRequest{},
					models.NewGenericPreConditionFailedError(
						fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			err = h.service.DeleteCustomer(ctx, cus.GetID())
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteCustomerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-customer"),
		)...,
	)
}
