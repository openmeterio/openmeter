package v4

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
)

// RuleConfigValidatorFunc validates a Rule configuration. Domain validators
// passed to the Registry only receive configurations that already satisfy the
// JSON Schema.
type RuleConfigValidatorFunc func(ctx context.Context, namespace string, configuration json.RawMessage) error

// RuleConfigValidatorBinding assigns a domain validator to the Event type whose
// Rule configurations it checks.
type RuleConfigValidatorBinding struct {
	EventType EventType
	Validate  RuleConfigValidatorFunc
}

// RuleConfigDefinition is the Rule configuration contract of an Event type.
// Domain validation is not part of the contract; it is passed to the Registry
// separately, because it depends on domain services.
type RuleConfigDefinition struct {
	JSONSchema json.RawMessage
}

var _ models.Equaler[RuleConfigDefinition] = (*RuleConfigDefinition)(nil)

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

// clone returns a copy that shares no mutable memory with d.
func (d EventTypeDefinition) clone() EventTypeDefinition {
	out := d
	out.PayloadJSONSchema = slices.Clone(d.PayloadJSONSchema)

	if d.Rule != nil {
		out.Rule = &RuleConfigDefinition{
			JSONSchema: slices.Clone(d.Rule.JSONSchema),
		}
	}

	if d.DeprecatedBy != nil {
		deprecatedBy := *d.DeprecatedBy
		out.DeprecatedBy = &deprecatedBy
	}

	return out
}

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
		// An empty prefix here comes from v1..*, which must not widen to v1.*.
		if prefix == "" {
			return EventTypePattern{}, models.NewGenericValidationError(fmt.Errorf("invalid event type pattern %q: empty name prefix", s))
		}

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

// EventRegistryConfig supplies every Event type definition served by the process.
type EventRegistryConfig struct {
	Logger      *slog.Logger
	Definitions []EventTypeDefinition

	// RuleConfigValidators holds optional domain validators of Rule
	// configurations. Each must name a registered type that Rules can produce,
	// and a type gets at most one: wiring collects validators from several
	// domains, and a second one must not silently replace the first.
	RuleConfigValidators []RuleConfigValidatorBinding
}

var _ models.Validator = EventRegistryConfig{}

// Validate rejects invalid definitions, duplicate Event types, replacement
// chains that are dangling or cyclic, and misplaced Rule validators.
func (c EventRegistryConfig) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	definitions := make(map[EventType]EventTypeDefinition, len(c.Definitions))

	for i, def := range c.Definitions {
		if err := def.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("definitions[%d]: %w", i, err))

			continue
		}

		if _, ok := definitions[def.EventType]; ok {
			errs = append(errs, fmt.Errorf("definitions[%d]: duplicate event type %s", i, def.EventType))

			continue
		}

		definitions[def.EventType] = def
	}

	for _, def := range c.Definitions {
		if def.DeprecatedBy == nil {
			continue
		}

		if _, ok := definitions[*def.DeprecatedBy]; !ok {
			errs = append(errs, fmt.Errorf("event type %s: deprecated by unregistered event type %s", def.EventType, def.DeprecatedBy))

			continue
		}

		// Each definition names at most one replacement, so a chain either ends
		// at a current definition or revisits an Event type.
		seen := map[EventType]struct{}{def.EventType: {}}

		for next := def.DeprecatedBy; next != nil; next = definitions[*next].DeprecatedBy {
			if _, ok := seen[*next]; ok {
				errs = append(errs, fmt.Errorf("event type %s: deprecation cycle through %s", def.EventType, next))

				break
			}

			seen[*next] = struct{}{}
		}
	}

	validated := make(map[EventType]struct{}, len(c.RuleConfigValidators))

	for i, binding := range c.RuleConfigValidators {
		t := binding.EventType
		def, ok := definitions[t]

		switch {
		case binding.Validate == nil:
			errs = append(errs, fmt.Errorf("rule config validators[%d]: validator of %s is nil", i, t))
		case !ok:
			errs = append(errs, fmt.Errorf("rule config validators[%d]: unregistered event type %s", i, t))
		case def.Rule == nil:
			errs = append(errs, fmt.Errorf("rule config validators[%d]: event type %s, which rules cannot produce", i, t))
		}

		if _, ok := validated[t]; ok {
			errs = append(errs, fmt.Errorf("rule config validators[%d]: duplicate event type %s", i, t))
		}

		validated[t] = struct{}{}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type registeredEventType struct {
	definition          EventTypeDefinition
	payloadSchema       *jsonschema.Schema
	ruleConfigSchema    *jsonschema.Schema
	ruleConfigValidator RuleConfigValidatorFunc
}

// EventRegistry holds the Event type definitions and compiled schemas fixed at
// construction. It is immutable and safe for concurrent use. Its zero value is
// not usable; construct it with NewEventRegistry.
type EventRegistry struct {
	logger     *slog.Logger
	eventTypes map[EventType]registeredEventType

	// sorted lists Event types in EventType.Compare order for deterministic
	// listing.
	sorted []EventType
}

// NewEventRegistry validates the definitions and compiles their schemas, so
// an invalid contract fails dependency injection instead of Event creation.
func NewEventRegistry(config EventRegistryConfig) (*EventRegistry, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid event registry configuration: %w", err)
	}

	r := &EventRegistry{
		logger:     config.Logger,
		eventTypes: make(map[EventType]registeredEventType, len(config.Definitions)),
		sorted:     make([]EventType, 0, len(config.Definitions)),
	}

	ruleConfigValidators := make(map[EventType]RuleConfigValidatorFunc, len(config.RuleConfigValidators))
	for _, binding := range config.RuleConfigValidators {
		ruleConfigValidators[binding.EventType] = binding.Validate
	}

	var errs []error

	for _, def := range config.Definitions {
		registered := registeredEventType{
			definition:          def.clone(),
			ruleConfigValidator: ruleConfigValidators[def.EventType],
		}

		payloadSchema, err := compileJSONSchema(def.EventType.String()+".payload", def.PayloadJSONSchema)
		if err != nil {
			errs = append(errs, fmt.Errorf("event type %s: payload JSON schema: %w", def.EventType, err))
		}

		registered.payloadSchema = payloadSchema

		if def.Rule != nil {
			ruleConfigSchema, err := compileJSONSchema(def.EventType.String()+".rule_config", def.Rule.JSONSchema)
			if err != nil {
				errs = append(errs, fmt.Errorf("event type %s: rule config JSON schema: %w", def.EventType, err))
			}

			registered.ruleConfigSchema = ruleConfigSchema
		}

		r.eventTypes[def.EventType] = registered
		r.sorted = append(r.sorted, def.EventType)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid event registry configuration: %w", models.NewGenericValidationError(err))
	}

	slices.SortFunc(r.sorted, EventType.Compare)

	return r, nil
}

