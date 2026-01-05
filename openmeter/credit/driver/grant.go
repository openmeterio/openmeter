package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entitlement_httpdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	entitlement_httpdriverv2 "github.com/openmeterio/openmeter/openmeter/entitlement/driver/v2"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/strcase"
)

type GrantHandler interface {
	ListGrants() ListGrantsHandler
	VoidGrant() VoidGrantHandler
	ListGrantsV2() ListGrantsV2Handler
}

type grantHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	grantConnector   credit.GrantConnector
	grantRepo        grant.Repo
	customerService  customer.Service
}

func NewGrantHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	grantConnector credit.GrantConnector,
	grantRepo grant.Repo,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) GrantHandler {
	return &grantHandler{
		namespaceDecoder: namespaceDecoder,
		grantConnector:   grantConnector,
		grantRepo:        grantRepo,
		customerService:  customerService,
		options:          options,
	}
}

type (
	ListGrantsHandlerRequest struct {
		params grant.ListParams
	}
	ListGrantsHandlerResponse = commonhttp.Union[[]api.EntitlementGrant, pagination.Result[api.EntitlementGrant]]
	ListGrantsHandlerParams   struct {
		Params api.ListGrantsParams
	}
)
type ListGrantsHandler httptransport.HandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams]

func (h *grantHandler) ListGrants() ListGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams](
		func(ctx context.Context, r *http.Request, params ListGrantsHandlerParams) (ListGrantsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListGrantsHandlerRequest{}, err
			}

			// validate OrderBy
			if params.Params.OrderBy != nil {
				if !slices.Contains(grant.OrderBy("").StrValues(), strcase.CamelToSnake(string(*params.Params.OrderBy))) {
					return ListGrantsHandlerRequest{}, commonhttp.NewHTTPError(http.StatusBadRequest, errors.New("invalid order by"))
				}
			}

			return ListGrantsHandlerRequest{
				params: grant.ListParams{
					Namespace:      ns,
					IncludeDeleted: defaultx.WithDefault(params.Params.IncludeDeleted, false),
					Page: pagination.Page{
						PageSize:   defaultx.WithDefault(params.Params.PageSize, 0),
						PageNumber: defaultx.WithDefault(params.Params.Page, 0),
					},
					Limit:  defaultx.WithDefault(params.Params.Limit, commonhttp.DefaultPageSize),
					Offset: defaultx.WithDefault(params.Params.Offset, 0),
					OrderBy: grant.OrderBy(
						strcase.CamelToSnake(defaultx.WithDefault((*string)(params.Params.OrderBy), string(grant.OrderByEffectiveAt))),
					),
					Order:            commonhttp.GetSortOrder(api.SortOrderASC, params.Params.Order),
					SubjectKeys:      convert.DerefHeaderPtr[string](params.Params.Subject),
					FeatureIdsOrKeys: convert.DerefHeaderPtr[string](params.Params.Feature),
				},
			}, nil
		},
		func(ctx context.Context, request ListGrantsHandlerRequest) (ListGrantsHandlerResponse, error) {
			// due to backward compatibility, if pagination is not provided we return a simple array
			response := ListGrantsHandlerResponse{
				Option1: &[]api.EntitlementGrant{},
				Option2: &pagination.Result[api.EntitlementGrant]{},
			}
			grants, err := h.grantRepo.ListGrants(ctx, request.params)
			if err != nil {
				return response, err
			}

			apiGrants := make([]api.EntitlementGrant, 0, len(grants.Items))
			for _, grant := range grants.Items {
				entitlementGrant, err := meteredentitlement.GrantFromCreditGrant(grant, clock.Now())
				if err != nil {
					return response, err
				}
				// FIXME: not elegant but good for now, entitlement grants are all we have...
				apiGrant := entitlement_httpdriver.MapEntitlementGrantToAPI(entitlementGrant)

				apiGrants = append(apiGrants, apiGrant)
			}

			if request.params.Page.IsZero() {
				response.Option1 = &apiGrants
			} else {
				response.Option1 = nil
				response.Option2 = &pagination.Result[api.EntitlementGrant]{
					Items:      apiGrants,
					TotalCount: grants.TotalCount,
					Page:       grants.Page,
				}
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListGrantsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
				if models.IsGenericValidationError(err) {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w)
			}),
		)...,
	)
}

type VoidGrantHandlerRequest struct {
	ID models.NamespacedID
}
type (
	VoidGrantHandlerResponse = interface{}
	VoidGrantHandlerParams   struct {
		ID string
	}
)
type VoidGrantHandler httptransport.HandlerWithArgs[VoidGrantHandlerRequest, VoidGrantHandlerResponse, VoidGrantHandlerParams]

