package meter

import (
	"context"
	"errors"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	MeterEventSubsystem  metadata.EventSubsystem = "meter"
	MeterCreateEventName metadata.EventName      = "meter.created"
	MeterUpdateEventName metadata.EventName      = "meter.updated"
	MeterDeleteEventName metadata.EventName      = "meter.deleted"
)

// NewMeterCreateEvent creates a new meter create event
func NewMeterCreateEvent(ctx context.Context, meter *Meter) MeterCreateEvent {
	return MeterCreateEvent{
		Meter:  meter,
		UserID: session.GetSessionUserID(ctx),
	}
}

// MeterCreateEvent is an event that is emitted when a meter is created
type MeterCreateEvent struct {
	Meter  *Meter  `json:"meter"`
	UserID *string `json:"userId,omitempty"`
}

func (e MeterCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: MeterEventSubsystem,
		Name:      MeterCreateEventName,
		Version:   "v1",
	})
}

func (e MeterCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Meter.ManagedResource.Namespace, metadata.EntityMeter, e.Meter.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Meter.CreatedAt,
	}
}

func (e MeterCreateEvent) Validate() error {
	var errs []error

	if e.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	return errors.Join(errs...)
}

// NewMeterUpdateEvent creates a new meter update event
func NewMeterUpdateEvent(ctx context.Context, meter *Meter) MeterUpdateEvent {
	return MeterUpdateEvent{
		Meter:  meter,
		UserID: session.GetSessionUserID(ctx),
	}
}

// MeterUpdateEvent is an event that is emitted when a meter is updated
type MeterUpdateEvent struct {
	Meter  *Meter  `json:"meter"`
	UserID *string `json:"userId,omitempty"`
}

func (e MeterUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: MeterEventSubsystem,
		Name:      MeterUpdateEventName,
		Version:   "v1",
	})
}

func (e MeterUpdateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Meter.ManagedResource.Namespace, metadata.EntityMeter, e.Meter.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Meter.UpdatedAt,
	}
}

func (e MeterUpdateEvent) Validate() error {
	var errs []error

	if e.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	return errors.Join(errs...)
}

// NewMeterDeleteEvent creates a new meter delete event
func NewMeterDeleteEvent(ctx context.Context, meter *Meter) MeterDeleteEvent {
	return MeterDeleteEvent{
		Meter:  meter,
		UserID: session.GetSessionUserID(ctx),
	}
}

// MeterDeleteEvent is an event that is emitted when a meter is deleted
type MeterDeleteEvent struct {
	Meter  *Meter  `json:"meter"`
	UserID *string `json:"userId,omitempty"`
}

func (e MeterDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: MeterEventSubsystem,
		Name:      MeterDeleteEventName,
		Version:   "v1",
	})
}

func (e MeterDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Meter.ManagedResource.Namespace, metadata.EntityMeter, e.Meter.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.Meter.DeletedAt,
	}
}

func (e MeterDeleteEvent) Validate() error {
	var errs []error

	if e.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	if e.Meter.DeletedAt == nil {
		errs = append(errs, errors.New("meter deleted at is required"))
	}

	return errors.Join(errs...)
}
