package httpdriver

import (
	"context"
	"errors"
	"net/http"

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

type CreateGrantParams struct {
	SubjectKey    string
	EntitlementID string
}

type CreateGrantInputs struct {
	inp         entitlement.CreateEntitlementGrantInputs
	entitlement models.NamespacedID
}

type CreateGrantHandler httptransport.HandlerWithArgs[CreateGrantInputs, api.EntitlementGrant, CreateGrantParams]

func (h *meteredEntitlementHandler) CreateGrant() CreateGrantHandler {
	return httptransport.NewHandlerWithArgs[CreateGrantInputs, api.EntitlementGrant, CreateGrantParams](
		func(ctx context.Context, r *http.Request, params CreateGrantParams) (CreateGrantInputs, error) {
			apiGrant := api.EntitlementGrantCreateInput{}
			inp := CreateGrantInputs{}

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

			// Metadata: apiGrant.Metadata,
			// Recurrence: ,
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
		func(ctx context.Context, request CreateGrantInputs) (api.EntitlementGrant, error) {
			grant, err := h.balanceConnector.CreateGrant(ctx, request.entitlement, request.inp)
			if err != nil {
				return api.EntitlementGrant{}, err
			}
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
				SubjectKey:        &request.entitlement.ID,
			}

			if grant.Recurrence != nil {
				apiGrant.Recurrence = &api.RecurringPeriod{
					Anchor:   grant.Recurrence.Anchor,
					Interval: api.RecurringPeriodEnum(grant.Recurrence.Period),
				}
			}

			return apiGrant, nil
		},
		commonhttp.JSONResponseEncoder[api.EntitlementGrant],
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