func (h *grantHandler) VoidGrant() VoidGrantHandler {
	return httptransport.NewHandlerWithArgs[VoidGrantHandlerRequest, VoidGrantHandlerResponse, VoidGrantHandlerParams](
		func(ctx context.Context, r *http.Request, params VoidGrantHandlerParams) (VoidGrantHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return VoidGrantHandlerRequest{}, err
			}

			return VoidGrantHandlerRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
			}, nil
		},
		func(ctx context.Context, request VoidGrantHandlerRequest) (interface{}, error) {
			err := h.grantConnector.VoidGrant(ctx, request.ID)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[interface{}](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
				if models.IsGenericValidationError(err) {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*credit.GrantNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			}),
		)...,
	)
}

func (h *grantHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

// V2 List Grants
type (
	ListGrantsV2HandlerRequest struct {
		params grant.ListParams
	}
	ListGrantsV2HandlerResponse = api.GrantV2PaginatedResponse
	ListGrantsV2HandlerParams   struct {
		Params api.ListGrantsV2Params
	}
)
type ListGrantsV2Handler httptransport.HandlerWithArgs[ListGrantsV2HandlerRequest, ListGrantsV2HandlerResponse, ListGrantsV2HandlerParams]

func (h *grantHandler) ListGrantsV2() ListGrantsV2Handler {
	return httptransport.NewHandlerWithArgs[ListGrantsV2HandlerRequest, ListGrantsV2HandlerResponse, ListGrantsV2HandlerParams](
		func(ctx context.Context, r *http.Request, p ListGrantsV2HandlerParams) (ListGrantsV2HandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListGrantsV2HandlerRequest{}, err
			}

			// Resolve customers by ID or Key
			var customerIDs []string

			if p.Params.Customer != nil && len(*p.Params.Customer) > 0 {
				customerIDs = make([]string, 0, len(*p.Params.Customer))

				for _, idOrKey := range *p.Params.Customer {
					cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
						CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: idOrKey},
					})
					if err != nil {
						return ListGrantsV2HandlerRequest{}, err
					}

					// Skip deleted customers
					if cus != nil && cus.IsDeleted() {
						continue
					}

					customerIDs = append(customerIDs, cus.ID)
				}
			}

			req := grant.ListParams{
				Namespace:        ns,
				IncludeDeleted:   defaultx.WithDefault(p.Params.IncludeDeleted, false),
				Order:            commonhttp.GetSortOrder(api.SortOrderASC, p.Params.Order),
				OrderBy:          grant.OrderBy(strcase.CamelToSnake(defaultx.WithDefault((*string)(p.Params.OrderBy), string(grant.OrderByEffectiveAt)))),
				FeatureIdsOrKeys: convert.DerefHeaderPtr[string](p.Params.Feature),
				CustomerIDs:      customerIDs,
			}

			// Pagination: support both page/pageSize and limit/offset
			if p.Params.Page != nil || p.Params.PageSize != nil {
				req.Page.PageNumber = defaultx.WithDefault(p.Params.Page, 1)
				req.Page.PageSize = defaultx.WithDefault(p.Params.PageSize, 100)
			} else {
				req.Limit = defaultx.WithDefault(p.Params.Limit, 100)
				req.Offset = defaultx.WithDefault(p.Params.Offset, 0)
			}

			return ListGrantsV2HandlerRequest{params: req}, nil
		},
		func(ctx context.Context, request ListGrantsV2HandlerRequest) (ListGrantsV2HandlerResponse, error) {
			grants, err := h.grantRepo.ListGrants(ctx, request.params)
			if err != nil {
				return ListGrantsV2HandlerResponse{}, err
			}

			apiGrants := make([]api.EntitlementGrantV2, 0, len(grants.Items))
			for _, g := range grants.Items {
				entitlementGrant, err := meteredentitlement.GrantFromCreditGrant(g, clock.Now())
				if err != nil {
					return ListGrantsV2HandlerResponse{}, err
				}
				a := entitlement_httpdriverv2.MapEntitlementGrantToAPIV2(entitlementGrant)
				apiGrants = append(apiGrants, a)
			}

			return api.GrantV2PaginatedResponse{
				Items:      apiGrants,
				TotalCount: grants.TotalCount,
				Page:       grants.Page.PageNumber,
				PageSize:   grants.Page.PageSize,
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListGrantsV2HandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
				if models.IsGenericValidationError(err) {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w)
			}),
		)...,
	)
}
