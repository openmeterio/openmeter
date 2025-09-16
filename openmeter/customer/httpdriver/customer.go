package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomersResponse = pagination.Result[api.Customer]
	ListCustomersParams   = api.ListCustomersParams
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)

type ListCustomersRequest struct {
	customer.ListCustomersInput
	Expand []api.CustomerExpand
}

// ListCustomers returns a handler for listing customers.
func (h *handler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomersParams) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			req := ListCustomersRequest{
				ListCustomersInput: customer.ListCustomersInput{
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
					Key:          params.Key,
					Name:         params.Name,
					PrimaryEmail: params.PrimaryEmail,
					Subject:      params.Subject,
					PlanKey:      params.PlanKey,

					// Modifiers
					IncludeDeleted: lo.FromPtrOr(params.IncludeDeleted, customer.IncludeDeleted),
				},

				// Expand
				Expand: lo.FromPtrOr(params.Expand, []api.CustomerExpand{}),
			}

			if err := req.Page.Validate(); err != nil {
				return ListCustomersRequest{}, err
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomersRequest) (ListCustomersResponse, error) {
			resp, err := h.service.ListCustomers(ctx, request.ListCustomersInput)
			if err != nil {
				return ListCustomersResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			// Get the customer's subscriptions
			var customerSubscriptions map[string][]subscription.Subscription

			if len(resp.Items) > 0 {
				customerIDs := lo.Map(resp.Items, func(item customer.Customer, _ int) string {
					return item.ID
				})

				subscriptions, err := h.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
					Namespaces: []string{request.Namespace},
					Customers:  customerIDs,
					ActiveAt:   lo.ToPtr(time.Now()),
				})
				if err != nil {
					return ListCustomersResponse{}, err
				}

				customerSubscriptions = lo.GroupBy(subscriptions.Items, func(item subscription.Subscription) string {
					return item.CustomerId
				})
			}

			// Map the customers to the API
			return pagination.MapResultErr(resp, func(customer customer.Customer) (api.Customer, error) {
				var item api.Customer

				subs, ok := customerSubscriptions[customer.ID]
				if !ok {
					subs = []subscription.Subscription{}
				}

				item, err = CustomerToAPI(customer, subs, request.Expand)
				if err != nil {
					return item, fmt.Errorf("failed to cast customer customer: %w", err)
				}

				return item, nil
			})
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomers"),
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
			body := api.CustomerCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateCustomerRequest{}, fmt.Errorf("field to decode create customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerRequest{}, err
			}

			req := CreateCustomerRequest{
				Namespace:      ns,
				CustomerMutate: MapCustomerCreate(body),
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

			return h.mapCustomerWithSubscriptionsToAPI(ctx, *customer, nil)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCustomer"),
		)...,
	)
}

type (
	UpdateCustomerRequest  = customer.UpdateCustomerInput
	UpdateCustomerResponse = api.Customer
	UpdateCustomerHandler  httptransport.HandlerWithArgs[UpdateCustomerRequest, UpdateCustomerResponse, string]
)

// UpdateCustomer returns a handler for updating a customer.
func (h *handler) UpdateCustomer() UpdateCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIDOrKey string) (UpdateCustomerRequest, error) {
			body := api.CustomerReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateCustomerRequest{}, fmt.Errorf("field to decode update customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerRequest{}, err
			}

			// TODO: we should not allow key identifier for mutable operations
			// Get the customer
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   customerIDOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return UpdateCustomerRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return UpdateCustomerRequest{},
					models.NewGenericPreConditionFailedError(
						fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			req := UpdateCustomerRequest{
				CustomerID:     cus.GetID(),
				CustomerMutate: MapCustomerReplaceUpdate(body),
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

			return h.mapCustomerWithSubscriptionsToAPI(ctx, *customer, nil)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateCustomer"),
		)...,
	)
}

type (
	DeleteCustomerRequest  = customer.DeleteCustomerInput
	DeleteCustomerResponse = interface{}
	DeleteCustomerHandler  httptransport.HandlerWithArgs[DeleteCustomerRequest, DeleteCustomerResponse, string]
)

// DeleteCustomer returns a handler for deleting a customer.
func (h *handler) DeleteCustomer() DeleteCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIDOrKey string) (DeleteCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerRequest{}, err
			}

			// TODO: we should not allow key identifier for mutable operations
			// Get the customer
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   customerIDOrKey,
					Namespace: ns,
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

			return cus.GetID(), nil
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
		)...,
	)
}

