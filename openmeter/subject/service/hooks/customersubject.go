package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

type contextKey struct{}

var skipSubjectCustomerContextKey contextKey

func NewContextWithSkipSubjectCustomer(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipSubjectCustomerContextKey, true)
}

func SkipSubjectCustomerFromContext(ctx context.Context) bool {
	u, ok := ctx.Value(skipSubjectCustomerContextKey).(bool)
	if !ok {
		return false
	}

	return u
}

type (
	CustomerSubjectHook     = models.ServiceHook[customer.Customer]
	NoopCustomerSubjectHook = models.NoopServiceHook[customer.Customer]
)

var _ models.ServiceHook[customer.Customer] = (*customerSubjectHook)(nil)

type customerSubjectHook struct {
	NoopCustomerSubjectHook

	provisioner *SubjectProvisioner
	service     subject.Service
	logger      *slog.Logger
	tracer      trace.Tracer
}

func (s customerSubjectHook) PostCreate(ctx context.Context, cus *customer.Customer) error {
	ctx, span := s.tracer.Start(ctx, "customer_subject_hook.post_create")
	defer span.End()

	err := s.provisioner.EnsureSubjects(ctx, cus)
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to provision subjects for customer")
		span.RecordError(err)
	} else {
		span.SetStatus(otelcodes.Ok, "subjects provisioned for customer")
	}

	return err
}

func (s customerSubjectHook) PostUpdate(ctx context.Context, cus *customer.Customer) error {
	ctx, span := s.tracer.Start(ctx, "customer_subject_hook.post_update")
	defer span.End()

	err := s.provisioner.EnsureSubjects(ctx, cus)
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to provision subjects for customer")
		span.RecordError(err)
	} else {
		span.SetStatus(otelcodes.Ok, "subjects provisioned for customer")
	}

	return err
}

func (s customerSubjectHook) PostDelete(ctx context.Context, cus *customer.Customer) error {
	ctx, span := s.tracer.Start(ctx, "customer_subject_hook.pre_delete")
	defer span.End()

	if SkipSubjectCustomerFromContext(ctx) {
		return nil
	}

	if cus == nil {
		return errors.New("failed to delete subjects for customer: customer is nil")
	}

	if cus.UsageAttribution == nil {
		return nil
	}

	if len(cus.UsageAttribution.SubjectKeys) == 0 {
		return nil
	}

	for _, subKey := range cus.UsageAttribution.SubjectKeys {
		sub, err := s.service.GetByIdOrKey(ctx, cus.Namespace, subKey)
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				continue
			}

			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())

			return err
		}

		if sub.IsDeleted() {
			continue
		}

		if err := s.service.Delete(ctx, models.NamespacedID{
			Namespace: cus.Namespace,
			ID:        sub.Id,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())

			return err
		}

		span.AddEvent("deleted subject", trace.WithAttributes(
			attribute.String("subject.id", sub.Id),
			attribute.String("subject.key", sub.Key),
		))
	}

	return nil
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
		service:     config.Subject,
		logger:      config.Logger.With("subsystem", "subject_customer_provisioner"),
		tracer:      config.Tracer,
	}, nil
}

type CustomerSubjectHookConfig = SubjectProvisionerConfig

var _ models.Validator = (*SubjectProvisionerConfig)(nil)

type SubjectProvisionerConfig struct {
	Subject subject.Service
	Logger  *slog.Logger
	Tracer  trace.Tracer
}

func (c SubjectProvisionerConfig) Validate() error {
	var errs []error

	if c.Subject == nil {
		errs = append(errs, fmt.Errorf("subject service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, fmt.Errorf("tracer is required"))
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
		tracer:  config.Tracer,
	}, nil
}

type SubjectProvisioner struct {
	subject subject.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

// EnsureSubjects ensures that Subjects are provisioned for each usage attribution entry.
func (p SubjectProvisioner) EnsureSubjects(ctx context.Context, cus *customer.Customer) error {
	if SkipSubjectCustomerFromContext(ctx) {
		return nil
	}

	if cus == nil {
		return errors.New("failed to provision subject for customer: customer is nil")
	}

	var err error

	ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subjects")
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(otelcodes.Ok, "subjects provisioned for customer")
		}

		span.End()
	}()

	span.SetAttributes(
		attribute.String("customer.id", cus.ID),
		attribute.String("customer.key", lo.FromPtrOr(cus.Key, "nil")),
	)

	var errs []error

	if cus.UsageAttribution != nil {
		for _, subKey := range cus.UsageAttribution.SubjectKeys {
			sub, err := p.EnsureSubject(ctx, cus, subKey)
			if err != nil {
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
	}

	err = errors.Join(errs...)

	return err
}

func (p SubjectProvisioner) EnsureSubject(ctx context.Context, cus *customer.Customer, subjectKey string) (*subject.Subject, error) {
	if cus == nil {
		return nil, errors.New("failed to provision subject for customer: customer is nil")
	}

	var err error

	ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subject")
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(otelcodes.Ok, "subject provisioned for customer")
		}

		span.End()
	}()

	span.SetAttributes(
		attribute.String("customer.id", cus.ID),
		attribute.String("customer.key", lo.FromPtrOr(cus.Key, "nil")),
	)

	// Check if the subject exists
	sub, err := p.subject.GetByIdOrKey(ctx, cus.Namespace, subjectKey)
	if err != nil && !models.IsGenericNotFoundError(err) {
		return nil, fmt.Errorf("failed to get subject for customer [namespace=%s customer.id=%s customer.usage_attribution_key: %s]: %w",
			cus.Namespace, cus.ID, subjectKey, err)
	}

	if models.IsGenericNotFoundError(err) || sub.IsDeleted() {
		// Create Subject if it does not exist
		sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx),
			subject.CreateInput{
				Namespace: cus.Namespace,
				Key:       subjectKey,
				Metadata: lo.ToPtr(map[string]interface{}{
					"createdBy":  "subject.provisioner",
					"customerId": cus.ID,
				}),
			})
		if err != nil {
			return nil, fmt.Errorf("failed to create subject for customer [namespace=%s customer.id=%s customer.usage_attribution_key: %s]: %w",
				cus.Namespace, cus.ID, subjectKey, err)
		}

		span.AddEvent("created subject", trace.WithAttributes(
			attribute.String("subject.id", sub.Id),
			attribute.String("subject.key", sub.Key),
		))

		return &sub, nil
	}

	span.AddEvent("found subject", trace.WithAttributes(
		attribute.String("subject.id", sub.Id),
		attribute.String("subject.key", sub.Key),
	))

	return &sub, nil
}
