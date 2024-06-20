package convert

import "time"

func ToPointer[T any](value T) *T {
	return &value
}

func ToStringLike[Source, Dest ~string](value *Source) *Dest {
	if value == nil {
		return nil
	}
	return ToPointer(Dest(*value))
}

func SafeConvert[T any, U any](value *T, fn func(T) U) *U {
	if value == nil {
		return nil
	}
	return ToPointer(fn(*value))
}

func SafeToUTC(t *time.Time) *time.Time {
	return SafeConvert(t, func(dt time.Time) time.Time {
		return dt.In(time.UTC)
	})
}
