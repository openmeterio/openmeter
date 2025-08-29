package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CustomerSubjectHook     = models.ServiceHook[customer.Customer]
	NoopCustomerSubjectHook = models.NoopServiceHook[customer.Customer]
)

var _ models.ServiceHook[customer.Customer] = (*customerSubjectHook)(nil)

type customerSubjectHook struct {
	NoopCustomerSubjectHook

	provisioner *SubjectProvisioner
	logger      *slog.Logger
}

func (s customerSubjectHook) PostCreate(ctx context.Context, cus *customer.Customer) error {
	return s.provisioner.EnsureSubjects(ctx, cus)
}

func (s customerSubjectHook) PostUpdate(ctx context.Context, cus *customer.Customer) error {
	return s.provisioner.EnsureSubjects(ctx, cus)
}

func NewCustomerSubjectHook(config CustomerSubjectHookConfig) (CustomerSubjectHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subject hook config: %w", err)
	}

	provisioner, err := NewSubjectProvisioner(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize customer provisioner: %w", err)
	}

	return &customerSubjectHook{
		provisioner: provisioner,
		logger:      config.Logger.With("subsystem", "subject_customer_provisioner"),
	}, nil
}

type CustomerSubjectHookConfig = SubjectProvisionerConfig

var _ models.Validator = (*SubjectProvisionerConfig)(nil)

type SubjectProvisionerConfig struct {
	Subject subject.Service
	Logger  *slog.Logger
}

func (c SubjectProvisionerConfig) Validate() error {
	var errs []error

	if c.Subject == nil {
		errs = append(errs, fmt.Errorf("subject service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	return errors.Join(errs...)
}

func NewSubjectProvisioner(config SubjectProvisionerConfig) (*SubjectProvisioner, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subject provisioner config: %w", err)
	}

	return &SubjectProvisioner{
		subject: config.Subject,
		logger:  config.Logger.With("subsystem", "subject.provisioner"),
	}, nil
}

type SubjectProvisioner struct {
	subject subject.Service
	logger  *slog.Logger
}

// EnsureSubjects ensures that Subjects are provisioned for each usage attribution entry.
func (p SubjectProvisioner) EnsureSubjects(ctx context.Context, cus *customer.Customer) error {
	if cus == nil {
		return errors.New("failed to provision subject for customer: customer is nil")
	}

	var errs []error

	for _, subKey := range cus.UsageAttribution.SubjectKeys {
		// Check if the subject exists
		sub, err := p.subject.GetByIdOrKey(ctx, cus.Namespace, subKey)
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				// Create Subject if it does not exist
				_, err = p.subject.Create(ctx, subject.CreateInput{
					Namespace: cus.Namespace,
					Key:       subKey,
					Metadata: lo.ToPtr(map[string]interface{}{
						"createdBy":  "subject.provisioner",
						"customerId": cus.ID,
					}),
				})
				if err != nil {
					errs = append(errs,
						fmt.Errorf("failed to create subject for customer [namespace=%s customer.id=%s customer.usage_attribution_key: %s]: %w",
							cus.Namespace, cus.ID, subKey, err),
					)
				}

				continue
			}

			errs = append(errs, err)

			continue
		}

		if sub.Key != subKey {
			errs = append(errs,
				models.NewGenericValidationError(
					fmt.Errorf("use subject key instead of id for usage attribution [namespace=%s customer.id=%s customer.usage_attribution_key: %s]",
						cus.Namespace, cus.ID, subKey),
				),
			)
		}
	}

	return errors.Join(errs...)
}
