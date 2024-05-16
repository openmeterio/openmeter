package creditdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type ResetLedgerHandler httptransport.HandlerWithArgs[credit.Reset, credit.Reset, api.LedgerID]

func (b *builder) ResetLedger() ResetLedgerHandler {
	return httptransport.NewHandlerWithArgs[credit.Reset, credit.Reset, api.LedgerID](
		func(ctx context.Context, r *http.Request, ledgerID ulid.ULID) (credit.Reset, error) {
			resetIn := credit.Reset{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &resetIn); err != nil {
				return resetIn, err
			}

			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return resetIn, err
			}

			resetIn.Namespace = ns

			if resetIn.EffectiveAt.After(time.Now()) {
				return resetIn, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					errors.New("reset date cannot be in the future"),
				)
			}

			resetIn.LedgerID = ledgerID
			return resetIn, nil
		},
		func(ctx context.Context, request credit.Reset) (credit.Reset, error) {
			reset, _, err := b.CreditConnector.Reset(ctx, request)
			return reset, err
		},
		commonhttp.JSONResponseEncoderWithStatus[credit.Reset](http.StatusCreated),
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("resetLedger"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.HighWatermarBeforeError); ok {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}

				if _, ok := err.(*credit.LockErrNotObtainedError); ok {
					commonhttp.NewHTTPError(
						http.StatusConflict,
						fmt.Errorf("credit is currently locked, try again: %w", err),
					).EncodeError(ctx, w)
					return true
				}

				return false
			},
			),
		)...,
	)
}
