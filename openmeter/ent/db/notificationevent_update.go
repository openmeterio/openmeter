// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/notificationevent"
	"github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// NotificationEventUpdate is the builder for updating NotificationEvent entities.
type NotificationEventUpdate struct {
	config
	hooks    []Hook
	mutation *NotificationEventMutation
}

// Where appends a list predicates to the NotificationEventUpdate builder.
func (_u *NotificationEventUpdate) Where(ps ...predicate.NotificationEvent) *NotificationEventUpdate {
	_u.mutation.Where(ps...)
	return _u
}

// SetPayload sets the "payload" field.
func (_u *NotificationEventUpdate) SetPayload(v string) *NotificationEventUpdate {
	_u.mutation.SetPayload(v)
	return _u
}

// SetNillablePayload sets the "payload" field if the given value is not nil.
func (_u *NotificationEventUpdate) SetNillablePayload(v *string) *NotificationEventUpdate {
	if v != nil {
		_u.SetPayload(*v)
	}
	return _u
}

// SetAnnotations sets the "annotations" field.
func (_u *NotificationEventUpdate) SetAnnotations(v map[string]interface{}) *NotificationEventUpdate {
	_u.mutation.SetAnnotations(v)
	return _u
}

// ClearAnnotations clears the value of the "annotations" field.
func (_u *NotificationEventUpdate) ClearAnnotations() *NotificationEventUpdate {
	_u.mutation.ClearAnnotations()
	return _u
}

// AddDeliveryStatusIDs adds the "delivery_statuses" edge to the NotificationEventDeliveryStatus entity by IDs.
func (_u *NotificationEventUpdate) AddDeliveryStatusIDs(ids ...string) *NotificationEventUpdate {
	_u.mutation.AddDeliveryStatusIDs(ids...)
	return _u
}

// AddDeliveryStatuses adds the "delivery_statuses" edges to the NotificationEventDeliveryStatus entity.
func (_u *NotificationEventUpdate) AddDeliveryStatuses(v ...*NotificationEventDeliveryStatus) *NotificationEventUpdate {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.AddDeliveryStatusIDs(ids...)
}

// Mutation returns the NotificationEventMutation object of the builder.
func (_u *NotificationEventUpdate) Mutation() *NotificationEventMutation {
	return _u.mutation
}

// ClearDeliveryStatuses clears all "delivery_statuses" edges to the NotificationEventDeliveryStatus entity.
func (_u *NotificationEventUpdate) ClearDeliveryStatuses() *NotificationEventUpdate {
	_u.mutation.ClearDeliveryStatuses()
	return _u
}

// RemoveDeliveryStatusIDs removes the "delivery_statuses" edge to NotificationEventDeliveryStatus entities by IDs.
func (_u *NotificationEventUpdate) RemoveDeliveryStatusIDs(ids ...string) *NotificationEventUpdate {
	_u.mutation.RemoveDeliveryStatusIDs(ids...)
	return _u
}

