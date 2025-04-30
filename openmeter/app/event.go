package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

// EventAppParser should be implemented by the app's meta contents to be parsable from an EventApp
type EventAppParser interface {
	FromEventAppData(EventApp) error
}

type EventAppData map[string]any

// NewEventAppData creates a new EventAppData from a given value
// TODO[later]: we need to refactor apps to be able to handle serialization more gracefully, e.g. having a proper
// union type for app instead of the interface
func NewEventAppData(v any) (EventAppData, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var data EventAppData
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// ParseInto parses the EventAppData into a given value, the value must be a pointer
func (e EventAppData) ParseInto(v any) error {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return err
	}

	return nil
}

type EventApp struct {
	AppBase
	AppData EventAppData `json:"appData"`
}

func NewEventApp(app App) (EventApp, error) {
	appBase := app.GetAppBase()

	appData, err := app.GetEventAppData()
	if err != nil {
		return EventApp{}, err
	}

	return EventApp{
		AppBase: appBase,
		AppData: appData,
	}, nil
}

const (
	AppEventSubsystem  metadata.EventSubsystem = "app"
	AppCreateEventName metadata.EventName      = "app.created"
	AppUpdateEventName metadata.EventName      = "app.updated"
	AppDeleteEventName metadata.EventName      = "app.deleted"
)

// NewAppCreateEvent creates a new app create event
// TODO[later]: We should use eventApp instead of AppBase, but the creation flow is somewhat tricky to change as the flow
// is that the app calls the AppCreate without having the configuration presisted.
func NewAppCreateEvent(ctx context.Context, appBase AppBase) AppCreateEvent {
	return AppCreateEvent{
		AppBase: appBase,
		UserID:  session.GetSessionUserID(ctx),
	}
}

// AppCreateEvent is an event that is emitted when an app is created
type AppCreateEvent struct {
	AppBase
	UserID *string `json:"userId,omitempty"`
}

func (e AppCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppCreateEventName,
		Version:   "v1",
	})
}

func (e AppCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.AppBase.Namespace, metadata.EntityApp, e.AppBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.AppBase.CreatedAt,
	}
}

func (e AppCreateEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}
	return nil
}

// NewAppUpdateEvent creates a new app update event
func NewAppUpdateEvent(ctx context.Context, app App) (AppUpdateEvent, error) {
	eventApp, err := NewEventApp(app)
	if err != nil {
		return AppUpdateEvent{}, err
	}

	return AppUpdateEvent{
		EventApp: eventApp,
		UserID:   session.GetSessionUserID(ctx),
	}, nil
}

// AppUpdateEvent is an event that is emitted when an app is updated
type AppUpdateEvent struct {
	EventApp
	UserID *string `json:"userId,omitempty"`
}

func (e AppUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppUpdateEventName,
		Version:   "v2",
	})
}

func (e AppUpdateEvent) EventMetadata() metadata.EventMetadata {
	appBase := e.AppBase.GetAppBase()
	resourcePath := metadata.ComposeResourcePath(appBase.Namespace, metadata.EntityApp, appBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    appBase.UpdatedAt,
	}
}

func (e AppUpdateEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}

	return nil
}

// NewAppDeleteEvent creates a new app delete event
func NewAppDeleteEvent(ctx context.Context, app AppBase, appData EventAppData) AppDeleteEvent {
	return AppDeleteEvent{
		EventApp: EventApp{
			AppBase: app,
			AppData: appData,
		},
		UserID: session.GetSessionUserID(ctx),
	}
}

// AppDeleteEvent is an event that is emitted when an app is deleted
type AppDeleteEvent struct {
	EventApp
	UserID *string `json:"userId,omitempty"`
}

func (e AppDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppDeleteEventName,
		Version:   "v2",
	})
}

func (e AppDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.AppBase.Namespace, metadata.EntityApp, e.AppBase.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.AppBase.DeletedAt,
	}
}

func (e AppDeleteEvent) Validate() error {
	if e.AppBase.ID == "" {
		return fmt.Errorf("app base is required")
	}

	return nil
}
