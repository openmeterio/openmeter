package entitlementdriverv2

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// CustomerEntitlementHandler exposes V2 customer entitlement endpoints
type CustomerEntitlementHandler interface {
	CreateCustomerEntitlement() CreateCustomerEntitlementHandler
}

type customerEntitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.Connector
	customerService  customer.Service
}

func NewCustomerEntitlementHandler(
	connector entitlement.Connector,
	customerService customer.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) CustomerEntitlementHandler {
	return &customerEntitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		customerService:  customerService,
	}
}

type (
	CreateCustomerEntitlementHandlerRequest  = entitlement.CreateEntitlementInputs
	CreateCustomerEntitlementHandlerResponse = api.EntitlementV2
	CreateCustomerEntitlementHandlerParams   = string // customerIdOrKey
)

type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]

func (h *customerEntitlementHandler) CreateCustomerEntitlement() CreateCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs[
		CreateCustomerEntitlementHandlerRequest,
		CreateCustomerEntitlementHandlerResponse,
		CreateCustomerEntitlementHandlerParams,
	](
		func(ctx context.Context, r *http.Request, customerIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			inp := &api.EntitlementCreateInputs{}
			request := entitlement.CreateEntitlementInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return request, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return request, err
			}

			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   customerIdOrKey,
				},
			})
			if err != nil {
				return request, err
			}

			// Resolve subject key from customer usage attribution (expects 1:1 mapping)
			subjectKey, err := cus.UsageAttribution.GetSubjectKey()
			if err != nil {
				return request, commonhttp.NewHTTPError(http.StatusConflict, err)
			}

			// Reuse v1 parser to build entitlement create inputs using the subject key
			return entitlementdriver.ParseAPICreateInput(inp, ns, subjectKey)
		},
		func(ctx context.Context, request CreateCustomerEntitlementHandlerRequest) (CreateCustomerEntitlementHandlerResponse, error) {
			ent, err := h.connector.CreateEntitlement(ctx, request)
			if err != nil {
				return api.EntitlementV2{}, err
			}

			if ent.Customer == nil {
				return api.EntitlementV2{}, commonhttp.NewHTTPError(http.StatusNotFound, errors.New("customer not found"))
			}

			v2, err := ParserV2.ToAPIGenericV2(ent, ent.Customer.ID, ent.Customer.Key)
			if err != nil {
				return api.EntitlementV2{}, err
			}
			return *v2, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

func (h *customerEntitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
