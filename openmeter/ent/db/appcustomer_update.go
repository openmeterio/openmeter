// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppCustomerUpdate is the builder for updating AppCustomer entities.
type AppCustomerUpdate struct {
	config
	hooks    []Hook
	mutation *AppCustomerMutation
}

// Where appends a list predicates to the AppCustomerUpdate builder.
func (acu *AppCustomerUpdate) Where(ps ...predicate.AppCustomer) *AppCustomerUpdate {
	acu.mutation.Where(ps...)
	return acu
}

// SetUpdatedAt sets the "updated_at" field.
func (acu *AppCustomerUpdate) SetUpdatedAt(t time.Time) *AppCustomerUpdate {
	acu.mutation.SetUpdatedAt(t)
	return acu
}

// SetDeletedAt sets the "deleted_at" field.
func (acu *AppCustomerUpdate) SetDeletedAt(t time.Time) *AppCustomerUpdate {
	acu.mutation.SetDeletedAt(t)
	return acu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (acu *AppCustomerUpdate) SetNillableDeletedAt(t *time.Time) *AppCustomerUpdate {
	if t != nil {
		acu.SetDeletedAt(*t)
	}
	return acu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (acu *AppCustomerUpdate) ClearDeletedAt() *AppCustomerUpdate {
	acu.mutation.ClearDeletedAt()
	return acu
}

// SetActions sets the "actions" field.
func (acu *AppCustomerUpdate) SetActions(ala []appentity.AppListenerAction) *AppCustomerUpdate {
	acu.mutation.SetActions(ala)
	return acu
}

// AppendActions appends ala to the "actions" field.
func (acu *AppCustomerUpdate) AppendActions(ala []appentity.AppListenerAction) *AppCustomerUpdate {
	acu.mutation.AppendActions(ala)
	return acu
}

// ClearActions clears the value of the "actions" field.
func (acu *AppCustomerUpdate) ClearActions() *AppCustomerUpdate {
	acu.mutation.ClearActions()
	return acu
}

// Mutation returns the AppCustomerMutation object of the builder.
func (acu *AppCustomerUpdate) Mutation() *AppCustomerMutation {
	return acu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (acu *AppCustomerUpdate) Save(ctx context.Context) (int, error) {
	acu.defaults()
	return withHooks(ctx, acu.sqlSave, acu.mutation, acu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (acu *AppCustomerUpdate) SaveX(ctx context.Context) int {
	affected, err := acu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (acu *AppCustomerUpdate) Exec(ctx context.Context) error {
	_, err := acu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (acu *AppCustomerUpdate) ExecX(ctx context.Context) {
	if err := acu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (acu *AppCustomerUpdate) defaults() {
	if _, ok := acu.mutation.UpdatedAt(); !ok {
		v := appcustomer.UpdateDefaultUpdatedAt()
		acu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (acu *AppCustomerUpdate) check() error {
	if acu.mutation.AppCleared() && len(acu.mutation.AppIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppCustomer.app"`)
	}
	if acu.mutation.CustomerCleared() && len(acu.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppCustomer.customer"`)
	}
	return nil
}

func (acu *AppCustomerUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := acu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(appcustomer.Table, appcustomer.Columns, sqlgraph.NewFieldSpec(appcustomer.FieldID, field.TypeInt))
	if ps := acu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := acu.mutation.UpdatedAt(); ok {
		_spec.SetField(appcustomer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := acu.mutation.DeletedAt(); ok {
		_spec.SetField(appcustomer.FieldDeletedAt, field.TypeTime, value)
	}
	if acu.mutation.DeletedAtCleared() {
		_spec.ClearField(appcustomer.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := acu.mutation.Actions(); ok {
		_spec.SetField(appcustomer.FieldActions, field.TypeJSON, value)
	}
	if value, ok := acu.mutation.AppendedActions(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, appcustomer.FieldActions, value)
		})
	}
	if acu.mutation.ActionsCleared() {
		_spec.ClearField(appcustomer.FieldActions, field.TypeJSON)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, acu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appcustomer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	acu.mutation.done = true
	return n, nil
}

// AppCustomerUpdateOne is the builder for updating a single AppCustomer entity.
type AppCustomerUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *AppCustomerMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (acuo *AppCustomerUpdateOne) SetUpdatedAt(t time.Time) *AppCustomerUpdateOne {
	acuo.mutation.SetUpdatedAt(t)
	return acuo
}

// SetDeletedAt sets the "deleted_at" field.
func (acuo *AppCustomerUpdateOne) SetDeletedAt(t time.Time) *AppCustomerUpdateOne {
	acuo.mutation.SetDeletedAt(t)
	return acuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (acuo *AppCustomerUpdateOne) SetNillableDeletedAt(t *time.Time) *AppCustomerUpdateOne {
	if t != nil {
		acuo.SetDeletedAt(*t)
	}
	return acuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (acuo *AppCustomerUpdateOne) ClearDeletedAt() *AppCustomerUpdateOne {
	acuo.mutation.ClearDeletedAt()
	return acuo
}

// SetActions sets the "actions" field.
func (acuo *AppCustomerUpdateOne) SetActions(ala []appentity.AppListenerAction) *AppCustomerUpdateOne {
	acuo.mutation.SetActions(ala)
	return acuo
}

// AppendActions appends ala to the "actions" field.
func (acuo *AppCustomerUpdateOne) AppendActions(ala []appentity.AppListenerAction) *AppCustomerUpdateOne {
	acuo.mutation.AppendActions(ala)
	return acuo
}

// ClearActions clears the value of the "actions" field.
func (acuo *AppCustomerUpdateOne) ClearActions() *AppCustomerUpdateOne {
	acuo.mutation.ClearActions()
	return acuo
}

// Mutation returns the AppCustomerMutation object of the builder.
func (acuo *AppCustomerUpdateOne) Mutation() *AppCustomerMutation {
	return acuo.mutation
}

// Where appends a list predicates to the AppCustomerUpdate builder.
func (acuo *AppCustomerUpdateOne) Where(ps ...predicate.AppCustomer) *AppCustomerUpdateOne {
	acuo.mutation.Where(ps...)
	return acuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (acuo *AppCustomerUpdateOne) Select(field string, fields ...string) *AppCustomerUpdateOne {
	acuo.fields = append([]string{field}, fields...)
	return acuo
}

// Save executes the query and returns the updated AppCustomer entity.
func (acuo *AppCustomerUpdateOne) Save(ctx context.Context) (*AppCustomer, error) {
	acuo.defaults()
	return withHooks(ctx, acuo.sqlSave, acuo.mutation, acuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (acuo *AppCustomerUpdateOne) SaveX(ctx context.Context) *AppCustomer {
	node, err := acuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (acuo *AppCustomerUpdateOne) Exec(ctx context.Context) error {
	_, err := acuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (acuo *AppCustomerUpdateOne) ExecX(ctx context.Context) {
	if err := acuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (acuo *AppCustomerUpdateOne) defaults() {
	if _, ok := acuo.mutation.UpdatedAt(); !ok {
		v := appcustomer.UpdateDefaultUpdatedAt()
		acuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (acuo *AppCustomerUpdateOne) check() error {
	if acuo.mutation.AppCleared() && len(acuo.mutation.AppIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppCustomer.app"`)
	}
	if acuo.mutation.CustomerCleared() && len(acuo.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppCustomer.customer"`)
	}
	return nil
}

func (acuo *AppCustomerUpdateOne) sqlSave(ctx context.Context) (_node *AppCustomer, err error) {
	if err := acuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(appcustomer.Table, appcustomer.Columns, sqlgraph.NewFieldSpec(appcustomer.FieldID, field.TypeInt))
	id, ok := acuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "AppCustomer.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := acuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, appcustomer.FieldID)
		for _, f := range fields {
			if !appcustomer.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != appcustomer.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := acuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := acuo.mutation.UpdatedAt(); ok {
		_spec.SetField(appcustomer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := acuo.mutation.DeletedAt(); ok {
		_spec.SetField(appcustomer.FieldDeletedAt, field.TypeTime, value)
	}
	if acuo.mutation.DeletedAtCleared() {
		_spec.ClearField(appcustomer.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := acuo.mutation.Actions(); ok {
		_spec.SetField(appcustomer.FieldActions, field.TypeJSON, value)
	}
	if value, ok := acuo.mutation.AppendedActions(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, appcustomer.FieldActions, value)
		})
	}
	if acuo.mutation.ActionsCleared() {
		_spec.ClearField(appcustomer.FieldActions, field.TypeJSON)
	}
	_node = &AppCustomer{config: acuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, acuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appcustomer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	acuo.mutation.done = true
	return _node, nil
}
