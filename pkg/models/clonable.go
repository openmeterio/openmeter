package models

type Clonable[T any] interface {
	Clone() T
}