type (
	GetCustomerRequest  = customer.GetCustomerInput
	GetCustomerResponse = api.Customer
	GetCustomerHandler  httptransport.HandlerWithArgs[GetCustomerRequest, GetCustomerResponse, GetCustomerParams]
)

type GetCustomerParams struct {
	CustomerIDOrKey string
	api.GetCustomerParams
}

// GetCustomer returns a handler for getting a customer.
func (h *handler) GetCustomer() GetCustomerHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerParams) (GetCustomerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerRequest{}, err
			}

			return GetCustomerRequest{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   params.CustomerIDOrKey,
				},

				// Expand
				Expand: lo.FromPtrOr(params.Expand, []api.CustomerExpand{}),
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

			return h.mapCustomerWithSubscriptionsToAPI(ctx, *cus, request.Expand)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomer"),
		)...,
	)
}

type (
	GetCustomerEntitlementValueRequest  = customer.GetEntitlementValueInput
	GetCustomerEntitlementValueResponse = api.EntitlementValue
	GetCustomerEntitlementValueParams   = struct {
		CustomerIDOrKey string
		FeatureKey      string
	}
	GetCustomerEntitlementValueHandler httptransport.HandlerWithArgs[GetCustomerEntitlementValueRequest, GetCustomerEntitlementValueResponse, GetCustomerEntitlementValueParams]
)

// GetCustomerEntitlementValue returns a handler for getting a customer.
func (h *handler) GetCustomerEntitlementValue() GetCustomerEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementValueParams) (GetCustomerEntitlementValueRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementValueRequest{}, err
			}

			// Get the customer
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIDOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return GetCustomerEntitlementValueRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return GetCustomerEntitlementValueRequest{},
					models.NewGenericPreConditionFailedError(
						fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			return GetCustomerEntitlementValueRequest{
				FeatureKey: params.FeatureKey,
				CustomerID: cus.GetID(),
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementValueRequest) (GetCustomerEntitlementValueResponse, error) {
			val, err := h.entitlementService.GetEntitlementValue(ctx, request.CustomerID.Namespace, request.CustomerID.ID, request.FeatureKey, clock.Now())
			if err != nil {
				if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
					val = &entitlement.NoAccessValue{}
					err = nil
				}
			}

			if err != nil {
				return GetCustomerEntitlementValueResponse{}, err
			}

			return entitlementdriver.MapEntitlementValueToAPI(val)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementValueResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomer"),
		)...,
	)
}

type (
	GetCustomerAccessRequest  = customer.GetCustomerInput
	GetCustomerAccessResponse = api.CustomerAccess
	GetCustomerAccessParams   = struct {
		CustomerIDOrKey string
	}
	GetCustomerAccessHandler httptransport.HandlerWithArgs[GetCustomerAccessRequest, GetCustomerAccessResponse, GetCustomerAccessParams]
)

// GetCustomerAccess returns a handler for getting a customer access.
func (h *handler) GetCustomerAccess() GetCustomerAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerAccessParams) (GetCustomerAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerAccessRequest{}, err
			}

			return GetCustomerAccessRequest{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   params.CustomerIDOrKey,
				},
			}, nil
		},
		func(ctx context.Context, request GetCustomerAccessRequest) (GetCustomerAccessResponse, error) {
			cus, err := h.service.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: request.CustomerIDOrKey,
			})
			if err != nil {
				return GetCustomerAccessResponse{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return GetCustomerAccessResponse{},
					models.NewGenericPreConditionFailedError(
						fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			access, err := h.entitlementService.GetAccess(ctx, cus.Namespace, cus.ID)
			if err != nil {
				return GetCustomerAccessResponse{}, err
			}

			apiAccess, err := MapAccessToAPI(access)
			if err != nil {
				return GetCustomerAccessResponse{}, err
			}

			return apiAccess, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomerAccess"),
		)...,
	)
}

// mapCustomerWithSubscriptionsToAPI maps a customer to the API with its subscriptions.
func (h *handler) mapCustomerWithSubscriptionsToAPI(ctx context.Context, customer customer.Customer, expand []api.CustomerExpand) (api.Customer, error) {
	// Get the customer's subscriptions
	subscriptions, err := h.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{customer.Namespace},
		Customers:  []string{customer.ID},
		ActiveAt:   lo.ToPtr(time.Now()),
	})
	if err != nil {
		return GetCustomerResponse{}, err
	}

	// Map the customer to the API
	return CustomerToAPI(customer, subscriptions.Items, expand)
}
