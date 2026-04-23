package filters

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

const (
	maxCommaSeparatedItems = 50
	maxFilterValueLength   = 1024
)

// Operator constants used in filter[field][op]=value query parameters.
const (
	OpEq        = "eq"
	OpNeq       = "neq"
	OpGt        = "gt"
	OpGte       = "gte"
	OpLt        = "lt"
	OpLte       = "lte"
	OpContains  = "contains"
	OpOeq       = "oeq"
	OpOcontains = "ocontains"
	OpExists    = "exists"
	OpNexists   = "nexists"
)

var (
	ErrTooManyItems        = fmt.Errorf("too many comma-separated items (max %d)", maxCommaSeparatedItems)
	ErrUnsupportedOperator = errors.New("unsupported operator")
	ErrInvalidNumber       = errors.New("invalid number")
	ErrInvalidDateTime     = errors.New("invalid datetime")
)

// Parse populates a filter struct from URL query values.
// The target must be a pointer to a struct (or pointer-to-pointer-to-struct)
// whose fields are filter types. Field names are resolved from json struct tags.
func Parse(qs url.Values, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("Parse: target must be a non-nil pointer")
	}

	v = v.Elem()
	if v.Kind() == reflect.Pointer {
		if v.Type().Elem().Kind() != reflect.Struct {
			return fmt.Errorf("Parse: target must point to a struct or *struct, got **%s", v.Type().Elem().Kind())
		}
		if !hasFilterKeys(qs) {
			return nil
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("Parse: target must point to a struct or *struct, got %s", v.Kind())
	}

	return parseFiltersValue(qs, v)
}

// fieldError produces a consistent "filter[field][op]: reason" error.
func fieldError(field, op string, err error) error {
	return fmt.Errorf("filter[%s][%s]: %w", field, op, err)
}

var (
	filterStringType      = reflect.TypeFor[*FilterString]()
	filterStringExactType = reflect.TypeFor[*FilterStringExact]()
	FilterULIDType        = reflect.TypeFor[*FilterULID]()
	filterNumericType     = reflect.TypeFor[*FilterNumeric]()
	filterDateTimeType    = reflect.TypeFor[*FilterDateTime]()
	filterBooleanType     = reflect.TypeFor[*FilterBoolean]()
	filterLabelsType      = reflect.TypeFor[FilterLabels]()
	stringPtrType         = reflect.TypeFor[*string]()
)

// parseFiltersValue iterates struct fields and dispatches to per-type parsers.
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

		// FilterLabels uses dot-notation: filter[labels.env][eq]=prod
		if fieldVal.Type() == filterLabelsType {
			labels, err := parseFilterLabels(qs, name)
			if err != nil {
				return err
			}
			if labels != nil {
				fieldVal.Set(reflect.ValueOf(labels))
			}
			continue
		}

		if !hasFieldKeys(qs, name) {
			continue
		}

		switch fieldVal.Type() {
		case filterStringType:
			parsed, err := parseFilterString(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case filterStringExactType:
			parsed, err := parseFilterStringExact(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case FilterULIDType:
			parsed, err := parseFilterULID(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case filterNumericType:
			parsed, err := parseFilterNumeric(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case filterDateTimeType:
			parsed, err := parseFilterDateTime(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case filterBooleanType:
			parsed, err := parseFilterBoolean(qs, name)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(&parsed))

		case stringPtrType:
			if err := parseStringPtr(qs, name, fieldVal); err != nil {
				return err
			}

		default:
			return fmt.Errorf("filter[%s]: unsupported filter field type %s", name, fieldVal.Type())
		}
	}

	return nil
}

// parseStringPtr handles simple filter[field]=value for *string fields.
func parseStringPtr(qs url.Values, name string, fieldVal reflect.Value) error {
	prefix := "filter[" + name + "]"
	for key, values := range qs {
		if key != prefix {
			continue
		}
		val, err := singleValue(key, values)
		if err != nil {
			return err
		}
		if val != "" {
			fieldVal.Set(reflect.ValueOf(&val))
		}
		break
	}
	return nil
}

// parseFilterString extracts a FilterString supporting all string operators.
func parseFilterString(qs url.Values, field string) (FilterString, error) {
	var f FilterString

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		if p.bare {
			// Bare filter[field] means an existence check.
			f.Exists = lo.ToPtr(true)
			return nil
		}

		switch p.op {
		case OpEq:
			f.Eq = &p.value
		case OpNeq:
			f.Neq = &p.value
		case OpContains:
			f.Contains = &p.value
		case OpGt:
			f.Gt = &p.value
		case OpGte:
			f.Gte = &p.value
		case OpLt:
			f.Lt = &p.value
		case OpLte:
			f.Lte = &p.value
		case OpOeq:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Oeq = items
		case OpOcontains:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Ocontains = items
		case OpExists:
			f.Exists = lo.ToPtr(true)
		case OpNexists:
			f.Exists = lo.ToPtr(false)
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
		return nil
	})

	return f, err
}

// parseFilterStringExact extracts a FilterStringExact supporting eq, neq, and oeq.
func parseFilterStringExact(qs url.Values, field string) (FilterStringExact, error) {
	var f FilterStringExact

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		switch p.op {
		case OpEq:
			f.Eq = &p.value
		case OpNeq:
			f.Neq = &p.value
		case OpOeq:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Oeq = items
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
		return nil
	})

	return f, err
}

