package convert

import (
	"fmt"
	"time"

	"github.com/samber/lo"
)

func ToPointer[T any](value T) *T {
	return &value
}

// MapToPointer converts a map to a pointer to a map, returning nil if the map is empty,
// this solves the problem of lo.EmptyableToPtr not handling maps correctly.
func MapToPointer[T comparable, U any](value map[T]U) *map[T]U {
	if len(value) == 0 {
		return nil
	}
	return &value
}

// SliceToPointer converts a slice to a pointer to a slice, returning nil if the slice is empty,
// this solves the problem of lo.EmptyableToPtr not handling slices correctly.
func SliceToPointer[T any](value []T) *[]T {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func ToStringLike[Source, Dest ~string](value *Source) *Dest {
	if value == nil {
		return nil
	}
	return ToPointer(Dest(*value))
}

// SafeDeRef is a helper function to safely dereference a pointer and apply a function to it.
func SafeDeRef[T any, U any](value *T, fn func(T) *U) *U {
	if value == nil {
		return nil
	}
	return fn(*value)
}

func SafeToUTC(t *time.Time) *time.Time {
	return SafeDeRef(t, func(dt time.Time) *time.Time {
		return ToPointer(dt.In(time.UTC))
	})
}

// Header represents generic primitives with a header like map, slice, array...
type Header[E any] interface {
	map[string]E | []E
}

// Safely dereference a pointer to a slice or map
func DerefHeaderPtr[E any, T Header[E]](header *T) T {
	if header == nil {
		return nil
	}
	return *header
}

// StringerPtrToStringPtr converts a pointer to a type with String() method to a pointer to a string
// It returns nil if the input is nil
func StringerPtrToStringPtr[T fmt.Stringer](value *T) *string {
	if value == nil {
		return nil
	}
	return lo.ToPtr((*value).String())
}
