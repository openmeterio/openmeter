package plan

import (
	"fmt"
	"hash/fnv"
)

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

func DiscountsEqual(left, right []Discount) (bool, error) {
	if len(left) != len(right) {
		return false, nil
	}

	leftSet := make(map[uint64]struct{}, len(left))
	for _, d := range left {
		h, err := hashDiscount(d)
		if err != nil {
			return false, err
		}

		leftSet[h] = struct{}{}
	}

	rightSet := make(map[uint64]struct{}, len(left))
	for _, d := range right {
		h, err := hashDiscount(d)
		if err != nil {
			return false, err
		}

		rightSet[h] = struct{}{}
	}

	visited := make([]uint64, 0, len(rightSet))
	for lh := range leftSet {
		if _, ok := rightSet[lh]; !ok {
			return false, nil
		}

		visited = append(visited, lh)
	}

	if len(visited) != len(rightSet) {
		return false, nil
	}

	return true, nil
}

// TODO(chrisgacsal): we ned to replace this with a more generic and well tested solution.
func hashDiscount(d Discount) (uint64, error) {
	var content string

	switch d.Type() {
	case PercentageDiscountType:
		p, err := d.AsPercentage()
		if err != nil {
			return 0, fmt.Errorf("failed to cast to Percentage Discount: %w", err)
		}

		content += string(PercentageDiscountType)
		content += p.Percentage.String()
		for _, r := range p.RateCards {
			content += r
		}
	}

	h := fnv.New64()
	_, err := h.Write([]byte(content))
	if err != nil {
		return 0, fmt.Errorf("failed to caculate content hash: %w", err)
	}

	return h.Sum64(), nil
}
