package models

import "reflect"

type Annotations map[string]interface{}

func (a Annotations) GetBool(key string) bool {
	if len(a) == 0 {
		return false
	}

	val, ok := a[key]
	if !ok {
		return false
	}

	boolVal, ok := val.(bool)
	if !ok {
		return false
	}

	return boolVal
}

func (a Annotations) Equal(other Annotations) bool {
	if a == nil || other == nil {
		return a == nil && other == nil
	}

	if len(a) != len(other) {
		return false
	}

	for k, v := range a {
		otherV, ok := other[k]
		if !ok {
			return false
		}

		if !reflect.DeepEqual(v, otherV) {
			return false
		}
	}
	return true
}
