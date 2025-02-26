package appstripe

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/mock"

	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ secret.SecretService = (*MockSecretService)(nil)

func NewMockSecretService() (*MockSecretService, error) {
	secretAdapter := secretadapter.New()

	secretService, err := secretservice.New(secretservice.Config{
		Adapter: secretAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret service")
	}

	return &MockSecretService{
		original:    secretService,
		mockEnabled: false,
	}, nil
}

type MockSecretService struct {
	mock.Mock

	mockEnabled bool
	original    secret.Service
}

func (s *MockSecretService) EnableMock() {
	s.mockEnabled = true
}

func (s *MockSecretService) DisableMock() {
	s.mockEnabled = false
}

func (s *MockSecretService) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	if s.mockEnabled {
		args := s.Called(input)
		if err := input.Validate(); err != nil {
			return secretentity.SecretID{}, models.NewGenericValidationError(
				fmt.Errorf("error create app secret: %w", err),
			)
		}

		return args.Get(0).(secretentity.SecretID), args.Error(1)
	}

	return s.original.CreateAppSecret(ctx, input)
}

func (s *MockSecretService) UpdateAppSecret(ctx context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error) {
	if s.mockEnabled {
		args := s.Called(input)
		if err := input.Validate(); err != nil {
			return input.SecretID, models.NewGenericValidationError(
				fmt.Errorf("error update app secret: %w", err),
			)
		}

		return input.SecretID, args.Error(0)
	}

	return s.original.UpdateAppSecret(ctx, input)
}

func (s *MockSecretService) GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	if s.mockEnabled {
		args := s.Called(input)
		if err := input.Validate(); err != nil {
			return secretentity.Secret{}, models.NewGenericValidationError(
				fmt.Errorf("error get app secret: %w", err),
			)
		}

		return args.Get(0).(secretentity.Secret), args.Error(1)
	}

	return s.original.GetAppSecret(ctx, input)
}

func (s *MockSecretService) DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error {
	if s.mockEnabled {
		args := s.Called(input)
		if err := input.Validate(); err != nil {
			return models.NewGenericValidationError(
				fmt.Errorf("error delete app secret: %w", err),
			)
		}

		return args.Error(0)
	}

	return s.original.DeleteAppSecret(ctx, input)
}
