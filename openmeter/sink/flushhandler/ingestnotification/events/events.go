package events

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "ingest"
)

type EventBatchedIngest struct {
	Namespace  models.NamespaceID `json:"namespace"`
	SubjectKey string             `json:"subjectKey"`

	// MeterSlugs contain the list of slugs that are affected by the event. We
	// should not use meterIDs as they are not something present in the open source
	// version, thus any code that is in opensource should not rely on them.
	MeterSlugs []string `json:"meterSlugs"`
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

	if len(b.MeterSlugs) == 0 {
		return errors.New("meterSlugs must not be empty")
	}

	return nil
}

func (b EventBatchedIngest) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePathRaw(string(EventSubsystem)),
		Subject: metadata.ComposeResourcePath(b.Namespace.ID, metadata.EntitySubjectKey, b.SubjectKey),
	}
}
