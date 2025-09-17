package entitlementdriverv2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type (
	ListCustomerEntitlementGrantsHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Params                    api.ListCustomerEntitlementGrantsV2Params
	}
	ListCustomerEntitlementGrantsHandlerRequest struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Namespace                 string
		Params                    api.ListCustomerEntitlementGrantsV2Params
	}
	ListCustomerEntitlementGrantsHandlerResponse = api.GrantPaginatedResponse
	ListCustomerEntitlementGrantsHandler         = httptransport.HandlerWithArgs[ListCustomerEntitlementGrantsHandlerRequest, ListCustomerEntitlementGrantsHandlerResponse, ListCustomerEntitlementGrantsHandlerParams]
)

func (h *entitlementHandler) ListCustomerEntitlementGrants() ListCustomerEntitlementGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListCustomerEntitlementGrantsHandlerRequest, ListCustomerEntitlementGrantsHandlerResponse, ListCustomerEntitlementGrantsHandlerParams](
		func(ctx context.Context, r *http.Request, params ListCustomerEntitlementGrantsHandlerParams) (ListCustomerEntitlementGrantsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerRequest{}, err
			}

			return ListCustomerEntitlementGrantsHandlerRequest{
				CustomerIDOrKey:           params.CustomerIDOrKey,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
			}, nil
		},
		func(ctx context.Context, request ListCustomerEntitlementGrantsHandlerRequest) (ListCustomerEntitlementGrantsHandlerResponse, error) {
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: request.Namespace,
					IDOrKey:   request.CustomerIDOrKey,
				},
			})
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerResponse{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return ListCustomerEntitlementGrantsHandlerResponse{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			grants, err := h.balanceConnector.ListEntitlementGrants(ctx, request.Namespace, meteredentitlement.ListEntitlementGrantsParams{
				CustomerID:                cus.ID,
				EntitlementIDOrFeatureKey: request.EntitlementIdOrFeatureKey,
				OrderBy:                   grant.OrderBy(lo.CoalesceOrEmpty(string(lo.FromPtr(request.Params.OrderBy)), string(grant.OrderByDefault))),
				Order:                     sortx.Order(lo.CoalesceOrEmpty(string(lo.FromPtr(request.Params.Order)), string(sortx.OrderDefault))),
				Page: pagination.NewPage(
					lo.FromPtrOr(request.Params.Page, 1),
					lo.FromPtrOr(request.Params.PageSize, 100),
				),
			})
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerResponse{}, err
			}

			mapped := pagination.MapResult(grants, func(grant meteredentitlement.EntitlementGrant) api.EntitlementGrant {
				return entitlementdriver.MapEntitlementGrantToAPI(&grant)
			})

			return ListCustomerEntitlementGrantsHandlerResponse{
				Items:      mapped.Items,
				Page:       mapped.Page.PageNumber,
				PageSize:   mapped.Page.PageSize,
				TotalCount: mapped.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListCustomerEntitlementGrantsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomerEntitlementGrantsV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

// CreateCustomerEntitlementGrant
// Creates a grant for a customer's entitlement
// (POST /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey}/grants)
type (
	CreateCustomerEntitlementGrantHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
	}
	CreateCustomerEntitlementGrantHandlerRequest struct {
		CustomerID                string
		EntitlementIdOrFeatureKey string
		Namespace                 string
		GrantInput                meteredentitlement.CreateEntitlementGrantInputs
	}
	CreateCustomerEntitlementGrantHandlerResponse = api.EntitlementGrant
	CreateCustomerEntitlementGrantHandler         = httptransport.HandlerWithArgs[CreateCustomerEntitlementGrantHandlerRequest, CreateCustomerEntitlementGrantHandlerResponse, CreateCustomerEntitlementGrantHandlerParams]
)

func (h *entitlementHandler) CreateCustomerEntitlementGrant() CreateCustomerEntitlementGrantHandler {
	return httptransport.NewHandlerWithArgs[
		CreateCustomerEntitlementGrantHandlerRequest,
		CreateCustomerEntitlementGrantHandlerResponse,
		CreateCustomerEntitlementGrantHandlerParams,
	](
		func(ctx context.Context, r *http.Request, params CreateCustomerEntitlementGrantHandlerParams) (CreateCustomerEntitlementGrantHandlerRequest, error) {
			var body api.EntitlementGrantCreateInput
			var req CreateCustomerEntitlementGrantHandlerRequest

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return req, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return req, err
			}

			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: params.CustomerIDOrKey},
			})
			if err != nil {
				return req, err
			}

			if cus != nil && cus.IsDeleted() {
				return req, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			grantInput := meteredentitlement.CreateEntitlementGrantInputs{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      body.Amount,
					Priority:    defaultx.WithDefault(body.Priority, 0),
					EffectiveAt: body.EffectiveAt,
					Expiration: grant.ExpirationPeriod{
						Count:    body.Expiration.Count,
						Duration: grant.ExpirationPeriodDuration(body.Expiration.Duration),
					},
					ResetMaxRollover: defaultx.WithDefault(body.MaxRolloverAmount, 0),
					ResetMinRollover: defaultx.WithDefault(body.MinRolloverAmount, 0),
				},
			}

			if body.Metadata != nil {
				grantInput.Metadata = *body.Metadata
			}

			if body.Recurrence != nil {
				iv, err := entitlementdriver.MapAPIPeriodIntervalToRecurrence(body.Recurrence.Interval)
				if err != nil {
					return req, err
				}
				grantInput.Recurrence = &timeutil.Recurrence{
					Interval: iv,
					Anchor:   defaultx.WithDefault(body.Recurrence.Anchor, body.EffectiveAt),
				}
			}

			req = CreateCustomerEntitlementGrantHandlerRequest{
				CustomerID:                cus.ID,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
				GrantInput:                grantInput,
			}
			return req, nil
		},
		func(ctx context.Context, request CreateCustomerEntitlementGrantHandlerRequest) (CreateCustomerEntitlementGrantHandlerResponse, error) {
			g, err := h.balanceConnector.CreateGrant(ctx, request.Namespace, request.CustomerID, request.EntitlementIdOrFeatureKey, request.GrantInput)
			if err != nil {
				return CreateCustomerEntitlementGrantHandlerResponse{}, err
			}
			return entitlementdriver.MapEntitlementGrantToAPI(&g), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerEntitlementGrantHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCustomerEntitlementGrantV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

// GetCustomerEntitlementHistory
// (GET /api/v2/customers/{customerIdOrKey}/entitlements/{featureKey}/history)
type (
	GetCustomerEntitlementHistoryHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Params                    api.GetCustomerEntitlementHistoryV2Params
	}
	GetCustomerEntitlementHistoryHandlerRequest struct {
		Namespace     string
		CustomerID    string
		EntitlementID string
		Params        api.GetCustomerEntitlementHistoryV2Params
	}
	GetCustomerEntitlementHistoryHandlerResponse = api.WindowedBalanceHistory
	GetCustomerEntitlementHistoryHandler         = httptransport.HandlerWithArgs[GetCustomerEntitlementHistoryHandlerRequest, GetCustomerEntitlementHistoryHandlerResponse, GetCustomerEntitlementHistoryHandlerParams]
)

