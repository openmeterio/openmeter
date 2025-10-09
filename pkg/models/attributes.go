package models

import (
	"fmt"
	"reflect"
)

type Attributes map[any]any

func (a Attributes) Clone() Attributes {
	if a == nil {
		return nil
	}

	m := make(Attributes)

	if len(a) == 0 {
		return m
	}

	for k, v := range a {
		m[k] = v
	}

	return m
}

// AsStringMap converts Attributes into a map[string]any by:
// - keeping string keys as-is
// - stringifying comparable non-string keys as "<type>:<value>"
func (a Attributes) AsStringMap() map[string]any {
	if len(a) == 0 {
		return nil
	}

	out := make(map[string]any, len(a))
	for k, v := range a {
		if sk, ok := k.(string); ok {
			out[sk] = v
			continue
		}

		t := reflect.TypeOf(k)
		if t == nil {
			continue
		}
		if t.Comparable() {
			key := fmt.Sprintf("%T:%v", k, k)
			out[key] = v
		}
	}

	return out
}

func (a Attributes) Merge(m Attributes) Attributes {
	if len(m) == 0 {
		return a.Clone()
	}

	r := make(Attributes, len(a)+len(m))

	for k, v := range a {
		r[k] = v
	}

	for k, v := range m {
		r[k] = v
	}

	return r
}