// parseFilterULID extracts a FilterULID supporting all string operators.
func parseFilterULID(qs url.Values, field string) (FilterULID, error) {
	var f FilterULID

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		if p.bare {
			f.Exists = lo.ToPtr(true)
			return nil
		}

		switch p.op {
		case OpEq:
			f.Eq = &p.value
		case OpNeq:
			f.Neq = &p.value
		case OpContains:
			f.Contains = &p.value
		case OpOeq:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Oeq = items
		case OpOcontains:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Ocontains = items
		case OpExists:
			f.Exists = lo.ToPtr(true)
		case OpNexists:
			f.Exists = lo.ToPtr(false)
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
		return nil
	})

	return f, err
}

// parseFilterNumeric extracts a FilterNumeric, parsing values as float64.
func parseFilterNumeric(qs url.Values, field string) (FilterNumeric, error) {
	var f FilterNumeric

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		if p.bare {
			return fmt.Errorf("filter[%s]: empty numeric value", field)
		}

		switch p.op {
		case OpEq:
			return parseFloat(field, p, &f.Eq)
		case OpNeq:
			return parseFloat(field, p, &f.Neq)
		case OpGt:
			return parseFloat(field, p, &f.Gt)
		case OpGte:
			return parseFloat(field, p, &f.Gte)
		case OpLt:
			return parseFloat(field, p, &f.Lt)
		case OpLte:
			return parseFloat(field, p, &f.Lte)
		case OpOeq:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			for _, s := range items {
				v, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return fmt.Errorf("filter[%s][oeq]: invalid number %q: %w", field, s, err)
				}
				f.Oeq = append(f.Oeq, v)
			}
			return nil
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
	})

	return f, err
}

// parseFilterDateTime extracts a FilterDateTime, parsing values as RFC-3339 timestamps.
func parseFilterDateTime(qs url.Values, field string) (FilterDateTime, error) {
	var f FilterDateTime

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		if p.bare {
			f.Exists = lo.ToPtr(true)
			return nil
		}

		switch p.op {
		case OpEq:
			return parseTime(field, p, &f.Eq)
		case OpGt:
			return parseTime(field, p, &f.Gt)
		case OpGte:
			return parseTime(field, p, &f.Gte)
		case OpLt:
			return parseTime(field, p, &f.Lt)
		case OpLte:
			return parseTime(field, p, &f.Lte)
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
	})

	return f, err
}

// parseFilterBoolean extracts a FilterBoolean supporting only eq.
func parseFilterBoolean(qs url.Values, field string) (FilterBoolean, error) {
	var f FilterBoolean

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		switch p.op {
		case OpEq:
			v, err := strconv.ParseBool(p.value)
			if err != nil {
				return fmt.Errorf("filter[%s][eq]: invalid boolean %q: %w", field, p.value, err)
			}
			f.Eq = &v
			return nil
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
	})

	return f, err
}

// parseFilterLabel extracts a FilterLabel for a single dot-notation label key.
func parseFilterLabel(qs url.Values, field string) (FilterLabel, error) {
	var f FilterLabel

	err := forEachFieldParam(qs, field, func(p parsedFilterParam) error {
		switch p.op {
		case OpEq:
			f.Eq = &p.value
		case OpNeq:
			f.Neq = &p.value
		case OpContains:
			f.Contains = &p.value
		case OpOeq:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Oeq = items
		case OpOcontains:
			items, err := parseCommaSeparatedField(field, p.op, p.value)
			if err != nil {
				return err
			}
			f.Ocontains = items
		default:
			return fieldError(field, p.op, ErrUnsupportedOperator)
		}
		return nil
	})

	return f, err
}

