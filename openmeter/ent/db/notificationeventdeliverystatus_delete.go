// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// NotificationEventDeliveryStatusDelete is the builder for deleting a NotificationEventDeliveryStatus entity.
type NotificationEventDeliveryStatusDelete struct {
	config
	hooks    []Hook
	mutation *NotificationEventDeliveryStatusMutation
}

// Where appends a list predicates to the NotificationEventDeliveryStatusDelete builder.
func (_d *NotificationEventDeliveryStatusDelete) Where(ps ...predicate.NotificationEventDeliveryStatus) *NotificationEventDeliveryStatusDelete {
	_d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (_d *NotificationEventDeliveryStatusDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, _d.sqlExec, _d.mutation, _d.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *NotificationEventDeliveryStatusDelete) ExecX(ctx context.Context) int {
	n, err := _d.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (_d *NotificationEventDeliveryStatusDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(notificationeventdeliverystatus.Table, sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString))
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

// NotificationEventDeliveryStatusDeleteOne is the builder for deleting a single NotificationEventDeliveryStatus entity.
type NotificationEventDeliveryStatusDeleteOne struct {
	_d *NotificationEventDeliveryStatusDelete
}

// Where appends a list predicates to the NotificationEventDeliveryStatusDelete builder.
func (_d *NotificationEventDeliveryStatusDeleteOne) Where(ps ...predicate.NotificationEventDeliveryStatus) *NotificationEventDeliveryStatusDeleteOne {
	_d._d.mutation.Where(ps...)
	return _d
}

// Exec executes the deletion query.
func (_d *NotificationEventDeliveryStatusDeleteOne) Exec(ctx context.Context) error {
	n, err := _d._d.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{notificationeventdeliverystatus.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (_d *NotificationEventDeliveryStatusDeleteOne) ExecX(ctx context.Context) {
	if err := _d.Exec(ctx); err != nil {
		panic(err)
	}
}
