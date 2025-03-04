package lazy

import "sync"

// OnceValueErr is a function that returns a function that will call the given function once and return the result.
// If the function returns an error, the error will be returned on subsequent calls.
func OnceValueErr[T any](f func() (T, error)) func() (T, error) {
	var (
		once      sync.Once
		result    T
		resultErr error
	)

	return func() (T, error) {
		once.Do(func() {
			result, resultErr = f()
		})

		return result, resultErr
	}
}
