package entitydiff

import "github.com/samber/lo"

type NestedEntity[T Entity, P Entity] struct {
	Entity T
	Parent P
}

func (w NestedEntity[T, P]) GetID() string {
	return w.Entity.GetID()
}

func (w NestedEntity[T, P]) IsDeleted() bool {
	return w.Entity.IsDeleted()
}

type EqualerNestedEntity[T EqualerEntity[T], P Entity] struct {
	Entity T
	Parent P
}

func (w EqualerNestedEntity[T, P]) GetID() string {
	return w.Entity.GetID()
}

func (w EqualerNestedEntity[T, P]) IsDeleted() bool {
	return w.Entity.IsDeleted()
}

func (w EqualerNestedEntity[T, P]) Equal(other EqualerNestedEntity[T, P]) bool {
	return w.Entity.Equal(other.Entity)
}

func NewEqualersWithParent[T EqualerEntity[T], P Entity](entity []T, parent P) []EqualerNestedEntity[T, P] {
	return lo.Map(entity, func(item T, _ int) EqualerNestedEntity[T, P] {
		return EqualerNestedEntity[T, P]{
			Entity: item,
			Parent: parent,
		}
	})
}
