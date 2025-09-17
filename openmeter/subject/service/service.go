package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ subject.Service = (*Service)(nil)

type Service struct {
	subjectAdapter subject.Adapter
	hooks          models.ServiceHookRegistry[subject.Subject]
}

func (s *Service) RegisterHooks(hooks ...models.ServiceHook[subject.Subject]) {
	s.hooks.RegisterHooks(hooks...)
}

// New creates a new subject service
func New(subjectAdapter subject.Adapter) (*Service, error) {
	if subjectAdapter == nil {
		return nil, fmt.Errorf("subject adapter is required")
	}

	return &Service{
		subjectAdapter: subjectAdapter,
		hooks:          models.ServiceHookRegistry[subject.Subject]{},
	}, nil
}

// Create creates a new subject
func (s *Service) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		sub, err := s.subjectAdapter.Create(ctx, input)
		if err != nil {
			return subject.Subject{}, err
		}

		if err = s.hooks.PostCreate(ctx, &sub); err != nil {
			return subject.Subject{}, err
		}

		return sub, nil
	})
}

// Update updates an existing subject
func (s *Service) Update(ctx context.Context, input subject.UpdateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}

	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		sub, err := s.subjectAdapter.GetById(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.ID,
		})
		if err != nil {
			return subject.Subject{}, err
		}

		if err = s.hooks.PreUpdate(ctx, &sub); err != nil {
			return subject.Subject{}, err
		}

		sub, err = s.subjectAdapter.Update(ctx, input)
		if err != nil {
			return subject.Subject{}, err
		}

		if err = s.hooks.PostUpdate(ctx, &sub); err != nil {
			return subject.Subject{}, err
		}

		return sub, nil
	})
}

// GetByIdOrKey gets a subject by ID or key
func (s *Service) GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (subject.Subject, error) {
	return s.subjectAdapter.GetByIdOrKey(ctx, orgId, idOrKey)
}

// GetById gets a subject by ID
func (s *Service) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return s.subjectAdapter.GetById(ctx, id)
}

// GetByKey gets a subject by key
func (s *Service) GetByKey(ctx context.Context, key models.NamespacedKey) (subject.Subject, error) {
	if err := key.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid key: %w", models.NewGenericValidationError(err))
	}

	return s.subjectAdapter.GetByKey(ctx, key)
}

// List lists subjects
func (s *Service) List(ctx context.Context, orgId string, params subject.ListParams) (pagination.Result[subject.Subject], error) {
	return s.subjectAdapter.List(ctx, orgId, params)
}

// Delete deletes a subject by ID
func (s *Service) Delete(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}

	return transaction.RunWithNoValue(ctx, s.subjectAdapter, func(ctx context.Context) error {
		sub, err := s.subjectAdapter.GetById(ctx, id)
		if err != nil {
			return err
		}

		if err = s.hooks.PreDelete(ctx, &sub); err != nil {
			return err
		}

		if err = s.subjectAdapter.Delete(ctx, id); err != nil {
			return err
		}

		if err = s.hooks.PostDelete(ctx, &sub); err != nil {
			return err
		}

		return nil
	})
}
