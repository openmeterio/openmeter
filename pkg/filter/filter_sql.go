package filter

import (
	"fmt"
	"regexp"
	"strings"
)

type CompareType string

var (
	// Equality
	CompareTypeEq  CompareType = "$eq"
	CompareTypeGt  CompareType = "$gt"
	CompareTypeGte CompareType = "$gte"
	CompareTypeLt  CompareType = "$lt"
	CompareTypeLte CompareType = "$lte"
	CompareTypeNe  CompareType = "$ne"

	// String
	CompareTypeLike    CompareType = "$like"
	CompareTypeNotLike CompareType = "$notLike"
	CompareTypeMatch   CompareType = "$match"

	// Array
	CompareTypeIn  CompareType = "$in"
	CompareTypeNin CompareType = "$nin"
)

func ToSQL(field string, filter Filter) (string, error) {
	return traverse(field, filter)
}

// TODO: optimize this function
func Validate(filter Filter) error {
	_, err := ToSQL("", filter)
	return err
}

func traverse(field string, filter Filter) (string, error) {
	var results []string
	if filter.And != nil || filter.Or != nil {
		var childFilters []Filter

		if filter.And != nil {
			childFilters = *filter.And
		} else if filter.Or != nil {
			childFilters = *filter.Or
		}

		for _, childFilter := range childFilters {
			result, err := traverse(field, childFilter)
			if err != nil {
				return "", err
			}
			results = append(results, result)
		}

		if filter.And != nil {
			return controlAnd(results), nil
		} else if filter.Or != nil {
			return controlOr(results), nil
		}
	} else if filter.Not != nil {
		result, err := traverse(field, *filter.Not)
		if err != nil {
			return "", err
		}
		return controlNot(result)
	} else if filter.Not != nil || filter.Eq != nil || filter.Ne != nil || filter.In != nil || filter.Nin != nil || filter.Gt != nil || filter.Gte != nil || filter.Lt != nil || filter.Lte != nil || filter.Like != nil || filter.NotLike != nil || filter.Match != nil {
		var value FilterValue
		var compareType CompareType

		if filter.Eq != nil {
			value = *filter.Eq
			compareType = CompareTypeEq
		} else if filter.Ne != nil {
			value = *filter.Ne
			compareType = CompareTypeNe
		} else if filter.In != nil {
			value = *filter.In
			compareType = CompareTypeIn
		} else if filter.Nin != nil {
			value = *filter.Nin
			compareType = CompareTypeNin
		} else if filter.Gt != nil {
			value = *filter.Gt
			compareType = CompareTypeGt
		} else if filter.Gte != nil {
			value = *filter.Gte
			compareType = CompareTypeGte
		} else if filter.Lt != nil {
			value = *filter.Lt
			compareType = CompareTypeLt
		} else if filter.Lte != nil {
			value = *filter.Lte
			compareType = CompareTypeLte
		} else if filter.Like != nil {
			value = *filter.Like
			compareType = CompareTypeLike
		} else if filter.NotLike != nil {
			value = *filter.NotLike
			compareType = CompareTypeNotLike
		} else if filter.Match != nil {
			value = *filter.Match
			compareType = CompareTypeMatch
		}

		// Value is number
		if v, err := value.AsFilterValueNumber(); err == nil {
			return filterPrimitive(compareType, field, v)
		}

		// Value is string
		if v, err := value.AsFilterValueString(); err == nil {
			return filterPrimitive(compareType, field, v)
		}

		// Values is string array
		if v, err := value.AsFilterValueArrayString(); err == nil {
			return filterArray(compareType, field, v)
		}

		// Values is number array
		if v, err := value.AsFilterValueArrayNumber(); err == nil {
			return filterArray(compareType, field, v)
		}

		return "", fmt.Errorf("invalid value: %s", value)
	}

	return "", fmt.Errorf("unsupported filter")
}

