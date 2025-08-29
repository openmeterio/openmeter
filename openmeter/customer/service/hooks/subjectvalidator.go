package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	SubjectValidatorHook     = models.ServiceHook[subject.Subject]
	NoopSubjectValidatorHook = models.NoopServiceHook[subject.Subject]
)

var _ SubjectValidatorHook = (*subjectValidatorHook)(nil)

type subjectValidatorHook struct {
	NoopSubjectValidatorHook

	customer customer.Service
	logger   *slog.Logger
}

func (s subjectValidatorHook) PreDelete(ctx context.Context, sub *subject.Subject) error {
	cus, err := s.customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace:  sub.Namespace,
		SubjectKey: sub.Key,
	})
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return nil
		}

		return err
	}

	if cus == nil {
		return nil
	}

	if cus.DeletedAt != nil && cus.DeletedAt.Before(clock.Now()) {
		return nil
	}

	return models.NewGenericValidationError(
		fmt.Errorf("subject is assigned to customer [namespace=%s subject.key=%s customer.id=%s customer.name=%s]",
			sub.Namespace, sub.Key, cus.ID, cus.Name),
	)
}

func NewSubjectValidatorHook(config SubjectValidatorHookConfig) (SubjectValidatorHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subject validator hook config: %w", err)
	}

	return &subjectValidatorHook{
		customer: config.Customer,
		logger:   config.Logger.With("subsystem", "customer_subject_validator"),
	}, nil
}

type SubjectValidatorHookConfig struct {
	Customer customer.Service
	Logger   *slog.Logger
}

func (c SubjectValidatorHookConfig) Validate() error {
	var errs []error

	if c.Customer == nil {
		errs = append(errs, fmt.Errorf("customer service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	return errors.Join(errs...)
}
