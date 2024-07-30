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
	"github.com/openmeterio/openmeter/internal/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/internal/ent/db/predicate"
)

// BalanceSnapshotUpdate is the builder for updating BalanceSnapshot entities.
type BalanceSnapshotUpdate struct {
	config
	hooks    []Hook
	mutation *BalanceSnapshotMutation
}

// Where appends a list predicates to the BalanceSnapshotUpdate builder.
func (bsu *BalanceSnapshotUpdate) Where(ps ...predicate.BalanceSnapshot) *BalanceSnapshotUpdate {
	bsu.mutation.Where(ps...)
	return bsu
}

// SetUpdatedAt sets the "updated_at" field.
func (bsu *BalanceSnapshotUpdate) SetUpdatedAt(t time.Time) *BalanceSnapshotUpdate {
	bsu.mutation.SetUpdatedAt(t)
	return bsu
}

// SetDeletedAt sets the "deleted_at" field.
func (bsu *BalanceSnapshotUpdate) SetDeletedAt(t time.Time) *BalanceSnapshotUpdate {
	bsu.mutation.SetDeletedAt(t)
	return bsu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bsu *BalanceSnapshotUpdate) SetNillableDeletedAt(t *time.Time) *BalanceSnapshotUpdate {
	if t != nil {
		bsu.SetDeletedAt(*t)
	}
	return bsu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bsu *BalanceSnapshotUpdate) ClearDeletedAt() *BalanceSnapshotUpdate {
	bsu.mutation.ClearDeletedAt()
	return bsu
}

// Mutation returns the BalanceSnapshotMutation object of the builder.
func (bsu *BalanceSnapshotUpdate) Mutation() *BalanceSnapshotMutation {
	return bsu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (bsu *BalanceSnapshotUpdate) Save(ctx context.Context) (int, error) {
	bsu.defaults()
	return withHooks(ctx, bsu.sqlSave, bsu.mutation, bsu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bsu *BalanceSnapshotUpdate) SaveX(ctx context.Context) int {
	affected, err := bsu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (bsu *BalanceSnapshotUpdate) Exec(ctx context.Context) error {
	_, err := bsu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bsu *BalanceSnapshotUpdate) ExecX(ctx context.Context) {
	if err := bsu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bsu *BalanceSnapshotUpdate) defaults() {
	if _, ok := bsu.mutation.UpdatedAt(); !ok {
		v := balancesnapshot.UpdateDefaultUpdatedAt()
		bsu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bsu *BalanceSnapshotUpdate) check() error {
	if bsu.mutation.EntitlementCleared() && len(bsu.mutation.EntitlementIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BalanceSnapshot.entitlement"`)
	}
	return nil
}

func (bsu *BalanceSnapshotUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := bsu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(balancesnapshot.Table, balancesnapshot.Columns, sqlgraph.NewFieldSpec(balancesnapshot.FieldID, field.TypeInt))
	if ps := bsu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bsu.mutation.UpdatedAt(); ok {
		_spec.SetField(balancesnapshot.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bsu.mutation.DeletedAt(); ok {
		_spec.SetField(balancesnapshot.FieldDeletedAt, field.TypeTime, value)
	}
	if bsu.mutation.DeletedAtCleared() {
		_spec.ClearField(balancesnapshot.FieldDeletedAt, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, bsu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{balancesnapshot.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	bsu.mutation.done = true
	return n, nil
}

// BalanceSnapshotUpdateOne is the builder for updating a single BalanceSnapshot entity.
type BalanceSnapshotUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BalanceSnapshotMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (bsuo *BalanceSnapshotUpdateOne) SetUpdatedAt(t time.Time) *BalanceSnapshotUpdateOne {
	bsuo.mutation.SetUpdatedAt(t)
	return bsuo
}

// SetDeletedAt sets the "deleted_at" field.
func (bsuo *BalanceSnapshotUpdateOne) SetDeletedAt(t time.Time) *BalanceSnapshotUpdateOne {
	bsuo.mutation.SetDeletedAt(t)
	return bsuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bsuo *BalanceSnapshotUpdateOne) SetNillableDeletedAt(t *time.Time) *BalanceSnapshotUpdateOne {
	if t != nil {
		bsuo.SetDeletedAt(*t)
	}
	return bsuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bsuo *BalanceSnapshotUpdateOne) ClearDeletedAt() *BalanceSnapshotUpdateOne {
	bsuo.mutation.ClearDeletedAt()
	return bsuo
}

// Mutation returns the BalanceSnapshotMutation object of the builder.
func (bsuo *BalanceSnapshotUpdateOne) Mutation() *BalanceSnapshotMutation {
	return bsuo.mutation
}

// Where appends a list predicates to the BalanceSnapshotUpdate builder.
func (bsuo *BalanceSnapshotUpdateOne) Where(ps ...predicate.BalanceSnapshot) *BalanceSnapshotUpdateOne {
	bsuo.mutation.Where(ps...)
	return bsuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (bsuo *BalanceSnapshotUpdateOne) Select(field string, fields ...string) *BalanceSnapshotUpdateOne {
	bsuo.fields = append([]string{field}, fields...)
	return bsuo
}

// Save executes the query and returns the updated BalanceSnapshot entity.
func (bsuo *BalanceSnapshotUpdateOne) Save(ctx context.Context) (*BalanceSnapshot, error) {
	bsuo.defaults()
	return withHooks(ctx, bsuo.sqlSave, bsuo.mutation, bsuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bsuo *BalanceSnapshotUpdateOne) SaveX(ctx context.Context) *BalanceSnapshot {
	node, err := bsuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (bsuo *BalanceSnapshotUpdateOne) Exec(ctx context.Context) error {
	_, err := bsuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bsuo *BalanceSnapshotUpdateOne) ExecX(ctx context.Context) {
	if err := bsuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bsuo *BalanceSnapshotUpdateOne) defaults() {
	if _, ok := bsuo.mutation.UpdatedAt(); !ok {
		v := balancesnapshot.UpdateDefaultUpdatedAt()
		bsuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bsuo *BalanceSnapshotUpdateOne) check() error {
	if bsuo.mutation.EntitlementCleared() && len(bsuo.mutation.EntitlementIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BalanceSnapshot.entitlement"`)
	}
	return nil
}

func (bsuo *BalanceSnapshotUpdateOne) sqlSave(ctx context.Context) (_node *BalanceSnapshot, err error) {
	if err := bsuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(balancesnapshot.Table, balancesnapshot.Columns, sqlgraph.NewFieldSpec(balancesnapshot.FieldID, field.TypeInt))
	id, ok := bsuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BalanceSnapshot.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := bsuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, balancesnapshot.FieldID)
		for _, f := range fields {
			if !balancesnapshot.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != balancesnapshot.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := bsuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bsuo.mutation.UpdatedAt(); ok {
		_spec.SetField(balancesnapshot.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bsuo.mutation.DeletedAt(); ok {
		_spec.SetField(balancesnapshot.FieldDeletedAt, field.TypeTime, value)
	}
	if bsuo.mutation.DeletedAtCleared() {
		_spec.ClearField(balancesnapshot.FieldDeletedAt, field.TypeTime)
	}
	_node = &BalanceSnapshot{config: bsuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, bsuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{balancesnapshot.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	bsuo.mutation.done = true
	return _node, nil
}