// parseFilterLabels collects dot-notation filter params into a FilterLabels map.
func parseFilterLabels(qs url.Values, field string) (FilterLabels, error) {
	prefix := "filter[" + field + "."

	labelKeys := make(map[string]struct{})
	for key := range qs {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := key[len(prefix):]
		end := strings.IndexByte(rest, ']')
		if end <= 0 {
			continue
		}
		labelKeys[rest[:end]] = struct{}{}
	}

	if len(labelKeys) == 0 {
		return nil, nil
	}

	result := make(FilterLabels, len(labelKeys))
	for labelKey := range labelKeys {
		parsed, err := parseFilterLabel(qs, field+"."+labelKey)
		if err != nil {
			return nil, err
		}
		result[labelKey] = parsed
	}
	return result, nil
}

// parseFloat parses the param value as float64 and assigns it to dst.
func parseFloat(field string, p parsedFilterParam, dst **float64) error {
	v, err := strconv.ParseFloat(p.value, 64)
	if err != nil {
		return fieldError(field, p.op, ErrInvalidNumber)
	}
	*dst = &v
	return nil
}

// parseTime parses the param value as RFC-3339 timestamp and assigns it to dst.
func parseTime(field string, p parsedFilterParam, dst **time.Time) error {
	v, err := time.Parse(time.RFC3339, p.value)
	if err != nil {
		return fieldError(field, p.op, ErrInvalidDateTime)
	}
	*dst = &v
	return nil
}

// parseCommaSeparatedField splits a comma-separated value with field-level error wrapping.
func parseCommaSeparatedField(field, op, value string) ([]string, error) {
	items, err := parseCommaSeparated(value)
	if err != nil {
		return nil, fieldError(field, op, err)
	}
	return items, nil
}

// parsedFilterParam holds a single parsed filter[field][op]=value entry.
type parsedFilterParam struct {
	op    string
	value string
	bare  bool
}

// forEachFieldParam iterates over all filter[field][op]=value entries for a given field.
func forEachFieldParam(qs url.Values, field string, visit func(parsedFilterParam) error) error {
	prefix := "filter[" + field + "]"

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

// hasFilterKeys reports whether any query key starts with "filter[".
func hasFilterKeys(qs url.Values) bool {
	for key := range qs {
		if strings.HasPrefix(key, "filter[") {
			return true
		}
	}
	return false
}

// hasFieldKeys reports whether qs contains filter[field] or filter[field][op] keys.
func hasFieldKeys(qs url.Values, field string) bool {
	prefix := "filter[" + field + "]"
	for key := range qs {
		if key == prefix || strings.HasPrefix(key, prefix+"[") {
			return true
		}
	}
	return false
}

// checkUnknownFilterKeys rejects filter keys not present in the knownFields set.
func checkUnknownFilterKeys(qs url.Values, knownFields map[string]struct{}) error {
	var unknown []string
	seen := make(map[string]struct{})

	for key := range qs {
		name, ok := filterFieldName(key)
		if !ok {
			continue
		}
		// Dot-notation: match base segment ("labels" in "labels.env").
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

// filterFieldName extracts the field name from a "filter[name]" or "filter[name][op]" key.
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

// jsonFieldName returns the json tag name for a struct field.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// parseOperator extracts the operator string from a "[op]" suffix.
func parseOperator(rest string) (string, error) {
	if rest == "" {
		return OpEq, nil
	}
	if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
		return "", fmt.Errorf("malformed operator segment %q", rest)
	}
	op := rest[1 : len(rest)-1]
	if op == "" {
		return "", fmt.Errorf("empty operator in %q", rest)
	}
	return op, nil
}

// parseCommaSeparated splits on commas, trims whitespace, and enforces the item cap.
func parseCommaSeparated(value string) ([]string, error) {
	items := lo.FilterMap(strings.Split(value, ","), func(p string, _ int) (string, bool) {
		s := strings.TrimSpace(p)
		if s == "" {
			return "", false
		}
		return s, true
	})
	if len(items) > maxCommaSeparatedItems {
		return nil, ErrTooManyItems
	}
	return items, nil
}

// singleValue enforces single-value and max-length constraints on a query parameter.
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
