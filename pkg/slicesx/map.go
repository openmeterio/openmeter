package slicesx

import (
	"errors"
)

// Map maps elements of a slice from T to M, returning a new slice.
func Map[T any, S any](s []T, f func(T) S) []S {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	n := make([]S, len(s))

	for i, v := range s {
		n[i] = f(v)
	}

	return n
}

// MapWithErr maps elements of a slice from T to M, returning a new slice and a joined error if there are any.
// If an error is returned from the mapping function, a nil array and the error is returned.
func MapWithErr[T any, S any](s []T, f func(T) (S, error)) ([]S, error) {
	// Nil input, return early.
	if s == nil {
		return nil, nil
	}

	var outErr error
	n := make([]S, 0, len(s))

	for _, v := range s {
		res, err := f(v)
		if err != nil {
			outErr = errors.Join(outErr, err)
			continue
		}

		n = append(n, res)
	}

	if outErr != nil {
		return nil, outErr
	}

	return n, nil
}

// MapWithErrPreservingResults maps every element, preserves each returned result
// at its input index, and joins all mapping errors.
func MapWithErrPreservingResults[T any, S any](s []T, f func(T, int) (S, error)) ([]S, error) {
	if len(s) == 0 {
		return nil, nil
	}

	results := make([]S, len(s))
	errs := make([]error, len(s))
	for index, value := range s {
		results[index], errs[index] = f(value, index)
	}

	return results, errors.Join(errs...)
}

func WrapMapFn[T, O any](fn func(T) (O, error)) func(T, int) (O, error) {
	return func(t T, _ int) (O, error) {
		return fn(t)
	}
}
