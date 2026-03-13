package charges

type WithIndex[T any] struct {
	Index int
	Value T
}
