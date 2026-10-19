// Package v4 holds the Notification vNext types, starting with the identity of
// Notification Event types. Every domain's generated notificationevents package
// imports it, so it must not depend on Notification or any domain package.
package v4

import (
	"errors"
	"fmt"
	"regexp"

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
