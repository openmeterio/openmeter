// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingProfileDelete is the builder for deleting a BillingProfile entity.
type BillingProfileDelete struct {
	config
	hooks    []Hook
	mutation *BillingProfileMutation
}

// Where appends a list predicates to the BillingProfileDelete builder.
func (_d *BillingProfileDelete) Where(ps ...predicate.BillingProfile) *BillingProfileDelete {
	_d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (_d *BillingProfileDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, _d.sqlExec, _d.mutation, _d.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *BillingProfileDelete) ExecX(ctx context.Context) int {
	n, err := _d.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (_d *BillingProfileDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(billingprofile.Table, sqlgraph.NewFieldSpec(billingprofile.FieldID, field.TypeString))
	if ps := _d.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, _d.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	_d.mutation.done = true
	return affected, err
}

// BillingProfileDeleteOne is the builder for deleting a single BillingProfile entity.
type BillingProfileDeleteOne struct {
	_d *BillingProfileDelete
}

// Where appends a list predicates to the BillingProfileDelete builder.
func (_d *BillingProfileDeleteOne) Where(ps ...predicate.BillingProfile) *BillingProfileDeleteOne {
	_d._d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query.
func (_d *BillingProfileDeleteOne) Exec(ctx context.Context) error {
	n, err := _d._d.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{billingprofile.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *BillingProfileDeleteOne) ExecX(ctx context.Context) {
	if err := _d.Exec(ctx); err != nil {
		panic(err)
	}
}
