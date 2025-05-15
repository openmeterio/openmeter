package slicesx

func ForEachUntilWithErr[T any](s []T, f func(T, int) (breaks bool, err error)) error {
	for i, v := range s {
		breaks, err := f(v, i)
		if err != nil {
			return err
		}

		if breaks {
			break
		}
	}

	return nil
}
