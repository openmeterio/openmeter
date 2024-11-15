package slicesx

type SliceDifference[T any] struct {
	MissingFromThis  []T
	MissingFromOther []T
}

type OptionalSlice[T any] struct {
	slice *[]T
}

// TODO: rename to match optional
func NewOptionalSlice[T any](slice []T) OptionalSlice[T] {
	if len(slice) == 0 && slice != nil {
		slice = nil
	}

	return OptionalSlice[T]{slice: &slice}
}

func (s OptionalSlice[T]) IsPresent() bool {
	return s.slice != nil
}

func (s OptionalSlice[T]) Get() []T {
	if s.slice == nil {
		return nil
	}

	return *s.slice
}

func (s *OptionalSlice[T]) Append(items ...T) {
	if s.slice == nil {
		slice := items
		s.slice = &slice
		return
	}

	*s.slice = append(*s.slice, items...)
}

func (s OptionalSlice[T]) Map(fn func(T) T) OptionalSlice[T] {
	if s.slice == nil {
		return OptionalSlice[T]{}
	}

	newSlice := make([]T, len(*s.slice))

	for i, item := range *s.slice {
		newSlice[i] = fn(item)
	}

	return OptionalSlice[T]{slice: &newSlice}
}
