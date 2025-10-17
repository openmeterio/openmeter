package balanceworker

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	BalanceWorkerEventSubsystem       metadata.EventSubsystem = "balanceWorker"
	BalanceWorkerRecalculateEventName metadata.EventName      = "triggerEntitlementRecalculation"
)

type RecalculateEvent struct {
	Entitlement entitlement.Entitlement `json:"entitlement"`
	AsOf        time.Time               `json:"asOf"`
}

func (e RecalculateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: BalanceWorkerEventSubsystem,
		Name:      BalanceWorkerRecalculateEventName,
		Version:   "v1",
	})
}

func (e RecalculateEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntityEntitlement, e.Entitlement.ID),
		Subject: metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntitySubjectKey, e.Entitlement.SubjectKey),
	}
}

func (e RecalculateEvent) Validate() error {
	var errs []error

	if e.AsOf.IsZero() {
		errs = append(errs, errors.New("asOf is required"))
	}

	if err := e.Entitlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("entitlement: %w", err))
	}

	return errors.Join(errs...)
}
