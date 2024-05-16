package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
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
				Namespace: ns,
				Subjects:  defaultx.WithDefault(params.Subject, nil),
				Offset:    defaultx.WithDefault(params.Offset, 0),
				Limit:     defaultx.WithDefault(params.Limit, DefaultLedgerQueryLimit),
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
	LedgerID  ulid.ULID
}

type GetLedgerHistoryHandler httptransport.HandlerWithArgs[GetLedgerHistoryRequest, credit.LedgerEntryList, GetLedgerHistoryRequest]

func (b *builder) GetLedgerHistory() GetLedgerHistoryHandler {
	return httptransport.NewHandlerWithArgs[GetLedgerHistoryRequest, credit.LedgerEntryList, GetLedgerHistoryRequest](
		func(ctx context.Context, r *http.Request, in GetLedgerHistoryRequest) (GetLedgerHistoryRequest, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return in, err
			}

			in.Namespace = ns
			return in, nil
		},
		func(ctx context.Context, req GetLedgerHistoryRequest) (credit.LedgerEntryList, error) {
			return b.CreditConnector.GetHistory(
				ctx,
				credit.NewNamespacedID(req.Namespace, req.LedgerID),
				req.From,
				defaultx.WithDefault(req.To, time.Now()),
				defaultx.WithDefault(req.Limit, DefaultLedgerQueryLimit),
			)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("getLedgerHistory"),
		)...,
	)
}
