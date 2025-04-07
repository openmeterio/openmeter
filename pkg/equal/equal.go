package equal

import (
	"github.com/openmeterio/openmeter/pkg/hasher"
)

// Equaler is an interface that can be used to compare two objects.
// This is already present in models, but we need it here so that we can avoid a circular dependency.
type Equaler[T any] interface {
	Equal(other T) bool
}

func PtrEqual[T Equaler[T]](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return (*a).Equal(*b)
}

func HasherPtrEqual[T hasher.Hasher](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return (*a).Hash() == (*b).Hash()
}
