package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	progressmanagerentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetProgressRequest  = progressmanagerentity.GetProgressInput
	GetProgressResponse = api.Progress
	GetProgressHandler  httptransport.HandlerWithArgs[GetProgressRequest, GetProgressResponse, string]
)

// GetProgress returns a handler for getting the progress by id.
func (h *handler) GetProgress() GetProgressHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, progressID string) (GetProgressRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetProgressRequest{}, err
			}

			return GetProgressRequest{
				ProgressID: progressmanagerentity.ProgressID{
					NamespacedModel: models.NamespacedModel{
						Namespace: ns,
					},
					ID: progressID,
				},
			}, nil
		},
		func(ctx context.Context, request GetProgressRequest) (GetProgressResponse, error) {
			progress, err := h.service.GetProgress(ctx, request)
			if err != nil {
				return GetProgressResponse{}, err
			}

			if progress == nil {
				return GetProgressResponse{}, fmt.Errorf("failed to get progress")
			}

			apiProgress := progressToAPI(*progress)

			return apiProgress, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetProgressResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getProgress"),
		)...,
	)
}

// progressToAPI converts a Progress to an API Progress
func progressToAPI(p progressmanagerentity.Progress) api.Progress {
	return api.Progress{
		Success:   p.Success,
		Failed:    p.Failed,
		Total:     p.Total,
		UpdatedAt: p.UpdatedAt,
	}
}
