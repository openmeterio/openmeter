package entutils

import (
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/mixin"
	"github.com/samber/lo"
)

type MixinWithAdditionalMixins interface {
	ent.Mixin

	// Additional mixins to include
	Mixin() []ent.Mixin
}

type emptyMixin struct {
	mixin.Schema
}

func (emptyMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{}
}

var _ ent.Mixin = RecursiveMixin[emptyMixin]{}

type RecursiveMixin[T MixinWithAdditionalMixins] struct{}

func (r RecursiveMixin[T]) Fields() []ent.Field {
	var base T

	return slices.Concat(
		base.Fields(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []ent.Field {
			return item.Fields()
		})),
	)
}

func (r RecursiveMixin[T]) Indexes() []ent.Index {
	var base T

	return slices.Concat(
		base.Indexes(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []ent.Index {
			return item.Indexes()
		})),
	)
}

func (r RecursiveMixin[T]) Edges() []ent.Edge {
	var base T

	return slices.Concat(
		base.Edges(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []ent.Edge {
			return item.Edges()
		})),
	)
}

func (r RecursiveMixin[T]) Hooks() []ent.Hook {
	var base T

	return slices.Concat(
		base.Hooks(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []ent.Hook {
			return item.Hooks()
		})),
	)
}

func (r RecursiveMixin[T]) Interceptors() []ent.Interceptor {
	var base T

	return slices.Concat(
		base.Interceptors(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []ent.Interceptor {
			return item.Interceptors()
		})),
	)
}

func (r RecursiveMixin[T]) Policy() ent.Policy {
	// TODO: properly support (e.g. evaluate all mixins)
	var base T

	return base.Policy()
}

func (r RecursiveMixin[T]) Annotations() []schema.Annotation {
	var base T

	return slices.Concat(
		base.Annotations(),
		lo.Flatten(lo.Map(base.Mixin(), func(item ent.Mixin, _ int) []schema.Annotation {
			return item.Annotations()
		})),
	)
}
