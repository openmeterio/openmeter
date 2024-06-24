package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
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
	entitlementConnector entitlement.EntitlementConnector
	balanceConnector     entitlement.EntitlementBalanceConnector
}

func NewMeteredEntitlementHandler(
	entitlementConnector entitlement.EntitlementConnector,
	balanceConnector entitlement.EntitlementBalanceConnector,
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

// The generated api.EntitlementMetered type doesn't really follow our openapi spec
// so we have to manually override some fields...
// FIXME: APIs can drift due to this

type CreateGrantHandlerRequest struct {
	inp         entitlement.CreateEntitlementGrantInputs
	entitlement models.NamespacedID
	subjectKey  string
}
type CreateGrantHandlerResponse = api.EntitlementGrant
type CreateGrantHandlerParams struct {
	SubjectKey    string
	EntitlementID string
}

type CreateGrantHandler httptransport.HandlerWithArgs[CreateGrantHandlerRequest, CreateGrantHandlerResponse, CreateGrantHandlerParams]

func (h *meteredEntitlementHandler) CreateGrant() CreateGrantHandler {
	return httptransport.NewHandlerWithArgs[CreateGrantHandlerRequest, CreateGrantHandlerResponse, CreateGrantHandlerParams](
		func(ctx context.Context, r *http.Request, params CreateGrantHandlerParams) (CreateGrantHandlerRequest, error) {
			apiGrant := api.EntitlementGrantCreateInput{}
			inp := CreateGrantHandlerRequest{
				subjectKey: params.SubjectKey,
			}

			if err := commonhttp.JSONRequestBodyDecoder(r, &apiGrant); err != nil {
				return inp, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return inp, err
			}

			// TODO: match subjectKey and entitlement
			inp.entitlement = models.NamespacedID{
				Namespace: ns,
				ID:        params.EntitlementID,
			}

			inp.inp = entitlement.CreateEntitlementGrantInputs{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      apiGrant.Amount,
					Priority:    uint8(defaultx.WithDefault(apiGrant.Priority, 0)),
					EffectiveAt: apiGrant.EffectiveAt,
					Expiration: credit.ExpirationPeriod{
						Count:    uint8(apiGrant.Expiration.Count),
						Duration: credit.ExpirationPeriodDuration(apiGrant.Expiration.Duration),
					},
					ResetMaxRollover: defaultx.WithDefault(apiGrant.MaxRolloverAmount, 0),
				},
			}

			if apiGrant.Metadata != nil {
				inp.inp.Metadata = *apiGrant.Metadata
			}

			if apiGrant.Recurrence != nil {
				inp.inp.Recurrence = &credit.Recurrence{
					Period: credit.RecurrencePeriod(apiGrant.Recurrence.Interval),
					Anchor: apiGrant.Recurrence.Anchor,
				}
			}

			return inp, nil
		},
		func(ctx context.Context, request CreateGrantHandlerRequest) (api.EntitlementGrant, error) {
			grant, err := h.balanceConnector.CreateGrant(ctx, request.entitlement, request.inp)
			if err != nil {
				return api.EntitlementGrant{}, err
			}
			apiGrant := MapEntitlementGrantToAPI(&request.subjectKey, &grant)

			return apiGrant, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[api.EntitlementGrant](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
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

type ListEntitlementGrantHandlerRequest struct {
	ID         models.NamespacedID
	SubjectKey string
}
type ListEntitlementGrantHandlerResponse = []api.EntitlementGrant
type ListEntitlementGrantsHandlerParams struct {
	EntitlementID string
	SubjectKey    string
}

type ListEntitlementGrantsHandler httptransport.HandlerWithArgs[ListEntitlementGrantHandlerRequest, ListEntitlementGrantHandlerResponse, ListEntitlementGrantsHandlerParams]

func (h *meteredEntitlementHandler) ListEntitlementGrants() ListEntitlementGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListEntitlementGrantHandlerRequest, ListEntitlementGrantHandlerResponse, ListEntitlementGrantsHandlerParams](
		func(ctx context.Context, r *http.Request, params ListEntitlementGrantsHandlerParams) (ListEntitlementGrantHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEntitlementGrantHandlerRequest{}, err
			}

			return ListEntitlementGrantHandlerRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.EntitlementID,
				},
				SubjectKey: params.SubjectKey,
			}, nil
		},
		func(ctx context.Context, request ListEntitlementGrantHandlerRequest) ([]api.EntitlementGrant, error) {
			// TODO: validate that entitlement belongs to subject
			grants, err := h.balanceConnector.ListEntitlementGrants(ctx, request.ID)
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
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
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

type ResetEntitlementUsageHandlerRequest struct {
	EntitlementID           string
	Namespace               string
	SubjectID               string
	At                      time.Time
	RetainUsagePeriodAnchor bool
}

type ResetEntitlementUsageHandlerResponse = interface{}

type ResetEntitlementUsageHandlerParams struct {
	EntitlementID string
	SubjectKey    string
}

type ResetEntitlementUsageHandler httptransport.HandlerWithArgs[ResetEntitlementUsageHandlerRequest, ResetEntitlementUsageHandlerResponse, ResetEntitlementUsageHandlerParams]

func (h *meteredEntitlementHandler) ResetEntitlementUsage() ResetEntitlementUsageHandler {
	return httptransport.NewHandlerWithArgs[ResetEntitlementUsageHandlerRequest, ResetEntitlementUsageHandlerResponse, ResetEntitlementUsageHandlerParams](
		func(ctx context.Context, r *http.Request, params ResetEntitlementUsageHandlerParams) (ResetEntitlementUsageHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResetEntitlementUsageHandlerRequest{}, err
			}

			var body api.ResetEntitlementUsageJSONBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return ResetEntitlementUsageHandlerRequest{}, err
			}

			return ResetEntitlementUsageHandlerRequest{
				EntitlementID:           params.EntitlementID,
				Namespace:               ns,
				SubjectID:               params.SubjectKey,
				At:                      defaultx.WithDefault(body.EffectiveAt, time.Now()),
				RetainUsagePeriodAnchor: defaultx.WithDefault(body.RetainUsagePeriodAnchor, false),
			}, nil
		},
		func(ctx context.Context, request ResetEntitlementUsageHandlerRequest) (interface{}, error) {
			_, err := h.balanceConnector.ResetEntitlementUsage(ctx, models.NamespacedID{
				Namespace: request.Namespace,
				ID:        request.EntitlementID,
			}, entitlement.ResetEntitlementUsageParams{
				ResetAt:                 request.At,
				RetainUsagePeriodAnchor: request.RetainUsagePeriodAnchor,
			})
			return nil, err
		},
		commonhttp.EmptyResponseEncoder[interface{}](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
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

type GetEntitlementBalanceHistoryHandlerRequest struct {
	ID     models.NamespacedID
	params entitlement.BalanceHistoryParams
}
type GetEntitlementBalanceHistoryHandlerResponse = api.WindowedBalanceHistory
type GetEntitlementBalanceHistoryHandlerParams struct {
	EntitlementID string
	SubjectKey    string
	Params        api.GetEntitlementHistoryParams
}
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
				params: entitlement.BalanceHistoryParams{
					From:           params.Params.From,
					To:             defaultx.WithDefault(params.Params.To, time.Now()),
					WindowSize:     entitlement.WindowSize(params.Params.WindowSize),
					WindowTimeZone: *tLocation,
				},
			}, nil
		},
		func(ctx context.Context, request GetEntitlementBalanceHistoryHandlerRequest) (api.WindowedBalanceHistory, error) {
			windowedHistory, burndownHistory, err := h.balanceConnector.GetEntitlementBalanceHistory(ctx, request.ID, request.params)
			windows := make([]api.BalanceHistoryWindow, 0, len(windowedHistory))
			for _, window := range windowedHistory {
				windows = append(windows, api.BalanceHistoryWindow{
					BalanceAtStart: &window.BalanceAtStart,
					Period: &api.Period{
						From: window.From,
						To:   window.To,
					},
					Usage: &window.UsageInPeriod,
				})
			}

			segments := burndownHistory.Segments()
			burndown := make([]api.GrantBurnDownHistorySegment, 0, len(segments))

			for _, segment := range segments {
				usages := make([]api.GrantUsageRecord, 0, len(segment.GrantUsages))
				for _, usage := range segment.GrantUsages {
					usages = append(usages, api.GrantUsageRecord{
						GrantId: &usage.GrantID,
						Usage:   &usage.Usage,
					})
				}

				burndown = append(burndown, api.GrantBurnDownHistorySegment{
					BalanceAtEnd:         convert.ToPointer(segment.ApplyUsage().Balance()),
					BalanceAtStart:       convert.ToPointer(segment.BalanceAtStart.Balance()),
					GrantBalancesAtEnd:   convert.ToPointer((map[string]float64)(segment.ApplyUsage())),
					GrantBalancesAtStart: convert.ToPointer((map[string]float64)(segment.BalanceAtStart)),
					GrantUsages:          &usages,
					Overage:              &segment.Overage,
					Usage:                &segment.TotalUsage,
					Period: &api.Period{
						From: segment.From,
						To:   segment.To,
					},
				})
			}

			return api.WindowedBalanceHistory{
				WindowedHistory: &windows,
				BurndownHistory: &burndown,
			}, err
		},
		commonhttp.JSONResponseEncoder[api.WindowedBalanceHistory],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
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

func (h *meteredEntitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func MapEntitlementGrantToAPI(subjectKey *string, grant *entitlement.EntitlementGrant) api.EntitlementGrant {
	apiGrant := api.EntitlementGrant{
		Amount:      grant.Amount,
		CreatedAt:   &grant.CreatedAt,
		EffectiveAt: grant.EffectiveAt,
		Expiration: api.ExpirationPeriod{
			Count:    int(grant.Expiration.Count),
			Duration: api.ExpirationPeriodDuration(grant.Expiration.Duration),
		},
		Id:                &grant.ID,
		Metadata:          &grant.Metadata,
		Priority:          convert.ToPointer(int(grant.Priority)),
		UpdatedAt:         &grant.UpdatedAt,
		DeletedAt:         grant.DeletedAt,
		EntitlementId:     &grant.EntitlementID,
		ExpiresAt:         &grant.ExpiresAt,
		MaxRolloverAmount: &grant.MaxRolloverAmount,
		NextRecurrence:    grant.NextRecurrence,
		VoidedAt:          grant.VoidedAt,
	}
	if subjectKey != nil {
		apiGrant.SubjectKey = subjectKey
	}

	if grant.Recurrence != nil {
		apiGrant.Recurrence = &api.RecurringPeriod{
			Anchor:   grant.Recurrence.Anchor,
			Interval: api.RecurringPeriodEnum(grant.Recurrence.Period),
		}
	}

	return apiGrant
}
