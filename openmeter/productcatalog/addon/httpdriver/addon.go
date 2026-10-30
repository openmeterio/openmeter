package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListAddonsRequest  = addon.ListAddonsInput
	ListAddonsResponse = api.AddonPaginatedResponse
	ListAddonsParams   = api.ListAddonsParams
	ListAddonsHandler  httptransport.HandlerWithArgs[ListAddonsRequest, ListAddonsResponse, ListAddonsParams]
)

func (h *handler) ListAddons() ListAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAddonsParams) (ListAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAddonsRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			var statusFilter []productcatalog.AddonStatus
			if params.Status != nil {
				statusFilter = lo.Map(*params.Status, func(status api.AddonStatus, _ int) productcatalog.AddonStatus {
					return productcatalog.AddonStatus(status)
				})
			}

			req := ListAddonsRequest{
				OrderBy: addon.OrderBy(lo.FromPtrOr(params.OrderBy, api.AddonOrderById)),
				Order:   sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderDESC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
				Namespaces:     []string{ns},
				KeyVersions:    lo.FromPtr(params.KeyVersion),
				IncludeDeleted: lo.FromPtr(params.IncludeDeleted),
				Status:         statusFilter,
				// The v1 API cannot represent unit_config, custom default currencies, or
				// rate-card currency overrides; exclude such add-ons at the query layer.
				ExcludeUnitConfig:                true,
				ExcludeUnrepresentableCurrencies: true,
			}

			if params.Id != nil {
				req.ID = &filter.FilterULID{FilterString: filter.FilterString{In: params.Id}}
			}

			if params.Key != nil {
				req.Key = &filter.FilterString{In: params.Key}
			}

			if params.Currency != nil {
				req.Currency = &filter.FilterString{In: params.Currency}
			}

			return req, nil
		},
		func(ctx context.Context, request ListAddonsRequest) (ListAddonsResponse, error) {
			// The decoder scopes the list to the caller's namespace; tax codes are looked up there.
			if len(request.Namespaces) != 1 {
				return ListAddonsResponse{}, fmt.Errorf("expected exactly one namespace, got %d", len(request.Namespaces))
			}

			resp, err := h.service.ListAddons(ctx, request)
			if err != nil {
				return ListAddonsResponse{}, fmt.Errorf("failed to list add-ons: %w", err)
			}

			taxCodes, err := h.taxCodesForAddons(ctx, request.Namespaces[0], resp.Items...)
			if err != nil {
				return ListAddonsResponse{}, err
			}

			items := make([]api.Addon, 0, len(resp.Items))

			for _, a := range resp.Items {
				var item api.Addon

				item, err = FromAddon(a, taxCodes)
				if err != nil {
					return ListAddonsResponse{}, fmt.Errorf("failed to cast add-on [namespace=%s key=%s]: %w", a.Namespace, a.Key, err)
				}

				items = append(items, item)
			}

			return ListAddonsResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listAddons"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	CreateAddonRequest = addon.CreateAddonInput
	// CreateAddonHandlerRequest holds the undecoded body: mapping it may create Tax Codes for
	// legacy stripe.code values, which has to happen in the same transaction as the create.
	CreateAddonHandlerRequest struct {
		Namespace string
		Body      api.AddonCreate
	}
	CreateAddonResponse = api.Addon
	CreateAddonHandler  httptransport.Handler[CreateAddonHandlerRequest, CreateAddonResponse]
)

