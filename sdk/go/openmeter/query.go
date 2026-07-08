package openmeter

import (
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
// filter[key][contains]=..., and so on for each set operator.
func addStringFilter(q url.Values, prefix string, f *StringFilter) {
	if f == nil {
		return
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
		setDeepObjectString(q, prefix, "oeq", strings.Join(f.Oeq, ","))
	}
	if len(f.Ocontains) > 0 {
		setDeepObjectString(q, prefix, "ocontains", strings.Join(f.Ocontains, ","))
	}
	if f.Exists != nil {
		setDeepObjectString(q, prefix, "$exists", strconv.FormatBool(*f.Exists))
	}
}
