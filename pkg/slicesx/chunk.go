package slicesx

// Chunk splits a slice into chunks of the given size. The last chunk may be smaller than the given size.
func Chunk[T any](s []T, size int) [][]T {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	// Nil or zero size, return the slice as is.
	if size <= 0 {
		return [][]T{s}
	}

	var chunks [][]T
	chunk := make([]T, 0, size)

	for i, item := range s {
		chunk = append(chunk, item)

		if len(chunk) == size || i == len(s)-1 {
			chunks = append(chunks, chunk)
			if i != len(s)-1 {
				chunk = make([]T, 0, size)
			}
		}
	}

	return chunks
}
