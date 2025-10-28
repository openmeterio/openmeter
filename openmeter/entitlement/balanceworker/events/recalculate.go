package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	EventSubsystem       metadata.EventSubsystem = "balanceWorker"
	RecalculateEventName metadata.EventName      = "triggerEntitlementRecalculation"
)

var (
	_ marshaler.Event = RecalculateEvent{}

	recalculateEventType = metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      RecalculateEventName,
		Version:   "v1",
	}
	recalculateEventName  = metadata.GetEventName(recalculateEventType)
	EventVersionSubsystem = recalculateEventType.VersionSubsystem()
)

type RecalculateEventEntitlement struct {
	models.NamespacedID

	SubjectKey string `json:"subjectKey"`
}

func (e RecalculateEventEntitlement) Validate() error {
	var errs []error

	if err := e.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced id: %w", err))
	}

	if e.SubjectKey == "" {
		errs = append(errs, errors.New("subject key is required"))
	}

	return errors.Join(errs...)
}

type RecalculateEvent struct {
	Entitlement RecalculateEventEntitlement `json:"entitlement"`
	AsOf        time.Time                   `json:"asOf"`
}

func (e RecalculateEvent) EventName() string {
	return recalculateEventName
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
