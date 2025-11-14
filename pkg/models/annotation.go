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

func (a Annotations) GetString(key string) (string, bool) {
	if len(a) == 0 {
		return "", false
	}

	val, ok := a[key]
	if !ok {
		return "", false
	}

	strVal, ok := val.(string)
	if !ok {
		return "", false
	}

	return strVal, true
}

func (a Annotations) GetInt(key string) (int, bool) {
	if len(a) == 0 {
		return 0, false
	}

	val, ok := a[key]
	if !ok {
		return 0, false
	}

	intVal, ok := val.(int)
	if !ok {
		return 0, false
	}

	return intVal, true
}

func (a Annotations) Clone() Annotations {
	if a == nil {
		return nil
	}

	result := make(Annotations)

	for k, v := range a {
		result[k] = v
	}

	return result
}

func (a Annotations) Merge(m Annotations) Annotations {
	if a == nil {
		return m
	}

	result := a.Clone()

	if len(m) == 0 {
		return result
	}

	for k, v := range m {
		result[k] = v
	}

	return result
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
