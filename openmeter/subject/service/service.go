package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ subject.Service = &Service{}

type Service struct {
	subjectAdapter subject.Adapter
}

// New creates a new subject service
func New(subjectAdapter subject.Adapter) *Service {
	return &Service{
		subjectAdapter: subjectAdapter,
	}
}

// Create creates a new subject
func (s *Service) Create(ctx context.Context, input subject.CreateInput) (*subject.Subject, error) {
	return s.subjectAdapter.Create(ctx, input)
}

// Update updates an existing subject
func (s *Service) Update(ctx context.Context, input subject.UpdateInput) (*subject.Subject, error) {
	return s.subjectAdapter.Update(ctx, input)
}

// GetByIdOrKey gets a subject by ID or key
func (s *Service) GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (*subject.Subject, error) {
	return s.subjectAdapter.GetByIdOrKey(ctx, orgId, idOrKey)
}

// GetByKeyWithFallback gets a subject by key with fallback
func (s *Service) GetByKeyWithFallback(ctx context.Context, namespacedKey models.NamespacedKey) (subject.Subject, error) {
	subj, err := s.GetByIdOrKey(ctx, namespacedKey.Namespace, namespacedKey.Key)
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return subject.Subject{
				Key: namespacedKey.Key,
			}, nil
		}
	}

	if subj == nil {
		return subject.Subject{}, fmt.Errorf("subject nil")
	}

	return *subj, nil
}

// List lists subjects
func (s *Service) List(ctx context.Context, orgId string, params subject.ListParams) (pagination.PagedResponse[*subject.Subject], error) {
	return s.subjectAdapter.List(ctx, orgId, params)
}

// DeleteById deletes a subject by ID
func (s *Service) DeleteById(ctx context.Context, id string) error {
	return s.subjectAdapter.DeleteById(ctx, id)
}
