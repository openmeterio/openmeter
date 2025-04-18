// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// PlanAddonDelete is the builder for deleting a PlanAddon entity.
type PlanAddonDelete struct {
	config
	hooks    []Hook
	mutation *PlanAddonMutation
}

// Where appends a list predicates to the PlanAddonDelete builder.
func (pad *PlanAddonDelete) Where(ps ...predicate.PlanAddon) *PlanAddonDelete {
	pad.mutation.Where(ps...)
	return pad
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (pad *PlanAddonDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, pad.sqlExec, pad.mutation, pad.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (pad *PlanAddonDelete) ExecX(ctx context.Context) int {
	n, err := pad.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (pad *PlanAddonDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(planaddon.Table, sqlgraph.NewFieldSpec(planaddon.FieldID, field.TypeString))
	if ps := pad.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, pad.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	pad.mutation.done = true
	return affected, err
}

// PlanAddonDeleteOne is the builder for deleting a single PlanAddon entity.
type PlanAddonDeleteOne struct {
	pad *PlanAddonDelete
}

// Where appends a list predicates to the PlanAddonDelete builder.
func (pado *PlanAddonDeleteOne) Where(ps ...predicate.PlanAddon) *PlanAddonDeleteOne {
	pado.pad.mutation.Where(ps...)
	return pado
}

// Exec executes the deletion query.
func (pado *PlanAddonDeleteOne) Exec(ctx context.Context) error {
	n, err := pado.pad.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{planaddon.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (pado *PlanAddonDeleteOne) ExecX(ctx context.Context) {
	if err := pado.Exec(ctx); err != nil {
		panic(err)
	}
}
