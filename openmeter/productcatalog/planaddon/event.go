package planaddon

import (
	"context"
	"errors"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	PlanAddonEventSubsystem  metadata.EventSubsystem = "planaddon"
	PlanAddonCreateEventName metadata.EventName      = "planaddon.created"
	PlanAddonUpdateEventName metadata.EventName      = "planaddon.updated"
	PlanAddonDeleteEventName metadata.EventName      = "planaddon.deleted"
)

// NewPlanAddonCreateEvent creates a new PlanAddon create event
func NewPlanAddonCreateEvent(ctx context.Context, planAddon *PlanAddon) PlanAddonCreateEvent {
	return PlanAddonCreateEvent{
		PlanAddon: planAddon,
		UserID:    session.GetSessionUserID(ctx),
	}
}

// PlanAddonCreateEvent is an event that is emitted when an PlanAddon is created
type PlanAddonCreateEvent struct {
	PlanAddon *PlanAddon `json:"planAddon"`
	UserID    *string    `json:"userId,omitempty"`
}

func (e PlanAddonCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanAddonEventSubsystem,
		Name:      PlanAddonCreateEventName,
		Version:   "v1",
	})
}

func (e PlanAddonCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.PlanAddon.Namespace, metadata.EntityAddon, e.PlanAddon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.PlanAddon.CreatedAt,
	}
}

func (e PlanAddonCreateEvent) Validate() error {
	var errs []error

	if e.PlanAddon == nil {
		errs = append(errs, errors.New("plan add-on assignment is required"))
	}

	return errors.Join(errs...)
}

// NewPlanAddonUpdateEvent creates a new PlanAddon update event
func NewPlanAddonUpdateEvent(ctx context.Context, addon *PlanAddon) PlanAddonUpdateEvent {
	return PlanAddonUpdateEvent{
		PlanAddon: addon,
		UserID:    session.GetSessionUserID(ctx),
	}
}

// PlanAddonUpdateEvent is an event that is emitted when an PlanAddon is updated
type PlanAddonUpdateEvent struct {
	PlanAddon *PlanAddon `json:"planAddon"`
	UserID    *string    `json:"userId,omitempty"`
}

func (e PlanAddonUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanAddonEventSubsystem,
		Name:      PlanAddonUpdateEventName,
		Version:   "v1",
	})
}

func (e PlanAddonUpdateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.PlanAddon.Namespace, metadata.EntityAddon, e.PlanAddon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.PlanAddon.UpdatedAt,
	}
}

func (e PlanAddonUpdateEvent) Validate() error {
	var errs []error

	if e.PlanAddon == nil {
		errs = append(errs, errors.New("plan add-on assignment is required"))
	}

	return errors.Join(errs...)
}

// NewPlanAddonDeleteEvent creates a new PlanAddon delete event
func NewPlanAddonDeleteEvent(ctx context.Context, addon *PlanAddon) PlanAddonDeleteEvent {
	return PlanAddonDeleteEvent{
		PlanAddon: addon,
		UserID:    session.GetSessionUserID(ctx),
	}
}

// PlanAddonDeleteEvent is an event that is emitted when an PlanAddon is deleted
type PlanAddonDeleteEvent struct {
	PlanAddon *PlanAddon `json:"planAddon"`
	UserID    *string    `json:"userId,omitempty"`
}

func (e PlanAddonDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: PlanAddonEventSubsystem,
		Name:      PlanAddonDeleteEventName,
		Version:   "v1",
	})
}

func (e PlanAddonDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.PlanAddon.Namespace, metadata.EntityAddon, e.PlanAddon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    lo.FromPtr(e.PlanAddon.DeletedAt),
	}
}

func (e PlanAddonDeleteEvent) Validate() error {
	var errs []error

	if e.PlanAddon == nil {
		errs = append(errs, errors.New("plan add-on assignment is required"))
	}

	if e.PlanAddon != nil && e.PlanAddon.DeletedAt == nil {
		errs = append(errs, errors.New(`"deleted at" attribute for plan add-on assignment is required`))
	}

	return errors.Join(errs...)
}
