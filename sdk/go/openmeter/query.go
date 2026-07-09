package openmeter

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// This file implements the query-string serialization styles the OpenMeter v3
// meter-list endpoint uses. Three distinct OpenAPI styles appear on one
// operation, and each is handled explicitly here:
//
//   - page   deepObject         -> page[size]=10&page[number]=2
//   - sort    form, explode=false -> sort=name,-created_at   (see MetersService.List)
//   - filter deepObject (nested) -> filter[key][eq]=foo
//
// A TypeSpec emitter targeting this SDK shape would generate the equivalent of
// these helpers from the parameter styles declared in the spec.

// setDeepObjectString sets a single-level deepObject member: prefix[key]=value.
func setDeepObjectString(q url.Values, prefix, key, value string) {
	q.Set(prefix+"["+key+"]", value)
}

// addStringFilter serializes a StringFilter as a nested deepObject under prefix,
// e.g. addStringFilter(q, "filter[key]", f) yields filter[key][eq]=...,
// filter[key][contains]=..., and so on for each set operator. It returns an
// error if a one-of value cannot be represented (see joinFilterList).
func addStringFilter(q url.Values, prefix string, f *StringFilter) error {
	if f == nil {
		return nil
	}

	if f.Eq != nil {
		setDeepObjectString(q, prefix, "eq", *f.Eq)
	}
	if f.Neq != nil {
		setDeepObjectString(q, prefix, "neq", *f.Neq)
	}
	if f.Contains != nil {
		setDeepObjectString(q, prefix, "contains", *f.Contains)
	}
	if f.Gt != nil {
		setDeepObjectString(q, prefix, "gt", *f.Gt)
	}
	if f.Gte != nil {
		setDeepObjectString(q, prefix, "gte", *f.Gte)
	}
	if f.Lt != nil {
		setDeepObjectString(q, prefix, "lt", *f.Lt)
	}
	if f.Lte != nil {
		setDeepObjectString(q, prefix, "lte", *f.Lte)
	}
	if len(f.Oeq) > 0 {
		v, err := joinFilterList(f.Oeq)
		if err != nil {
			return fmt.Errorf("%s[oeq]: %w", prefix, err)
		}
		setDeepObjectString(q, prefix, "oeq", v)
	}
	if len(f.Ocontains) > 0 {
		v, err := joinFilterList(f.Ocontains)
		if err != nil {
			return fmt.Errorf("%s[ocontains]: %w", prefix, err)
		}
		setDeepObjectString(q, prefix, "ocontains", v)
	}
	if f.Exists != nil {
		setDeepObjectString(q, prefix, "$exists", strconv.FormatBool(*f.Exists))
	}

	return nil
}

// joinFilterList renders a one-of list (oeq/ocontains) as the comma-separated
// value the API expects, e.g. filter[key][oeq]=a,b. The API splits this list on
// commas and provides no escape for a comma within a value — repeated query
// parameters for the same key are rejected server-side — so a value containing
// a comma cannot be represented. Rather than silently send it as multiple
// values, joinFilterList rejects it so the caller gets a clear error.
func joinFilterList(values []string) (string, error) {
	for _, v := range values {
		if strings.Contains(v, ",") {
			return "", fmt.Errorf("value %q contains a comma, which the comma-separated one-of filter encoding cannot represent", v)
		}
	}

	return strings.Join(values, ","), nil
}
