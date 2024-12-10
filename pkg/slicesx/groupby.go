package slicesx

import "github.com/samber/lo"

func UniqueGroupBy[T any, U comparable, Slice ~[]T](collection Slice, iteratee func(item T) U) (map[U]T, bool) {
	res := lo.GroupBy(collection, iteratee)
	out := make(map[U]T, len(res))

	for k, v := range res {
		if len(v) > 1 {
			return nil, false
		}

		out[k] = v[0]
	}

	return out, true
}
