package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Provider owns external Namespace and Channel state and asynchronous delivery. Implementations
// must be safe for concurrent use, honor context deadlines, and make mutations
// idempotent. Notification owns persistence and reconciliation scheduling.
//
// ProviderData maps contain internal operational metadata. Each key must use the
// owning Provider's prefix, such as "svix.message_id". Results replace that
// Provider's complete subset; Notification preserves other Providers' keys.
// Values must not contain secrets.
type Provider[C v4.ChannelConfig, U v4.ChannelConfigUpdate] interface {
	// RoutingMode describes how destinations are selected for initial sends,
	// not targeted resends. It is fixed for the lifetime of the provider instance.
	RoutingMode() ProviderRoutingMode

	// GetNamespace retrieves the provider-side namespace by its Notification ID.
	GetNamespace(context.Context, GetNamespaceInput) (*GetNamespaceResult, error)

	// CreateNamespace provisions a provider-side namespace using the supplied ID,
	// name, and metadata. Retrying must not create a duplicate namespace.
	CreateNamespace(context.Context, CreateNamespaceInput) (*CreateNamespaceResult, error)

	// UpdateNamespace applies only supplied fields to an existing namespace.
	// Provided metadata replaces the complete map; omitted metadata is preserved.
	UpdateNamespace(context.Context, UpdateNamespaceInput) (*UpdateNamespaceResult, error)

	// DeleteNamespace removes the provider-side namespace. Repeated deletion of
	// an already absent namespace must succeed.
	DeleteNamespace(context.Context, DeleteNamespaceInput) error

	// GetChannel retrieves provider-side Channel operation state and operational
	// metadata without mutating the remote resource.
	GetChannel(context.Context, GetChannelInput) (*GetChannelResult, error)

	// CreateChannel provisions the desired provider-side Channel idempotently.
	// The result indicates whether provisioning is complete or still pending.
	CreateChannel(context.Context, CreateChannelInput[C]) (*CreateChannelResult, error)

	// UpdateChannel applies only supplied fields to an existing Channel.
	// Success confirms the supplied changes, not the state of omitted fields.
	UpdateChannel(context.Context, UpdateChannelInput[U]) (*UpdateChannelResult, error)

	// DeleteChannel removes the provider-side Channel. Repeated deletion of an
	// already absent Channel must succeed. Notification owns its local soft deletion.
	DeleteChannel(context.Context, DeleteChannelInput) error

	// GetEvent retrieves delivery state and attempts for the requested destinations
	// without initiating another send. Notification persists the returned observations.
	GetEvent(context.Context, GetEventInput) (*GetEventResult, error)

	// SendEvent submits the persisted Event for asynchronous delivery, using its ID
	// as the external identity where supported.
	// A nil destination Error means acceptance, not successful delivery.
	SendEvent(context.Context, SendEventInput) (*SendEventResult, error)

	// ResendEvent resubmits the same Event for the supplied dispatch generations.
	// As with SendEvent, successful submission indicates acceptance only.
	ResendEvent(context.Context, ResendEventInput) (*ResendEventResult, error)
}

type ProviderType string

type ProviderRoutingMode string

const (
	ProviderRoutingModeExplicit  ProviderRoutingMode = "explicit"
	ProviderRoutingModeEventType ProviderRoutingMode = "event_type"
)

type Namespace struct {
	ID           string
	Name         string
	Metadata     models.Metadata
	ProviderData map[string]string
}

var _ models.Validator = Namespace{}

