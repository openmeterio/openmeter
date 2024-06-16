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
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db/grant"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db/predicate"
)

// GrantUpdate is the builder for updating Grant entities.
type GrantUpdate struct {
	config
	hooks    []Hook
	mutation *GrantMutation
}

// Where appends a list predicates to the GrantUpdate builder.
func (gu *GrantUpdate) Where(ps ...predicate.Grant) *GrantUpdate {
	gu.mutation.Where(ps...)
	return gu
}

// SetMetadata sets the "metadata" field.
func (gu *GrantUpdate) SetMetadata(m map[string]string) *GrantUpdate {
	gu.mutation.SetMetadata(m)
	return gu
}

// ClearMetadata clears the value of the "metadata" field.
func (gu *GrantUpdate) ClearMetadata() *GrantUpdate {
	gu.mutation.ClearMetadata()
	return gu
}

// SetUpdatedAt sets the "updated_at" field.
func (gu *GrantUpdate) SetUpdatedAt(t time.Time) *GrantUpdate {
	gu.mutation.SetUpdatedAt(t)
	return gu
}

// SetDeletedAt sets the "deleted_at" field.
func (gu *GrantUpdate) SetDeletedAt(t time.Time) *GrantUpdate {
	gu.mutation.SetDeletedAt(t)
	return gu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (gu *GrantUpdate) SetNillableDeletedAt(t *time.Time) *GrantUpdate {
	if t != nil {
		gu.SetDeletedAt(*t)
	}
	return gu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (gu *GrantUpdate) ClearDeletedAt() *GrantUpdate {
	gu.mutation.ClearDeletedAt()
	return gu
}

// Mutation returns the GrantMutation object of the builder.
func (gu *GrantUpdate) Mutation() *GrantMutation {
	return gu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (gu *GrantUpdate) Save(ctx context.Context) (int, error) {
	gu.defaults()
	return withHooks(ctx, gu.sqlSave, gu.mutation, gu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (gu *GrantUpdate) SaveX(ctx context.Context) int {
	affected, err := gu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (gu *GrantUpdate) Exec(ctx context.Context) error {
	_, err := gu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gu *GrantUpdate) ExecX(ctx context.Context) {
	if err := gu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (gu *GrantUpdate) defaults() {
	if _, ok := gu.mutation.UpdatedAt(); !ok {
		v := grant.UpdateDefaultUpdatedAt()
		gu.mutation.SetUpdatedAt(v)
	}
}

func (gu *GrantUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(grant.Table, grant.Columns, sqlgraph.NewFieldSpec(grant.FieldID, field.TypeString))
	if ps := gu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := gu.mutation.Metadata(); ok {
		_spec.SetField(grant.FieldMetadata, field.TypeJSON, value)
	}
	if gu.mutation.MetadataCleared() {
		_spec.ClearField(grant.FieldMetadata, field.TypeJSON)
	}
	if value, ok := gu.mutation.UpdatedAt(); ok {
		_spec.SetField(grant.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := gu.mutation.DeletedAt(); ok {
		_spec.SetField(grant.FieldDeletedAt, field.TypeTime, value)
	}
	if gu.mutation.DeletedAtCleared() {
		_spec.ClearField(grant.FieldDeletedAt, field.TypeTime)
	}
	if gu.mutation.VoidedAtCleared() {
		_spec.ClearField(grant.FieldVoidedAt, field.TypeTime)
	}
	if gu.mutation.RecurrencePeriodCleared() {
		_spec.ClearField(grant.FieldRecurrencePeriod, field.TypeEnum)
	}
	if gu.mutation.RecurrenceAnchorCleared() {
		_spec.ClearField(grant.FieldRecurrenceAnchor, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, gu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{grant.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	gu.mutation.done = true
	return n, nil
}

// GrantUpdateOne is the builder for updating a single Grant entity.
type GrantUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *GrantMutation
}

// SetMetadata sets the "metadata" field.
func (guo *GrantUpdateOne) SetMetadata(m map[string]string) *GrantUpdateOne {
	guo.mutation.SetMetadata(m)
	return guo
}

// ClearMetadata clears the value of the "metadata" field.
func (guo *GrantUpdateOne) ClearMetadata() *GrantUpdateOne {
	guo.mutation.ClearMetadata()
	return guo
}

// SetUpdatedAt sets the "updated_at" field.
func (guo *GrantUpdateOne) SetUpdatedAt(t time.Time) *GrantUpdateOne {
	guo.mutation.SetUpdatedAt(t)
	return guo
}

// SetDeletedAt sets the "deleted_at" field.
func (guo *GrantUpdateOne) SetDeletedAt(t time.Time) *GrantUpdateOne {
	guo.mutation.SetDeletedAt(t)
	return guo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (guo *GrantUpdateOne) SetNillableDeletedAt(t *time.Time) *GrantUpdateOne {
	if t != nil {
		guo.SetDeletedAt(*t)
	}
	return guo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (guo *GrantUpdateOne) ClearDeletedAt() *GrantUpdateOne {
	guo.mutation.ClearDeletedAt()
	return guo
}

// Mutation returns the GrantMutation object of the builder.
func (guo *GrantUpdateOne) Mutation() *GrantMutation {
	return guo.mutation
}

// Where appends a list predicates to the GrantUpdate builder.
func (guo *GrantUpdateOne) Where(ps ...predicate.Grant) *GrantUpdateOne {
	guo.mutation.Where(ps...)
	return guo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (guo *GrantUpdateOne) Select(field string, fields ...string) *GrantUpdateOne {
	guo.fields = append([]string{field}, fields...)
	return guo
}

// Save executes the query and returns the updated Grant entity.
func (guo *GrantUpdateOne) Save(ctx context.Context) (*Grant, error) {
	guo.defaults()
	return withHooks(ctx, guo.sqlSave, guo.mutation, guo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (guo *GrantUpdateOne) SaveX(ctx context.Context) *Grant {
	node, err := guo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (guo *GrantUpdateOne) Exec(ctx context.Context) error {
	_, err := guo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (guo *GrantUpdateOne) ExecX(ctx context.Context) {
	if err := guo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (guo *GrantUpdateOne) defaults() {
	if _, ok := guo.mutation.UpdatedAt(); !ok {
		v := grant.UpdateDefaultUpdatedAt()
		guo.mutation.SetUpdatedAt(v)
	}
}

func (guo *GrantUpdateOne) sqlSave(ctx context.Context) (_node *Grant, err error) {
	_spec := sqlgraph.NewUpdateSpec(grant.Table, grant.Columns, sqlgraph.NewFieldSpec(grant.FieldID, field.TypeString))
	id, ok := guo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "Grant.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := guo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, grant.FieldID)
		for _, f := range fields {
			if !grant.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != grant.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := guo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := guo.mutation.Metadata(); ok {
		_spec.SetField(grant.FieldMetadata, field.TypeJSON, value)
	}
	if guo.mutation.MetadataCleared() {
		_spec.ClearField(grant.FieldMetadata, field.TypeJSON)
	}
	if value, ok := guo.mutation.UpdatedAt(); ok {
		_spec.SetField(grant.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := guo.mutation.DeletedAt(); ok {
		_spec.SetField(grant.FieldDeletedAt, field.TypeTime, value)
	}
	if guo.mutation.DeletedAtCleared() {
		_spec.ClearField(grant.FieldDeletedAt, field.TypeTime)
	}
	if guo.mutation.VoidedAtCleared() {
		_spec.ClearField(grant.FieldVoidedAt, field.TypeTime)
	}
	if guo.mutation.RecurrencePeriodCleared() {
		_spec.ClearField(grant.FieldRecurrencePeriod, field.TypeEnum)
	}
	if guo.mutation.RecurrenceAnchorCleared() {
		_spec.ClearField(grant.FieldRecurrenceAnchor, field.TypeTime)
	}
	_node = &Grant{config: guo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, guo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{grant.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	guo.mutation.done = true
	return _node, nil
}
