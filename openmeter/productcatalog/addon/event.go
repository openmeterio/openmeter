package addon

import (
	"context"
	"errors"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	AddonEventSubsystem   metadata.EventSubsystem = "addon"
	AddonCreateEventName  metadata.EventName      = "addon.created"
	AddonUpdateEventName  metadata.EventName      = "addon.updated"
	AddonDeleteEventName  metadata.EventName      = "addon.deleted"
	AddonPublishEventName metadata.EventName      = "addon.published"
	AddonArchiveEventName metadata.EventName      = "addon.archived"
)

// NewAddonCreateEvent creates a new Addon create event
func NewAddonCreateEvent(ctx context.Context, addon *Addon) AddonCreateEvent {
	return AddonCreateEvent{
		Addon:  addon,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AddonCreateEvent is an event that is emitted when an Addon is created
type AddonCreateEvent struct {
	Addon  *Addon  `json:"addon"`
	UserID *string `json:"userId,omitempty"`
}

func (e AddonCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AddonEventSubsystem,
		Name:      AddonCreateEventName,
		Version:   "v1",
	})
}

func (e AddonCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Addon.CreatedAt,
	}
}

func (e AddonCreateEvent) Validate() error {
	var errs []error

	if e.Addon == nil {
		errs = append(errs, errors.New("add-on is required"))
	}

	return errors.Join(errs...)
}

// NewAddonUpdateEvent creates a new Addon update event
func NewAddonUpdateEvent(ctx context.Context, Addon *Addon) AddonUpdateEvent {
	return AddonUpdateEvent{
		Addon:  Addon,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AddonUpdateEvent is an event that is emitted when an Addon is updated
type AddonUpdateEvent struct {
	Addon  *Addon  `json:"addon"`
	UserID *string `json:"userId,omitempty"`
}

func (e AddonUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AddonEventSubsystem,
		Name:      AddonUpdateEventName,
		Version:   "v1",
	})
}

func (e AddonUpdateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Addon.UpdatedAt,
	}
}

func (e AddonUpdateEvent) Validate() error {
	var errs []error

	if e.Addon == nil {
		errs = append(errs, errors.New("add-on is required"))
	}

	return errors.Join(errs...)
}

// NewAddonDeleteEvent creates a new Addon delete event
func NewAddonDeleteEvent(ctx context.Context, Addon *Addon) AddonDeleteEvent {
	return AddonDeleteEvent{
		Addon:  Addon,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AddonDeleteEvent is an event that is emitted when an Addon is deleted
type AddonDeleteEvent struct {
	Addon  *Addon  `json:"addon"`
	UserID *string `json:"userId,omitempty"`
}

func (e AddonDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AddonEventSubsystem,
		Name:      AddonDeleteEventName,
		Version:   "v1",
	})
}

func (e AddonDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    lo.FromPtr(e.Addon.DeletedAt),
	}
}

func (e AddonDeleteEvent) Validate() error {
	var errs []error

	if e.Addon == nil {
		errs = append(errs, errors.New("add-on is required"))
	}

	if e.Addon.DeletedAt == nil {
		errs = append(errs, errors.New("add-on deleted at is required"))
	}

	return errors.Join(errs...)
}

// NewAddonPublishEvent creates a new Addon publish event
func NewAddonPublishEvent(ctx context.Context, Addon *Addon) AddonPublishEvent {
	return AddonPublishEvent{
		Addon:  Addon,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AddonPublishEvent is an event that is emitted when an Addon is published
type AddonPublishEvent struct {
	Addon  *Addon  `json:"addon"`
	UserID *string `json:"userId,omitempty"`
}

func (e AddonPublishEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AddonEventSubsystem,
		Name:      AddonPublishEventName,
		Version:   "v1",
	})
}

func (e AddonPublishEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Addon.UpdatedAt,
	}
}

func (e AddonPublishEvent) Validate() error {
	var errs []error

	if e.Addon == nil {
		errs = append(errs, errors.New("add-on is required"))
	}

	return errors.Join(errs...)
}

// NewAddonArchiveEvent creates a new Addon archive event
func NewAddonArchiveEvent(ctx context.Context, Addon *Addon) AddonArchiveEvent {
	return AddonArchiveEvent{
		Addon:  Addon,
		UserID: session.GetSessionUserID(ctx),
	}
}

// AddonArchiveEvent is an event that is emitted when an Addon is archived
type AddonArchiveEvent struct {
	Addon  *Addon  `json:"addon"`
	UserID *string `json:"userId,omitempty"`
}

func (e AddonArchiveEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AddonEventSubsystem,
		Name:      AddonArchiveEventName,
		Version:   "v1",
	})
}

func (e AddonArchiveEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Addon.UpdatedAt,
	}
}

func (e AddonArchiveEvent) Validate() error {
	var errs []error

	if e.Addon == nil {
		errs = append(errs, errors.New("add-on is required"))
	}

	return errors.Join(errs...)
}
