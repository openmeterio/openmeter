// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppCustomInvoicingDelete is the builder for deleting a AppCustomInvoicing entity.
type AppCustomInvoicingDelete struct {
	config
	hooks    []Hook
	mutation *AppCustomInvoicingMutation
}

// Where appends a list predicates to the AppCustomInvoicingDelete builder.
func (acid *AppCustomInvoicingDelete) Where(ps ...predicate.AppCustomInvoicing) *AppCustomInvoicingDelete {
	acid.mutation.Where(ps...)
	return acid
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (acid *AppCustomInvoicingDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, acid.sqlExec, acid.mutation, acid.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (acid *AppCustomInvoicingDelete) ExecX(ctx context.Context) int {
	n, err := acid.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (acid *AppCustomInvoicingDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(appcustominvoicing.Table, sqlgraph.NewFieldSpec(appcustominvoicing.FieldID, field.TypeString))
	if ps := acid.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, acid.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	acid.mutation.done = true
	return affected, err
}

// AppCustomInvoicingDeleteOne is the builder for deleting a single AppCustomInvoicing entity.
type AppCustomInvoicingDeleteOne struct {
	acid *AppCustomInvoicingDelete
}

// Where appends a list predicates to the AppCustomInvoicingDelete builder.
func (acido *AppCustomInvoicingDeleteOne) Where(ps ...predicate.AppCustomInvoicing) *AppCustomInvoicingDeleteOne {
	acido.acid.mutation.Where(ps...)
	return acido
}

// Exec executes the deletion query.
func (acido *AppCustomInvoicingDeleteOne) Exec(ctx context.Context) error {
	n, err := acido.acid.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{appcustominvoicing.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (acido *AppCustomInvoicingDeleteOne) ExecX(ctx context.Context) {
	if err := acido.Exec(ctx); err != nil {
		panic(err)
	}
}
