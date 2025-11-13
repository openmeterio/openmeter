package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
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
		Version:   "v2",
	}
	recalculateEventName  = metadata.GetEventName(recalculateEventType)
	EventVersionSubsystem = recalculateEventType.VersionSubsystem()
)

type RecalculateEvent struct {
	Entitlement         models.NamespacedID                  `json:"entitlement"`
	AsOf                time.Time                            `json:"asOf"`
	OriginalEventSource string                               `json:"originalEventSource"`
	SourceOperation     snapshot.ValueOperationType          `json:"sourceOperation"`
	RawIngestedEvents   []serializer.CloudEventsKafkaPayload `json:"rawIngestedEvents"`
}

func (e RecalculateEvent) EventName() string {
	return recalculateEventName
}

func (e RecalculateEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  e.OriginalEventSource,
		Subject: metadata.ComposeResourcePath(e.Entitlement.Namespace, metadata.EntityEntitlement, e.Entitlement.ID),
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
