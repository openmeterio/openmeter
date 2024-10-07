package entitlementdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type MeteredEntitlementHandler interface {
	CreateGrant() CreateGrantHandler
	ListEntitlementGrants() ListEntitlementGrantsHandler
	ResetEntitlementUsage() ResetEntitlementUsageHandler
	GetEntitlementBalanceHistory() GetEntitlementBalanceHistoryHandler
}

type meteredEntitlementHandler struct {
	namespaceDecoder     namespacedriver.NamespaceDecoder
	options              []httptransport.HandlerOption
	entitlementConnector entitlement.Connector
	balanceConnector     meteredentitlement.Connector
}

func NewMeteredEntitlementHandler(
	entitlementConnector entitlement.Connector,
	balanceConnector meteredentitlement.Connector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) MeteredEntitlementHandler {
	return &meteredEntitlementHandler{
		entitlementConnector: entitlementConnector,
		balanceConnector:     balanceConnector,
		namespaceDecoder:     namespaceDecoder,
		options:              options,
	}
}

type (
	CreateGrantHandlerRequest struct {
		GrantInput                meteredentitlement.CreateEntitlementGrantInputs
		EntitlementIdOrFeatureKey string
		Namespace                 string
		SubjectKey                string
	}
	CreateGrantHandlerResponse = api.EntitlementGrant
	CreateGrantHandlerParams   struct {
		SubjectKey                string
		EntitlementIdOrFeatureKey string
	}
)

type CreateGrantHandler httptransport.HandlerWithArgs[CreateGrantHandlerRequest, CreateGrantHandlerResponse, CreateGrantHandlerParams]

