package convert

func ToPointer[T any](value T) *T {
	return &value
}
