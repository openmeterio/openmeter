package set

type Set[T comparable] map[T]struct{}

func New[T comparable](items ...T) Set[T] {
	s := Set[T]{}
	s.Add(items...)
	return s
}

func (s Set[T]) Add(items ...T) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

func (s Set[T]) Remove(items ...T) {
	for _, item := range items {
		delete(s, item)
	}
}

func (s Set[T]) AsSlice() []T {
	result := make([]T, 0, len(s))

	for item := range s {
		result = append(result, item)
	}

	return result
}

// Subtract removes all items from a that are also in b
func Subtract[T comparable](a Set[T], b ...Set[T]) Set[T] {
	result := Set[T]{}
	for item := range a {
		result[item] = struct{}{}
	}

	for _, set := range b {
		for item := range set {
			delete(result, item)
		}
	}

	return result
}

func Union[T comparable](sets ...Set[T]) Set[T] {
	result := Set[T]{}

	for _, set := range sets {
		for item := range set {
			result[item] = struct{}{}
		}
	}

	return result
}
