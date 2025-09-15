package meteredentitlement

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type hook struct {
	EntitlementHook models.ServiceHook[entitlement.Entitlement]
}

var _ models.ServiceHook[Entitlement] = (*hook)(nil)

func ConvertHook(h models.ServiceHook[entitlement.Entitlement]) models.ServiceHook[Entitlement] {
	return &hook{
		EntitlementHook: h,
	}
}

func (h *hook) PreDelete(ctx context.Context, ent *Entitlement) error {
	return h.EntitlementHook.PreDelete(ctx, ent.ToGenericEntitlement())
}

func (h *hook) PreUpdate(ctx context.Context, ent *Entitlement) error {
	return h.EntitlementHook.PreUpdate(ctx, ent.ToGenericEntitlement())
}

func (h *hook) PostCreate(ctx context.Context, ent *Entitlement) error {
	return h.EntitlementHook.PostCreate(ctx, ent.ToGenericEntitlement())
}

func (h *hook) PostUpdate(ctx context.Context, ent *Entitlement) error {
	return h.EntitlementHook.PostUpdate(ctx, ent.ToGenericEntitlement())
}

func (h *hook) PostDelete(ctx context.Context, ent *Entitlement) error {
	return h.EntitlementHook.PostDelete(ctx, ent.ToGenericEntitlement())
}
