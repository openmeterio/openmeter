package types

import (
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
)

type EventTypeSpec struct {
	// Subsystem defines which connector/component is responsible for the event (e.g. ingest, entitlements, etc)
	Subsystem string

	// Type is the actual event type (e.g. ingestion, flush, etc)
	Name string

	// Version is the version of the event (e.g. v1, v2, etc)
	Version string

	// SpecVersion is the version of the event spec (e.g. 1.0, 1.1, etc)
	SpecVersion string

	// SubjectKind specifies the kind of the subject. Used to construct the subject field in /namespace/<id>/kind/<entryID>
	SubjectKind string

	// cloudEventType is the actual cloud event type, so that we don't have the calculate it
	// for each message
	cloudEventType string
}

func (s *EventTypeSpec) FillEvent(ev event.Event) {
	ev.SetType(s.Type())
	ev.SetSpecVersion(s.SpecVersion)
}

func (s *EventTypeSpec) Type() string {
	if s.cloudEventType != "" {
		return s.cloudEventType
	}
	s.cloudEventType = fmt.Sprintf("openmeter.%s.%s.%s", s.Subsystem, s.Version, s.Name)
	return s.cloudEventType
}

type EventSpec struct {
	// Source of the event
	Source string

	// ID of the event
	ID string

	// Time specifies when the event occurred
	Time time.Time

	// Subject meta (optional, references the entity/subject of the event)
	Namespace string
	SubjectID string
}

func (c *EventSpec) FillEvent(ev event.Event, subjectKind string) {
	if c.Time.IsZero() {
		ev.SetTime(time.Now())
	} else {
		ev.SetTime(c.Time)
	}

	// Subject is optional and can be empty
	if c.SubjectID != "" && c.Namespace != "" {
		ev.SetSubject(fmt.Sprintf("/namespace/%s/%s/%s", c.Namespace, subjectKind, c.SubjectID))
	}

	ev.SetSource(c.Source)
	ev.SetID(c.ID)
}
