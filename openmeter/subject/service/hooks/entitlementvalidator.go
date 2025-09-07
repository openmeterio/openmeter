package hooks

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	EntitlementValidatorHook     = models.ServiceHook[subject.Subject]
	NoopEntitlementValidatorHook = models.NoopServiceHook[subject.Subject]
)

var _ models.ServiceHook[subject.Subject] = (*entitlementValidatorHook)(nil)

type entitlementValidatorHook struct {
	NoopEntitlementValidatorHook

	entitlementService entitlement.Connector
}

type EntitlementValidatorHookConfig struct {
	EntitlementService entitlement.Connector
}

func (e EntitlementValidatorHookConfig) Validate() error {
	if e.EntitlementService == nil {
		return fmt.Errorf("entitlement service is required")
	}

	return nil
}

func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid entitlement validator hook config: %w", err)
	}

	return &entitlementValidatorHook{
		entitlementService: config.EntitlementService,
	}, nil
}

func (e *entitlementValidatorHook) PreDelete(ctx context.Context, subject *subject.Subject) error {
	entitlements, err := e.entitlementService.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:  []string{subject.Namespace},
		SubjectKeys: []string{subject.Key},
		Page: pagination.Page{
			PageSize:   1,
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get entitlements of customer: %w", err)
	}

	if entitlements.TotalCount > 0 {
		return models.NewGenericValidationError(fmt.Errorf("subject has entitlements, please remove them before deleting the subject"))
	}

	return nil
}
