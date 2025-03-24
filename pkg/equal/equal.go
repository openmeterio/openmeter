package equal

type Comparable[T any] interface {
	Equal(other T) bool
}

func PtrEqual[T Comparable[T]](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return (*a).Equal(*b)
}
