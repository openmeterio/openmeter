// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonquantity"
)

// SubscriptionAddonQuantityDelete is the builder for deleting a SubscriptionAddonQuantity entity.
type SubscriptionAddonQuantityDelete struct {
	config
	hooks    []Hook
	mutation *SubscriptionAddonQuantityMutation
}

// Where appends a list predicates to the SubscriptionAddonQuantityDelete builder.
func (saqd *SubscriptionAddonQuantityDelete) Where(ps ...predicate.SubscriptionAddonQuantity) *SubscriptionAddonQuantityDelete {
	saqd.mutation.Where(ps...)
	return saqd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (saqd *SubscriptionAddonQuantityDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, saqd.sqlExec, saqd.mutation, saqd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (saqd *SubscriptionAddonQuantityDelete) ExecX(ctx context.Context) int {
	n, err := saqd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (saqd *SubscriptionAddonQuantityDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(subscriptionaddonquantity.Table, sqlgraph.NewFieldSpec(subscriptionaddonquantity.FieldID, field.TypeString))
	if ps := saqd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, saqd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	saqd.mutation.done = true
	return affected, err
}

// SubscriptionAddonQuantityDeleteOne is the builder for deleting a single SubscriptionAddonQuantity entity.
type SubscriptionAddonQuantityDeleteOne struct {
	saqd *SubscriptionAddonQuantityDelete
}

// Where appends a list predicates to the SubscriptionAddonQuantityDelete builder.
func (saqdo *SubscriptionAddonQuantityDeleteOne) Where(ps ...predicate.SubscriptionAddonQuantity) *SubscriptionAddonQuantityDeleteOne {
	saqdo.saqd.mutation.Where(ps...)
	return saqdo
}

// Exec executes the deletion query.
func (saqdo *SubscriptionAddonQuantityDeleteOne) Exec(ctx context.Context) error {
	n, err := saqdo.saqd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{subscriptionaddonquantity.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (saqdo *SubscriptionAddonQuantityDeleteOne) ExecX(ctx context.Context) {
	if err := saqdo.Exec(ctx); err != nil {
		panic(err)
	}
}
