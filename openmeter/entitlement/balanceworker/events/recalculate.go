package events

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	EventSubsystem       metadata.EventSubsystem = "balanceWorker"
	RecalculateEventName metadata.EventName      = "triggerEntitlementRecalculation"
)

type OperationType string

const (
	OperationTypeEntitlementCreated      OperationType = "entitlement_created"
	OperationTypeEntitlementDeleted      OperationType = "entitlement_deleted"
	OperationTypeGrantCreated            OperationType = "grant_created"
	OperationTypeGrantDeleted            OperationType = "grant_deleted"
	OperationTypeGrantVoided             OperationType = "grant_voided"
	OperationTypeMeteredEntitlementReset OperationType = "metered_entitlement_reset"
	OperationTypeIngest                  OperationType = "ingest"
	OperationTypeRecalculate             OperationType = "recalculate"
)

func (o OperationType) Values() []OperationType {
	return []OperationType{
		OperationTypeEntitlementCreated,
		OperationTypeEntitlementDeleted,
		OperationTypeGrantCreated,
		OperationTypeGrantDeleted,
		OperationTypeGrantVoided,
		OperationTypeMeteredEntitlementReset,
		OperationTypeIngest,
		OperationTypeRecalculate,
	}
}

func (o OperationType) Validate() error {
	if !slices.Contains(o.Values(), o) {
		return fmt.Errorf("invalid operation type: %s", o)
	}
	return nil
}

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
	SourceOperation     OperationType                        `json:"sourceOperation"`
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

	if err := e.SourceOperation.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("sourceOperation: %w", err))
	}

	return errors.Join(errs...)
}
