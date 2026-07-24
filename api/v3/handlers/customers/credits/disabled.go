package customerscredits

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

const creditsDisabledPrecondition = "credits is not enabled"

type disabledHandler struct {
	options []httptransport.HandlerOption
}

func NewDisabled(options ...httptransport.HandlerOption) Handler {
	return disabledHandler{
		options: options,
	}
}

func (h disabledHandler) GetCustomerCreditBalance() GetCustomerCreditBalanceHandler {
	return disabledHandlerWithArgs[GetCustomerCreditBalanceRequest, GetCustomerCreditBalanceResponse, GetCustomerCreditBalanceParams](
		"get-customer-credit-balance",
		h.options,
	)
}

func (h disabledHandler) ListCreditGrants() ListCreditGrantsHandler {
	return disabledHandlerWithArgs[ListCreditGrantsRequest, ListCreditGrantsResponse, ListCreditGrantsParams](
		"list-credit-grants",
		h.options,
	)
}

func (h disabledHandler) CreateCreditGrant() CreateCreditGrantHandler {
	return disabledHandlerWithArgs[CreateCreditGrantRequest, CreateCreditGrantResponse, CreateCreditGrantParams](
		"create-credit-grant",
		h.options,
	)
}

func (h disabledHandler) GetCreditGrant() GetCreditGrantHandler {
	return disabledHandlerWithArgs[GetCreditGrantRequest, GetCreditGrantResponse, GetCreditGrantParams](
		"get-credit-grant",
		h.options,
	)
}

func (h disabledHandler) VoidCreditGrant() VoidCreditGrantHandler {
	return disabledHandlerWithArgs[VoidCreditGrantRequest, VoidCreditGrantResponse, VoidCreditGrantParams](
		"void-credit-grant",
		h.options,
	)
}

func (h disabledHandler) UpdateCreditGrantExternalSettlement() UpdateCreditGrantExternalSettlementHandler {
	return disabledHandlerWithArgs[UpdateCreditGrantExternalSettlementRequest, UpdateCreditGrantExternalSettlementResponse, UpdateCreditGrantExternalSettlementParams](
		"update-credit-grant-external-settlement",
		h.options,
	)
}

func (h disabledHandler) ListCreditTransactions() ListCreditTransactionsHandler {
	return disabledHandlerWithArgs[ListCreditTransactionsRequest, ListCreditTransactionsResponse, ListCreditTransactionsParams](
		"list-credit-transactions",
		h.options,
	)
}

func disabledHandlerWithArgs[Request, Response, Params any](
	operationName string,
	options []httptransport.HandlerOption,
) httptransport.HandlerWithArgs[Request, Response, Params] {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, _ *http.Request, _ Params) (Request, error) {
			var request Request
			return request, apierrors.NewPreconditionFailedError(ctx, creditsDisabledPrecondition)
		},
		func(context.Context, Request) (Response, error) {
			var response Response
			return response, nil
		},
		commonhttp.JSONResponseEncoder[Response],
		httptransport.AppendOptions(
			options,
			httptransport.WithOperationName(operationName),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
