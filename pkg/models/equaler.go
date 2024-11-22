package models

type Equaler[T any] interface {
	// Equal returns true in case all attributes of T are strictly equal.
	Equal(T) bool
}
