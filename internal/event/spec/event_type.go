package spec

// TODO: move to metadata
import (
	"fmt"
	"time"
)

type (
	EventSubsystem   string
	EventName        string
	EventVersion     string
	EventSubjectKind string
)

type EventTypeSpec struct {
	// Subsystem defines which connector/component is responsible for the event (e.g. ingest, entitlements, etc)
	Subsystem EventSubsystem

	// Type is the actual event type (e.g. ingestion, flush, etc)
	Name EventName

	// Version is the version of the event (e.g. v1, v2, etc)
	Version EventVersion
}

func (s *EventTypeSpec) EventName() string {
	return fmt.Sprintf("io.openmeter.%s.%s.%s", s.Subsystem, s.Version, s.Name)
}

func (s *EventTypeSpec) VersionSubsystem() string {
	return fmt.Sprintf("io.openmeter.%s.%s", s.Subsystem, s.Version)
}

func GetEventName(spec EventTypeSpec) string {
	return spec.EventName()
}

type EventMetadata struct {
	// ID of the event
	ID string

	// Time specifies when the event occurred
	Time time.Time

	// Subject meta
	// Examples for source and subject pairs
	//  grant:
	//      source: //openmeter.io/namespace/<id>/entitlement/<id>/grant/<id>
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	//
	//  entitlement:
	//      source: //openmeter.io/namespace/<id>/entitlement/<id>
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	//
	//  ingest:
	//      source: //openmeter.io/namespace/<id>/event
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	Subject string
	Source  string
}
