package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	GetSubjectParams   = string
	GetSubjectResponse = api.Subject
	GetSubjectHandler  httptransport.HandlerWithArgs[GetSubjectRequest, GetSubjectResponse, GetSubjectParams]
)

type GetSubjectRequest struct {
	namespace      string
	subjectIdOrKey string
}

// GetSubject returns a handler for getting a subject by ID or key.
func (h *handler) GetSubject() GetSubjectHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subjectIdOrKey GetSubjectParams) (GetSubjectRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubjectRequest{}, err
			}

			return GetSubjectRequest{
				namespace:      ns,
				subjectIdOrKey: subjectIdOrKey,
			}, nil
		},
		func(ctx context.Context, request GetSubjectRequest) (GetSubjectResponse, error) {
			// Get subject
			sub, err := h.subjectService.GetByIdOrKey(ctx, request.namespace, request.subjectIdOrKey)
			if err != nil {
				return GetSubjectResponse{}, fmt.Errorf("failed to get subject: %w", err)
			}

			// Respond with subject
			return FromSubject(sub), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubjectResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getSubject"),
		)...,
	)
}

type (
	ListSubjectsResponse = []api.Subject
	ListSubjectsHandler  httptransport.Handler[ListSubjectsRequest, ListSubjectsResponse]
)

type ListSubjectsRequest struct {
	namespace string
}

// ListSubjects returns a handler for listing subjects.
func (h *handler) ListSubjects() ListSubjectsHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubjectsRequest{}, err
			}

			return ListSubjectsRequest{
				namespace: ns,
			}, nil
		},
		func(ctx context.Context, request ListSubjectsRequest) (ListSubjectsResponse, error) {
			result, err := h.subjectService.List(ctx, request.namespace, subject.ListParams{})
			if err != nil {
				return ListSubjectsResponse{}, fmt.Errorf("failed to list subjects in repository: %w", err)
			}

			// Response
			resp := pagination.MapResult(result, func(sub subject.Subject) api.Subject {
				return FromSubject(sub)
			})

			return resp.Items, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubjectsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listSubjects"),
		)...,
	)
}

// Upserts a subject (creates or updates).
// If the subject doesn't exist, it will be created.
// If the subject exists, it will be partially updated with the provided fields.
type (
	UpsertSubjectResponse = []api.Subject
	UpsertSubjectHandler  httptransport.Handler[UpsertSubjectRequest, UpsertSubjectResponse]
)

type UpsertSubjectRequest struct {
	namespace   string
	rawPayloads []map[string]interface{}
	subjects    []api.SubjectUpsert
}

