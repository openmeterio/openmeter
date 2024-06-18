package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlement_httpdriver "github.com/openmeterio/openmeter/internal/entitlement/httpdriver"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type GrantHandler interface {
	ListGrants() ListGrantsHandler
	VoidGrant() VoidGrantHandler
}

type grantHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	grantConnector   credit.GrantConnector
}

func NewGrantHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	grantConnector credit.GrantConnector,
	options ...httptransport.HandlerOption,
) GrantHandler {
	return &grantHandler{
		namespaceDecoder: namespaceDecoder,
		grantConnector:   grantConnector,
		options:          options,
	}
}

type ListGrantsInputs struct {
	params credit.ListGrantsParams
}
type ListGrantsParams struct {
	Params api.ListGrantsParams
}
type ListGrantsHandler httptransport.HandlerWithArgs[ListGrantsInputs, []api.EntitlementGrant, ListGrantsParams]

func (h *grantHandler) ListGrants() ListGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListGrantsInputs, []api.EntitlementGrant, ListGrantsParams](
		func(ctx context.Context, r *http.Request, params ListGrantsParams) (ListGrantsInputs, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListGrantsInputs{}, err
			}

			return ListGrantsInputs{
				params: credit.ListGrantsParams{
					Namespace:      ns,
					IncludeDeleted: defaultx.WithDefault(params.Params.IncludeDeleted, false),
					Offset:         defaultx.WithDefault(params.Params.Offset, 0),
					Limit:          defaultx.WithDefault(params.Params.Limit, 1000),
					OrderBy:        credit.GrantOrderBy(defaultx.WithDefault((*string)(params.Params.OrderBy), string(credit.GrantOrderByCreatedAt))),
				},
			}, nil
		},
		func(ctx context.Context, request ListGrantsInputs) ([]api.EntitlementGrant, error) {
			grants, err := h.grantConnector.ListGrants(ctx, request.params)
			if err != nil {
				return nil, err
			}

			apiGrants := make([]api.EntitlementGrant, 0, len(grants))
			for _, grant := range grants {
				entitlementGrant, err := entitlement.GrantFromCreditGrant(grant)
				if err != nil {
					return nil, err
				}
				// FIXME: not elegant but good for now, entitlement grants are all we have...
				apiGrant := entitlement_httpdriver.MapEntitlementGrantToAPI(nil, entitlementGrant)

				apiGrants = append(apiGrants, apiGrant)
			}

			return apiGrants, nil
		},
		commonhttp.JSONResponseEncoder[[]api.EntitlementGrant],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*models.GenericUserError); ok {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			}),
		)...,
	)
}

type VoidGrantInputs struct {
	ID models.NamespacedID
}
type VoidGrantParams struct {
	ID string
}
type VoidGrantHandler httptransport.HandlerWithArgs[VoidGrantInputs, interface{}, VoidGrantParams]

func (h *grantHandler) VoidGrant() VoidGrantHandler {
	return httptransport.NewHandlerWithArgs[VoidGrantInputs, interface{}, VoidGrantParams](
		func(ctx context.Context, r *http.Request, params VoidGrantParams) (VoidGrantInputs, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return VoidGrantInputs{}, err
			}

			return VoidGrantInputs{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
			}, nil
		},
		func(ctx context.Context, request VoidGrantInputs) (interface{}, error) {
			err := h.grantConnector.VoidGrant(ctx, request.ID)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[interface{}](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*models.GenericUserError); ok {
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
