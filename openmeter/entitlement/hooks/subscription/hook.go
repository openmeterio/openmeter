package entitlementsubscriptionhook

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type hook struct {
	NoopEntitlementSubscriptionHook
}

type EntitlementSubscriptionHookConfig struct{}

type (
	EntitlementSubscriptionHook     = models.ServiceHook[entitlement.Entitlement]
	NoopEntitlementSubscriptionHook = models.NoopServiceHook[entitlement.Entitlement]
)

var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)

func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook {
	return &hook{
		NoopEntitlementSubscriptionHook: NoopEntitlementSubscriptionHook{},
	}
}

// The methods of entitlement.Service are not conventionally named, so check where this is used
func (h *hook) PreDelete(ctx context.Context, ent *entitlement.Entitlement) error {
	if subscription.IsSubscriptionOperation(ctx) {
		return nil
	}

	if subscription.AnnotationParser.HasSubscription(ent.Annotations) {
		return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))
	}
	return nil
}

// The methods of entitlement.Service are not conventionally named, so check where this is used
func (h *hook) PreUpdate(ctx context.Context, ent *entitlement.Entitlement) error {
	if subscription.IsSubscriptionOperation(ctx) {
		return nil
	}

	if subscription.AnnotationParser.HasSubscription(ent.Annotations) {
		return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))
	}
	return nil
}
