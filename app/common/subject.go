package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subject/adapter"
	"github.com/openmeterio/openmeter/openmeter/subject/service"
	subjecthooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
)

var Subject = wire.NewSet(
	NewSubjectService,
	NewSubjectAdapter,
)

func NewSubjectService(
	adapter subject.Adapter,
) (subject.Service, error) {
	return service.New(adapter)
}

func NewSubjectAdapter(
	db *entdb.Client,
) (subject.Adapter, error) {
	return adapter.New(db)
}

func NewSubjectCustomerHook(
	subject subject.Service,
	customer customer.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) (subjecthooks.CustomerSubjectHook, error) {
	h, err := subjecthooks.NewCustomerSubjectHook(subjecthooks.CustomerSubjectHookConfig{
		Subject: subject,
		Logger:  logger,
		Tracer:  tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer subject hook: %w", err)
	}

	customer.RegisterHooks(h)

	return h, nil
}

func NewSubjectEntitlementValidatorHook(
	logger *slog.Logger,
	entitlementRegistry *registry.Entitlement,
	subjectService subject.Service,
) (subjecthooks.EntitlementValidatorHook, error) {
	h, err := subjecthooks.NewEntitlementValidatorHook(subjecthooks.EntitlementValidatorHookConfig{
		EntitlementService: entitlementRegistry.Entitlement,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subject entitlement validator hook: %w", err)
	}

	subjectService.RegisterHooks(h)

	return h, nil
}
