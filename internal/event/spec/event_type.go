package spec

import (
	"fmt"
	"time"
)

type (
	EventSubsystem   string
	EventName        string
	EventVersion     string
	EventSubjectKind string
	EventSpecVersion string
)

type EventTypeSpec struct {
	// Subsystem defines which connector/component is responsible for the event (e.g. ingest, entitlements, etc)
	Subsystem EventSubsystem

	// Type is the actual event type (e.g. ingestion, flush, etc)
	Name EventName

	// Version is the version of the event (e.g. v1, v2, etc)
	Version EventVersion

	// SpecVersion is the version of the event spec (e.g. 1.0, 1.1, etc)
	SpecVersion EventSpecVersion

	// cloudEventType is the actual cloud event type, so that we don't have the calculate it
	// for each message
	cloudEventType string
}

func (s *EventTypeSpec) Type() string {
	if s.cloudEventType != "" {
		return s.cloudEventType
	}
	s.cloudEventType = fmt.Sprintf("openmeter.%s.%s.%s", s.Subsystem, s.Version, s.Name)
	return s.cloudEventType
}

type EventSpec struct {
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
