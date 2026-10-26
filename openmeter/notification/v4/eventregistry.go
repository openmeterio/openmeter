package v4

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/openmeterio/openmeter/pkg/models"
)

// EventRegistryConfig supplies every Event type definition served by the process.
type EventRegistryConfig struct {
	Logger      *slog.Logger
	Definitions []EventTypeDefinition
}

var _ models.Validator = EventRegistryConfig{}

// Validate rejects invalid definitions, duplicate Event types, and replacement
// chains that are dangling or cyclic.
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type registeredEventType struct {
	definition       EventTypeDefinition
	payloadSchema    *jsonschema.Schema
	ruleConfigSchema *jsonschema.Schema
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

	var errs []error

	for _, def := range config.Definitions {
		registered := registeredEventType{definition: def.clone()}

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

// RuleConfigValidator returns the Rule configuration validator of t. It runs
// schema validation and then the optional domain validator. It fails for
// Event types that no Rule can produce.
func (r *EventRegistry) RuleConfigValidator(t EventType) (RuleConfigValidatorFunc, error) {
	registered, err := r.get(t)
	if err != nil {
		return nil, err
	}

	if registered.definition.Rule == nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("event type %s cannot be produced by rules", t))
	}

	domainValidator := registered.definition.Rule.Validator

	return func(ctx context.Context, namespace string, configuration json.RawMessage) error {
		if err := validateJSON(registered.ruleConfigSchema, configuration); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("invalid %s rule config: %w", t, err))
		}

		if domainValidator == nil {
			return nil
		}

		return domainValidator(ctx, namespace, configuration)
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
