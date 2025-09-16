package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListTokensParams   = api.ListPortalTokensParams
	ListTokensResponse = []api.PortalToken
	ListTokensHandler  httptransport.HandlerWithArgs[ListTokensRequest, ListTokensResponse, ListTokensParams]
)

type ListTokensRequest struct {
	namespace string
	page      pagination.Page
}

// ListTokens returns a handler for listing portal tokens.
func (h *handler) ListTokens() ListTokensHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListTokensParams) (ListTokensRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListTokensRequest{}, err
			}

			pageSize := 25

			if params.Limit != nil {
				pageSize = *params.Limit
			}

			return ListTokensRequest{
				namespace: ns,
				page:      pagination.NewPage(1, pageSize),
			}, nil
		},
		func(ctx context.Context, request ListTokensRequest) (ListTokensResponse, error) {
			result, err := h.portalService.ListTokens(ctx, portal.ListTokensInput{
				Namespace: request.namespace,
				Page:      request.page,
			})
			if err != nil {
				return ListTokensResponse{}, fmt.Errorf("failed to list tokens: %w", err)
			}

			// Response
			resp := pagination.MapResult(result, toAPIPortalToken)

			return resp.Items, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListTokensResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listPortalTokens"),
		)...,
	)
}

type (
	CreateTokenRequest  = portal.CreateTokenInput
	CreateTokenResponse = *api.PortalToken
	CreateTokenHandler  httptransport.Handler[CreateTokenRequest, CreateTokenResponse]
)

// CreateToken returns a new httptransport.Handler for creating a token.
func (h *handler) CreateToken() CreateTokenHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateTokenRequest, error) {
			body := api.CreatePortalTokenJSONRequestBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateTokenRequest{}, fmt.Errorf("field to decode create token request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateTokenRequest{}, err
			}

			if body.AllowedMeterSlugs != nil && len(*body.AllowedMeterSlugs) == 0 {
				return CreateTokenRequest{}, models.NewGenericValidationError(
					fmt.Errorf("slug filter cannot be an empty array when provided"),
				)
			}

			req := CreateTokenRequest{
				Namespace:         ns,
				Subject:           body.Subject,
				ExpiresAt:         body.ExpiresAt,
				AllowedMeterSlugs: body.AllowedMeterSlugs,
			}

			return req, nil
		},
		func(ctx context.Context, request CreateTokenRequest) (CreateTokenResponse, error) {
			// If allowed meter slugs are provided, validate them.
			if request.AllowedMeterSlugs != nil {
				meterList, err := h.meterService.ListMeters(ctx, meter.ListMetersParams{
					Namespace:  request.Namespace,
					SlugFilter: request.AllowedMeterSlugs,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to list meters by slug: %w", err)
				}

				metersBySlug := lo.KeyBy(meterList.Items, func(m meter.Meter) string {
					return m.Key
				})

				for _, slug := range *request.AllowedMeterSlugs {
					if _, ok := metersBySlug[slug]; !ok {
						return nil, meter.NewMeterNotFoundError(slug)
					}
				}
			}

			// Create token
			token, err := h.portalService.CreateToken(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to create token: %w", err)
			}

			if token == nil {
				return nil, fmt.Errorf("failed to create token")
			}

			portalToken := toAPIPortalToken(token)
			portalToken.Token = token.Token

			return &portalToken, nil
		},
		// TODO: status code is currently 200 in API definition, should be 201
		commonhttp.JSONResponseEncoderWithStatus[CreateTokenResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createPortalToken"),
		)...,
	)
}

type (
	InvalidateTokenRequest  = portal.InvalidateTokenInput
	InvalidateTokenResponse = interface{}
	InvalidateTokenHandler  httptransport.Handler[InvalidateTokenRequest, InvalidateTokenResponse]
)

// InvalidateToken returns a new httptransport.Handler for creating a token.
func (h *handler) InvalidateToken() InvalidateTokenHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (InvalidateTokenRequest, error) {
			body := api.InvalidatePortalTokensJSONBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return InvalidateTokenRequest{}, fmt.Errorf("field to decode create token request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return InvalidateTokenRequest{}, err
			}

			req := InvalidateTokenRequest{
				Namespace: ns,
				ID:        body.Id,
				Subject:   body.Subject,
			}

			return req, nil
		},
		func(ctx context.Context, request InvalidateTokenRequest) (InvalidateTokenResponse, error) {
			err := h.portalService.InvalidateToken(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to invalidate token: %w", err)
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[InvalidateTokenResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("invalidatePortalTokens"),
		)...,
	)
}
