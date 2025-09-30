package subscriptionvalidators

// a result is simply a wrapper around either a value or an error, with some convenience methods
type result[T any] struct {
	value T
	err   error
}

func (r result[T]) Value() T {
	return r.value
}

func (r result[T]) Error() error {
	return r.err
}

func (r result[T]) FlatMap(fn func(T) result[T]) result[T] {
	if r.err != nil {
		return r
	}

	return fn(r.value)
}

func (r result[T]) FlatMapErr(fn func(error) result[T]) result[T] {
	if r.err == nil {
		return r
	}

	return fn(r.err)
}

func resultFromTouple[T any](value T, err error) result[T] {
	return result[T]{
		value: value,
		err:   err,
	}
}

func errResult[T any](err error) result[T] {
	var def T

	return result[T]{
		value: def,
		err:   err,
	}
}

func okResult[T any](value T) result[T] {
	return result[T]{
		value: value,
		err:   nil,
	}
}

// flatMap can be used to convert from Result[T] to Result[U]
func flatMap[T any, U any](fn func(T) result[U]) func(result[T]) result[U] {
	return func(r result[T]) result[U] {
		if r.err != nil {
			return errResult[U](r.err)
		}

		return fn(r.value)
	}
}
