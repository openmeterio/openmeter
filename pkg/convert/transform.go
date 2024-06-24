package convert

func Transform[IN any, OUT any](input []IN, f func(IN) OUT) []OUT {
	result := make([]OUT, 0, len(input))
	for _, i := range input {
		result = append(result, f(i))
	}
	return result
}