// RemoveDeliveryStatuses removes "delivery_statuses" edges to NotificationEventDeliveryStatus entities.
func (_u *NotificationEventUpdate) RemoveDeliveryStatuses(v ...*NotificationEventDeliveryStatus) *NotificationEventUpdate {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.RemoveDeliveryStatusIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (_u *NotificationEventUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *NotificationEventUpdate) SaveX(ctx context.Context) int {
	affected, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (_u *NotificationEventUpdate) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *NotificationEventUpdate) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *NotificationEventUpdate) check() error {
	if _u.mutation.RulesCleared() && len(_u.mutation.RulesIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "NotificationEvent.rules"`)
	}
	return nil
}

func (_u *NotificationEventUpdate) sqlSave(ctx context.Context) (_node int, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(notificationevent.Table, notificationevent.Columns, sqlgraph.NewFieldSpec(notificationevent.FieldID, field.TypeString))
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.Payload(); ok {
		_spec.SetField(notificationevent.FieldPayload, field.TypeString, value)
	}
	if value, ok := _u.mutation.Annotations(); ok {
		vv, err := notificationevent.ValueScanner.Annotations.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(notificationevent.FieldAnnotations, field.TypeString, vv)
	}
	if _u.mutation.AnnotationsCleared() {
		_spec.ClearField(notificationevent.FieldAnnotations, field.TypeString)
	}
	if _u.mutation.DeliveryStatusesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.RemovedDeliveryStatusesIDs(); len(nodes) > 0 && !_u.mutation.DeliveryStatusesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.DeliveryStatusesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if _node, err = sqlgraph.UpdateNodes(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{notificationevent.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	_u.mutation.done = true
	return _node, nil
}

// NotificationEventUpdateOne is the builder for updating a single NotificationEvent entity.
type NotificationEventUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *NotificationEventMutation
}

// SetPayload sets the "payload" field.
func (_u *NotificationEventUpdateOne) SetPayload(v string) *NotificationEventUpdateOne {
	_u.mutation.SetPayload(v)
	return _u
}

// SetNillablePayload sets the "payload" field if the given value is not nil.
func (_u *NotificationEventUpdateOne) SetNillablePayload(v *string) *NotificationEventUpdateOne {
	if v != nil {
		_u.SetPayload(*v)
	}
	return _u
}

// SetAnnotations sets the "annotations" field.
func (_u *NotificationEventUpdateOne) SetAnnotations(v map[string]interface{}) *NotificationEventUpdateOne {
	_u.mutation.SetAnnotations(v)
	return _u
}

// ClearAnnotations clears the value of the "annotations" field.
func (_u *NotificationEventUpdateOne) ClearAnnotations() *NotificationEventUpdateOne {
	_u.mutation.ClearAnnotations()
	return _u
}

// AddDeliveryStatusIDs adds the "delivery_statuses" edge to the NotificationEventDeliveryStatus entity by IDs.
func (_u *NotificationEventUpdateOne) AddDeliveryStatusIDs(ids ...string) *NotificationEventUpdateOne {
	_u.mutation.AddDeliveryStatusIDs(ids...)
	return _u
}

// AddDeliveryStatuses adds the "delivery_statuses" edges to the NotificationEventDeliveryStatus entity.
func (_u *NotificationEventUpdateOne) AddDeliveryStatuses(v ...*NotificationEventDeliveryStatus) *NotificationEventUpdateOne {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.AddDeliveryStatusIDs(ids...)
}

// Mutation returns the NotificationEventMutation object of the builder.
func (_u *NotificationEventUpdateOne) Mutation() *NotificationEventMutation {
	return _u.mutation
}

// ClearDeliveryStatuses clears all "delivery_statuses" edges to the NotificationEventDeliveryStatus entity.
func (_u *NotificationEventUpdateOne) ClearDeliveryStatuses() *NotificationEventUpdateOne {
	_u.mutation.ClearDeliveryStatuses()
	return _u
}

// RemoveDeliveryStatusIDs removes the "delivery_statuses" edge to NotificationEventDeliveryStatus entities by IDs.
func (_u *NotificationEventUpdateOne) RemoveDeliveryStatusIDs(ids ...string) *NotificationEventUpdateOne {
	_u.mutation.RemoveDeliveryStatusIDs(ids...)
	return _u
}

// RemoveDeliveryStatuses removes "delivery_statuses" edges to NotificationEventDeliveryStatus entities.
func (_u *NotificationEventUpdateOne) RemoveDeliveryStatuses(v ...*NotificationEventDeliveryStatus) *NotificationEventUpdateOne {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.RemoveDeliveryStatusIDs(ids...)
}

// Where appends a list predicates to the NotificationEventUpdate builder.
func (_u *NotificationEventUpdateOne) Where(ps ...predicate.NotificationEvent) *NotificationEventUpdateOne {
	_u.mutation.Where(ps...)
	return _u
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (_u *NotificationEventUpdateOne) Select(field string, fields ...string) *NotificationEventUpdateOne {
	_u.fields = append([]string{field}, fields...)
	return _u
}

// Save executes the query and returns the updated NotificationEvent entity.
func (_u *NotificationEventUpdateOne) Save(ctx context.Context) (*NotificationEvent, error) {
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *NotificationEventUpdateOne) SaveX(ctx context.Context) *NotificationEvent {
	node, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (_u *NotificationEventUpdateOne) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *NotificationEventUpdateOne) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *NotificationEventUpdateOne) check() error {
	if _u.mutation.RulesCleared() && len(_u.mutation.RulesIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "NotificationEvent.rules"`)
	}
	return nil
}

func (_u *NotificationEventUpdateOne) sqlSave(ctx context.Context) (_node *NotificationEvent, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(notificationevent.Table, notificationevent.Columns, sqlgraph.NewFieldSpec(notificationevent.FieldID, field.TypeString))
	id, ok := _u.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "NotificationEvent.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := _u.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, notificationevent.FieldID)
		for _, f := range fields {
			if !notificationevent.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != notificationevent.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.Payload(); ok {
		_spec.SetField(notificationevent.FieldPayload, field.TypeString, value)
	}
	if value, ok := _u.mutation.Annotations(); ok {
		vv, err := notificationevent.ValueScanner.Annotations.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(notificationevent.FieldAnnotations, field.TypeString, vv)
	}
	if _u.mutation.AnnotationsCleared() {
		_spec.ClearField(notificationevent.FieldAnnotations, field.TypeString)
	}
	if _u.mutation.DeliveryStatusesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.RemovedDeliveryStatusesIDs(); len(nodes) > 0 && !_u.mutation.DeliveryStatusesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.DeliveryStatusesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   notificationevent.DeliveryStatusesTable,
			Columns: notificationevent.DeliveryStatusesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(notificationeventdeliverystatus.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &NotificationEvent{config: _u.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{notificationevent.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	_u.mutation.done = true
	return _node, nil
}
