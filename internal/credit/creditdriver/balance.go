package creditdriver

import (
	"context"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type GetLedgerBalaceHandlerParams struct {
	LedgerID    api.LedgerID
	QueryParams api.GetLedgerBalanceParams
}

type GetLedgerBalanceRequest struct {
	LedgerID credit.NamespacedLedgerID
	Cutline  time.Time
}

type GetLedgerBalanceHandlerResponse struct {
	credit.Balance
	LastReset time.Time `json:"lastReset"`
	LedgerID  string    `json:"ledgerID"`
}

type GetLedgerBalanceHandler httptransport.HandlerWithArgs[GetLedgerBalanceRequest, api.LedgerBalance, GetLedgerBalaceHandlerParams]

func (b *builder) GetLedgerBalance() GetLedgerBalanceHandler {
	return httptransport.NewHandlerWithArgs[GetLedgerBalanceRequest, api.LedgerBalance, GetLedgerBalaceHandlerParams](
		func(ctx context.Context, r *http.Request, queryIn GetLedgerBalaceHandlerParams) (GetLedgerBalanceRequest, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return GetLedgerBalanceRequest{}, err
			}

			return GetLedgerBalanceRequest{
				LedgerID: credit.NewNamespacedLedgerID(ns, queryIn.LedgerID),
				Cutline:  defaultx.WithDefault(queryIn.QueryParams.Time, time.Now()),
			}, nil
		},
		func(ctx context.Context, request GetLedgerBalanceRequest) (api.LedgerBalance, error) {
			balance, err := b.CreditConnector.GetBalance(ctx, request.LedgerID, request.Cutline)
			if err != nil {
				return api.LedgerBalance{}, err
			}
			highWatermark, err := b.CreditConnector.GetHighWatermark(ctx, request.LedgerID)
			return mapBalanceToAPI(balance, highWatermark), err
		},
		commonhttp.JSONResponseEncoder[api.LedgerBalance],
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("getLedgerBalance"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.HighWatermarBeforeError); ok {
					commonhttp.NewHTTPError(http.StatusBadRequest, err).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*credit.LedgerNotFoundError); ok {
					commonhttp.NewHTTPError(http.StatusNotFound, err).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*credit.LockErrNotObtainedError); ok {
					commonhttp.NewHTTPError(http.StatusConflict, err).EncodeError(ctx, w)
					return true
				}
				return false
			}))...,
	)
}

func mapBalanceToAPI(balance credit.Balance, highwatermark credit.HighWatermark) api.LedgerBalance {
	var featureBalances []api.FeatureBalance = make([]api.FeatureBalance, 0, len(balance.FeatureBalances))
	for _, featureBalance := range balance.FeatureBalances {
		featureBalances = append(featureBalances, mapFeatureBalanceToAPI(featureBalance))
	}

	var grantBalances []api.LedgerGrantBalance = make([]api.LedgerGrantBalance, 0, len(balance.GrantBalances))
	for _, grantBalance := range balance.GrantBalances {
		grantBalances = append(grantBalances, mapGrantBalanceToAPI(grantBalance))
	}

	return api.LedgerBalance{
		FeatureBalances: featureBalances,
		GrantBalances:   grantBalances,
		LastReset:       &highwatermark.Time,
		Metadata:        &balance.Metadata,
		Subject:         balance.Subject,
	}
}

func mapFeatureBalanceToAPI(featureBalance credit.FeatureBalance) api.FeatureBalance {
	id := new(string)
	if featureBalance.ID != nil {
		*id = string(*featureBalance.ID)
	}
	return api.FeatureBalance{
		Archived:            featureBalance.Archived,
		CreatedAt:           featureBalance.CreatedAt,
		Id:                  id,
		MeterGroupByFilters: featureBalance.MeterGroupByFilters,
		MeterSlug:           featureBalance.MeterSlug,
		Name:                featureBalance.Name,
		UpdatedAt:           featureBalance.UpdatedAt,
		Usage:               float32(featureBalance.Usage),
		Balance:             float32(featureBalance.Balance),
	}
}

func mapGrantBalanceToAPI(grantBalance credit.GrantBalance) api.LedgerGrantBalance {
	id := new(string)
	if grantBalance.ID != nil {
		*id = string(*grantBalance.ID)
	}
	parentId := new(string)
	if grantBalance.ParentID != nil {
		*parentId = string(*grantBalance.ParentID)
	}
	priority := int(grantBalance.Priority)

	return api.LedgerGrantBalance{
		Amount:      float32(grantBalance.Amount),
		Balance:     float32(grantBalance.Balance),
		CreatedAt:   grantBalance.CreatedAt,
		EffectiveAt: grantBalance.EffectiveAt,
		Expiration: &api.LedgerGrantExpirationPeriod{
			Count:    int(grantBalance.Expiration.Count),
			Duration: api.LedgerGrantExpirationPeriodDuration(grantBalance.Expiration.Duration),
		},
		ExpiresAt: &grantBalance.ExpiresAt,
		FeatureID: string(*grantBalance.FeatureID),
		Id:        id,
		Metadata:  &grantBalance.Metadata,
		ParentId:  parentId,
		Priority:  &priority,
		Rollover:  grantBalance.Rollover,
		Type:      api.LedgerGrantType(grantBalance.Type),
		UpdatedAt: grantBalance.UpdatedAt,
	}
}
