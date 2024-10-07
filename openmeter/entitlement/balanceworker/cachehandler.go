package balanceworker

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/edge"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (w *Worker) handleCacheMissEvent(ctx context.Context, event *edge.EntitlementCacheMissEvent, source string) (marshaler.Event, error) {
	currentTime := clock.Now()

	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	ent, err := w.entitlement.Entitlement.GetEntitlement(ctx, event.EntitlementNamespace, event.EntitlementIdOrFeatureKey)
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = w.entitlement.EntitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, event.EntitlementNamespace, event.SubjectKey, event.EntitlementIdOrFeatureKey, event.At)
	}

	if err != nil {
		return nil, err
	}

	return w.processEntitlementEntity(ctx, ent, currentTime, source)
}
