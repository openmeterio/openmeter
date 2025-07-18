// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicesplitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceSplitLineGroupDelete is the builder for deleting a BillingInvoiceSplitLineGroup entity.
type BillingInvoiceSplitLineGroupDelete struct {
	config
	hooks    []Hook
	mutation *BillingInvoiceSplitLineGroupMutation
}

// Where appends a list predicates to the BillingInvoiceSplitLineGroupDelete builder.
func (_d *BillingInvoiceSplitLineGroupDelete) Where(ps ...predicate.BillingInvoiceSplitLineGroup) *BillingInvoiceSplitLineGroupDelete {
	_d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (_d *BillingInvoiceSplitLineGroupDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, _d.sqlExec, _d.mutation, _d.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *BillingInvoiceSplitLineGroupDelete) ExecX(ctx context.Context) int {
	n, err := _d.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (_d *BillingInvoiceSplitLineGroupDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(billinginvoicesplitlinegroup.Table, sqlgraph.NewFieldSpec(billinginvoicesplitlinegroup.FieldID, field.TypeString))
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

// BillingInvoiceSplitLineGroupDeleteOne is the builder for deleting a single BillingInvoiceSplitLineGroup entity.
type BillingInvoiceSplitLineGroupDeleteOne struct {
	_d *BillingInvoiceSplitLineGroupDelete
}

// Where appends a list predicates to the BillingInvoiceSplitLineGroupDelete builder.
func (_d *BillingInvoiceSplitLineGroupDeleteOne) Where(ps ...predicate.BillingInvoiceSplitLineGroup) *BillingInvoiceSplitLineGroupDeleteOne {
	_d._d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query.
func (_d *BillingInvoiceSplitLineGroupDeleteOne) Exec(ctx context.Context) error {
	n, err := _d._d.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{billinginvoicesplitlinegroup.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *BillingInvoiceSplitLineGroupDeleteOne) ExecX(ctx context.Context) {
	if err := _d.Exec(ctx); err != nil {
		panic(err)
	}
}
