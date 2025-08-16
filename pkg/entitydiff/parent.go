package entitydiff

import "github.com/samber/lo"

type WithParent[T Entity, P Entity] struct {
	Entity T
	Parent P
}

func (w WithParent[T, P]) GetID() string {
	return w.Entity.GetID()
}

func (w WithParent[T, P]) IsDeleted() bool {
	return w.Entity.IsDeleted()
}

type EqualerWithParent[T EqualerEntity[T], P Entity] struct {
	Entity T
	Parent P
}

func (w EqualerWithParent[T, P]) GetID() string {
	return w.Entity.GetID()
}

func (w EqualerWithParent[T, P]) IsDeleted() bool {
	return w.Entity.IsDeleted()
}

func (w EqualerWithParent[T, P]) Equal(other EqualerWithParent[T, P]) bool {
	return w.Entity.Equal(other.Entity)
}

func NewEqualersWithParent[T EqualerEntity[T], P Entity](entity []T, parent P) []EqualerWithParent[T, P] {
	return lo.Map(entity, func(item T, _ int) EqualerWithParent[T, P] {
		return EqualerWithParent[T, P]{
			Entity: item,
			Parent: parent,
		}
	})
}
