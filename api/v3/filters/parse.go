package filters

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// maxCommaSeparatedItems caps the number of values accepted in a comma-separated
// filter parameter (oeq/ocontains).
const maxCommaSeparatedItems = 50

// maxFilterValueLength caps the byte length of any single filter value.
const maxFilterValueLength = 1024

// ErrTooManyItems is returned when a comma-separated filter value exceeds
// maxCommaSeparatedItems after splitting and trimming.
var ErrTooManyItems = fmt.Errorf("too many comma-separated items (max %d)", maxCommaSeparatedItems)

// Parse populates a filter struct from URL query values.
// The target must be a pointer to a struct (or pointer-to-pointer-to-struct)
// whose fields are filter types. Field names are resolved from json struct tags.
//
// Supported field types:
//   - *FilterString: parsed via parseFilterString
//   - *FilterStringExact: parsed via parseFilterStringExact
//   - *FilterNumeric: parsed via parseFilterNumeric
//   - *FilterDateTime: parsed via parseFilterDateTime
//   - *FilterBoolean: parsed via parseFilterBoolean
//   - *string: parsed as shorthand eq (filter[field]=value)
//   - any type implementing json.Unmarshaler: marshals filter ops to JSON, then unmarshals
func Parse(qs url.Values, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("Parse: target must be a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Pointer {
		return parseFiltersValue(qs, v)
	}
	// target is **Struct — allocate the inner struct if any filter[...] keys exist
	if !hasFilterKeys(qs) {
		return nil
	}
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	return parseFiltersValue(qs, v.Elem())
}

func hasFilterKeys(qs url.Values) bool {
	for key := range qs {
		if strings.HasPrefix(key, "filter[") {
			return true
		}
	}
	return false
}

// hasFieldKeys reports whether qs contains any filter[<field>] or
// filter[<field>][op] key for the given field name.
func hasFieldKeys(qs url.Values, field string) bool {
	prefix := "filter[" + field + "]"
	for key := range qs {
		if key == prefix || strings.HasPrefix(key, prefix+"[") {
			return true
		}
	}
	return false
}

// checkUnknownFilterKeys returns an error if any filter[<name>]... key in qs
// refers to a <name> that is not in the knownFields set. Label-style
// dot-notation keys (e.g. "filter[labels.env]") are matched against their
// base segment before the first dot, since label map fields use that shape.
// Unknown field names are reported as a deterministic, comma-separated list
// so the error can be surfaced to API clients (e.g. as a 400 response).
func checkUnknownFilterKeys(qs url.Values, knownFields map[string]struct{}) error {
	var unknown []string
	seen := make(map[string]struct{})
	for key := range qs {
		name, ok := filterFieldName(key)
		if !ok {
			continue
		}
		// Allow dot-notation for labels-style map filters: the known field is
		// the base segment ("labels" in "labels.env").
		base := name
		if dot := strings.IndexByte(name, '.'); dot > 0 {
			base = name[:dot]
		}
		if _, known := knownFields[base]; known {
			continue
		}
		if _, already := seen[name]; already {
			continue
		}
		seen[name] = struct{}{}
		unknown = append(unknown, name)
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("unknown filter field(s): %s", strings.Join(unknown, ", "))
}

// filterFieldName extracts the <name> portion from a URL query key shaped like
// "filter[<name>]" or "filter[<name>][op]". It returns false for keys that do
// not match the filter[...] prefix or that are malformed (e.g. "filter[" with
// no closing bracket or an empty name).
func filterFieldName(key string) (string, bool) {
	const prefix = "filter["
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	rest := key[len(prefix):]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return "", false
	}
	return rest[:end], true
}

func parseFiltersValue(qs url.Values, v reflect.Value) error {
	t := v.Type()

	knownFields := make(map[string]struct{}, t.NumField())
	for i := range t.NumField() {
		if name := jsonFieldName(t.Field(i)); name != "" && name != "-" {
			knownFields[name] = struct{}{}
		}
	}
	if err := checkUnknownFilterKeys(qs, knownFields); err != nil {
		return err
	}

	for i := range t.NumField() {
		field := t.Field(i)
		fieldVal := v.Field(i)

		name := jsonFieldName(field)
		if name == "" || name == "-" {
			continue
		}

		// Skip fields with no matching filter[<name>]... keys so the target
		// field stays nil and callers can distinguish "no filter" from
		// "filter with all zero ops".
		if !hasFieldKeys(qs, name) {
			continue
		}

		if !fieldVal.CanAddr() {
			return fmt.Errorf("filter[%s]: target field is not addressable", name)
		}

		if err := dispatchFieldParse(qs, name, fieldVal.Addr().Interface()); err != nil {
			return err
		}
	}

	return nil
}

