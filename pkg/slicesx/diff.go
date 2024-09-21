// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