func filterPrimitive[T JSONUnmarshald](compareType CompareType, field string, value T) (string, error) {
	switch compareType {
	case CompareTypeEq:
		return filterArithmetic(compareType, field, "=", value)
	case CompareTypeNe:
		return filterArithmetic(compareType, field, "!=", value)
	case CompareTypeGt:
		return filterArithmetic(compareType, field, ">", value)
	case CompareTypeGte:
		return filterArithmetic(compareType, field, ">=", value)
	case CompareTypeLt:
		return filterArithmetic(compareType, field, "<", value)
	case CompareTypeLte:
		return filterArithmetic(compareType, field, "<=", value)
	case CompareTypeLike:
		return filterLike(field, value)
	case CompareTypeNotLike:
		return filterNotLike(field, value)
	case CompareTypeMatch:
		return filterMatch(field, value)
	}

	return "", fmt.Errorf("invalid filter type: %s", compareType)
}

func filterArray[T JSONUnmarshald](compareType CompareType, field string, value []T) (string, error) {
	switch compareType {
	case CompareTypeIn:
		return filterIn(field, value)
	case CompareTypeNin:
		return filterNin(field, value)
	}

	return "", fmt.Errorf("invalid filter type: %s", compareType)
}

func controlAnd(results []string) string {
	return fmt.Sprintf("(%s)", strings.Join(results, " AND "))
}
func controlOr(results []string) string {
	return fmt.Sprintf("(%s)", strings.Join(results, " OR "))
}

func controlNot(result string) (string, error) {
	return fmt.Sprintf(`NOT (%v)`, result), nil
}

func filterLike[T JSONUnmarshald](field string, value T) (string, error) {
	switch v := any(value).(type) {
	case string:
		return fmt.Sprintf(`%s LIKE %s`, field, wrapString(v)), nil
	}
	return "", fmt.Errorf("unsupported $like value")
}

func filterNotLike[T JSONUnmarshald](field string, value T) (string, error) {
	switch v := any(value).(type) {
	case string:
		return fmt.Sprintf(`%s NOT LIKE %v`, field, wrapString(v)), nil
	}
	return "", fmt.Errorf("unsupported $notLike value")
}

func filterMatch[T JSONUnmarshald](field string, value T) (string, error) {
	switch v := any(value).(type) {
	case string:
		_, err := regexp.Compile(v)
		if err != nil {
			return "", fmt.Errorf("$match value has to be a valid regexp string")
		}

		return fmt.Sprintf(`match(%s, /%v/)`, field, value), nil
	}
	return "", fmt.Errorf("unsupported $match value")
}

func filterIn[T JSONUnmarshald](field string, values []T) (string, error) {
	items := []string{}

	for _, value := range values {
		switch v := any(value).(type) {
		case float64:
			items = append(items, fmt.Sprintf(`%v`, v))
		case string:
			items = append(items, wrapString(v))
		default:
			return "", fmt.Errorf("unsupported $in value")
		}
	}

	return fmt.Sprintf(`%s IN (%s)`, field, strings.Join(items, ", ")), nil
}

func filterNin[T JSONUnmarshald](field string, values []T) (string, error) {
	items := []string{}

	for _, value := range values {
		switch v := any(value).(type) {
		case float64:
			items = append(items, fmt.Sprintf(`%v`, v))
		case string:
			items = append(items, wrapString(v))
		default:
			return "", fmt.Errorf("unsupported $nin value")
		}
	}

	return fmt.Sprintf(`%s NOT IN (%s)`, field, strings.Join(items, ", ")), nil
}

func filterArithmetic[T JSONUnmarshald](compareType CompareType, field string, arithmetic string, value T) (string, error) {
	switch v := any(value).(type) {
	case float64:
		return fmt.Sprintf(`%s %s %v`, field, arithmetic, v), nil
	case string:
		return fmt.Sprintf(`%s %s %s`, field, arithmetic, wrapString(v)), nil
	}
	return "", fmt.Errorf("unsupported value %v for %s", value, compareType)
}

func wrapString(value string) string {
	return fmt.Sprintf(`'%s'`, value)
}
