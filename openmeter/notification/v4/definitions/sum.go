package definitions

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/jsonx"
)

// sum is the committed definitions.sum. Regenerate it with
// `make update-notification-definitions-sum`, never as part of `make generate`:
// rewriting it must be a deliberate decision that a changed definition was
// never released.
//
//go:embed definitions.sum
var sum []byte

// Generate writes one line per definition, sorted by Event type: the Event type,
// its fingerprint and, if deprecated, its replacement. The replacement is kept
// out of the fingerprint but recorded here, so setting, changing or removing a
// deprecation also shows up in review.
func Generate(w io.Writer) error {
	return generate(w, All())
}

// Verify checks that the committed definitions.sum matches the current
// definitions.
func Verify() error {
	return verify(sum, All())
}

func generate(w io.Writer, defs []v4.EventTypeDefinition) error {
	defs = slices.Clone(defs)
	slices.SortFunc(defs, func(a, b v4.EventTypeDefinition) int {
		return a.EventType.Compare(b.EventType)
	})

	for _, def := range defs {
		fingerprint, err := Fingerprint(def)
		if err != nil {
			return fmt.Errorf("event type %s: %w", def.EventType, err)
		}

		line := fmt.Sprintf("%s %s", def.EventType, fingerprint)
		if def.DeprecatedBy != nil {
			line += fmt.Sprintf(" deprecated_by=%s", def.DeprecatedBy)
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}

func verify(committed []byte, defs []v4.EventTypeDefinition) error {
	var want bytes.Buffer
	if err := generate(&want, defs); err != nil {
		return err
	}

	if bytes.Equal(want.Bytes(), committed) {
		return nil
	}

	return fmt.Errorf("definitions.sum does not match the current definitions\ncommitted:\n%s\ncurrent:\n%s", committed, want.Bytes())
}

// Fingerprint identifies what must not change once the definition is
// released: the Event type, the payload schema and the Rule configuration
// schema, if any. Schemas are hashed in canonical form without annotations, so
// formatting, object key order and documentation edits do not change it.
// DeprecatedBy is left out because it is set on released definitions.
func Fingerprint(d v4.EventTypeDefinition) (string, error) {
	payload, err := fingerprintSchema(d.PayloadJSONSchema)
	if err != nil {
		return "", fmt.Errorf("payload JSON schema: %w", err)
	}

	ruleConfig := json.RawMessage("null")
	if d.Rule != nil {
		ruleConfig, err = fingerprintSchema(d.Rule.JSONSchema)
		if err != nil {
			return "", fmt.Errorf("rule config JSON schema: %w", err)
		}
	}

	doc, err := json.Marshal(struct {
		EventType  string          `json:"event_type"`
		Payload    json.RawMessage `json:"payload"`
		RuleConfig json.RawMessage `json:"rule_config"`
	}{
		EventType:  d.EventType.String(),
		Payload:    payload,
		RuleConfig: ruleConfig,
	})
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(doc)

	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Annotation keywords only document a schema: validation ignores them, so
// editing the docs of a released contract is not a contract change.
var annotationKeywords = []string{"$comment", "deprecated", "description", "examples", "title"}

// Keywords whose value is a map from names to subschemas. Their keys are
// property or definition names, so a property called "description" stays.
// The values of draft-07 dependencies can also be arrays of property names,
// which are kept as they are.
var namedSubschemaKeywords = []string{"$defs", "definitions", "dependencies", "dependentSchemas", "patternProperties", "properties"}

// Keywords whose value is a subschema or an array of subschemas.
var subschemaKeywords = []string{
	"additionalItems", "additionalProperties", "allOf", "anyOf", "contains", "contentSchema", "else", "if", "items",
	"not", "oneOf", "prefixItems", "propertyNames", "then", "unevaluatedItems", "unevaluatedProperties",
}

func fingerprintSchema(schema []byte) (json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(schema))
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after JSON document")
	}

	stripped, err := json.Marshal(withoutAnnotations(value))
	if err != nil {
		return nil, err
	}

	// Marshal escapes HTML characters; Canonical writes them as they are.
	return jsonx.Canonical(stripped)
}

// withoutAnnotations removes the annotation keywords from a schema and its
// subschemas. Values of other keywords, such as enum, const or default, are
// data and are kept as they are.
func withoutAnnotations(schema any) any {
	obj, ok := schema.(map[string]any)
	if !ok {
		return schema
	}

	out := make(map[string]any, len(obj))
	for k, v := range obj {
		switch {
		case slices.Contains(annotationKeywords, k):
			continue
		case slices.Contains(namedSubschemaKeywords, k):
			if named, ok := v.(map[string]any); ok {
				m := make(map[string]any, len(named))
				for name, sub := range named {
					m[name] = withoutAnnotations(sub)
				}

				v = m
			}
		case slices.Contains(subschemaKeywords, k):
			if list, ok := v.([]any); ok {
				l := make([]any, len(list))
				for i, sub := range list {
					l[i] = withoutAnnotations(sub)
				}

				v = l
			} else {
				v = withoutAnnotations(v)
			}
		}

		out[k] = v
	}

	return out
}
