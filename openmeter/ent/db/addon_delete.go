// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AddonDelete is the builder for deleting a Addon entity.
type AddonDelete struct {
	config
	hooks    []Hook
	mutation *AddonMutation
}

// Where appends a list predicates to the AddonDelete builder.
func (ad *AddonDelete) Where(ps ...predicate.Addon) *AddonDelete {
	ad.mutation.Where(ps...)
	return ad
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (ad *AddonDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, ad.sqlExec, ad.mutation, ad.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (ad *AddonDelete) ExecX(ctx context.Context) int {
	n, err := ad.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (ad *AddonDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(addon.Table, sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString))
	if ps := ad.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, ad.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	ad.mutation.done = true
	return affected, err
}

// AddonDeleteOne is the builder for deleting a single Addon entity.
type AddonDeleteOne struct {
	ad *AddonDelete
}

// Where appends a list predicates to the AddonDelete builder.
func (ado *AddonDeleteOne) Where(ps ...predicate.Addon) *AddonDeleteOne {
	ado.ad.mutation.Where(ps...)
	return ado
}

// Exec executes the deletion query.
func (ado *AddonDeleteOne) Exec(ctx context.Context) error {
	n, err := ado.ad.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{addon.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (ado *AddonDeleteOne) ExecX(ctx context.Context) {
	if err := ado.Exec(ctx); err != nil {
		panic(err)
	}
}
