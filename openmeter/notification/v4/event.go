package v4

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// namePattern matches <resource>.<operation> or <subsystem>.<resource>.<operation>
// with lowercase snake_case segments. Keep it in sync with the @eventType check
// in api/spec/packages/typespec-notification-events.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

var _ models.Validator = (*EventType)(nil)

// EventType is a versioned Event type. Its canonical string form,
// v<version>.<name>, crosses API, Kafka, persistence and Provider boundaries.
// Versions have no compatibility guarantee: a changed contract is a new
// version.
type EventType struct {
	Name    string
	Version int
}

// String returns the canonical form, e.g. v1.billing.invoice.created.
func (t EventType) String() string {
	return fmt.Sprintf("v%d.%s", t.Version, t.Name)
}

func (t EventType) Validate() error {
	var errs []error

	if !namePattern.MatchString(t.Name) {
		errs = append(errs, fmt.Errorf("name %q must have at least two lowercase snake_case segments", t.Name))
	}

	if t.Version < 1 {
		errs = append(errs, fmt.Errorf("version must be positive, got %d", t.Version))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Event is the persisted snapshot passed unchanged to a Provider. Providers must
// not resolve its payload or type through live domain resources or a local Registry.
type Event struct {
	models.NamespacedID

	EventType    EventType
	CreatedAt    time.Time
	Payload      json.RawMessage
	Annotations  models.Annotations
	RuleSnapshot *RuleSnapshot
	ProviderData ProviderData
}

func (e Event) String() string {
	return fmt.Sprintf("event:%s (id=%s namespace=%s)", e.EventType, e.NamespacedID.ID, e.NamespacedID.Namespace)
}

// RuleSnapshot captures evaluated conditions without referring to a live Rule
// or its Channels. System and test Events omit it.
type RuleSnapshot struct {
	models.NamespacedID

	Name          string
	EventType     EventType
	Revision      uint64
	Configuration json.RawMessage
}

func (s RuleSnapshot) String() string {
	return fmt.Sprintf("rule_snapshot:%s (event_type=%s id=%s namespace=%s)", s.Name, s.EventType, s.NamespacedID.ID, s.NamespacedID.Namespace)
}
