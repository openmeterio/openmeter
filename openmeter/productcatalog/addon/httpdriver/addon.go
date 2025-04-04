package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/defaultx"
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
				IDs:            lo.FromPtrOr(params.Id, nil),
				Keys:           lo.FromPtrOr(params.Key, nil),
				KeyVersions:    lo.FromPtrOr(params.KeyVersion, nil),
				IncludeDeleted: lo.FromPtrOr(params.IncludeDeleted, false),
				Currencies:     lo.FromPtrOr(params.Currency, nil),
				Status:         statusFilter,
			}

			return req, nil
		},
		func(ctx context.Context, request ListAddonsRequest) (ListAddonsResponse, error) {
			resp, err := h.service.ListAddons(ctx, request)
			if err != nil {
				return ListAddonsResponse{}, fmt.Errorf("failed to list add-ons: %w", err)
			}

			items := make([]api.Addon, 0, len(resp.Items))

			for _, a := range resp.Items {
				var item api.Addon

				item, err = FromAddon(a)
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
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreateAddonRequest  = addon.CreateAddonInput
	CreateAddonResponse = api.Addon
	CreateAddonHandler  httptransport.Handler[CreateAddonRequest, CreateAddonResponse]
)

func (h *handler) CreateAddon() CreateAddonHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateAddonRequest, error) {
			body := api.AddonCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateAddonRequest{}, fmt.Errorf("failed to decode create add-on request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			req, err := AsCreateAddonRequest(body, ns)
			if err != nil {
				return CreateAddonRequest{}, fmt.Errorf("failed to parse add-on request [namespace=%s key=%s]: %w", ns, body.Key, err)
			}

			req.NamespacedModel = models.NamespacedModel{
				Namespace: ns,
			}

			return req, nil
		},
		func(ctx context.Context, request CreateAddonRequest) (CreateAddonResponse, error) {
			a, err := h.service.CreateAddon(ctx, request)
			if err != nil {
				return CreateAddonResponse{}, fmt.Errorf("failed to create add-on [namespace=%s key=%s]: %w", request.Namespace, request.Key, err)
			}

			return FromAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateAddonRequest  = addon.UpdateAddonInput
	UpdateAddonResponse = api.Addon
	UpdateAddonHandler  httptransport.HandlerWithArgs[UpdateAddonRequest, UpdateAddonResponse, string]
)

func (h *handler) UpdateAddon() UpdateAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (UpdateAddonRequest, error) {
			body := api.AddonReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateAddonRequest{}, fmt.Errorf("failed to decode update add-on request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			req, err := AsUpdateAddonRequest(body, ns, addonID)
			if err != nil {
				return UpdateAddonRequest{}, fmt.Errorf("failed to parse update add-on request [namespace=%s id=%s]: %w", ns, addonID, err)
			}

			req.NamespacedID = models.NamespacedID{
				Namespace: ns,
				ID:        addonID,
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateAddonRequest) (UpdateAddonResponse, error) {
			a, err := h.service.UpdateAddon(ctx, request)
			if err != nil {
				return UpdateAddonResponse{}, fmt.Errorf("failed to update add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			return FromAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
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
			httptransport.WithErrorEncoder(errorEncoder()),
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

			return FromAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
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
					EffectiveFrom: lo.ToPtr(time.Now()),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request PublishAddonRequest) (PublishAddonResponse, error) {
			a, err := h.service.PublishAddon(ctx, request)
			if err != nil {
				return PublishAddonResponse{}, fmt.Errorf("failed to punlish add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			return FromAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[PublishAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publishAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
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
				EffectiveTo: time.Now(),
			}

			return req, nil
		},
		func(ctx context.Context, request ArchiveAddonRequest) (ArchiveAddonResponse, error) {
			p, err := h.service.ArchiveAddon(ctx, request)
			if err != nil {
				return ArchiveAddonResponse{}, fmt.Errorf("failed to archive add-on [namespace=%s id=%s]: %w", request.Namespace, request.ID, err)
			}

			return FromAddon(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[ArchiveAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archiveAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