func (h *entitlementHandler) GetCustomerEntitlementHistory() GetCustomerEntitlementHistoryHandler {
	return httptransport.NewHandlerWithArgs[
		GetCustomerEntitlementHistoryHandlerRequest,
		GetCustomerEntitlementHistoryHandlerResponse,
		GetCustomerEntitlementHistoryHandlerParams,
	](
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementHistoryHandlerParams) (GetCustomerEntitlementHistoryHandlerRequest, error) {
			var def GetCustomerEntitlementHistoryHandlerRequest

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return def, err
			}

			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: params.CustomerIDOrKey},
			})
			if err != nil {
				return def, err
			}

			if cus != nil && cus.IsDeleted() {
				return def, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Resolve entitlement ID from feature key for the given customer
			ent, err := h.connector.GetEntitlementOfCustomerAt(ctx, ns, cus.ID, params.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil {
				return def, err
			}

			return GetCustomerEntitlementHistoryHandlerRequest{
				Namespace:     ns,
				CustomerID:    cus.ID,
				EntitlementID: ent.ID,
				Params:        params.Params,
			}, nil
		},
		func(ctx context.Context, req GetCustomerEntitlementHistoryHandlerRequest) (GetCustomerEntitlementHistoryHandlerResponse, error) {
			// Time zone handling similar to v1
			tLocation := time.UTC
			if req.Params.WindowTimeZone != nil {
				tz, err := time.LoadLocation(*req.Params.WindowTimeZone)
				if err != nil {
					return api.WindowedBalanceHistory{}, commonhttp.NewHTTPError(http.StatusBadRequest, err)
				}
				tLocation = tz
			}

			windowedHistory, burndownHistory, err := h.balanceConnector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: req.Namespace, ID: req.EntitlementID}, meteredentitlement.BalanceHistoryParams{
				From:           req.Params.From,
				To:             req.Params.To,
				WindowSize:     meteredentitlement.WindowSize(req.Params.WindowSize),
				WindowTimeZone: *tLocation,
			})
			if err != nil {
				return api.WindowedBalanceHistory{}, err
			}

			windows := make([]api.BalanceHistoryWindow, 0, len(windowedHistory))
			for _, window := range windowedHistory {
				windows = append(windows, api.BalanceHistoryWindow{
					BalanceAtStart: window.BalanceAtStart,
					Period:         api.Period{From: window.From, To: window.To},
					Usage:          window.UsageInPeriod,
				})
			}

			segments := burndownHistory.Segments()
			burndown := make([]api.GrantBurnDownHistorySegment, 0, len(segments))
			for _, segment := range segments {
				usages := make([]api.GrantUsageRecord, 0, len(segment.GrantUsages))
				for _, usage := range segment.GrantUsages {
					usages = append(usages, api.GrantUsageRecord{GrantId: usage.GrantID, Usage: usage.Usage})
				}
				burndown = append(burndown, api.GrantBurnDownHistorySegment{
					BalanceAtEnd:         segment.ApplyUsage().Balance(),
					BalanceAtStart:       segment.BalanceAtStart.Balance(),
					GrantBalancesAtEnd:   (map[string]float64)(segment.ApplyUsage()),
					GrantBalancesAtStart: (map[string]float64)(segment.BalanceAtStart),
					GrantUsages:          usages,
					Overage:              segment.Overage,
					Usage:                segment.TotalUsage,
					Period:               api.Period{From: segment.From, To: segment.To},
				})
			}

			return api.WindowedBalanceHistory{WindowedHistory: windows, BurndownHistory: burndown}, nil
		},
		commonhttp.JSONResponseEncoder[GetCustomerEntitlementHistoryHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomerEntitlementHistoryV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

// ResetCustomerEntitlementUsage
// (POST /api/v2/customers/{customerId}/entitlements/{featureKey}/reset)
type (
	ResetCustomerEntitlementUsageHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
	}
	ResetCustomerEntitlementUsageHandlerRequest struct {
		EntitlementID   string
		Namespace       string
		At              time.Time
		RetainAnchor    bool
		PreserveOverage *bool
	}
	ResetCustomerEntitlementUsageHandlerResponse = interface{}
	ResetCustomerEntitlementUsageHandler         = httptransport.HandlerWithArgs[ResetCustomerEntitlementUsageHandlerRequest, ResetCustomerEntitlementUsageHandlerResponse, ResetCustomerEntitlementUsageHandlerParams]
)

