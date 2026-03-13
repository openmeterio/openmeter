// Package aip160 implements an AIP-160 compatible filter expression parser.
//
// Two encoding formats are supported:
//
// # Simple format (field as top-level key)
//
// Used when the filter is encoded as a standalone string:
//
//	name[eq]=Foo+User
//	name[eq]=Foo+User&age[gt]=30
//	deleted_time                         (existence: field is non-null)
//	labels.activity[nexists]             (non-existence)
//	city[oeq]=London,Paris               (or-equal, CSV)
//
// # Deep-object format (field nested under a param name)
//
// Standard AIP-160 URL query params where the filter param name is the base key:
//
//	?filter[name][eq]=Foo+User
//	?filter[name][eq]=Foo+User&filter[age][gt]=30
//	?filter[deleted_time]                          (existence)
//	?filter[labels.key_1][eq]=val_A                (dot notation)
//	?filter[labels.activity][nexists]              (non-existence)
//	?filter[city][oeq]=London,Paris                (or-equal, CSV)
//
// Use [ParseFromValues] to parse from [url.Values] (e.g. r.URL.Query()).
// Use [Parse] for the simple string format.
// [Filter.Parse] auto-detects the format.
package aip160

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Filter represents an AIP-160 compatible filter expression string.
// It auto-detects between the simple format ("name[eq]=foo") and
// the deep-object format ("filter[name][eq]=foo") on [Filter.Parse].
type Filter string

// Parse parses the filter expression into a list of field conditions.
// It auto-detects the format:
//   - simple: "name[eq]=foo&age[gt]=30"
//   - deep-object: "filter[name][eq]=foo&filter[age][gt]=30"
//
// All conditions are implicitly combined with AND.
func (f Filter) Parse() ([]FieldFilter, error) {
	s := string(f)
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	rawParams, err := url.ParseQuery(s)
	if err != nil {
		return nil, fmt.Errorf("aip160: invalid filter expression: %w", err)
	}

	// Auto-detect format: if every key matches "word[...]..." use deep-object parsing.
	if base, ok := detectDeepObjectBase(rawParams); ok {
		return parseDeepObject(rawParams, base)
	}
	return parseSimple(rawParams)
}

// IsEmpty returns true if the filter string is blank.
func (f Filter) IsEmpty() bool {
	return strings.TrimSpace(string(f)) == ""
}

// String implements fmt.Stringer.
func (f Filter) String() string {
	return string(f)
}

// UnmarshalText implements encoding.TextUnmarshaler so oapi-codegen can bind
// the raw query-parameter value into a Filter.
func (f *Filter) UnmarshalText(text []byte) error {
	*f = Filter(text)
	return nil
}

// Operator is an AIP-160 filter operator.
type Operator string

const (
	// OpEq returns records that exactly match the given value (default when no operator is specified).
	OpEq Operator = "eq"
	// OpNeq returns records that do not match the given value.
	OpNeq Operator = "neq"
	// OpOEq (or-equal) returns records that match any of the comma-separated values.
	OpOEq Operator = "oeq"
	// OpContains returns records that contain the given string.
	OpContains Operator = "contains"
	// OpOContains (or-contains) returns records that contain any of the comma-separated strings.
	OpOContains Operator = "ocontains"
	// OpLt returns records where the field value is less than the given value.
	OpLt Operator = "lt"
	// OpLte returns records where the field value is less than or equal to the given value.
	OpLte Operator = "lte"
	// OpGt returns records where the field value is greater than the given value.
	OpGt Operator = "gt"
	// OpGte returns records where the field value is greater than or equal to the given value.
	OpGte Operator = "gte"
	// OpExists checks that the field is non-null.
	// Expressed as a bare field name with no value (?filter[field]) or with the explicit bracket (?filter[field][exists]).
	OpExists Operator = "exists"
	// OpNexists checks that the field is null/absent.
	OpNexists Operator = "nexists"
)

// validValueOperators is the set of operators that require a value.
var validValueOperators = map[Operator]bool{
	OpEq:        true,
	OpNeq:       true,
	OpOEq:       true,
	OpContains:  true,
	OpOContains: true,
	OpLt:        true,
	OpLte:       true,
	OpGt:        true,
	OpGte:       true,
}

// FieldFilter represents a single parsed filter condition.
type FieldFilter struct {
	// Field is the resource field name. Dot notation is used for nested fields (e.g. "labels.owner").
	Field string
	// Operator is the comparison operator.
	Operator Operator
	// Value is the comparison value for single-value operators (eq, neq, contains, lt, lte, gt, gte).
	Value string
	// Values holds the list of values for multi-value operators (oeq, ocontains).
	Values []string
}

// Parse parses an AIP-160 filter expression string in the simple format into a
// slice of FieldFilter conditions.
//
// The input must use URL query string encoding with field as the top-level key:
//
//	field[op]=value
//	field[op]=value&field2[op2]=value2
//	field                              (existence: field is non-null)
//	field[nexists]                     (non-existence: field is null/absent)
//
// Values for oeq and ocontains operators are comma-separated.
// All conditions are implicitly ANDed.
//
// For the standard deep-object URL format (?filter[field][op]=value) use
// [ParseFromValues] instead.
func Parse(s string) ([]FieldFilter, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	rawParams, err := url.ParseQuery(s)
	if err != nil {
		return nil, fmt.Errorf("aip160: invalid filter expression: %w", err)
	}

	return parseSimple(rawParams)
}

