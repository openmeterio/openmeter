package customerscredits

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCreditGrantRequest  = creditgrant.CreateInput
	CreateCreditGrantResponse = apiv3models.BillingCreditGrant
	CreateCreditGrantParams   struct {
		CustomerID apiv3models.ULID
	}
	CreateCreditGrantHandler httptransport.HandlerWithArgs[CreateCreditGrantRequest, CreateCreditGrantResponse, CreateCreditGrantParams]
)

func (h *handler) CreateCreditGrant() CreateCreditGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args CreateCreditGrantParams) (CreateCreditGrantRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCreditGrantRequest{}, err
			}

			var body apiv3models.CreateCreditGrantRequest
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCreditGrantRequest{}, err
			}

			req, err := fromAPICreateCreditGrantRequest(ns, args.CustomerID, body)
			if err != nil {
				return CreateCreditGrantRequest{}, apierrors.NewBadRequestError(ctx, err, nil)
			}

			return req, nil
		},
		func(ctx context.Context, request CreateCreditGrantRequest) (CreateCreditGrantResponse, error) {
			charge, err := h.creditGrantService.Create(ctx, request)
			if err != nil {
				return CreateCreditGrantResponse{}, err
			}

			return toAPIBillingCreditGrant(charge)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCreditGrantResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-credit-grant"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
