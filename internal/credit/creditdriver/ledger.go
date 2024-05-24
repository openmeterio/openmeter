// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package creditdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

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
				credit.NewNamespacedLedgerID(req.Namespace, req.LedgerID),
				req.From,
				defaultx.WithDefault(req.To, time.Now()),
				credit.Pagination{
					Limit:  defaultx.WithDefault(req.Limit, DefaultLedgerQueryLimit),
					Offset: defaultx.WithDefault(req.Offset, 0),
				},
			)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("getLedgerHistory"),
		)...,
	)
}
