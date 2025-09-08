package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	customerservicehooks "github.com/openmeterio/openmeter/openmeter/customer/service/hooks"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementvalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Customer = wire.NewSet(
	NewCustomerService,
)

func NewCustomerService(
	logger *slog.Logger,
	db *entdb.Client,
	entRegistry *registry.Entitlement,
	eventPublisher eventbus.Publisher,
) (customer.Service, error) {
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: db,
		Logger: logger.WithGroup("customer.postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	service, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: eventPublisher,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer service: %w", err)
	}

	// Create and register the entitlement validator
	validator, err := entitlementvalidator.NewValidator(entRegistry.EntitlementRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement validator: %w", err)
	}

	service.RegisterRequestValidator(validator)

	return service, nil
}

type CustomerSubjectHook customerservicehooks.SubjectCustomerHook

func NewCustomerSubjectServiceHook(
	config config.CustomerConfiguration,
	logger *slog.Logger,
	tracer trace.Tracer,
	subjectService subject.Service,
	customerService customer.Service,
) (CustomerSubjectHook, error) {
	if !config.EnableSubjectHook {
		return new(customerservicehooks.NoopSubjectCustomerHook), nil
	}

	// Initialize the subject customer hook and register it for Subject service
	h, err := customerservicehooks.NewSubjectCustomerHook(customerservicehooks.SubjectCustomerHookConfig{
		Customer:     customerService,
		Logger:       logger,
		Tracer:       tracer,
		IgnoreErrors: config.IgnoreErrors,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer subject hook: %w", err)
	}

	subjectService.RegisterHooks(h)

	return h, nil
}

type CustomerSubjectValidatorHook customerservicehooks.SubjectValidatorHook

func NewCustomerSubjectValidatorServiceHook(
	logger *slog.Logger,
	subjectService subject.Service,
	customerService customer.Service,
) (CustomerSubjectValidatorHook, error) {
	// Initialize the customer subject validator hook and register it for Subject service
	h, err := customerservicehooks.NewSubjectValidatorHook(customerservicehooks.SubjectValidatorHookConfig{
		Customer: customerService,
		Logger:   logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer subject validator hook: %w", err)
	}

	subjectService.RegisterHooks(h)

	return h, nil
}

type CustomerEntitlementValidatorHook customerservicehooks.EntitlementValidatorHook

func NewCustomerEntitlementValidatorServiceHook(
	logger *slog.Logger,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
) (CustomerEntitlementValidatorHook, error) {
	h, err := customerservicehooks.NewEntitlementValidatorHook(customerservicehooks.EntitlementValidatorHookConfig{
		EntitlementService: entitlementRegistry.Entitlement,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer entitlement validator hook: %w", err)
	}

	customerService.RegisterHooks(h)

	return h, nil
}
