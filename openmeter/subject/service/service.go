package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
func (s *Service) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		return s.subjectAdapter.Create(ctx, input)
	})
}

// Update updates an existing subject
func (s *Service) Update(ctx context.Context, input subject.UpdateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		return s.subjectAdapter.Update(ctx, input)
	})
}

// GetByIdOrKey gets a subject by ID or key
func (s *Service) GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (subject.Subject, error) {
	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		return s.subjectAdapter.GetByIdOrKey(ctx, orgId, idOrKey)
	})
}

// GetById gets a subject by ID
func (s *Service) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		return s.subjectAdapter.GetById(ctx, id)
	})
}

// GetByKey gets a subject by key
func (s *Service) GetByKey(ctx context.Context, key models.NamespacedKey) (subject.Subject, error) {
	if err := key.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid key: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		return s.subjectAdapter.GetByKey(ctx, key)
	})
}

// List lists subjects
func (s *Service) List(ctx context.Context, orgId string, params subject.ListParams) (pagination.PagedResponse[subject.Subject], error) {
	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (pagination.PagedResponse[subject.Subject], error) {
		return s.subjectAdapter.List(ctx, orgId, params)
	})
}

// DeleteById deletes a subject by ID
func (s *Service) Delete(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return transaction.RunWithNoValue(ctx, s.subjectAdapter, func(ctx context.Context) error {
		return s.subjectAdapter.Delete(ctx, id)
	})
}
