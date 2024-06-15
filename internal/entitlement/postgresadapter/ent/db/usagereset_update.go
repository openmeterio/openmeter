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
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/predicate"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/usagereset"
)

// UsageResetUpdate is the builder for updating UsageReset entities.
type UsageResetUpdate struct {
	config
	hooks    []Hook
	mutation *UsageResetMutation
}

// Where appends a list predicates to the UsageResetUpdate builder.
func (uru *UsageResetUpdate) Where(ps ...predicate.UsageReset) *UsageResetUpdate {
	uru.mutation.Where(ps...)
	return uru
}

// SetUpdatedAt sets the "updated_at" field.
func (uru *UsageResetUpdate) SetUpdatedAt(t time.Time) *UsageResetUpdate {
	uru.mutation.SetUpdatedAt(t)
	return uru
}

// SetDeletedAt sets the "deleted_at" field.
func (uru *UsageResetUpdate) SetDeletedAt(t time.Time) *UsageResetUpdate {
	uru.mutation.SetDeletedAt(t)
	return uru
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (uru *UsageResetUpdate) SetNillableDeletedAt(t *time.Time) *UsageResetUpdate {
	if t != nil {
		uru.SetDeletedAt(*t)
	}
	return uru
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (uru *UsageResetUpdate) ClearDeletedAt() *UsageResetUpdate {
	uru.mutation.ClearDeletedAt()
	return uru
}

// Mutation returns the UsageResetMutation object of the builder.
func (uru *UsageResetUpdate) Mutation() *UsageResetMutation {
	return uru.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (uru *UsageResetUpdate) Save(ctx context.Context) (int, error) {
	uru.defaults()
	return withHooks(ctx, uru.sqlSave, uru.mutation, uru.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (uru *UsageResetUpdate) SaveX(ctx context.Context) int {
	affected, err := uru.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (uru *UsageResetUpdate) Exec(ctx context.Context) error {
	_, err := uru.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (uru *UsageResetUpdate) ExecX(ctx context.Context) {
	if err := uru.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (uru *UsageResetUpdate) defaults() {
	if _, ok := uru.mutation.UpdatedAt(); !ok {
		v := usagereset.UpdateDefaultUpdatedAt()
		uru.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (uru *UsageResetUpdate) check() error {
	if _, ok := uru.mutation.EntitlementID(); uru.mutation.EntitlementCleared() && !ok {
		return errors.New(`db: clearing a required unique edge "UsageReset.entitlement"`)
	}
	return nil
}

func (uru *UsageResetUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := uru.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(usagereset.Table, usagereset.Columns, sqlgraph.NewFieldSpec(usagereset.FieldID, field.TypeString))
	if ps := uru.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := uru.mutation.UpdatedAt(); ok {
		_spec.SetField(usagereset.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := uru.mutation.DeletedAt(); ok {
		_spec.SetField(usagereset.FieldDeletedAt, field.TypeTime, value)
	}
	if uru.mutation.DeletedAtCleared() {
		_spec.ClearField(usagereset.FieldDeletedAt, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, uru.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{usagereset.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	uru.mutation.done = true
	return n, nil
}

// UsageResetUpdateOne is the builder for updating a single UsageReset entity.
type UsageResetUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *UsageResetMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (uruo *UsageResetUpdateOne) SetUpdatedAt(t time.Time) *UsageResetUpdateOne {
	uruo.mutation.SetUpdatedAt(t)
	return uruo
}

// SetDeletedAt sets the "deleted_at" field.
func (uruo *UsageResetUpdateOne) SetDeletedAt(t time.Time) *UsageResetUpdateOne {
	uruo.mutation.SetDeletedAt(t)
	return uruo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (uruo *UsageResetUpdateOne) SetNillableDeletedAt(t *time.Time) *UsageResetUpdateOne {
	if t != nil {
		uruo.SetDeletedAt(*t)
	}
	return uruo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (uruo *UsageResetUpdateOne) ClearDeletedAt() *UsageResetUpdateOne {
	uruo.mutation.ClearDeletedAt()
	return uruo
}

// Mutation returns the UsageResetMutation object of the builder.
func (uruo *UsageResetUpdateOne) Mutation() *UsageResetMutation {
	return uruo.mutation
}

// Where appends a list predicates to the UsageResetUpdate builder.
func (uruo *UsageResetUpdateOne) Where(ps ...predicate.UsageReset) *UsageResetUpdateOne {
	uruo.mutation.Where(ps...)
	return uruo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (uruo *UsageResetUpdateOne) Select(field string, fields ...string) *UsageResetUpdateOne {
	uruo.fields = append([]string{field}, fields...)
	return uruo
}

// Save executes the query and returns the updated UsageReset entity.
func (uruo *UsageResetUpdateOne) Save(ctx context.Context) (*UsageReset, error) {
	uruo.defaults()
	return withHooks(ctx, uruo.sqlSave, uruo.mutation, uruo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (uruo *UsageResetUpdateOne) SaveX(ctx context.Context) *UsageReset {
	node, err := uruo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (uruo *UsageResetUpdateOne) Exec(ctx context.Context) error {
	_, err := uruo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (uruo *UsageResetUpdateOne) ExecX(ctx context.Context) {
	if err := uruo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (uruo *UsageResetUpdateOne) defaults() {
	if _, ok := uruo.mutation.UpdatedAt(); !ok {
		v := usagereset.UpdateDefaultUpdatedAt()
		uruo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (uruo *UsageResetUpdateOne) check() error {
	if _, ok := uruo.mutation.EntitlementID(); uruo.mutation.EntitlementCleared() && !ok {
		return errors.New(`db: clearing a required unique edge "UsageReset.entitlement"`)
	}
	return nil
}

func (uruo *UsageResetUpdateOne) sqlSave(ctx context.Context) (_node *UsageReset, err error) {
	if err := uruo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(usagereset.Table, usagereset.Columns, sqlgraph.NewFieldSpec(usagereset.FieldID, field.TypeString))
	id, ok := uruo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "UsageReset.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := uruo.fields; len(fields) > 0 {
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
	if ps := uruo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := uruo.mutation.UpdatedAt(); ok {
		_spec.SetField(usagereset.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := uruo.mutation.DeletedAt(); ok {
		_spec.SetField(usagereset.FieldDeletedAt, field.TypeTime, value)
	}
	if uruo.mutation.DeletedAtCleared() {
		_spec.ClearField(usagereset.FieldDeletedAt, field.TypeTime)
	}
	_node = &UsageReset{config: uruo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, uruo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{usagereset.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	uruo.mutation.done = true
	return _node, nil
}
