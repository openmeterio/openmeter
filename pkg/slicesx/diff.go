package slicesx

import "github.com/samber/lo"

type Diff[T comparable, S ~[]T] struct {
	additionsMap map[T]struct{}
	additions    S

	removalsMap map[T]struct{}
	removals    S
}

func (d Diff[T, S]) Has(item T) bool {
	return d.InAdditions(item) || d.InRemovals(item)
}

func (d Diff[T, S]) InAdditions(item T) bool {
	if _, ok := d.additionsMap[item]; ok {
		return true
	}

	return false
}

func (d Diff[T, S]) InRemovals(item T) bool {
	if _, ok := d.removalsMap[item]; ok {
		return true
	}

	return false
}

func (d Diff[T, S]) Additions() S {
	return d.additions
}

func (d Diff[T, S]) Removals() S {
	return d.removals
}

func (d Diff[T, S]) HasChanged() bool {
	return len(d.additions) > 0 || len(d.removals) > 0
}

func (d Diff[T, S]) Changed() S {
	return append(d.additions, d.removals...)
}

func NewDiff[T comparable, S ~[]T](base, new S) *Diff[T, S] {
	additions, removals := lo.Difference(base, new)

	additionsMap := lo.SliceToMap(additions, func(item T) (T, struct{}) {
		return item, struct{}{}
	})

	removalsMap := lo.SliceToMap(removals, func(item T) (T, struct{}) {
		return item, struct{}{}
	})

	return &Diff[T, S]{
		additionsMap: additionsMap,
		additions:    additions,
		removalsMap:  removalsMap,
		removals:     removals,
	}
}
