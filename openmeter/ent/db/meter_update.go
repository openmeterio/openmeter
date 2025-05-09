// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	dbmeter "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// MeterUpdate is the builder for updating Meter entities.
type MeterUpdate struct {
	config
	hooks    []Hook
	mutation *MeterMutation
}

// Where appends a list predicates to the MeterUpdate builder.
func (_u *MeterUpdate) Where(ps ...predicate.Meter) *MeterUpdate {
	_u.mutation.Where(ps...)
	return _u
}

// SetMetadata sets the "metadata" field.
func (_u *MeterUpdate) SetMetadata(v map[string]string) *MeterUpdate {
	_u.mutation.SetMetadata(v)
	return _u
}

// ClearMetadata clears the value of the "metadata" field.
func (_u *MeterUpdate) ClearMetadata() *MeterUpdate {
	_u.mutation.ClearMetadata()
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *MeterUpdate) SetUpdatedAt(v time.Time) *MeterUpdate {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *MeterUpdate) SetDeletedAt(v time.Time) *MeterUpdate {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *MeterUpdate) SetNillableDeletedAt(v *time.Time) *MeterUpdate {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *MeterUpdate) ClearDeletedAt() *MeterUpdate {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetName sets the "name" field.
func (_u *MeterUpdate) SetName(v string) *MeterUpdate {
	_u.mutation.SetName(v)
	return _u
}

// SetNillableName sets the "name" field if the given value is not nil.
func (_u *MeterUpdate) SetNillableName(v *string) *MeterUpdate {
	if v != nil {
		_u.SetName(*v)
	}
	return _u
}

// SetDescription sets the "description" field.
func (_u *MeterUpdate) SetDescription(v string) *MeterUpdate {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *MeterUpdate) SetNillableDescription(v *string) *MeterUpdate {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *MeterUpdate) ClearDescription() *MeterUpdate {
	_u.mutation.ClearDescription()
	return _u
}

// SetValueProperty sets the "value_property" field.
func (_u *MeterUpdate) SetValueProperty(v string) *MeterUpdate {
	_u.mutation.SetValueProperty(v)
	return _u
}

// SetNillableValueProperty sets the "value_property" field if the given value is not nil.
func (_u *MeterUpdate) SetNillableValueProperty(v *string) *MeterUpdate {
	if v != nil {
		_u.SetValueProperty(*v)
	}
	return _u
}

// ClearValueProperty clears the value of the "value_property" field.
func (_u *MeterUpdate) ClearValueProperty() *MeterUpdate {
	_u.mutation.ClearValueProperty()
	return _u
}

// SetGroupBy sets the "group_by" field.
func (_u *MeterUpdate) SetGroupBy(v map[string]string) *MeterUpdate {
	_u.mutation.SetGroupBy(v)
	return _u
}

// ClearGroupBy clears the value of the "group_by" field.
func (_u *MeterUpdate) ClearGroupBy() *MeterUpdate {
	_u.mutation.ClearGroupBy()
	return _u
}

// SetEventFrom sets the "event_from" field.
func (_u *MeterUpdate) SetEventFrom(v time.Time) *MeterUpdate {
	_u.mutation.SetEventFrom(v)
	return _u
}

// SetNillableEventFrom sets the "event_from" field if the given value is not nil.
func (_u *MeterUpdate) SetNillableEventFrom(v *time.Time) *MeterUpdate {
	if v != nil {
		_u.SetEventFrom(*v)
	}
	return _u
}

// ClearEventFrom clears the value of the "event_from" field.
func (_u *MeterUpdate) ClearEventFrom() *MeterUpdate {
	_u.mutation.ClearEventFrom()
	return _u
}

// Mutation returns the MeterMutation object of the builder.
func (_u *MeterUpdate) Mutation() *MeterMutation {
	return _u.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (_u *MeterUpdate) Save(ctx context.Context) (int, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *MeterUpdate) SaveX(ctx context.Context) int {
	affected, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (_u *MeterUpdate) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *MeterUpdate) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *MeterUpdate) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := dbmeter.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

func (_u *MeterUpdate) sqlSave(ctx context.Context) (_node int, err error) {
	_spec := sqlgraph.NewUpdateSpec(dbmeter.Table, dbmeter.Columns, sqlgraph.NewFieldSpec(dbmeter.FieldID, field.TypeString))
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.Metadata(); ok {
		_spec.SetField(dbmeter.FieldMetadata, field.TypeJSON, value)
	}
	if _u.mutation.MetadataCleared() {
		_spec.ClearField(dbmeter.FieldMetadata, field.TypeJSON)
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(dbmeter.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(dbmeter.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(dbmeter.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.Name(); ok {
		_spec.SetField(dbmeter.FieldName, field.TypeString, value)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(dbmeter.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(dbmeter.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.ValueProperty(); ok {
		_spec.SetField(dbmeter.FieldValueProperty, field.TypeString, value)
	}
	if _u.mutation.ValuePropertyCleared() {
		_spec.ClearField(dbmeter.FieldValueProperty, field.TypeString)
	}
	if value, ok := _u.mutation.GroupBy(); ok {
		_spec.SetField(dbmeter.FieldGroupBy, field.TypeJSON, value)
	}
	if _u.mutation.GroupByCleared() {
		_spec.ClearField(dbmeter.FieldGroupBy, field.TypeJSON)
	}
	if value, ok := _u.mutation.EventFrom(); ok {
		_spec.SetField(dbmeter.FieldEventFrom, field.TypeTime, value)
	}
	if _u.mutation.EventFromCleared() {
		_spec.ClearField(dbmeter.FieldEventFrom, field.TypeTime)
	}
	if _node, err = sqlgraph.UpdateNodes(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{dbmeter.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	_u.mutation.done = true
	return _node, nil
}

// MeterUpdateOne is the builder for updating a single Meter entity.
type MeterUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *MeterMutation
}

// SetMetadata sets the "metadata" field.
func (_u *MeterUpdateOne) SetMetadata(v map[string]string) *MeterUpdateOne {
	_u.mutation.SetMetadata(v)
	return _u
}

// ClearMetadata clears the value of the "metadata" field.
func (_u *MeterUpdateOne) ClearMetadata() *MeterUpdateOne {
	_u.mutation.ClearMetadata()
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *MeterUpdateOne) SetUpdatedAt(v time.Time) *MeterUpdateOne {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *MeterUpdateOne) SetDeletedAt(v time.Time) *MeterUpdateOne {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *MeterUpdateOne) SetNillableDeletedAt(v *time.Time) *MeterUpdateOne {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *MeterUpdateOne) ClearDeletedAt() *MeterUpdateOne {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetName sets the "name" field.
func (_u *MeterUpdateOne) SetName(v string) *MeterUpdateOne {
	_u.mutation.SetName(v)
	return _u
}

// SetNillableName sets the "name" field if the given value is not nil.
func (_u *MeterUpdateOne) SetNillableName(v *string) *MeterUpdateOne {
	if v != nil {
		_u.SetName(*v)
	}
	return _u
}

// SetDescription sets the "description" field.
func (_u *MeterUpdateOne) SetDescription(v string) *MeterUpdateOne {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *MeterUpdateOne) SetNillableDescription(v *string) *MeterUpdateOne {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *MeterUpdateOne) ClearDescription() *MeterUpdateOne {
	_u.mutation.ClearDescription()
	return _u
}

// SetValueProperty sets the "value_property" field.
func (_u *MeterUpdateOne) SetValueProperty(v string) *MeterUpdateOne {
	_u.mutation.SetValueProperty(v)
	return _u
}

// SetNillableValueProperty sets the "value_property" field if the given value is not nil.
func (_u *MeterUpdateOne) SetNillableValueProperty(v *string) *MeterUpdateOne {
	if v != nil {
		_u.SetValueProperty(*v)
	}
	return _u
}

// ClearValueProperty clears the value of the "value_property" field.
func (_u *MeterUpdateOne) ClearValueProperty() *MeterUpdateOne {
	_u.mutation.ClearValueProperty()
	return _u
}

// SetGroupBy sets the "group_by" field.
func (_u *MeterUpdateOne) SetGroupBy(v map[string]string) *MeterUpdateOne {
	_u.mutation.SetGroupBy(v)
	return _u
}

// ClearGroupBy clears the value of the "group_by" field.
func (_u *MeterUpdateOne) ClearGroupBy() *MeterUpdateOne {
	_u.mutation.ClearGroupBy()
	return _u
}

// SetEventFrom sets the "event_from" field.
func (_u *MeterUpdateOne) SetEventFrom(v time.Time) *MeterUpdateOne {
	_u.mutation.SetEventFrom(v)
	return _u
}

// SetNillableEventFrom sets the "event_from" field if the given value is not nil.
func (_u *MeterUpdateOne) SetNillableEventFrom(v *time.Time) *MeterUpdateOne {
	if v != nil {
		_u.SetEventFrom(*v)
	}
	return _u
}

// ClearEventFrom clears the value of the "event_from" field.
func (_u *MeterUpdateOne) ClearEventFrom() *MeterUpdateOne {
	_u.mutation.ClearEventFrom()
	return _u
}

// Mutation returns the MeterMutation object of the builder.
func (_u *MeterUpdateOne) Mutation() *MeterMutation {
	return _u.mutation
}

// Where appends a list predicates to the MeterUpdate builder.
func (_u *MeterUpdateOne) Where(ps ...predicate.Meter) *MeterUpdateOne {
	_u.mutation.Where(ps...)
	return _u
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (_u *MeterUpdateOne) Select(field string, fields ...string) *MeterUpdateOne {
	_u.fields = append([]string{field}, fields...)
	return _u
}

// Save executes the query and returns the updated Meter entity.
func (_u *MeterUpdateOne) Save(ctx context.Context) (*Meter, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *MeterUpdateOne) SaveX(ctx context.Context) *Meter {
	node, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (_u *MeterUpdateOne) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *MeterUpdateOne) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *MeterUpdateOne) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := dbmeter.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

func (_u *MeterUpdateOne) sqlSave(ctx context.Context) (_node *Meter, err error) {
	_spec := sqlgraph.NewUpdateSpec(dbmeter.Table, dbmeter.Columns, sqlgraph.NewFieldSpec(dbmeter.FieldID, field.TypeString))
	id, ok := _u.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "Meter.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := _u.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, dbmeter.FieldID)
		for _, f := range fields {
			if !dbmeter.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != dbmeter.FieldID {
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
	if value, ok := _u.mutation.Metadata(); ok {
		_spec.SetField(dbmeter.FieldMetadata, field.TypeJSON, value)
	}
	if _u.mutation.MetadataCleared() {
		_spec.ClearField(dbmeter.FieldMetadata, field.TypeJSON)
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(dbmeter.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(dbmeter.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(dbmeter.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.Name(); ok {
		_spec.SetField(dbmeter.FieldName, field.TypeString, value)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(dbmeter.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(dbmeter.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.ValueProperty(); ok {
		_spec.SetField(dbmeter.FieldValueProperty, field.TypeString, value)
	}
	if _u.mutation.ValuePropertyCleared() {
		_spec.ClearField(dbmeter.FieldValueProperty, field.TypeString)
	}
	if value, ok := _u.mutation.GroupBy(); ok {
		_spec.SetField(dbmeter.FieldGroupBy, field.TypeJSON, value)
	}
	if _u.mutation.GroupByCleared() {
		_spec.ClearField(dbmeter.FieldGroupBy, field.TypeJSON)
	}
	if value, ok := _u.mutation.EventFrom(); ok {
		_spec.SetField(dbmeter.FieldEventFrom, field.TypeTime, value)
	}
	if _u.mutation.EventFromCleared() {
		_spec.ClearField(dbmeter.FieldEventFrom, field.TypeTime)
	}
	_node = &Meter{config: _u.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{dbmeter.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	_u.mutation.done = true
	return _node, nil
}