func (h *handler) CreateAddon() CreateAddonHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateAddonHandlerRequest, error) {
			body := api.AddonCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateAddonHandlerRequest{}, fmt.Errorf("failed to decode create add-on request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateAddonHandlerRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			return CreateAddonHandlerRequest{
				Namespace: ns,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, request CreateAddonHandlerRequest) (CreateAddonResponse, error) {
			var resp CreateAddonResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				req, err := AsCreateAddonRequest(ctx, resolver, request.Body, request.Namespace)
				if err != nil {
					return fmt.Errorf("failed to parse add-on request [namespace=%s key=%s]: %w", request.Namespace, request.Body.Key, err)
				}

				req.NamespacedModel = models.NamespacedModel{
					Namespace: request.Namespace,
				}

				req.IgnoreNonCriticalIssues = true

				a, err := h.service.CreateAddon(ctx, req)
				if err != nil {
					return fmt.Errorf("failed to create add-on [namespace=%s key=%s]: %w", req.Namespace, req.Key, err)
				}

				// Inside the transaction so Tax Codes created for this request are visible.
				taxCodes, err := h.taxCodesForAddons(ctx, a.Namespace, *a)
				if err != nil {
					return err
				}

				resp, err = FromAddon(*a, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	UpdateAddonRequest = addon.UpdateAddonInput
	// UpdateAddonHandlerRequest holds the undecoded body for the same reason as
	// CreateAddonHandlerRequest.
	UpdateAddonHandlerRequest struct {
		models.NamespacedID
		Body api.AddonReplaceUpdate
	}
	UpdateAddonResponse = api.Addon
	UpdateAddonHandler  httptransport.HandlerWithArgs[UpdateAddonHandlerRequest, UpdateAddonResponse, string]
)

func (h *handler) UpdateAddon() UpdateAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (UpdateAddonHandlerRequest, error) {
			body := api.AddonReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateAddonHandlerRequest{}, fmt.Errorf("failed to decode update add-on request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateAddonHandlerRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			return UpdateAddonHandlerRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
				Body: body,
			}, nil
		},
		func(ctx context.Context, request UpdateAddonHandlerRequest) (UpdateAddonResponse, error) {
			var resp UpdateAddonResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				req, err := AsUpdateAddonRequest(ctx, resolver, request.Body, request.Namespace, request.ID)
				if err != nil {
					return fmt.Errorf("failed to parse update add-on request [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
				}

				req.NamespacedID = request.NamespacedID

				req.IgnoreNonCriticalIssues = true

				req.RejectUnitConfig = true
				req.RejectUnrepresentableCurrencies = true

				a, err := h.service.UpdateAddon(ctx, req)
				if err != nil {
					return fmt.Errorf("failed to update add-on [namespace=%s id=%s]: %w", req.Namespace, req.ID, err)
				}

				// Inside the transaction so Tax Codes created for this request are visible.
				taxCodes, err := h.taxCodesForAddons(ctx, a.Namespace, *a)
				if err != nil {
					return err
				}

				resp, err = FromAddon(*a, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	DeleteAddonRequest  = addon.DeleteAddonInput
	DeleteAddonResponse = interface{}
	DeleteAddonHandler  httptransport.HandlerWithArgs[DeleteAddonRequest, DeleteAddonResponse, string]
)

func (h *handler) DeleteAddon() DeleteAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (DeleteAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			return DeleteAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteAddonRequest) (DeleteAddonResponse, error) {
			err := h.service.DeleteAddon(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteAddonResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	GetAddonRequest       = addon.GetAddonInput
	GetAddonRequestParams struct {
		// AddonID or Key.
		IDOrKey string

		// Version is the version of the add-on.
		// If not set the latest version is assumed.
		Version int

		// AllowLatest defines whether return the latest version regardless of its AddonStatus or with ActiveStatus only if
		// Version is not set.
		IncludeLatest bool
	}
	GetAddonResponse = api.Addon
	GetAddonHandler  httptransport.HandlerWithArgs[GetAddonRequest, GetAddonResponse, GetAddonRequestParams]
)

func (h *handler) GetAddon() GetAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetAddonRequestParams) (GetAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			// Try to detect whether the IdOrKey is an ID in ULID format or Key.
			idOrKey := ref.ParseIDOrKey(params.IDOrKey)

			return GetAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        idOrKey.ID,
				},
				Key:           idOrKey.Key,
				Version:       params.Version,
				IncludeLatest: params.IncludeLatest,
			}, nil
		},
		func(ctx context.Context, request GetAddonRequest) (GetAddonResponse, error) {
			a, err := h.service.GetAddon(ctx, request)
			if err != nil {
				return GetAddonResponse{}, fmt.Errorf("failed to get add-on [namespace=%s key=%s id=%s]: %w", request.Namespace, request.Key, request.ID, err)
			}

			if a.AsProductCatalogAddon().HasUnitConfig() {
				return GetAddonResponse{}, productcatalog.ErrUnitConfigNotRepresentable
			}

			if a.Currency.IsCustom() {
				return GetAddonResponse{}, productcatalog.ErrCurrencyNotRepresentable
			}

			if a.AsProductCatalogAddon().HasCurrencyOverrides() {
				return GetAddonResponse{}, productcatalog.ErrRateCardCurrencyNotRepresentable
			}

			taxCodes, err := h.taxCodesForAddons(ctx, a.Namespace, *a)
			if err != nil {
				return GetAddonResponse{}, err
			}

			return FromAddon(*a, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	PublishAddonRequest  = addon.PublishAddonInput
	PublishAddonResponse = api.Addon
	PublishAddonHandler  httptransport.HandlerWithArgs[PublishAddonRequest, PublishAddonResponse, string]
)

func (h *handler) PublishAddon() PublishAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (PublishAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return PublishAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod

			req := PublishAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
				},
				RejectUnitConfig:                true,
				RejectUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request PublishAddonRequest) (PublishAddonResponse, error) {
			a, err := h.service.PublishAddon(ctx, request)
			if err != nil {
				return PublishAddonResponse{}, fmt.Errorf("failed to punlish add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			taxCodes, err := h.taxCodesForAddons(ctx, a.Namespace, *a)
			if err != nil {
				return PublishAddonResponse{}, err
			}

			return FromAddon(*a, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[PublishAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publishAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}

type (
	ArchiveAddonRequest  = addon.ArchiveAddonInput
	ArchiveAddonResponse = api.Addon
	ArchiveAddonHandler  httptransport.HandlerWithArgs[ArchiveAddonRequest, ArchiveAddonResponse, string]
)

func (h *handler) ArchiveAddon() ArchiveAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (ArchiveAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchiveAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod.To

			req := ArchiveAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
				EffectiveTo: clock.Now(),
				// The v1 API cannot represent unit_config; reject archiving such an add-on.
				RejectUnitConfig:                true,
				RejectUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request ArchiveAddonRequest) (ArchiveAddonResponse, error) {
			p, err := h.service.ArchiveAddon(ctx, request)
			if err != nil {
				return ArchiveAddonResponse{}, fmt.Errorf("failed to archive add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			taxCodes, err := h.taxCodesForAddons(ctx, p.Namespace, *p)
			if err != nil {
				return ArchiveAddonResponse{}, err
			}

			return FromAddon(*p, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[ArchiveAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archiveAddon"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),
		)...,
	)
}
