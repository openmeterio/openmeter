package plan

// MetadataEqual returns false if the two metadata hashmaps are differ
func MetadataEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}

	visited := make([]string, 0, len(right))
	for lk, lv := range left {
		rv, ok := right[lk]
		if !ok {
			return false
		}

		if lv != rv {
			return false
		}

		visited = append(visited, lk)
	}

	return len(visited) == len(right)
}
