package defaultx

func WithDefault[T any](value *T, def T) T {
	if value != nil {
		return *value
	}

	return def
}
