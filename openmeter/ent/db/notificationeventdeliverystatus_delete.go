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
func (nedsd *NotificationEventDeliveryStatusDelete) Where(ps ...predicate.NotificationEventDeliveryStatus) *NotificationEventDeliveryStatusDelete {
	nedsd.mutation.Where(ps...)
	return nedsd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (nedsd *NotificationEventDeliveryStatusDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, nedsd.sqlExec, nedsd.mutation, nedsd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (nedsd *NotificationEventDeliveryStatusDelete) ExecX(ctx context.Context) int {
	n, err := nedsd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (nedsd *NotificationEventDeliveryStatusDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(notificationeventdeliverystatus.Table, sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString))
	if ps := nedsd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, nedsd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	nedsd.mutation.done = true
	return affected, err
}

// NotificationEventDeliveryStatusDeleteOne is the builder for deleting a single NotificationEventDeliveryStatus entity.
type NotificationEventDeliveryStatusDeleteOne struct {
	nedsd *NotificationEventDeliveryStatusDelete
}

// Where appends a list predicates to the NotificationEventDeliveryStatusDelete builder.
func (nedsdo *NotificationEventDeliveryStatusDeleteOne) Where(ps ...predicate.NotificationEventDeliveryStatus) *NotificationEventDeliveryStatusDeleteOne {
	nedsdo.nedsd.mutation.Where(ps...)
	return nedsdo
}

// Exec executes the deletion query.
func (nedsdo *NotificationEventDeliveryStatusDeleteOne) Exec(ctx context.Context) error {
	n, err := nedsdo.nedsd.Exec(ctx)
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
func (nedsdo *NotificationEventDeliveryStatusDeleteOne) ExecX(ctx context.Context) {
	if err := nedsdo.Exec(ctx); err != nil {
		panic(err)
	}
}