// dispatchFieldParse populates a single target field from filter query
// parameters. fieldPtr is always a pointer to the addressable field (e.g.
// **FilterString, **string, *SomeUnion). The type switch handles every
// known filter type statically; reflect is only used for the generic
// json.Unmarshaler fallback, which exists to support generated union types
// (TypeSpec-produced oneOf fields) that cannot be enumerated at compile time.
//
// Adding a new filter type means adding a case here — the compiler will not
// force it, but parseUnknownField's fallback will silently ignore unknown
// concrete types, so test coverage must catch the miss.
func dispatchFieldParse(qs url.Values, name string, fieldPtr any) error {
	switch p := fieldPtr.(type) {
	case **FilterString:
		parsed, err := parseFilterString(qs, name)
		if err != nil {
			return err
		}
		*p = &parsed
		return nil

	case **FilterStringExact:
		parsed, err := parseFilterStringExact(qs, name)
		if err != nil {
			return err
		}
		*p = &parsed
		return nil

	case **FilterNumeric:
		parsed, err := parseFilterNumeric(qs, name)
		if err != nil {
			return err
		}
		*p = &parsed
		return nil

	case **FilterDateTime:
		parsed, err := parseFilterDateTime(qs, name)
		if err != nil {
			return err
		}
		*p = &parsed
		return nil

	case **FilterBoolean:
		parsed, err := parseFilterBoolean(qs, name)
		if err != nil {
			return err
		}
		*p = &parsed
		return nil

	case **string:
		// Simple string field: filter[field]=value (shorthand eq).
		prefix := fmt.Sprintf("filter[%s]", name)
		for key, values := range qs {
			if key != prefix {
				continue
			}
			val, err := singleValue(key, values)
			if err != nil {
				return err
			}
			if val == "" {
				return nil
			}
			*p = &val
			return nil
		}
		return nil

	default:
		return parseUnknownField(qs, name, fieldPtr)
	}
}

// parseUnknownField is the reflect-based escape hatch for filter types not
// covered by dispatchFieldParse's static cases. It builds a JSON object from
// the query params and delegates to json.Unmarshaler if the field type
// implements it. This path exists to support generated union types
// (TypeSpec-produced oneOf fields) whose concrete type is not known at
// compile time and therefore cannot be added to the type switch.
func parseUnknownField(qs url.Values, name string, fieldPtr any) error {
	obj, err := buildFilterJSON(qs, name)
	if err != nil {
		return err
	}
	if obj == nil {
		return nil
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("filter[%s]: %w", name, err)
	}

	// fieldPtr is a pointer to the field; the field itself may already be a
	// pointer (fieldPtr is then a pointer-to-pointer). Unwrap one level to get
	// the value we want to unmarshal into, and allocate it if the inner
	// pointer is nil.
	fieldVal := reflect.ValueOf(fieldPtr).Elem()
	target := fieldVal
	if fieldVal.Kind() == reflect.Pointer {
		target = reflect.New(fieldVal.Type().Elem())
	}
	unmarshaler, ok := target.Interface().(json.Unmarshaler)
	if !ok {
		return nil
	}
	if err := unmarshaler.UnmarshalJSON(data); err != nil {
		return fmt.Errorf("filter[%s]: %w", name, err)
	}
	fieldVal.Set(target)
	return nil
}

// buildFilterJSON collects filter[field][op]=value entries into a JSON-serializable
// object. If the param is filter[field]=value (no operator), it returns the plain
// string value (shorthand for eq). If operators are present, it returns a
// map[string]any. When both the shorthand and one or more operator forms are
// present for the same field, the operator form wins (deterministic regardless
// of Go map iteration order) and the shorthand value is stored as the "eq"
// operator on the resulting object.
//
// Enforces singleValue semantics on every key — repeated parameters are a 400.
func buildFilterJSON(qs url.Values, field string) (any, error) {
	prefix := fmt.Sprintf("filter[%s]", field)

	var (
		obj          map[string]any
		shorthand    string
		hasShorthand bool
	)

	// Collect matching keys first so the final object is deterministic even
	// though url.Values iteration order is random.
	type entry struct {
		key, rest, value string
	}
	var entries []entry

	for key, values := range qs {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		v, err := singleValue(key, values)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry{key: key, rest: key[len(prefix):], value: v})
	}
	// Sort for determinism.
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

	for _, e := range entries {
		if e.rest == "" {
			if e.value == "" {
				// Bare key existence: filter[field] with no value → exists: true
				if obj == nil {
					obj = make(map[string]any)
				}
				obj["exists"] = true
				continue
			}
			// Shorthand: filter[field]=value → candidate eq (operator wins later).
			shorthand = e.value
			hasShorthand = true
			continue
		}

		// rest should be "[op]"
		if strings.HasPrefix(e.rest, "[") && strings.HasSuffix(e.rest, "]") {
			op := e.rest[1 : len(e.rest)-1]
			if obj == nil {
				obj = make(map[string]any)
			}
			if e.value != "" {
				obj[op] = e.value
			} else {
				obj[op] = true
			}
		}
	}

	if obj != nil {
		return obj, nil
	}
	if hasShorthand {
		return shorthand, nil
	}
	return nil, nil
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

