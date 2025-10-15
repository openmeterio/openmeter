package slicesx

import "github.com/samber/lo"

func SliceToPtrSlice[T any](s []T) []*T {
	return lo.Map(s, func(_ T, i int) *T {
		return &s[i]
	})
}