// ParseFromValues parses AIP-160 filter conditions from the deep-object URL
// query params produced by ?filter[field][op]=value.
//
// paramName is the base query parameter name, typically "filter".
// Only keys of the form paramName[field] or paramName[field][op] are processed;
// all other keys in values are ignored.
//
// Example URL: ?filter[name][eq]=Foo+User&filter[age][gt]=30&filter[deleted_time]
//
//	conditions, err := aip160.ParseFromValues(r.URL.Query(), "filter")
func ParseFromValues(values url.Values, paramName string) ([]FieldFilter, error) {
	prefix := paramName + "["
	conditions := make([]FieldFilter, 0)

	for key, vals := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		// Strip paramName, leaving "[field][op]" or "[field]".
		rest := key[len(paramName):]
		field, op, err := parseDeepSuffix(rest)
		if err != nil {
			return nil, fmt.Errorf("aip160: invalid filter key %q: %w", key, err)
		}
		cond, err := buildFieldFilter(field, op, vals)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}

	return conditions, nil
}

// ── internal helpers ─────────────────────────────────────────────────────────

// simpleKeyRe matches "field" or "field[op]".
var simpleKeyRe = regexp.MustCompile(`^([^\[]+)(?:\[([^\]]*)\])?$`)

// deepSuffixRe matches "[field][op]" or "[field]" (existence/nexists).
var deepSuffixRe = regexp.MustCompile(`^\[([^\]]+)\](?:\[([^\]]*)\])?$`)

// deepObjectKeyRe matches "base[...]..." — any key starting with word + bracket.
var deepObjectKeyRe = regexp.MustCompile(`^([^\[]+)\[`)

func parseSimple(rawParams url.Values) ([]FieldFilter, error) {
	conditions := make([]FieldFilter, 0, len(rawParams))
	for key, vals := range rawParams {
		matches := simpleKeyRe.FindStringSubmatch(key)
		if matches == nil {
			return nil, fmt.Errorf("aip160: invalid filter key %q", key)
		}
		field := matches[1]
		op := Operator(matches[2])
		cond, err := buildFieldFilter(field, op, vals)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}
	return conditions, nil
}

// detectDeepObjectBase returns the common base name (e.g. "filter") when all
// keys in rawParams use the deep-object format "base[field][op]".
// Returns ("", false) if the format is not consistently deep-object.
func detectDeepObjectBase(rawParams url.Values) (string, bool) {
	var base string
	for key := range rawParams {
		m := deepObjectKeyRe.FindStringSubmatch(key)
		if m == nil {
			return "", false
		}
		if base == "" {
			base = m[1]
		} else if base != m[1] {
			// Mixed bases — not unambiguous deep-object.
			return "", false
		}
		// Verify the remainder is a valid deep suffix "[field]" or "[field][op]".
		rest := key[len(m[1]):]
		if !deepSuffixRe.MatchString(rest) {
			return "", false
		}
	}
	return base, base != ""
}

func parseDeepObject(rawParams url.Values, base string) ([]FieldFilter, error) {
	return ParseFromValues(rawParams, base)
}

// parseDeepSuffix parses "[field][op]" or "[field]" returning (field, op).
func parseDeepSuffix(rest string) (field string, op Operator, err error) {
	matches := deepSuffixRe.FindStringSubmatch(rest)
	if matches == nil {
		return "", "", fmt.Errorf("expected [field] or [field][op], got %q", rest)
	}
	return matches[1], Operator(matches[2]), nil
}

// buildFieldFilter constructs a FieldFilter from a parsed field name, operator,
// and raw URL param values.
func buildFieldFilter(field string, op Operator, vals []string) (FieldFilter, error) {
	// Bare field name (no operator bracket, op is empty string) with no value → existence check.
	if op == "" {
		if len(vals) == 0 || (len(vals) == 1 && vals[0] == "") {
			return FieldFilter{Field: field, Operator: OpExists}, nil
		}
		// "field=value" without an explicit operator defaults to eq.
		return FieldFilter{Field: field, Operator: OpEq, Value: vals[0]}, nil
	}

	// Explicit existence checks — value is ignored.
	if op == OpExists {
		return FieldFilter{Field: field, Operator: OpExists}, nil
	}

	// field[nexists] — value is ignored.
	if op == OpNexists {
		return FieldFilter{Field: field, Operator: OpNexists}, nil
	}

	if !validValueOperators[op] {
		return FieldFilter{}, fmt.Errorf("aip160: unsupported operator %q for field %q", op, field)
	}

	if len(vals) == 0 || (len(vals) == 1 && vals[0] == "") {
		return FieldFilter{}, fmt.Errorf("aip160: operator %q requires a non-empty value for field %q", op, field)
	}

	value := vals[0]

	switch op {
	case OpOEq:
		return FieldFilter{Field: field, Operator: OpOEq, Values: splitCSV(value)}, nil
	case OpOContains:
		return FieldFilter{Field: field, Operator: OpOContains, Values: splitCSV(value)}, nil
	default:
		return FieldFilter{Field: field, Operator: op, Value: value}, nil
	}
}

// splitCSV splits a comma-separated string, trimming spaces around each element.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
