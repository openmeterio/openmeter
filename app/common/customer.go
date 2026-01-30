package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	customerservicehooks "github.com/openmeterio/openmeter/openmeter/customer/service/hooks"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
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

	return service, nil
}

type CustomerSubjectHook customerservicehooks.SubjectCustomerHook

func NewCustomerSubjectServiceHook(
	config config.CustomerConfiguration,
	logger *slog.Logger,
	tracer trace.Tracer,
	subjectService subject.Service,
	customerService customer.Service,
	customerOverrideService billing.CustomerOverrideService,
) (CustomerSubjectHook, error) {
	if !config.EnableSubjectHook {
		return new(customerservicehooks.NoopSubjectCustomerHook), nil
	}

	// Initialize the subject customer hook and register it for Subject service
	h, err := customerservicehooks.NewSubjectCustomerHook(customerservicehooks.SubjectCustomerHookConfig{
		Customer:         customerService,
		CustomerOverride: customerOverrideService,
		Logger:           logger,
		Tracer:           tracer,
		IgnoreErrors:     config.IgnoreErrors,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer subject hook: %w", err)
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