// UpsertSubject returns a new httptransport.Handler for creating a report.
func (h *handler) UpsertSubject() UpsertSubjectHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (UpsertSubjectRequest, error) {
			// Resolve namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertSubjectRequest{}, err
			}

			// TODO: https://github.com/deepmap/oapi-codegen/issues/1039
			// OpenAPI generator cannot handle optional body parameters, so we have to check it manually
			// In both cases when JSON field is `null` or `undefined` it results in `nil`.
			// This means we cannot differentiate between user wanting to erase value or leave it as is.
			// To work this around we parse the body twice. First time we decode it to map and then we decode it to struct.
			// This way we can check if field is present in body and if it is we can set it to nil in update query.
			// Copy body to buffer so we can read it twice
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body.Close() //  must close
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			body := []api.SubjectUpsert{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpsertSubjectRequest{}, fmt.Errorf("failed to decode update report request: %w", err)
			}

			// We decode to body so we can check if field is present in JSON body (undefined vs. null)
			rawPayloads := []map[string]interface{}{}

			err = json.Unmarshal(bodyBytes, &rawPayloads)
			if err != nil {
				return UpsertSubjectRequest{}, fmt.Errorf("failed to decode update report request: %w", err)
			}

			// Check if stripeCustomerId is valid for each payload
			for _, payload := range body {
				if payload.StripeCustomerId != nil && !strings.HasPrefix(*payload.StripeCustomerId, "cus_") {
					err := errors.New("stripeCustomerId must start with cus_ when set")
					return UpsertSubjectRequest{}, commonhttp.NewHTTPError(http.StatusBadRequest, err)
				}
			}

			return UpsertSubjectRequest{
				namespace:   ns,
				rawPayloads: rawPayloads,
				subjects:    body,
			}, nil
		},
		func(ctx context.Context, request UpsertSubjectRequest) (UpsertSubjectResponse, error) {
			// Get subjects by keys
			var keys []string
			for _, body := range request.subjects {
				keys = append(keys, body.Key)
			}

			result, err := h.subjectService.List(ctx, request.namespace, subject.ListParams{
				Keys: keys,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list subjects: %w", err)
			}

			subjectsEntityMap := lo.KeyBy(result.Items, func(sub subject.Subject) string {
				return sub.Key
			})

			// TODO: this is a workaround for batch upserts, we should optimize it
			var subjects []subject.Subject

			for idx, payload := range request.subjects {
				existingSubject, subjectExists := subjectsEntityMap[payload.Key]
				rawPayload := request.rawPayloads[idx]

				// Create subject if not found
				if !subjectExists {
					createdSubject, err := h.subjectService.Create(ctx, subject.CreateInput{
						Namespace:        request.namespace,
						Key:              payload.Key,
						DisplayName:      payload.DisplayName,
						StripeCustomerId: payload.StripeCustomerId,
						Metadata:         payload.Metadata,
					})
					if err != nil {
						return nil, fmt.Errorf("failed to create subject: %w", err)
					}

					subjects = append(subjects, createdSubject)
				} else {
					// Update subject
					updateInput := subject.UpdateInput{
						ID:        existingSubject.Id,
						Namespace: request.namespace,
					}

					if _, ok := rawPayload["displayName"]; ok {
						updateInput.DisplayName = subject.OptionalNullable[string]{
							IsSet: true,
							Value: payload.DisplayName,
						}
					}

					if _, ok := rawPayload["stripeCustomerId"]; ok {
						updateInput.StripeCustomerId = subject.OptionalNullable[string]{
							IsSet: true,
							Value: payload.StripeCustomerId,
						}
					}

					if _, ok := rawPayload["metadata"]; ok {
						updateInput.Metadata = subject.OptionalNullable[map[string]interface{}]{
							IsSet: true,
							Value: payload.Metadata,
						}
					}

					updatedSubject, err := h.subjectService.Update(ctx, updateInput)
					if err != nil {
						return nil, fmt.Errorf("failed to update subject in repository: %w", err)
					}

					subjects = append(subjects, updatedSubject)
				}
			}

			// Respond with updated subject(s)
			var list []api.Subject

			for _, sub := range subjects {
				list = append(list, FromSubject(sub))
			}

			return list, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpsertSubjectResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsertSubject"),
		)...,
	)
}

type (
	DeleteSubjectParams   = string
	DeleteSubjectResponse = interface{}
	DeleteSubjectHandler  httptransport.HandlerWithArgs[DeleteSubjectRequest, DeleteSubjectResponse, DeleteSubjectParams]
)

type DeleteSubjectRequest struct {
	namespace      string
	SubjectIdOrKey string
}

// DeleteSubject returns a handler for deleting a token.
func (h *handler) DeleteSubject() DeleteSubjectHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subjectIdOrKey DeleteSubjectParams) (DeleteSubjectRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteSubjectRequest{}, err
			}

			return DeleteSubjectRequest{
				namespace:      ns,
				SubjectIdOrKey: subjectIdOrKey,
			}, nil
		},
		func(ctx context.Context, request DeleteSubjectRequest) (DeleteSubjectResponse, error) {
			// Get subject
			subjectEntity, err := h.subjectService.GetByIdOrKey(ctx, request.namespace, request.SubjectIdOrKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get subject: %w", err)
			}

			// Delete subject from database
			if err := h.subjectService.Delete(ctx, models.NamespacedID{
				Namespace: request.namespace,
				ID:        subjectEntity.Id,
			}); err != nil {
				return nil, fmt.Errorf("failed to delete subject in repository: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteSubjectResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteSubject"),
		)...,
	)
}
