// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingCustomerOverrideDelete is the builder for deleting a BillingCustomerOverride entity.
type BillingCustomerOverrideDelete struct {
	config
	hooks    []Hook
	mutation *BillingCustomerOverrideMutation
}

// Where appends a list predicates to the BillingCustomerOverrideDelete builder.
func (bcod *BillingCustomerOverrideDelete) Where(ps ...predicate.BillingCustomerOverride) *BillingCustomerOverrideDelete {
	bcod.mutation.Where(ps...)
	return bcod
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (bcod *BillingCustomerOverrideDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, bcod.sqlExec, bcod.mutation, bcod.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (bcod *BillingCustomerOverrideDelete) ExecX(ctx context.Context) int {
	n, err := bcod.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (bcod *BillingCustomerOverrideDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(billingcustomeroverride.Table, sqlgraph.NewFieldSpec(billingcustomeroverride.FieldID, field.TypeString))
	if ps := bcod.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, bcod.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	bcod.mutation.done = true
	return affected, err
}

// BillingCustomerOverrideDeleteOne is the builder for deleting a single BillingCustomerOverride entity.
type BillingCustomerOverrideDeleteOne struct {
	bcod *BillingCustomerOverrideDelete
}

// Where appends a list predicates to the BillingCustomerOverrideDelete builder.
func (bcodo *BillingCustomerOverrideDeleteOne) Where(ps ...predicate.BillingCustomerOverride) *BillingCustomerOverrideDeleteOne {
	bcodo.bcod.mutation.Where(ps...)
	return bcodo
}

// Exec executes the deletion query.
func (bcodo *BillingCustomerOverrideDeleteOne) Exec(ctx context.Context) error {
	n, err := bcodo.bcod.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{billingcustomeroverride.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (bcodo *BillingCustomerOverrideDeleteOne) ExecX(ctx context.Context) {
	if err := bcodo.Exec(ctx); err != nil {
		panic(err)
	}
}
