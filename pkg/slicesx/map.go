package slicesx

import "errors"

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
