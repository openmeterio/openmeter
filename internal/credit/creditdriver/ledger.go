package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type CreateLedgerHandler httptransport.Handler[credit.Ledger, credit.Ledger]

func (b *builder) CreateLedger() CreateLedgerHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (credit.Ledger, error) {
			ledgerIn := credit.Ledger{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &ledgerIn); err != nil {
				return ledgerIn, err
			}

			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return ledgerIn, err
			}

			ledgerIn.Namespace = ns

			if ledgerIn.Subject == "" {
				return ledgerIn, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					errors.New("subject must be non-empty when creating a new ledger"),
				)
			}
			return ledgerIn, nil
		},
		b.CreditConnector.CreateLedger,
		commonhttp.JSONResponseEncoderWithStatus[credit.Ledger](http.StatusCreated),
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("createLedger"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if existsError, ok := err.(*credit.LedgerAlreadyExistsError); ok {
					credit.NewLedgerAlreadyExistsProblem(
						ctx,
						err,
						existsError.Ledger,
					).Respond(w)
					return true
				}
				return false
			}),
		)...,
	)
}

type ListLedgersHandler httptransport.HandlerWithArgs[credit.ListLedgersParams, []credit.Ledger, api.ListLedgersParams]

func (b *builder) ListLedgers() ListLedgersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params api.ListLedgersParams) (credit.ListLedgersParams, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.ListLedgersParams{}, err
			}

			req := credit.ListLedgersParams{
				Namespace:   ns,
				Subjects:    defaultx.WithDefault(params.Subject, nil),
				SubjectLike: defaultx.WithDefault(params.SubjectSimilarTo, ""),
				Offset:      defaultx.WithDefault(params.Offset, 0),
				Limit:       defaultx.WithDefault(params.Limit, DefaultLedgerQueryLimit),
				OrderBy:     defaultx.WithDefault((*credit.LedgerOrderBy)(params.OrderBy), credit.LedgerOrderByID),
			}
			return req, nil
		},
		b.CreditConnector.ListLedgers,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("listLedgers"),
		)...,
	)
}

type GetLedgerHistoryRequest struct {
	api.GetLedgerHistoryParams
	// Namespace is filled by the request encoder
	Namespace string
	LedgerID  api.LedgerID
}

type GetLedgerHistoryHandler httptransport.HandlerWithArgs[GetLedgerHistoryRequest, []api.LedgerEntry, GetLedgerHistoryRequest]

func (b *builder) GetLedgerHistory() GetLedgerHistoryHandler {
	return httptransport.NewHandlerWithArgs[GetLedgerHistoryRequest, []api.LedgerEntry, GetLedgerHistoryRequest](
		func(ctx context.Context, r *http.Request, in GetLedgerHistoryRequest) (GetLedgerHistoryRequest, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return in, err
			}

			in.Namespace = ns
			return in, nil
		},
		func(ctx context.Context, req GetLedgerHistoryRequest) ([]api.LedgerEntry, error) {
			ledgerEntryList, err := b.CreditConnector.GetHistory(
				ctx,
				credit.NewNamespacedLedgerID(req.Namespace, req.LedgerID),
				req.From,
				defaultx.WithDefault(req.To, time.Now()),
				credit.Pagination{
					Limit:  defaultx.WithDefault(req.Limit, DefaultLedgerQueryLimit),
					Offset: defaultx.WithDefault(req.Offset, 0),
				},
			)
			if err != nil {
				return nil, err
			}
			res := make([]api.LedgerEntry, 0, ledgerEntryList.Len())
			for _, entry := range ledgerEntryList.GetEntries() {
				res = append(res, mapLedgerEntry(entry))
			}
			return res, nil
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("getLedgerHistory"),
		)...,
	)
}

func mapLedgerEntry(entry credit.LedgerEntry) api.LedgerEntry {
	var period *api.Period
	if entry.Period != nil {
		period = &api.Period{
			From: entry.Period.From,
			To:   entry.Period.To,
		}
	}
	return api.LedgerEntry{
		Id:        convert.ToStringLike[credit.GrantID, string](entry.ID),
		Type:      api.LedgerEntryType(entry.Type),
		Time:      entry.Time,
		FeatureID: string(defaultx.WithDefault(entry.FeatureID, credit.FeatureID(""))),
		Amount:    defaultx.WithDefault(entry.Amount, 0),
		Period:    period,
	}
}
