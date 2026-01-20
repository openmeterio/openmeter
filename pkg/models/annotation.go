package models

import (
	"math"
	"reflect"

	"github.com/brunoga/deep"
)

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

	switch t := val.(type) {
	case int:
		return t, true
	case float32:
		f := float64(t)

		if f != math.Trunc(f) || float64(math.MaxInt) < f || f < float64(math.MinInt) {
			return 0, false
		}

		return int(t), true
	case float64:
		if t != math.Trunc(t) || float64(math.MaxInt) < t || t < float64(math.MinInt) {
			return 0, false
		}

		return int(t), true
	default:
		return 0, false
	}
}

func (a Annotations) Reset() {
	for k := range a {
		delete(a, k)
	}
}

func (a Annotations) Clone() (Annotations, error) {
	if a == nil {
		return nil, nil
	}

	return deep.Copy[Annotations](a)
}

func (a Annotations) Merge(m Annotations) (Annotations, error) {
	if a == nil {
		return m, nil
	}

	result, err := a.Clone()
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return result, nil
	}

	for k, v := range m {
		vv, err := deep.Copy(v)
		if err != nil {
			return nil, err
		}

		result[k] = vv
	}

	return result, nil
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
