package subscriptionaddons

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListSubscriptionAddonsParams struct {
	SubscriptionID apiv3.ULID
	Params         apiv3.ListSubscriptionAddonsParams
}

type listSubscriptionAddonsRequest struct {
	SubscriptionID models.NamespacedID
	Input          subscriptionaddon.ListSubscriptionAddonsInput
}

type (
	ListSubscriptionAddonsRequest  = listSubscriptionAddonsRequest
	ListSubscriptionAddonsResponse = response.PagePaginationResponse[apiv3.SubscriptionAddon]
	ListSubscriptionAddonsHandler  = httptransport.HandlerWithArgs[ListSubscriptionAddonsRequest, ListSubscriptionAddonsResponse, ListSubscriptionAddonsParams]
)

func (h *handler) ListSubscriptionAddons() ListSubscriptionAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubscriptionAddonsParams) (ListSubscriptionAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubscriptionAddonsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Params.Page.Number, 1),
					lo.FromPtrOr(params.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListSubscriptionAddonsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			input := subscriptionaddon.ListSubscriptionAddonsInput{
				SubscriptionID: params.SubscriptionID,
				Page:           page,
			}

			if params.Params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Params.Sort)
				if err != nil {
					return ListSubscriptionAddonsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}

				input.OrderBy = subscriptionaddon.OrderBy(sort.Field)
				input.Order = sort.Order.ToSortxOrder()

				if err := input.OrderBy.Validate(); err != nil {
					return ListSubscriptionAddonsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: fmt.Sprintf("unsupported sort field %q", sort.Field), Source: apierrors.InvalidParamSourceQuery},
					})
				}
			}

			return ListSubscriptionAddonsRequest{
				SubscriptionID: models.NamespacedID{Namespace: ns, ID: params.SubscriptionID},
				Input:          input,
			}, nil
		},
		func(ctx context.Context, req ListSubscriptionAddonsRequest) (ListSubscriptionAddonsResponse, error) {
			res, err := h.addonService.List(ctx, req.SubscriptionID.Namespace, req.Input)
			if err != nil {
				return ListSubscriptionAddonsResponse{}, fmt.Errorf("failed to list subscription addons: %w", err)
			}

			items := make([]apiv3.SubscriptionAddon, 0, len(res.Items))
			for _, item := range res.Items {
				converted, err := toAPISubscriptionAddon(item)
				if err != nil {
					return ListSubscriptionAddonsResponse{}, fmt.Errorf("failed to convert subscription addon: %w", err)
				}
				items = append(items, converted)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Input.Page.PageSize,
				Number: req.Input.Page.PageNumber,
				Total:  lo.ToPtr(res.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubscriptionAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-subscription-addons"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