func (h *entitlementHandler) ResetCustomerEntitlementUsage() ResetCustomerEntitlementUsageHandler {
	return httptransport.NewHandlerWithArgs[
		ResetCustomerEntitlementUsageHandlerRequest,
		ResetCustomerEntitlementUsageHandlerResponse,
		ResetCustomerEntitlementUsageHandlerParams,
	](
		func(ctx context.Context, r *http.Request, params ResetCustomerEntitlementUsageHandlerParams) (ResetCustomerEntitlementUsageHandlerRequest, error) {
			var body api.ResetEntitlementUsageInput

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResetCustomerEntitlementUsageHandlerRequest{}, err
			}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return ResetCustomerEntitlementUsageHandlerRequest{}, err
			}

			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: params.CustomerIDOrKey},
			})
			if err != nil {
				return ResetCustomerEntitlementUsageHandlerRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return ResetCustomerEntitlementUsageHandlerRequest{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Resolve entitlement
			ent, err := h.connector.GetEntitlementOfCustomerAt(ctx, ns, cus.ID, params.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil {
				return ResetCustomerEntitlementUsageHandlerRequest{}, err
			}

			return ResetCustomerEntitlementUsageHandlerRequest{
				EntitlementID:   ent.ID,
				Namespace:       ns,
				At:              defaultx.WithDefault(body.EffectiveAt, clock.Now()),
				RetainAnchor:    defaultx.WithDefault(body.RetainAnchor, false),
				PreserveOverage: body.PreserveOverage,
			}, nil
		},
		func(ctx context.Context, req ResetCustomerEntitlementUsageHandlerRequest) (ResetCustomerEntitlementUsageHandlerResponse, error) {
			_, err := h.balanceConnector.ResetEntitlementUsage(ctx, models.NamespacedID{Namespace: req.Namespace, ID: req.EntitlementID}, meteredentitlement.ResetEntitlementUsageParams{
				At:              req.At,
				RetainAnchor:    req.RetainAnchor,
				PreserveOverage: req.PreserveOverage,
			})
			return nil, err
		},
		commonhttp.EmptyResponseEncoder[ResetCustomerEntitlementUsageHandlerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("resetCustomerEntitlementUsageV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}