type parsedFilterParam struct {
	op    string
	value string
	bare  bool
}

func forEachFieldParam(qs url.Values, field string, visit func(parsedFilterParam) error) error {
	prefix := fmt.Sprintf("filter[%s]", field)

	for key, values := range qs {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		value, err := singleValue(key, values)
		if err != nil {
			return err
		}

		rest := key[len(prefix):]
		op, err := parseOperator(rest)
		if err != nil {
			return fmt.Errorf("invalid filter parameter %q: %w", key, err)
		}

		if err := visit(parsedFilterParam{
			op:    op,
			value: value,
			bare:  rest == "" && value == "",
		}); err != nil {
			return err
		}
	}

	return nil
}

// parseFilterString extracts a FilterString from URL query values for
// the given field name.  It understands the deepObject encoding used by our API:
//
//	filter[field]=value          → eq (shorthand)
//	filter[field][eq]=value      → eq
//	filter[field][neq]=value     → neq
//	filter[field][contains]=value → contains
//	filter[field][oeq]=a,b,c     → oeq (comma-separated)
//	filter[field][ocontains]=a,b → ocontains (comma-separated)
//	filter[field][gt]=value      → gt
//	filter[field][gte]=value     → gte
//	filter[field][lt]=value      → lt
//	filter[field][lte]=value     → lte
//	filter[field][exists]        → exists (true)
//	filter[field][nexists]       → exists (false)
func parseFilterString(qs url.Values, field string) (FilterString, error) {
	var f FilterString

	err := forEachFieldParam(qs, field, func(param parsedFilterParam) error {
		// Bare key with no value: ?filter[field] → exists
		if param.bare {
			t := true
			f.Exists = &t
			return nil
		}

		switch param.op {
		case "eq":
			f.Eq = &param.value
		case "neq":
			f.Neq = &param.value
		case "contains":
			f.Contains = &param.value
		case "oeq":
			items, err := parseCommaSeparated(param.value)
			if err != nil {
				return fmt.Errorf("filter[%s][oeq]: %w", field, err)
			}
			f.Oeq = items
		case "ocontains":
			items, err := parseCommaSeparated(param.value)
			if err != nil {
				return fmt.Errorf("filter[%s][ocontains]: %w", field, err)
			}
			f.Ocontains = items
		case "gt":
			f.Gt = &param.value
		case "gte":
			f.Gte = &param.value
		case "lt":
			f.Lt = &param.value
		case "lte":
			f.Lte = &param.value
		case "exists":
			t := true
			f.Exists = &t
		case "nexists":
			fa := false
			f.Exists = &fa
		default:
			return fmt.Errorf("unsupported operator %q for string filter on field %q", param.op, field)
		}

		return nil
	})

	return f, err
}

// parseFilterStringExact extracts a FilterStringExact from URL query values.
// Supports eq, neq, and oeq operators.
func parseFilterStringExact(qs url.Values, field string) (FilterStringExact, error) {
	var f FilterStringExact

	err := forEachFieldParam(qs, field, func(param parsedFilterParam) error {
		switch param.op {
		case "eq":
			f.Eq = &param.value
		case "neq":
			f.Neq = &param.value
		case "oeq":
			items, err := parseCommaSeparated(param.value)
			if err != nil {
				return fmt.Errorf("filter[%s][oeq]: %w", field, err)
			}
			f.Oeq = items
		default:
			return fmt.Errorf("unsupported operator %q for exact string filter on field %q", param.op, field)
		}

		return nil
	})

	return f, err
}

