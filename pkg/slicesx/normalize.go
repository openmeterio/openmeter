package slicesx

import (
	"cmp"
	"slices"
)

func Normalize[S ~[]E, E cmp.Ordered](s S) S {
	out := slices.Clone(s)
	slices.Sort(out)

	return slices.Compact(out)
}