// EventType returns a copy of the definition registered for t.
func (r *EventRegistry) EventType(t EventType) (EventTypeDefinition, error) {
	registered, err := r.get(t)
	if err != nil {
		return EventTypeDefinition{}, err
	}

	return registered.definition.clone(), nil
}

// EventTypes returns copies of the definitions matching filter, or all when
// nil, in EventType.Compare order.
func (r *EventRegistry) EventTypes(filter *EventTypePattern) []EventTypeDefinition {
	out := make([]EventTypeDefinition, 0, len(r.sorted))

	for _, t := range r.sorted {
		if filter == nil || filter.Matches(t) {
			out = append(out, r.eventTypes[t].definition.clone())
		}
	}

	return out
}

// PayloadValidatorFunc validates an Event payload against its registered
// contract. Namespace is currently unused; it keeps the signature stable for
// namespace-aware payload checks.
type PayloadValidatorFunc func(ctx context.Context, namespace string, payload json.RawMessage) error

// PayloadValidator returns the payload validator of t. Event creation is the
// only payload validation boundary: dispatch trusts persisted payloads.
func (r *EventRegistry) PayloadValidator(t EventType) (PayloadValidatorFunc, error) {
	registered, err := r.get(t)
	if err != nil {
		return nil, err
	}

	return func(_ context.Context, _ string, payload json.RawMessage) error {
		if err := validateJSON(registered.payloadSchema, payload); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("invalid %s payload: %w", t, err))
		}

		return nil
	}, nil
}

// RuleConfigValidator returns the validator used when a Rule is created or
// updated. It runs schema validation and then the optional domain validator.
// It fails for Event types that no Rule can produce.
func (r *EventRegistry) RuleConfigValidator(t EventType) (RuleConfigValidatorFunc, error) {
	validateSchema, err := r.RuleConfigSchemaValidator(t)
	if err != nil {
		return nil, err
	}

	domainValidator := r.eventTypes[t].ruleConfigValidator

	return func(ctx context.Context, namespace string, configuration json.RawMessage) error {
		if err := validateSchema(ctx, namespace, configuration); err != nil {
			return err
		}

		if domainValidator == nil {
			return nil
		}

		return domainValidator(ctx, namespace, configuration)
	}, nil
}

// RuleConfigSchemaValidator returns a validator that checks only the JSON
// Schema. Event creation uses it for the Rule snapshot: the domain already
// evaluated that configuration, and domain checks against current state could
// reject a snapshot of a Rule that was valid when it ran. It fails for Event
// types that no Rule can produce.
func (r *EventRegistry) RuleConfigSchemaValidator(t EventType) (RuleConfigValidatorFunc, error) {
	registered, err := r.get(t)
	if err != nil {
		return nil, err
	}

	if registered.definition.Rule == nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("event type %s cannot be produced by rules", t))
	}

	return func(_ context.Context, _ string, configuration json.RawMessage) error {
		if err := validateJSON(registered.ruleConfigSchema, configuration); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("invalid %s rule config: %w", t, err))
		}

		return nil
	}, nil
}

func (r *EventRegistry) get(t EventType) (registeredEventType, error) {
	registered, ok := r.eventTypes[t]
	if !ok {
		return registeredEventType{}, models.NewGenericNotFoundError(fmt.Errorf("event type %s is not registered", t))
	}

	return registered, nil
}

// compileJSONSchema compiles a self-contained schema. Draft 2020-12 applies
// when the document does not declare $schema; external references are not
// loaded. The name only identifies the schema in errors.
func compileJSONSchema(name string, schema json.RawMessage) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	// Built-in metaschemas still resolve; any other reference fails to load.
	c.UseLoader(jsonschema.SchemeURLLoader{})

	// A non-file base URL keeps relative references from resolving against
	// the working directory.
	url := "mem:///" + name

	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}

	return c.Compile(url)
}

func validateJSON(schema *jsonschema.Schema, raw json.RawMessage) error {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("must be valid JSON: %w", err)
	}

	return schema.Validate(doc)
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
