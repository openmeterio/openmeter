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

type GetLedgerBalanceHandler httptransport.HandlerWithArgs[GetLedgerBalanceRequest, credit.Balance, GetLedgerBalaceHandlerParams]

func (b *builder) GetLedgerBalance() GetLedgerBalanceHandler {
	return httptransport.NewHandlerWithArgs[GetLedgerBalanceRequest, credit.Balance, GetLedgerBalaceHandlerParams](
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
		func(ctx context.Context, request GetLedgerBalanceRequest) (credit.Balance, error) {
			return b.CreditConnector.GetBalance(ctx, request.LedgerID, request.Cutline)
		},
		commonhttp.JSONResponseEncoder[credit.Balance],
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
