package set

import "sync"

type Set[T comparable] struct {
	mu      sync.RWMutex
	content map[T]struct{}
}

func New[T comparable](items ...T) *Set[T] {
	s := Set[T]{
		content: make(map[T]struct{}, len(items)),
	}
	s.Add(items...)
	return &s
}

func (s *Set[T]) Add(items ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range items {
		s.content[item] = struct{}{}
	}
}

func (s *Set[T]) Remove(items ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range items {
		delete(s.content, item)
	}
}

func (s *Set[T]) AsSlice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]T, 0, len(s.content))

	for item := range s.content {
		result = append(result, item)
	}

	return result
}

func (s *Set[T]) IsEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.content) == 0
}

// Subtract removes all items from a that are also in b
func Subtract[T comparable](a *Set[T], b ...*Set[T]) *Set[T] {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, set := range b {
		set.mu.RLock()
		defer set.mu.RUnlock()
	}

	result := Set[T]{
		content: make(map[T]struct{}, len(a.content)),
	}
	for item := range a.content {
		result.content[item] = struct{}{}
	}

	for _, set := range b {
		for item := range set.content {
			delete(result.content, item)
		}
	}

	return &result
}

func Union[T comparable](sets ...*Set[T]) *Set[T] {
	for _, set := range sets {
		set.mu.RLock()
		defer set.mu.RUnlock()
	}

	outLen := 0
	for _, set := range sets {
		outLen += len(set.content)
	}

	result := Set[T]{
		content: make(map[T]struct{}, outLen),
	}

	for _, set := range sets {
		for item := range set.content {
			result.content[item] = struct{}{}
		}
	}

	return &result
}
