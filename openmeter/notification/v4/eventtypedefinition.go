package v4

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
)

// RuleConfigValidatorFunc performs namespace-aware domain validation of a Rule
// configuration that already satisfies its JSON Schema.
type RuleConfigValidatorFunc func(ctx context.Context, namespace string, configuration json.RawMessage) error

// RuleConfigDefinition is the Rule configuration contract of an Event type.
type RuleConfigDefinition struct {
	JSONSchema json.RawMessage

	// Validator is optional and runs after schema validation succeeds.
	Validator RuleConfigValidatorFunc `json:"-"`
}

var _ models.Equaler[RuleConfigDefinition] = (*RuleConfigDefinition)(nil)

// Equal compares the schema only; Validator is a function and not comparable.
func (c RuleConfigDefinition) Equal(other RuleConfigDefinition) bool {
	return bytes.Equal(c.JSONSchema, other.JSONSchema)
}

var (
	_ models.Validator                            = (*EventTypeDefinition)(nil)
	_ models.CustomValidator[EventTypeDefinition] = (*EventTypeDefinition)(nil)
	_ models.Equaler[EventTypeDefinition]         = (*EventTypeDefinition)(nil)
)

// EventTypeDefinition binds an Event type to the contracts generated from its
// TypeSpec models. Released definitions are immutable: a changed contract is a
// new Event type version.
type EventTypeDefinition struct {
	EventType         EventType
	PayloadJSONSchema json.RawMessage

	// Rule is nil when no Rule can produce the Event type, e.g. system Events.
	Rule *RuleConfigDefinition

	// DeprecatedBy is advisory and names the immediate replacement. Deprecated
	// definitions remain registered and usable; Events are never converted.
	DeprecatedBy *EventType
}

func (d EventTypeDefinition) Deprecated() bool {
	return d.DeprecatedBy != nil
}

// Validate checks the definition in isolation. Schemas are only checked to be
// JSON here; the Registry compiles them and validates references between
// definitions.
func (d EventTypeDefinition) Validate() error {
	var errs []error

	if err := d.EventType.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("event type: %w", err))
	}

	if err := validateJSONSchemaDocument(d.PayloadJSONSchema); err != nil {
		errs = append(errs, fmt.Errorf("payload JSON schema: %w", err))
	}

	if d.Rule != nil {
		if err := validateJSONSchemaDocument(d.Rule.JSONSchema); err != nil {
			errs = append(errs, fmt.Errorf("rule config JSON schema: %w", err))
		}
	}

	if d.DeprecatedBy != nil {
		if err := d.DeprecatedBy.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("deprecated by: %w", err))
		} else if *d.DeprecatedBy == d.EventType {
			errs = append(errs, errors.New("deprecated by: event type cannot deprecate itself"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (d EventTypeDefinition) ValidateWith(validators ...models.ValidatorFunc[EventTypeDefinition]) error {
	return models.Validate(d, validators...)
}

// Equal compares the contract: identity, byte-equal schemas, Rule capability,
// and replacement.
func (d EventTypeDefinition) Equal(other EventTypeDefinition) bool {
	return d.EventType == other.EventType &&
		bytes.Equal(d.PayloadJSONSchema, other.PayloadJSONSchema) &&
		equal.PtrEqual(d.Rule, other.Rule) &&
		equal.ComparablePtrEqual(d.DeprecatedBy, other.DeprecatedBy)
}

// clone returns a copy that shares no mutable memory with d. The Rule
// validator is a function value and is shared.
func (d EventTypeDefinition) clone() EventTypeDefinition {
	out := d
	out.PayloadJSONSchema = slices.Clone(d.PayloadJSONSchema)

	if d.Rule != nil {
		out.Rule = &RuleConfigDefinition{
			JSONSchema: slices.Clone(d.Rule.JSONSchema),
			Validator:  d.Rule.Validator,
		}
	}

	if d.DeprecatedBy != nil {
		deprecatedBy := *d.DeprecatedBy
		out.DeprecatedBy = &deprecatedBy
	}

	return out
}

func validateJSONSchemaDocument(schema json.RawMessage) error {
	if len(schema) == 0 {
		return errors.New("is required")
	}

	if !json.Valid(schema) {
		return errors.New("must be valid JSON")
	}

	return nil
}