func (h *meteredEntitlementHandler) CreateGrant() CreateGrantHandler {
	return httptransport.NewHandlerWithArgs[CreateGrantHandlerRequest, CreateGrantHandlerResponse, CreateGrantHandlerParams](
		func(ctx context.Context, r *http.Request, params CreateGrantHandlerParams) (CreateGrantHandlerRequest, error) {
			apiGrant := api.EntitlementGrantCreateInput{}
			req := CreateGrantHandlerRequest{
				SubjectKey: params.SubjectKey,
			}

			if err := commonhttp.JSONRequestBodyDecoder(r, &apiGrant); err != nil {
				return req, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return req, err
			}

			// TODO: match subjectKey and entitlement
			req.Namespace = ns
			req.EntitlementIdOrFeatureKey = params.EntitlementIdOrFeatureKey

			req.GrantInput = meteredentitlement.CreateEntitlementGrantInputs{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      apiGrant.Amount,
					Priority:    uint8(defaultx.WithDefault(apiGrant.Priority, 0)),
					EffectiveAt: apiGrant.EffectiveAt,
					Expiration: grant.ExpirationPeriod{
						Count:    uint8(apiGrant.Expiration.Count),
						Duration: grant.ExpirationPeriodDuration(apiGrant.Expiration.Duration),
					},
					ResetMaxRollover: defaultx.WithDefault(apiGrant.MaxRolloverAmount, 0),
					ResetMinRollover: defaultx.WithDefault(apiGrant.MinRolloverAmount, 0),
				},
			}

			if apiGrant.Metadata != nil {
				req.GrantInput.Metadata = *apiGrant.Metadata
			}

			if apiGrant.Recurrence != nil {
				req.GrantInput.Recurrence = &recurrence.Recurrence{
					Interval: recurrence.RecurrenceInterval(apiGrant.Recurrence.Interval),
					Anchor:   defaultx.WithDefault(apiGrant.Recurrence.Anchor, apiGrant.EffectiveAt),
				}
			}

			return req, nil
		},
		func(ctx context.Context, request CreateGrantHandlerRequest) (api.EntitlementGrant, error) {
			grant, err := h.balanceConnector.CreateGrant(ctx, request.Namespace, request.SubjectKey, request.EntitlementIdOrFeatureKey, request.GrantInput)
			if err != nil {
				return api.EntitlementGrant{}, err
			}
			apiGrant := MapEntitlementGrantToAPI(&request.SubjectKey, &grant)

			return apiGrant, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[api.EntitlementGrant](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type ListEntitlementGrantHandlerRequest struct {
	EntitlementIdOrFeatureKey string
	Namespace                 string
	SubjectKey                string
}
type (
	ListEntitlementGrantHandlerResponse = []api.EntitlementGrant
	ListEntitlementGrantsHandlerParams  struct {
		EntitlementIdOrFeatureKey string
		SubjectKey                string
	}
)

type ListEntitlementGrantsHandler httptransport.HandlerWithArgs[ListEntitlementGrantHandlerRequest, ListEntitlementGrantHandlerResponse, ListEntitlementGrantsHandlerParams]

func (h *meteredEntitlementHandler) ListEntitlementGrants() ListEntitlementGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListEntitlementGrantHandlerRequest, ListEntitlementGrantHandlerResponse, ListEntitlementGrantsHandlerParams](
		func(ctx context.Context, r *http.Request, params ListEntitlementGrantsHandlerParams) (ListEntitlementGrantHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEntitlementGrantHandlerRequest{}, err
			}

			return ListEntitlementGrantHandlerRequest{
				Namespace:                 ns,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				SubjectKey:                params.SubjectKey,
			}, nil
		},
		func(ctx context.Context, request ListEntitlementGrantHandlerRequest) ([]api.EntitlementGrant, error) {
			// TODO: validate that entitlement belongs to subject
			grants, err := h.balanceConnector.ListEntitlementGrants(ctx, request.Namespace, request.SubjectKey, request.EntitlementIdOrFeatureKey)
			if err != nil {
				return nil, err
			}

			apiGrants := make([]api.EntitlementGrant, 0, len(grants))
			for _, grant := range grants {
				apiGrant := MapEntitlementGrantToAPI(&request.SubjectKey, &grant)

				apiGrants = append(apiGrants, apiGrant)
			}

			return apiGrants, nil
		},
		commonhttp.JSONResponseEncoder[[]api.EntitlementGrant],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type ResetEntitlementUsageHandlerRequest struct {
	EntitlementID   string
	Namespace       string
	SubjectID       string
	At              time.Time
	RetainAnchor    bool
	PreserveOverage *bool
}
type (
	ResetEntitlementUsageHandlerResponse = interface{}
	ResetEntitlementUsageHandlerParams   struct {
		EntitlementID string
		SubjectKey    string
	}
)
type ResetEntitlementUsageHandler httptransport.HandlerWithArgs[ResetEntitlementUsageHandlerRequest, ResetEntitlementUsageHandlerResponse, ResetEntitlementUsageHandlerParams]

func (h *meteredEntitlementHandler) ResetEntitlementUsage() ResetEntitlementUsageHandler {
	return httptransport.NewHandlerWithArgs[ResetEntitlementUsageHandlerRequest, ResetEntitlementUsageHandlerResponse, ResetEntitlementUsageHandlerParams](
		func(ctx context.Context, r *http.Request, params ResetEntitlementUsageHandlerParams) (ResetEntitlementUsageHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResetEntitlementUsageHandlerRequest{}, err
			}

			var body api.ResetEntitlementUsageInput

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return ResetEntitlementUsageHandlerRequest{}, err
			}

			return ResetEntitlementUsageHandlerRequest{
				EntitlementID:   params.EntitlementID,
				Namespace:       ns,
				SubjectID:       params.SubjectKey,
				At:              defaultx.WithDefault(body.EffectiveAt, clock.Now()),
				RetainAnchor:    defaultx.WithDefault(body.RetainAnchor, false),
				PreserveOverage: body.PreserveOverage,
			}, nil
		},
		func(ctx context.Context, request ResetEntitlementUsageHandlerRequest) (interface{}, error) {
			_, err := h.balanceConnector.ResetEntitlementUsage(ctx, models.NamespacedID{
				Namespace: request.Namespace,
				ID:        request.EntitlementID,
			}, meteredentitlement.ResetEntitlementUsageParams{
				At:              request.At,
				RetainAnchor:    request.RetainAnchor,
				PreserveOverage: request.PreserveOverage,
			})
			return nil, err
		},
		commonhttp.EmptyResponseEncoder[interface{}](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type GetEntitlementBalanceHistoryHandlerRequest struct {
	ID     models.NamespacedID
	params meteredentitlement.BalanceHistoryParams
}
type (
	GetEntitlementBalanceHistoryHandlerResponse = api.WindowedBalanceHistory
	GetEntitlementBalanceHistoryHandlerParams   struct {
		EntitlementID string
		SubjectKey    string
		Params        api.GetEntitlementHistoryParams
	}
)
type GetEntitlementBalanceHistoryHandler httptransport.HandlerWithArgs[GetEntitlementBalanceHistoryHandlerRequest, GetEntitlementBalanceHistoryHandlerResponse, GetEntitlementBalanceHistoryHandlerParams]

func (h *meteredEntitlementHandler) GetEntitlementBalanceHistory() GetEntitlementBalanceHistoryHandler {
	return httptransport.NewHandlerWithArgs[GetEntitlementBalanceHistoryHandlerRequest, GetEntitlementBalanceHistoryHandlerResponse, GetEntitlementBalanceHistoryHandlerParams](
		func(ctx context.Context, r *http.Request, params GetEntitlementBalanceHistoryHandlerParams) (GetEntitlementBalanceHistoryHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementBalanceHistoryHandlerRequest{}, err
			}

			tLocation := time.UTC
			if params.Params.WindowTimeZone != nil {
				tz, err := time.LoadLocation(*params.Params.WindowTimeZone)
				if err != nil {
					err := fmt.Errorf("invalid time zone: %w", err)

					return GetEntitlementBalanceHistoryHandlerRequest{}, err
				}
				tLocation = tz
			}

			return GetEntitlementBalanceHistoryHandlerRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.EntitlementID,
				},
				params: meteredentitlement.BalanceHistoryParams{
					From:           params.Params.From,
					To:             params.Params.To,
					WindowSize:     meteredentitlement.WindowSize(params.Params.WindowSize),
					WindowTimeZone: *tLocation,
				},
			}, nil
		},
		func(ctx context.Context, request GetEntitlementBalanceHistoryHandlerRequest) (api.WindowedBalanceHistory, error) {
			windowedHistory, burndownHistory, err := h.balanceConnector.GetEntitlementBalanceHistory(ctx, request.ID, request.params)
			windows := make([]api.BalanceHistoryWindow, 0, len(windowedHistory))
			for _, window := range windowedHistory {
				windows = append(windows, api.BalanceHistoryWindow{
					BalanceAtStart: window.BalanceAtStart,
					Period: api.Period{
						From: window.From,
						To:   window.To,
					},
					Usage: window.UsageInPeriod,
				})
			}

			segments := burndownHistory.Segments()
			burndown := make([]api.GrantBurnDownHistorySegment, 0, len(segments))

			for _, segment := range segments {
				usages := make([]api.GrantUsageRecord, 0, len(segment.GrantUsages))
				for _, usage := range segment.GrantUsages {
					usages = append(usages, api.GrantUsageRecord{
						GrantId: usage.GrantID,
						Usage:   usage.Usage,
					})
				}

				burndown = append(burndown, api.GrantBurnDownHistorySegment{
					BalanceAtEnd:         segment.ApplyUsage().Balance(),
					BalanceAtStart:       segment.BalanceAtStart.Balance(),
					GrantBalancesAtEnd:   (map[string]float64)(segment.ApplyUsage()),
					GrantBalancesAtStart: (map[string]float64)(segment.BalanceAtStart),
					GrantUsages:          usages,
					Overage:              segment.Overage,
					Usage:                segment.TotalUsage,
					Period: api.Period{
						From: segment.From,
						To:   segment.To,
					},
				})
			}

			return api.WindowedBalanceHistory{
				WindowedHistory: windows,
				BurndownHistory: burndown,
			}, err
		},
		commonhttp.JSONResponseEncoder[api.WindowedBalanceHistory],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

func (h *meteredEntitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func MapEntitlementGrantToAPI(subjectKey *string, grant *meteredentitlement.EntitlementGrant) api.EntitlementGrant {
	apiGrant := api.EntitlementGrant{
		Amount:      grant.Amount,
		CreatedAt:   grant.CreatedAt,
		EffectiveAt: grant.EffectiveAt,
		Expiration: api.ExpirationPeriod{
			Count:    int(grant.Expiration.Count),
			Duration: api.ExpirationDuration(grant.Expiration.Duration),
		},
		Id:                grant.ID,
		Metadata:          &grant.Metadata,
		Priority:          convert.ToPointer(int8(grant.Priority)),
		UpdatedAt:         grant.UpdatedAt,
		DeletedAt:         grant.DeletedAt,
		EntitlementId:     grant.EntitlementID,
		ExpiresAt:         &grant.ExpiresAt,
		MaxRolloverAmount: &grant.MaxRolloverAmount,
		MinRolloverAmount: &grant.MinRolloverAmount,
		NextRecurrence:    grant.NextRecurrence,
		VoidedAt:          grant.VoidedAt,
	}

	if grant.Recurrence != nil {
		apiGrant.Recurrence = api.RecurringPeriod{
			Anchor:   grant.Recurrence.Anchor,
			Interval: api.RecurringPeriodInterval(grant.Recurrence.Interval),
		}
	}

	return apiGrant
}
