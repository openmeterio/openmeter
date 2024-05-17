package creditdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type ListLedgerGrantsHandler httptransport.HandlerWithArgs[credit.ListGrantsParams, []credit.Grant, api.ListLedgerGrantsParams]

func (b *builder) ListLedgerGrants() ListLedgerGrantsHandler {
	return httptransport.NewHandlerWithArgs[credit.ListGrantsParams, []credit.Grant, api.ListLedgerGrantsParams](
		func(ctx context.Context, r *http.Request, queryIn api.ListLedgerGrantsParams) (credit.ListGrantsParams, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.ListGrantsParams{}, err
			}

			request := credit.ListGrantsParams{
				Namespace:         ns,
				FromHighWatermark: true,
				IncludeVoid:       true,
				Limit:             defaultx.WithDefault(queryIn.Limit, DefaultLedgerQueryLimit),
			}

			if queryIn.LedgerID != nil {
				request.LedgerIDs = []credit.LedgerID{*queryIn.LedgerID}
			}
			return request, nil
		},
		b.CreditConnector.ListGrants,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("listLedgerGrants"),
		)...,
	)
}

type ListLedgerGrantsByLedgerParams struct {
	LedgerID credit.LedgerID
	Params   api.ListLedgerGrantsByLedgerParams
}

type ListLedgerGrantsByLedgerHandler httptransport.HandlerWithArgs[credit.ListGrantsParams, []credit.Grant, ListLedgerGrantsByLedgerParams]

func (b *builder) ListLedgerGrantsByLedger() ListLedgerGrantsByLedgerHandler {
	return httptransport.NewHandlerWithArgs[credit.ListGrantsParams, []credit.Grant, ListLedgerGrantsByLedgerParams](
		func(ctx context.Context, r *http.Request, queryIn ListLedgerGrantsByLedgerParams) (credit.ListGrantsParams, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.ListGrantsParams{}, err
			}

			request := credit.ListGrantsParams{
				Namespace:         ns,
				LedgerIDs:         []credit.LedgerID{queryIn.LedgerID},
				FromHighWatermark: true,
				IncludeVoid:       true,
				Limit:             defaultx.WithDefault(queryIn.Params.Limit, DefaultLedgerQueryLimit),
			}
			return request, nil
		},
		b.CreditConnector.ListGrants,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("listLedgerGrantsByLedger"),
		)...,
	)
}

type CreateLedgerGrantHandler httptransport.HandlerWithArgs[credit.Grant, credit.Grant, api.LedgerID]

func (b *builder) CreateLedgerGrant() CreateLedgerGrantHandler {
	return httptransport.NewHandlerWithArgs[credit.Grant, credit.Grant, api.LedgerID](
		func(ctx context.Context, r *http.Request, arg api.LedgerID) (credit.Grant, error) {
			grant := credit.Grant{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &grant); err != nil {
				return grant, err
			}

			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return grant, err
			}

			grant.LedgerID = arg
			grant.Namespace = ns
			return grant, nil
		},
		b.CreditConnector.CreateGrant,
		commonhttp.JSONResponseEncoderWithStatus[credit.Grant](http.StatusCreated),
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("createLedgerGrant"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.LedgerNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}

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

type GrantPathParams struct {
	LedgerID api.LedgerID
	GrantID  api.LedgerGrantID
}

type VoidLedgerGrantHandler httptransport.HandlerWithArgs[credit.Grant, credit.Grant, GrantPathParams]

func (b *builder) VoidLedgerGrant() VoidLedgerGrantHandler {
	return httptransport.NewHandlerWithArgs[credit.Grant, credit.Grant, GrantPathParams](
		func(ctx context.Context, r *http.Request, in GrantPathParams) (credit.Grant, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.Grant{}, err
			}

			grant, err := b.CreditConnector.GetGrant(ctx, credit.NewNamespacedGrantID(ns, in.GrantID))
			if err != nil {
				if _, ok := err.(*credit.GrantNotFoundError); ok {
					return credit.Grant{}, &credit.GrantNotFoundError{GrantID: in.GrantID}
				}
				return credit.Grant{}, err
			}

			if grant.Namespace != ns {
				return credit.Grant{}, &credit.GrantNotFoundError{GrantID: in.GrantID}
			}

			if grant.LedgerID != in.LedgerID {
				return credit.Grant{}, &credit.GrantNotFoundError{GrantID: in.GrantID}
			}

			if grant.Void {
				return grant, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("grant already voided: %s", in.GrantID),
				)
			}

			return grant, nil
		},
		b.CreditConnector.VoidGrant,
		commonhttp.EmptyResponseEncoder[credit.Grant](http.StatusNoContent),
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("voidLedgerGrant"),
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

				if _, ok := err.(*credit.GrantNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			},
			),
		)...,
	)
}

type GetLedgerGrantHandler httptransport.HandlerWithArgs[credit.NamespacedGrantID, credit.Grant, GrantPathParams]

func (b *builder) GetLedgerGrant() GetLedgerGrantHandler {
	return httptransport.NewHandlerWithArgs[credit.NamespacedGrantID, credit.Grant, GrantPathParams](
		func(ctx context.Context, r *http.Request, in GrantPathParams) (credit.NamespacedGrantID, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.NamespacedGrantID{}, err
			}

			return credit.NewNamespacedGrantID(ns, in.GrantID), nil
		},
		b.CreditConnector.GetGrant,
		commonhttp.JSONResponseEncoder[credit.Grant],
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("getLedgerGrant"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.GrantNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			},
			),
		)...,
	)
}
