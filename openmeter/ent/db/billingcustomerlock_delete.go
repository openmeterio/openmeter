// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingCustomerLockDelete is the builder for deleting a BillingCustomerLock entity.
type BillingCustomerLockDelete struct {
	config
	hooks    []Hook
	mutation *BillingCustomerLockMutation
}

// Where appends a list predicates to the BillingCustomerLockDelete builder.
func (bcld *BillingCustomerLockDelete) Where(ps ...predicate.BillingCustomerLock) *BillingCustomerLockDelete {
	bcld.mutation.Where(ps...)
	return bcld
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (bcld *BillingCustomerLockDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, bcld.sqlExec, bcld.mutation, bcld.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (bcld *BillingCustomerLockDelete) ExecX(ctx context.Context) int {
	n, err := bcld.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (bcld *BillingCustomerLockDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(billingcustomerlock.Table, sqlgraph.NewFieldSpec(billingcustomerlock.FieldID, field.TypeString))
	if ps := bcld.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, bcld.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	bcld.mutation.done = true
	return affected, err
}

// BillingCustomerLockDeleteOne is the builder for deleting a single BillingCustomerLock entity.
type BillingCustomerLockDeleteOne struct {
	bcld *BillingCustomerLockDelete
}

// Where appends a list predicates to the BillingCustomerLockDelete builder.
func (bcldo *BillingCustomerLockDeleteOne) Where(ps ...predicate.BillingCustomerLock) *BillingCustomerLockDeleteOne {
	bcldo.bcld.mutation.Where(ps...)
	return bcldo
}

// Exec executes the deletion query.
func (bcldo *BillingCustomerLockDeleteOne) Exec(ctx context.Context) error {
	n, err := bcldo.bcld.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{billingcustomerlock.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (bcldo *BillingCustomerLockDeleteOne) ExecX(ctx context.Context) {
	if err := bcldo.Exec(ctx); err != nil {
		panic(err)
	}
}
