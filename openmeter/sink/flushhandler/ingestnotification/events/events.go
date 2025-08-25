package events

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "ingest"
)

type EventBatchedIngest struct {
	Namespace  models.NamespaceID `json:"namespace"`
	SubjectKey string             `json:"subjectKey"`
	// We could add the customerID here, could make sense if something apart from the balanceworker is interested in the customerID
	// CustomerID string `json:"customerID"`

	// MeterSlugs contain the list of slugs that are affected by the event. We
	// should not use meterIDs as they are not something present in the open source
	// version, thus any code that is in opensource should not rely on them.
	MeterSlugs []string `json:"meterSlugs"`

	RawEvents []serializer.CloudEventsKafkaPayload `json:"rawEvents"`
	StoredAt  time.Time                            `json:"storedAt"`
}

var (
	_ marshaler.Event = EventBatchedIngest{}

	batchIngestEventType = metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "events.ingested",
		Version:   "v2",
	}
	batchIngestEventName  = metadata.GetEventName(batchIngestEventType)
	EventVersionSubsystem = batchIngestEventType.VersionSubsystem()
)

func (b EventBatchedIngest) EventName() string {
	return batchIngestEventName
}

func (b EventBatchedIngest) Validate() error {
	if err := b.Namespace.Validate(); err != nil {
		return err
	}

	if b.SubjectKey == "" {
		return errors.New("subjectKey must be set")
	}

	return nil
}

func (b EventBatchedIngest) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePathRaw(string(EventSubsystem)),
		Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey),
	}
}
