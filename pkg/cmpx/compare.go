package cmpx

// Comparable is implemented by values with a deterministic ordering.
type Comparable[T any] interface {
	Compare(T) int
}

// Compare orders two comparable values.
func Compare[T Comparable[T]](left, right T) int {
	return left.Compare(right)
}
