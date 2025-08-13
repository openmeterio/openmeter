package entutils

func NoopValidator[T any](_ T) error {
	return nil
}