func (n Namespace) Validate() error {
	var errs []error

	if err := (GetNamespaceInput{ID: n.ID}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if strings.TrimSpace(n.Name) == "" {
		errs = append(errs, errors.New("namespace name is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetNamespaceInput struct {
	ID string
}

var _ models.Validator = GetNamespaceInput{}

func (i GetNamespaceInput) Validate() error {
	var errs []error

	if strings.TrimSpace(i.ID) == "" {
		errs = append(errs, errors.New("namespace ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateNamespaceInput struct {
	Namespace

	// IdempotencyKey identifies the same logical creation across retries.
	// Nil selects the provider default; a supplied key must be nonblank.
	IdempotencyKey *string
}

var _ models.Validator = CreateNamespaceInput{}

func (i CreateNamespaceInput) Validate() error {
	var errs []error

	if err := i.Namespace.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.IdempotencyKey != nil && strings.TrimSpace(*i.IdempotencyKey) == "" {
		errs = append(errs, errors.New("idempotency key must not be blank when provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateNamespaceInput struct {
	ID   string
	Name *string

	// Metadata replaces the complete map when provided. A pointer to a nil or
	// empty map clears existing metadata; nil preserves it.
	Metadata *models.Metadata

	ProviderData map[string]string
}

var _ models.Validator = UpdateNamespaceInput{}

func (i UpdateNamespaceInput) Validate() error {
	var errs []error

	if err := (GetNamespaceInput{ID: i.ID}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name != nil && strings.TrimSpace(*i.Name) == "" {
		errs = append(errs, errors.New("invalid namespace name: must not be blank when provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type (
	DeleteNamespaceInput = GetNamespaceInput

	GetNamespaceResult    = Namespace
	CreateNamespaceResult = Namespace
	UpdateNamespaceResult = Namespace
)

type ChannelOperationState string

const (
	ChannelOperationStatePending ChannelOperationState = "pending"
	ChannelOperationStateInSync  ChannelOperationState = "in_sync"
)

type CreateChannelInput[C v4.ChannelConfig] struct {
	Namespace string
	ChannelID string
	Name      string
	Config    C

	// Disabled is the effective state supplied by Notification, including when
	// a nonempty user filter currently resolves to no Event types.
	Disabled bool

	// EventTypes contains concrete types; Providers do not expand wildcards.
	// Nil or empty means catch-all. Disabled takes precedence over filtering.
	EventTypes   []v4.EventType
	ProviderData map[string]string

	// IdempotencyKey identifies the same logical creation across retries.
	// Nil selects the provider default; a supplied key must be nonblank.
	IdempotencyKey *string
}

var _ models.Validator = CreateChannelInput[v4.ChannelConfig]{}

func (i CreateChannelInput[C]) Validate() error {
	var errs []error

	if err := (GetChannelInput{Namespace: i.Namespace, ChannelID: i.ChannelID}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if strings.TrimSpace(i.Name) == "" {
		errs = append(errs, errors.New("channel name is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid channel configuration: %w", err))
	}

	for idx, eventType := range i.EventTypes {
		if err := eventType.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid eventTypes[%d]: %w", idx, err))
		}
	}

	if i.IdempotencyKey != nil && strings.TrimSpace(*i.IdempotencyKey) == "" {
		errs = append(errs, errors.New("idempotency key must not be blank when provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateChannelInput[U v4.ChannelConfigUpdate] struct {
	Namespace string
	ChannelID string
	Name      *string

	// Config contains independently optional configuration fields.
	Config U

	// Disabled is the effective state supplied by Notification, including when
	// a nonempty user filter currently resolves to no Event types.
	Disabled *bool

	// EventTypes replaces the concrete routing filter when provided. A pointer
	// to a nil or empty slice enables catch-all routing; nil preserves filtering.
	EventTypes   *[]v4.EventType
	ProviderData map[string]string
}

var _ models.Validator = UpdateChannelInput[v4.WebhookConfigUpdate]{}

func (i UpdateChannelInput[U]) Validate() error {
	var errs []error

	if err := (GetChannelInput{Namespace: i.Namespace, ChannelID: i.ChannelID}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name != nil && strings.TrimSpace(*i.Name) == "" {
		errs = append(errs, errors.New("invalid channel name: must not be blank when provided"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid channel configuration: %w", err))
	}

	if i.EventTypes != nil {
		for idx, eventType := range *i.EventTypes {
			if err := eventType.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("invalid eventTypes[%d]: %w", idx, err))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetChannelInput struct {
	Namespace    string
	ChannelID    string
	ProviderData map[string]string
}

var _ models.Validator = GetChannelInput{}

func (i GetChannelInput) Validate() error {
	var errs []error

	if strings.TrimSpace(i.Namespace) == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if strings.TrimSpace(i.ChannelID) == "" {
		errs = append(errs, errors.New("channel ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeleteChannelInput = GetChannelInput

// ChannelOperationResult reports convergence.
type ChannelOperationResult struct {
	State             ChannelOperationState
	ProviderData      map[string]string
	NextObservationAt *time.Time
}

type (
	GetChannelResult    = ChannelOperationResult
	CreateChannelResult = ChannelOperationResult
	UpdateChannelResult = ChannelOperationResult
)

type SendEventInput struct {
	Event        v4.Event
	Destinations []EventDestination

	// IdempotencyKey identifies the same logical send across retries, not its external identity.
	// Nil selects the provider default; a supplied key must be nonblank.
	IdempotencyKey *string
}

var _ models.Validator = SendEventInput{}

func (i SendEventInput) Validate() error {
	var errs []error

	destinations := make([]GetEventDestination, 0, len(i.Destinations))

	for _, destination := range i.Destinations {
		destinations = append(destinations, GetEventDestination{
			ChannelID:          destination.ChannelID,
			DispatchGeneration: 1,
		})
	}

	if err := (GetEventInput{Event: i.Event, Destinations: destinations}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Event.EventType.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid event type: %w", err))
	}

	if i.Event.CreatedAt.IsZero() {
		errs = append(errs, errors.New("event creation time is required"))
	}

	if !json.Valid(i.Event.Payload) {
		errs = append(errs, errors.New("event payload must be valid JSON"))
	}

	if i.IdempotencyKey != nil && strings.TrimSpace(*i.IdempotencyKey) == "" {
		errs = append(errs, errors.New("idempotency key must not be blank when provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// EventDestination identifies a persisted delivery destination. For event_type
// routing, it defines tracking and result cardinality, not explicit targeting.
type EventDestination struct {
	ChannelID           string
	ChannelProviderData map[string]string
}

// SendEventResult must include every requested destination exactly once.
// Destination errors permit partial acceptance; a method-level error means
// Notification must not apply any part of the result.
type SendEventResult struct {
	EventProviderData map[string]string
	Destinations      []SendEventDestinationResult
}

type SendEventDestinationResult struct {
	ChannelID                       string
	Error                           error
	EventDeliveryStatusProviderData map[string]string
	NextObservationAt               *time.Time
}

type GetEventDestination struct {
	ChannelID                       string
	ChannelProviderData             map[string]string
	EventDeliveryStatusProviderData map[string]string
	// DispatchGeneration is 1 for initial dispatch and increments only on an
	// explicit resend, never for automatic retries or Provider attempts.
	DispatchGeneration uint64
}

type ResendEventInput struct {
	Event        v4.Event
	Destinations []GetEventDestination

	// IdempotencyKey identifies the same logical resend across retries.
	// Nil selects the provider default; a supplied key must be nonblank.
	IdempotencyKey *string
}

var _ models.Validator = ResendEventInput{}

func (i ResendEventInput) Validate() error {
	var errs []error

	if i.IdempotencyKey != nil && strings.TrimSpace(*i.IdempotencyKey) == "" {
		errs = append(errs, errors.New("idempotency key must not be blank when provided"))
	}

	if err := (GetEventInput{Event: i.Event, Destinations: i.Destinations}).Validate(); err != nil {
		errs = append(errs, err)
	}

	for idx, destination := range i.Destinations {
		if destination.DispatchGeneration < 2 {
			errs = append(errs, fmt.Errorf("destinations[%d]: resend dispatch generation must be at least 2", idx))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ResendEventResult struct {
	EventProviderData map[string]string
	Destinations      []ResendEventDestinationResult
}

type ResendEventDestinationResult = SendEventDestinationResult

type GetEventInput struct {
	Event        v4.Event
	Destinations []GetEventDestination
}

var _ models.Validator = GetEventInput{}

func (i GetEventInput) Validate() error {
	var errs []error

	if strings.TrimSpace(i.Event.Namespace) == "" {
		errs = append(errs, errors.New("event namespace is required"))
	}

	if strings.TrimSpace(i.Event.ID) == "" {
		errs = append(errs, errors.New("event ID is required"))
	}

	if len(i.Destinations) == 0 {
		errs = append(errs, errors.New("at least one destination is required"))
	}

	seen := make(map[string]struct{}, len(i.Destinations))

	for idx, destination := range i.Destinations {
		if strings.TrimSpace(destination.ChannelID) == "" {
			errs = append(errs, fmt.Errorf("destinations[%d]: channel ID is required", idx))
		}

		if _, ok := seen[destination.ChannelID]; ok {
			errs = append(errs, fmt.Errorf("destinations[%d]: duplicate channel ID", idx))
		}

		seen[destination.ChannelID] = struct{}{}

		if destination.DispatchGeneration == 0 {
			errs = append(errs, fmt.Errorf("destinations[%d]: dispatch generation must be positive", idx))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetEventResult follows the same destination completeness and partial-error
// contract as SendEventResult. Retrieval does not initiate another send.
type GetEventResult struct {
	EventProviderData map[string]string
	Destinations      []GetEventDestinationResult
}

type GetEventDestinationResult struct {
	ChannelID                       string
	Error                           error
	State                           v4.EventDeliveryStatusState
	EventDeliveryStatusProviderData map[string]string
	Attempts                        []v4.EventDeliveryAttempt
	NextObservationAt               *time.Time
}
