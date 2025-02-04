package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateStripeAPIKeyRequest  = appstripeentity.UpdateAPIKeyInput
	UpdateStripeAPIKeyResponse = struct{}
	UpdateStripeAPIKeyHandler  httptransport.HandlerWithArgs[UpdateStripeAPIKeyRequest, UpdateStripeAPIKeyResponse, string]
)

// UpdateStripeAPIKeyHandler returns a handler for replacing stripe API key
func (h *handler) UpdateStripeAPIKey() UpdateStripeAPIKeyHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appID string) (UpdateStripeAPIKeyRequest, error) {
			body := api.UpdateStripeAPIKeyJSONRequestBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateStripeAPIKeyRequest{}, fmt.Errorf("field to decode replace stripe api key request: %w", err)
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateStripeAPIKeyRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := UpdateStripeAPIKeyRequest{
				AppID:  app.AppID{Namespace: namespace, ID: appID},
				APIKey: body.SecretAPIKey,
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateStripeAPIKeyRequest) (UpdateStripeAPIKeyResponse, error) {
			err := h.service.UpdateAPIKey(ctx, request)
			if err != nil {
				return UpdateStripeAPIKeyResponse{}, fmt.Errorf("failed to replace stripe api key: %w", err)
			}

			return UpdateStripeAPIKeyResponse{}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateStripeAPIKeyResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("replaceStripeAPIKey"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
