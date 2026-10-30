package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListPlansRequest  = plan.ListPlansInput
	ListPlansResponse = api.PlanPaginatedResponse
	ListPlansParams   = api.ListPlansParams
	ListPlansHandler  httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]
)

func (h *handler) ListPlans() ListPlansHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlansParams) (ListPlansRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlansRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var statusFilter []productcatalog.PlanStatus
			if params.Status != nil {
				statusFilter = lo.Map(*params.Status, func(status api.PlanStatus, _ int) productcatalog.PlanStatus {
					return productcatalog.PlanStatus(status)
				})
			}

			req := ListPlansRequest{
				OrderBy: plan.OrderBy(lo.FromPtrOr(params.OrderBy, api.PlanOrderById)),
				Order:   sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderDESC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
				Namespaces:                       []string{ns},
				IDs:                              lo.FromPtr(params.Id),
				Keys:                             lo.FromPtr(params.Key),
				KeyVersions:                      lo.FromPtr(params.KeyVersion),
				IncludeDeleted:                   lo.FromPtr(params.IncludeDeleted),
				Currencies:                       lo.FromPtr(params.Currency),
				Status:                           statusFilter,
				ExcludeUnitConfig:                true,
				ExcludeUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request ListPlansRequest) (ListPlansResponse, error) {
			// The decoder scopes the list to the caller's namespace; tax codes are looked up there.
			if len(request.Namespaces) != 1 {
				return ListPlansResponse{}, fmt.Errorf("expected exactly one namespace, got %d", len(request.Namespaces))
			}

			resp, err := h.service.ListPlans(ctx, request)
			if err != nil {
				return ListPlansResponse{}, fmt.Errorf("failed to list plans: %w", err)
			}

			taxCodes, err := h.taxCodesForPlans(ctx, request.Namespaces[0], resp.Items...)
			if err != nil {
				return ListPlansResponse{}, err
			}

			items := make([]api.Plan, 0, len(resp.Items))

			for _, p := range resp.Items {
				var item api.Plan

				item, err = FromPlan(p, taxCodes)
				if err != nil {
					return ListPlansResponse{}, fmt.Errorf("failed to cast plan: %w", err)
				}

				items = append(items, item)
			}

			return ListPlansResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlansResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listPlans"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	CreatePlanRequest = plan.CreatePlanInput
	// CreatePlanHandlerRequest holds the undecoded body: mapping it may create Tax Codes for
	// legacy stripe.code values, which has to happen in the same transaction as the create.
	CreatePlanHandlerRequest struct {
		Namespace string
		Body      api.PlanCreate
	}
	CreatePlanResponse = api.Plan
	CreatePlanHandler  httptransport.Handler[CreatePlanHandlerRequest, CreatePlanResponse]
)

func (h *handler) CreatePlan() CreatePlanHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreatePlanHandlerRequest, error) {
			body := api.PlanCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreatePlanHandlerRequest{}, fmt.Errorf("failed to decode create plan request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePlanHandlerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return CreatePlanHandlerRequest{
				Namespace: ns,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, request CreatePlanHandlerRequest) (CreatePlanResponse, error) {
			var resp CreatePlanResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				req, err := AsCreatePlanRequest(ctx, resolver, request.Body, request.Namespace)
				if err != nil {
					var resolveErr *productcataloghttp.TaxCodeResolutionError
					if errors.As(err, &resolveErr) {
						return err
					}

					return models.NewGenericValidationError(fmt.Errorf("failed to create plan request: %w", err))
				}

				req.NamespacedModel = models.NamespacedModel{
					Namespace: request.Namespace,
				}

				req.IgnoreNonCriticalIssues = true

				creditEnabled := featuregate.ContextResolver().Credits(ctx)
				if !creditEnabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode {
					return models.NewGenericValidationError(fmt.Errorf("credits are not enabled on this deployment of OpenMeter"))
				}

				p, err := h.service.CreatePlan(ctx, req)
				if err != nil {
					return fmt.Errorf("failed to create plan: %w", err)
				}

				// Inside the transaction so Tax Codes created for this request are visible.
				taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
				if err != nil {
					return err
				}

				resp, err = FromPlan(*p, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePlanResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createPlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	UpdatePlanRequest = plan.UpdatePlanInput
	// UpdatePlanHandlerRequest holds the undecoded body for the same reason as
	// CreatePlanHandlerRequest.
	UpdatePlanHandlerRequest struct {
		models.NamespacedID
		Body api.PlanReplaceUpdate
	}
	UpdatePlanResponse = api.Plan
	UpdatePlanHandler  httptransport.HandlerWithArgs[UpdatePlanHandlerRequest, UpdatePlanResponse, string]
)

func (h *handler) UpdatePlan() UpdatePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (UpdatePlanHandlerRequest, error) {
			body := api.PlanReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdatePlanHandlerRequest{}, fmt.Errorf("failed to decode update plan request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanHandlerRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return UpdatePlanHandlerRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				Body: body,
			}, nil
		},
		func(ctx context.Context, request UpdatePlanHandlerRequest) (UpdatePlanResponse, error) {
			var resp UpdatePlanResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				req, err := AsUpdatePlanRequest(ctx, resolver, request.Body, request.Namespace, request.ID)
				if err != nil {
					return fmt.Errorf("failed to parse update plan request: %w", err)
				}

				req.NamespacedID = request.NamespacedID

				req.IgnoreNonCriticalIssues = true
				req.RejectUnitConfig = true
				req.RejectUnrepresentableCurrencies = true

				// Validate credit_only settlement mode when credits are disabled
				creditEnabled := featuregate.ContextResolver().Credits(ctx)
				if !creditEnabled && req.SettlementMode != nil && *req.SettlementMode == productcatalog.CreditOnlySettlementMode {
					return models.NewGenericValidationError(fmt.Errorf("credits are not enabled on this deployment of OpenMeter"))
				}

				p, err := h.service.UpdatePlan(ctx, req)
				if err != nil {
					return fmt.Errorf("failed to update plan: %w", err)
				}

				// Inside the transaction so Tax Codes created for this request are visible.
				taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
				if err != nil {
					return err
				}

				resp, err = FromPlan(*p, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updatePlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	DeletePlanRequest  = plan.DeletePlanInput
	DeletePlanResponse = interface{}
	DeletePlanHandler  httptransport.HandlerWithArgs[DeletePlanRequest, DeletePlanResponse, string]
)

func (h *handler) DeletePlan() DeletePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (DeletePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeletePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
			}, nil
		},
		func(ctx context.Context, request DeletePlanRequest) (DeletePlanResponse, error) {
			err := h.service.DeletePlan(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete plan: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePlanResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deletePlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	GetPlanRequest       = plan.GetPlanInput
	GetPlanRequestParams struct {
		// PlanID or Key.
		IDOrKey string

		// Version is the version of the Plan.
		// If not set the latest version is assumed.
		Version int

		// AllowLatest defines whether return the latest version regardless of its PlanStatus or with ActiveStatus only if
		// Version is not set.
		IncludeLatest bool
	}
	GetPlanResponse = api.Plan
	GetPlanHandler  httptransport.HandlerWithArgs[GetPlanRequest, GetPlanResponse, GetPlanRequestParams]
)

func (h *handler) GetPlan() GetPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetPlanRequestParams) (GetPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// Try to detect whether the IdOrKey is an ID in ULID format or Key.
			idOrKey := ref.ParseIDOrKey(params.IDOrKey)

			return GetPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        idOrKey.ID,
				},
				Key:           idOrKey.Key,
				Version:       params.Version,
				IncludeLatest: params.IncludeLatest,
			}, nil
		},
		func(ctx context.Context, request GetPlanRequest) (GetPlanResponse, error) {
			p, err := h.service.GetPlan(ctx, request)
			if err != nil {
				return GetPlanResponse{}, fmt.Errorf("failed to get plan: %w", err)
			}

			if p.HasUnitConfig() {
				return GetPlanResponse{}, productcatalog.ErrUnitConfigNotRepresentable
			}

			if p.Currency.IsCustom() {
				return GetPlanResponse{}, productcatalog.ErrCurrencyNotRepresentable
			}

			if p.HasCurrencyOverrides() {
				return GetPlanResponse{}, productcatalog.ErrRateCardCurrencyNotRepresentable
			}

			taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
			if err != nil {
				return GetPlanResponse{}, err
			}

			return FromPlan(*p, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getPlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	PublishPlanRequest  = plan.PublishPlanInput
	PublishPlanResponse = api.Plan
	PublishPlanHandler  httptransport.HandlerWithArgs[PublishPlanRequest, PublishPlanResponse, string]
)

func (h *handler) PublishPlan() PublishPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (PublishPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return PublishPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod

			req := PublishPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
				},
				RejectUnitConfig:                true,
				RejectUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request PublishPlanRequest) (PublishPlanResponse, error) {
			p, err := h.service.PublishPlan(ctx, request)
			if err != nil {
				return PublishPlanResponse{}, fmt.Errorf("failed to Publish plan: %w", err)
			}

			taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
			if err != nil {
				return PublishPlanResponse{}, err
			}

			return FromPlan(*p, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publishPlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	ArchivePlanRequest  = plan.ArchivePlanInput
	ArchivePlanResponse = api.Plan
	ArchivePlanHandler  httptransport.HandlerWithArgs[ArchivePlanRequest, ArchivePlanResponse, string]
)

func (h *handler) ArchivePlan() ArchivePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (ArchivePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchivePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod.To

			req := ArchivePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectiveTo:                     clock.Now(),
				RejectUnitConfig:                true,
				RejectUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request ArchivePlanRequest) (ArchivePlanResponse, error) {
			p, err := h.service.ArchivePlan(ctx, request)
			if err != nil {
				return ArchivePlanResponse{}, fmt.Errorf("failed to archive plan: %w", err)
			}

			taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
			if err != nil {
				return ArchivePlanResponse{}, err
			}

			return FromPlan(*p, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archivePlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}

type (
	NextPlanRequest  = plan.NextPlanInput
	NextPlanResponse = api.Plan
	NextPlanHandler  httptransport.HandlerWithArgs[NextPlanRequest, NextPlanResponse, string]
)

func (h *handler) NextPlan() NextPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planIdOrKey string) (NextPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return NextPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod.To

			// Try to detect whether the IdOrKey is an ID in ULID format or Key.
			idOrKey := ref.ParseIDOrKey(planIdOrKey)

			req := NextPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        idOrKey.ID,
				},
				Key:                             idOrKey.Key,
				Version:                         0,
				RejectUnitConfig:                true,
				RejectUnrepresentableCurrencies: true,
			}

			return req, nil
		},
		func(ctx context.Context, request NextPlanRequest) (NextPlanResponse, error) {
			p, err := h.service.NextPlan(ctx, request)
			if err != nil {
				return NextPlanResponse{}, fmt.Errorf("failed to create next version of Plan: %w", err)
			}

			// TODO(chrisgacsal): update api.Response in TypeSpec definition to allow returning Plan

			taxCodes, err := h.taxCodesForPlans(ctx, p.Namespace, *p)
			if err != nil {
				return NextPlanResponse{}, err
			}

			return FromPlan(*p, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("nextPlan"),
			httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)),
		)...,
	)
}
