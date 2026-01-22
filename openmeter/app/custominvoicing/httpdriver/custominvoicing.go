package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DraftSyncronizedRequest  = appcustominvoicing.SyncDraftInvoiceInput
	DraftSyncronizedResponse = api.Invoice
	DraftSyncronizedParams   = struct {
		InvoiceID string `json:"invoiceId"`
	}
	DraftSyncronizedHandler httptransport.HandlerWithArgs[DraftSyncronizedRequest, DraftSyncronizedResponse, DraftSyncronizedParams]
)

func (h *handler) DraftSyncronized() DraftSyncronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DraftSyncronizedParams) (DraftSyncronizedRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return DraftSyncronizedRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.CustomInvoicingDraftSynchronizedRequest
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return DraftSyncronizedRequest{}, fmt.Errorf("failed to decode draft synchronized request: %w", err)
			}

			return DraftSyncronizedRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: namespace,
				},
				UpsertInvoiceResults: mapUpsertStandardInvoiceResultFromAPI(body.Invoicing),
			}, nil
		},
		func(ctx context.Context, request DraftSyncronizedRequest) (DraftSyncronizedResponse, error) {
			if err := request.Validate(); err != nil {
				return DraftSyncronizedResponse{}, err
			}

			invoice, err := h.service.SyncDraftInvoice(ctx, request)
			if err != nil {
				return DraftSyncronizedResponse{}, err
			}

			return billinghttpdriver.MapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[DraftSyncronizedResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("DraftSyncronized"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	IssuingSyncronizedRequest  = appcustominvoicing.SyncIssuingInvoiceInput
	IssuingSyncronizedResponse = api.Invoice
	IssuingSyncronizedParams   = struct {
		InvoiceID string `json:"invoiceId"`
	}
	IssuingSyncronizedHandler httptransport.HandlerWithArgs[IssuingSyncronizedRequest, IssuingSyncronizedResponse, IssuingSyncronizedParams]
)

func (h *handler) IssuingSyncronized() IssuingSyncronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params IssuingSyncronizedParams) (IssuingSyncronizedRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return IssuingSyncronizedRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.CustomInvoicingFinalizedRequest
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return IssuingSyncronizedRequest{}, fmt.Errorf("failed to decode issuing synchronized request: %w", err)
			}

			return IssuingSyncronizedRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: namespace,
				},
				FinalizeInvoiceResult: mapFinalizeStandardInvoiceResultFromAPI(body),
			}, nil
		},
		func(ctx context.Context, request IssuingSyncronizedRequest) (IssuingSyncronizedResponse, error) {
			if err := request.Validate(); err != nil {
				return IssuingSyncronizedResponse{}, err
			}

			invoice, err := h.service.SyncIssuingInvoice(ctx, request)
			if err != nil {
				return IssuingSyncronizedResponse{}, err
			}

			return billinghttpdriver.MapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[IssuingSyncronizedResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("IssuingSyncronized"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdatePaymentStatusRequest  = appcustominvoicing.HandlePaymentTriggerInput
	UpdatePaymentStatusResponse = api.Invoice
	UpdatePaymentStatusParams   = struct {
		InvoiceID string `json:"invoiceId"`
	}
	UpdatePaymentStatusHandler httptransport.HandlerWithArgs[UpdatePaymentStatusRequest, UpdatePaymentStatusResponse, UpdatePaymentStatusParams]
)

func (h *handler) UpdatePaymentStatus() UpdatePaymentStatusHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdatePaymentStatusParams) (UpdatePaymentStatusRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePaymentStatusRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.CustomInvoicingUpdatePaymentStatusRequest
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdatePaymentStatusRequest{}, fmt.Errorf("failed to decode handle payment trigger request: %w", err)
			}

			trigger, err := mapPaymentTriggerFromAPI(body.Trigger)
			if err != nil {
				return UpdatePaymentStatusRequest{}, fmt.Errorf("failed to map payment trigger: %w", err)
			}

			return UpdatePaymentStatusRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: namespace,
				},
				Trigger: trigger,
			}, nil
		},
		func(ctx context.Context, request UpdatePaymentStatusRequest) (UpdatePaymentStatusResponse, error) {
			if err := request.Validate(); err != nil {
				return UpdatePaymentStatusResponse{}, err
			}

			invoice, err := h.service.HandlePaymentTrigger(ctx, request)
			if err != nil {
				return UpdatePaymentStatusResponse{}, err
			}

			return billinghttpdriver.MapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePaymentStatusResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("UpdatePaymentStatus"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
