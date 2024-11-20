package models

type Optional[T any] struct {
	content *T
}

func OptionalWithValue[T any](t T) Optional[T] {
	return Optional[T]{
		content: &t,
	}
}

func (o Optional[T]) IsPresent() bool {
	return o.content != nil
}

func (o Optional[T]) Get() T {
	if o.content == nil {
		var empty T
		return empty
	}

	return *o.content
}
