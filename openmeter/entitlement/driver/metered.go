package entitlementdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
	entitlementConnector entitlement.Service
	balanceConnector     meteredentitlement.Connector
	subjectService       subject.Service
	customerService      customer.Service
}

func NewMeteredEntitlementHandler(
	entitlementConnector entitlement.Service,
	balanceConnector meteredentitlement.Connector,
	customerService customer.Service,
	subjectService subject.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) MeteredEntitlementHandler {
	return &meteredEntitlementHandler{
		entitlementConnector: entitlementConnector,
		balanceConnector:     balanceConnector,
		customerService:      customerService,
		subjectService:       subjectService,
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
					Priority:    defaultx.WithDefault(apiGrant.Priority, 0),
					EffectiveAt: apiGrant.EffectiveAt,
					Expiration: &grant.ExpirationPeriod{
						Count:    apiGrant.Expiration.Count,
						Duration: grant.ExpirationPeriodDuration(apiGrant.Expiration.Duration),
					},
					ResetMaxRollover: defaultx.WithDefault(apiGrant.MaxRolloverAmount, 0),
					ResetMinRollover: defaultx.WithDefault(apiGrant.MinRolloverAmount, 0),
				},
			}

			if apiGrant.Metadata != nil {
				req.GrantInput.Metadata = lo.FromPtr(apiGrant.Metadata)
			}

			if apiGrant.Recurrence != nil {
				iv, err := MapAPIPeriodIntervalToRecurrence(apiGrant.Recurrence.Interval)
				if err != nil {
					return req, fmt.Errorf("invalid interval: %w", err)
				}

				req.GrantInput.Recurrence = &timeutil.Recurrence{
					Interval: iv,
					Anchor:   defaultx.WithDefault(apiGrant.Recurrence.Anchor, apiGrant.EffectiveAt),
				}
			}

			return req, nil
		},
		func(ctx context.Context, request CreateGrantHandlerRequest) (api.EntitlementGrant, error) {
			cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectKey)
			if err != nil {
				return api.EntitlementGrant{}, err
			}

			grant, err := h.balanceConnector.CreateGrant(ctx, request.Namespace, cust.ID, request.EntitlementIdOrFeatureKey, request.GrantInput)
			if err != nil {
				return api.EntitlementGrant{}, err
			}
			apiGrant := MapEntitlementGrantToAPI(&grant)

			return apiGrant, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[api.EntitlementGrant](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(GetErrorEncoder()),
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
			cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectKey)
			if err != nil {
				return nil, err
			}

			// TODO: validate that entitlement belongs to subject
			grants, err := h.balanceConnector.ListEntitlementGrants(ctx, request.Namespace, meteredentitlement.ListEntitlementGrantsParams{
				CustomerID:                cust.ID,
				EntitlementIDOrFeatureKey: request.EntitlementIdOrFeatureKey,
				Page:                      pagination.NewPage(1, 1000),
			})
			if err != nil {
				return nil, err
			}

			apiGrants := make([]api.EntitlementGrant, 0, len(grants.Items))
			for _, grant := range grants.Items {
				apiGrant := MapEntitlementGrantToAPI(&grant)

				apiGrants = append(apiGrants, apiGrant)
			}

			return apiGrants, nil
		},
		commonhttp.JSONResponseEncoder[[]api.EntitlementGrant],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(GetErrorEncoder()),
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
			httptransport.WithErrorEncoder(GetErrorEncoder()),
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
			httptransport.WithErrorEncoder(GetErrorEncoder()),
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

func (h *meteredEntitlementHandler) resolveCustomerFromSubject(ctx context.Context, namespace string, subjectIdOrKey string) (*customer.Customer, error) {
	subj, err := h.subjectService.GetByIdOrKey(ctx, namespace, subjectIdOrKey)
	if err != nil {
		return nil, err
	}

	if subj.IsDeleted() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("subject is deleted [namespace: %s, subject: %s]", namespace, subjectIdOrKey),
		)
	}

	cust, err := h.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: namespace,
		Key:       subj.Key,
	})
	if err != nil {
		return nil, err
	}

	return cust, nil
}

func MapEntitlementGrantToAPI(grant *meteredentitlement.EntitlementGrant) api.EntitlementGrant {
	apiGrant := api.EntitlementGrant{
		Amount:        grant.Amount,
		CreatedAt:     grant.CreatedAt,
		EffectiveAt:   grant.EffectiveAt,
		Id:            grant.ID,
		Metadata:      convert.MapToPointer(grant.Metadata),
		Annotations:   (*api.Annotations)(convert.MapToPointer(grant.Annotations)),
		Priority:      convert.ToPointer(grant.Priority),
		UpdatedAt:     grant.UpdatedAt,
		DeletedAt:     grant.DeletedAt,
		EntitlementId: grant.EntitlementID,
		Expiration: func() api.ExpirationPeriod {
			if grant.Expiration == nil {
				return api.ExpirationPeriod{
					Count:    100,
					Duration: api.ExpirationDuration("YEAR"),
				}
			}

			return api.ExpirationPeriod{
				Count:    grant.Expiration.Count,
				Duration: api.ExpirationDuration(grant.Expiration.Duration),
			}
		}(),
		ExpiresAt:         lo.ToPtr(lo.FromPtrOr(grant.ExpiresAt, clock.Now().AddDate(100, 0, 0))), // V1 API expects all grants to have an expiresAt so we'll artificially set a very far future date. This is a hack that our users were already doing to get this behavior and we will sunset it later...
		MaxRolloverAmount: &grant.MaxRolloverAmount,
		MinRolloverAmount: &grant.MinRolloverAmount,
		NextRecurrence:    grant.NextRecurrence,
		VoidedAt:          grant.VoidedAt,
	}

	if grant.Recurrence != nil {
		apiGrant.Recurrence = &api.RecurringPeriod{
			Anchor:      grant.Recurrence.Anchor,
			Interval:    MapRecurrenceToAPI(grant.Recurrence.Interval),
			IntervalISO: grant.Recurrence.Interval.ISOString().String(),
		}
	}

	return apiGrant
}