// parseFilterNumeric extracts a FilterNumeric from URL query values.
func parseFilterNumeric(qs url.Values, field string) (FilterNumeric, error) {
	var f FilterNumeric

	err := forEachFieldParam(qs, field, func(param parsedFilterParam) error {
		if param.bare {
			return nil
		}

		switch param.op {
		case "eq":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Eq = &v
		case "neq":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Neq = &v
		case "oeq":
			items, err := parseCommaSeparated(param.value)
			if err != nil {
				return fmt.Errorf("filter[%s][oeq]: %w", field, err)
			}
			for _, s := range items {
				v, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return fmt.Errorf("filter[%s][oeq]: invalid number %q: %w", field, s, err)
				}
				f.Oeq = append(f.Oeq, v)
			}
		case "gt":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Gt = &v
		case "gte":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Gte = &v
		case "lt":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Lt = &v
		case "lte":
			v, err := parseFloatFilterValue(field, param.op, param.value)
			if err != nil {
				return err
			}
			f.Lte = &v
		default:
			return fmt.Errorf("unsupported operator %q for numeric filter on field %q", param.op, field)
		}

		return nil
	})

	return f, err
}

// parseFilterDateTime extracts a FilterDateTime from URL query values.
// Values are kept as strings (expected to be RFC-3339 timestamps).
func parseFilterDateTime(qs url.Values, field string) (FilterDateTime, error) {
	var f FilterDateTime

	err := forEachFieldParam(qs, field, func(param parsedFilterParam) error {
		if param.bare {
			return nil
		}

		switch param.op {
		case "eq":
			f.Eq = &param.value
		case "gt":
			f.Gt = &param.value
		case "gte":
			f.Gte = &param.value
		case "lt":
			f.Lt = &param.value
		case "lte":
			f.Lte = &param.value
		default:
			return fmt.Errorf("unsupported operator %q for datetime filter on field %q", param.op, field)
		}

		return nil
	})

	return f, err
}

// parseFilterBoolean extracts a FilterBoolean from URL query values.
func parseFilterBoolean(qs url.Values, field string) (FilterBoolean, error) {
	var f FilterBoolean

	err := forEachFieldParam(qs, field, func(param parsedFilterParam) error {
		switch param.op {
		case "eq":
			v, err := strconv.ParseBool(param.value)
			if err != nil {
				return fmt.Errorf("filter[%s][eq]: invalid boolean %q: %w", field, param.value, err)
			}
			f.Eq = &v
		default:
			return fmt.Errorf("unsupported operator %q for boolean filter on field %q", param.op, field)
		}

		return nil
	})

	return f, err
}

func parseFloatFilterValue(field, op, value string) (float64, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("filter[%s][%s]: invalid number %q: %w", field, op, value, err)
	}

	return v, nil
}

// parseOperator extracts the operator from the rest of a deep-object key.
//
//	""          → "eq"  (shorthand: filter[field]=value means eq)
//	"[eq]"      → "eq"
//	"[contains]" → "contains"
func parseOperator(rest string) (string, error) {
	if rest == "" {
		return "eq", nil
	}

	if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
		return "", fmt.Errorf("malformed operator segment %q", rest)
	}

	op := rest[1 : len(rest)-1]
	if op == "" {
		return "", fmt.Errorf("empty operator in %q", rest)
	}

	switch op {
	case "eq", "neq", "contains", "oeq", "ocontains", "gt", "gte", "lt", "lte", "exists", "nexists":
		return op, nil
	default:
		return "", fmt.Errorf("unknown filter operator %q", op)
	}
}

// parseCommaSeparated splits a filter value on commas, trims whitespace, drops
// empty entries, and enforces maxCommaSeparatedItems. Returns an error when the
// input would produce more than maxCommaSeparatedItems values.
func parseCommaSeparated(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
			if len(out) > maxCommaSeparatedItems {
				return nil, ErrTooManyItems
			}
		}
	}
	return out, nil
}

// singleValue enforces that a repeated query parameter only carries a single
// value. url.Values silently preserves duplicates (`?f=a&f=a`), and the parser
// would otherwise drop everything after values[0], causing silent behavioral
// divergence between client intent and server state. Returning an error forces
// callers to disambiguate at the boundary.
//
// It also enforces maxFilterValueLength on the accepted value.
func singleValue(key string, values []string) (string, error) {
	if len(values) > 1 {
		return "", fmt.Errorf("filter parameter %q: repeated query parameter not allowed (got %d values)", key, len(values))
	}
	if len(values) == 0 {
		return "", nil
	}
	v := values[0]
	if len(v) > maxFilterValueLength {
		return "", fmt.Errorf("filter parameter %q: value too long (max %d bytes)", key, maxFilterValueLength)
	}
	return v, nil
}
