package balanceworker

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement/edge"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (w *Worker) handleCacheMissEvent(ctx context.Context, event *edge.EntitlementCacheMissEvent, source string) (marshaler.Event, error) {
	currentTime := clock.Now()

	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	ent, err := w.entitlement.EntitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, event.EntitlementNamespace, event.SubjectKey, event.EntitlementIdOrFeatureKey, currentTime)
	if err != nil {
		return nil, err
	}

	return w.processEntitlementEntity(ctx, ent, currentTime, source)
}
