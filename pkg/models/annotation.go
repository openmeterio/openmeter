package models

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
