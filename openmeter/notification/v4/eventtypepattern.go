package v4

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
)

const eventTypeWildcard = "*"

var _ models.Validator = (*EventTypePattern)(nil)

// EventTypePattern selects Event types by exact canonical identity, such as
// v1.billing.invoice.created, or by an anchored terminal wildcard, such as
// v1.billing.* or v1.*. The version is always explicit and matching is
// case-sensitive. A pattern may match types registered only in a future
// release, so parsing never consults the Registry.
type EventTypePattern struct {
	Version int

	// Name is the full Event type name of an exact pattern, or the possibly
	// empty name prefix of a wildcard pattern.
	Name string

	// Wildcard matches every name below Name, i.e. at least one further segment.
	Wildcard bool
}

// ParseEventTypePattern parses an exact canonical Event type or a pattern
// whose final segment is *.
func ParseEventTypePattern(s string) (EventTypePattern, error) {
	version, rest, err := splitCanonicalEventType(s)
	if err != nil {
		return EventTypePattern{}, err
	}

	p := EventTypePattern{Version: version, Name: rest}

	if rest == eventTypeWildcard {
		p = EventTypePattern{Version: version, Wildcard: true}
	} else if prefix, ok := strings.CutSuffix(rest, "."+eventTypeWildcard); ok {
		p = EventTypePattern{Version: version, Name: prefix, Wildcard: true}
	}

	if err := p.Validate(); err != nil {
		return EventTypePattern{}, fmt.Errorf("invalid event type pattern %q: %w", s, err)
	}

	return p, nil
}

// String returns the canonical form accepted by ParseEventTypePattern.
func (p EventTypePattern) String() string {
	if !p.Wildcard {
		return EventType{Name: p.Name, Version: p.Version}.String()
	}

	if p.Name == "" {
		return fmt.Sprintf("v%d.%s", p.Version, eventTypeWildcard)
	}

	return fmt.Sprintf("v%d.%s.%s", p.Version, p.Name, eventTypeWildcard)
}

func (p EventTypePattern) Validate() error {
	if !p.Wildcard {
		return EventType{Name: p.Name, Version: p.Version}.Validate()
	}

	var errs []error

	if p.Version < 1 {
		errs = append(errs, fmt.Errorf("version must be positive, got %d", p.Version))
	}

	// A wildcard prefix may have a single segment, e.g. v1.billing.*, so it
	// is checked as a one-segment extension of a valid name.
	if p.Name != "" && !namePattern.MatchString(p.Name+".x") {
		errs = append(errs, fmt.Errorf("name prefix %q must be lowercase snake_case segments", p.Name))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Matches reports whether t is selected by the pattern.
func (p EventTypePattern) Matches(t EventType) bool {
	if t.Version != p.Version {
		return false
	}

	if !p.Wildcard {
		return t.Name == p.Name
	}

	return p.Name == "" || strings.HasPrefix(t.Name, p.Name+".")
}
