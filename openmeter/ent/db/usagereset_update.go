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
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
)

// UsageResetUpdate is the builder for updating UsageReset entities.
type UsageResetUpdate struct {
	config
	hooks    []Hook
	mutation *UsageResetMutation
}

// Where appends a list predicates to the UsageResetUpdate builder.
func (_u *UsageResetUpdate) Where(ps ...predicate.UsageReset) *UsageResetUpdate {
	_u.mutation.Where(ps...)
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *UsageResetUpdate) SetUpdatedAt(v time.Time) *UsageResetUpdate {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *UsageResetUpdate) SetDeletedAt(v time.Time) *UsageResetUpdate {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *UsageResetUpdate) SetNillableDeletedAt(v *time.Time) *UsageResetUpdate {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *UsageResetUpdate) ClearDeletedAt() *UsageResetUpdate {
	_u.mutation.ClearDeletedAt()
	return _u
}

// Mutation returns the UsageResetMutation object of the builder.
func (_u *UsageResetUpdate) Mutation() *UsageResetMutation {
	return _u.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (_u *UsageResetUpdate) Save(ctx context.Context) (int, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *UsageResetUpdate) SaveX(ctx context.Context) int {
	affected, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (_u *UsageResetUpdate) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *UsageResetUpdate) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *UsageResetUpdate) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := usagereset.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *UsageResetUpdate) check() error {
	if _u.mutation.EntitlementCleared() && len(_u.mutation.EntitlementIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "UsageReset.entitlement"`)
	}
	return nil
}

func (_u *UsageResetUpdate) sqlSave(ctx context.Context) (_node int, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(usagereset.Table, usagereset.Columns, sqlgraph.NewFieldSpec(usagereset.FieldID, field.TypeString))
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(usagereset.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(usagereset.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(usagereset.FieldDeletedAt, field.TypeTime)
	}
	if _node, err = sqlgraph.UpdateNodes(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{usagereset.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	_u.mutation.done = true
	return _node, nil
}

// UsageResetUpdateOne is the builder for updating a single UsageReset entity.
type UsageResetUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *UsageResetMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *UsageResetUpdateOne) SetUpdatedAt(v time.Time) *UsageResetUpdateOne {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *UsageResetUpdateOne) SetDeletedAt(v time.Time) *UsageResetUpdateOne {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *UsageResetUpdateOne) SetNillableDeletedAt(v *time.Time) *UsageResetUpdateOne {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *UsageResetUpdateOne) ClearDeletedAt() *UsageResetUpdateOne {
	_u.mutation.ClearDeletedAt()
	return _u
}

// Mutation returns the UsageResetMutation object of the builder.
func (_u *UsageResetUpdateOne) Mutation() *UsageResetMutation {
	return _u.mutation
}

// Where appends a list predicates to the UsageResetUpdate builder.
func (_u *UsageResetUpdateOne) Where(ps ...predicate.UsageReset) *UsageResetUpdateOne {
	_u.mutation.Where(ps...)
	return _u
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (_u *UsageResetUpdateOne) Select(field string, fields ...string) *UsageResetUpdateOne {
	_u.fields = append([]string{field}, fields...)
	return _u
}

// Save executes the query and returns the updated UsageReset entity.
func (_u *UsageResetUpdateOne) Save(ctx context.Context) (*UsageReset, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *UsageResetUpdateOne) SaveX(ctx context.Context) *UsageReset {
	node, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (_u *UsageResetUpdateOne) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *UsageResetUpdateOne) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *UsageResetUpdateOne) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := usagereset.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *UsageResetUpdateOne) check() error {
	if _u.mutation.EntitlementCleared() && len(_u.mutation.EntitlementIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "UsageReset.entitlement"`)
	}
	return nil
}

func (_u *UsageResetUpdateOne) sqlSave(ctx context.Context) (_node *UsageReset, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(usagereset.Table, usagereset.Columns, sqlgraph.NewFieldSpec(usagereset.FieldID, field.TypeString))
	id, ok := _u.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "UsageReset.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := _u.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, usagereset.FieldID)
		for _, f := range fields {
			if !usagereset.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != usagereset.FieldID {
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
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(usagereset.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(usagereset.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(usagereset.FieldDeletedAt, field.TypeTime)
	}
	_node = &UsageReset{config: _u.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{usagereset.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	_u.mutation.done = true
	return _node, nil
}
