package appobserver

import "context"

type Observer[T any] interface {
	PostCreate(context.Context, *T) error
	PostUpdate(context.Context, *T) error
	PostDelete(context.Context, *T) error
}

type Publisher[T any] interface {
	// Register allows an instance to register itself to listen/observe
	// events.
	Register(Observer[T]) error
	// Deregister allows an instance to remove itself from the collection
	// of observers/listeners.
	Deregister(Observer[T]) error
}
