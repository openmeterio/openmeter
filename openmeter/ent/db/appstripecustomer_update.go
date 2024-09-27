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
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppStripeCustomerUpdate is the builder for updating AppStripeCustomer entities.
type AppStripeCustomerUpdate struct {
	config
	hooks    []Hook
	mutation *AppStripeCustomerMutation
}

// Where appends a list predicates to the AppStripeCustomerUpdate builder.
func (ascu *AppStripeCustomerUpdate) Where(ps ...predicate.AppStripeCustomer) *AppStripeCustomerUpdate {
	ascu.mutation.Where(ps...)
	return ascu
}

// SetUpdatedAt sets the "updated_at" field.
func (ascu *AppStripeCustomerUpdate) SetUpdatedAt(t time.Time) *AppStripeCustomerUpdate {
	ascu.mutation.SetUpdatedAt(t)
	return ascu
}

// SetDeletedAt sets the "deleted_at" field.
func (ascu *AppStripeCustomerUpdate) SetDeletedAt(t time.Time) *AppStripeCustomerUpdate {
	ascu.mutation.SetDeletedAt(t)
	return ascu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (ascu *AppStripeCustomerUpdate) SetNillableDeletedAt(t *time.Time) *AppStripeCustomerUpdate {
	if t != nil {
		ascu.SetDeletedAt(*t)
	}
	return ascu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (ascu *AppStripeCustomerUpdate) ClearDeletedAt() *AppStripeCustomerUpdate {
	ascu.mutation.ClearDeletedAt()
	return ascu
}

// SetStripeCustomerID sets the "stripe_customer_id" field.
func (ascu *AppStripeCustomerUpdate) SetStripeCustomerID(s string) *AppStripeCustomerUpdate {
	ascu.mutation.SetStripeCustomerID(s)
	return ascu
}

// SetNillableStripeCustomerID sets the "stripe_customer_id" field if the given value is not nil.
func (ascu *AppStripeCustomerUpdate) SetNillableStripeCustomerID(s *string) *AppStripeCustomerUpdate {
	if s != nil {
		ascu.SetStripeCustomerID(*s)
	}
	return ascu
}

// ClearStripeCustomerID clears the value of the "stripe_customer_id" field.
func (ascu *AppStripeCustomerUpdate) ClearStripeCustomerID() *AppStripeCustomerUpdate {
	ascu.mutation.ClearStripeCustomerID()
	return ascu
}

// Mutation returns the AppStripeCustomerMutation object of the builder.
func (ascu *AppStripeCustomerUpdate) Mutation() *AppStripeCustomerMutation {
	return ascu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (ascu *AppStripeCustomerUpdate) Save(ctx context.Context) (int, error) {
	ascu.defaults()
	return withHooks(ctx, ascu.sqlSave, ascu.mutation, ascu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (ascu *AppStripeCustomerUpdate) SaveX(ctx context.Context) int {
	affected, err := ascu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (ascu *AppStripeCustomerUpdate) Exec(ctx context.Context) error {
	_, err := ascu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ascu *AppStripeCustomerUpdate) ExecX(ctx context.Context) {
	if err := ascu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ascu *AppStripeCustomerUpdate) defaults() {
	if _, ok := ascu.mutation.UpdatedAt(); !ok {
		v := appstripecustomer.UpdateDefaultUpdatedAt()
		ascu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ascu *AppStripeCustomerUpdate) check() error {
	if ascu.mutation.AppCleared() && len(ascu.mutation.AppIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.app"`)
	}
	if ascu.mutation.AppStripeCleared() && len(ascu.mutation.AppStripeIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.app_stripe"`)
	}
	if ascu.mutation.CustomerCleared() && len(ascu.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.customer"`)
	}
	return nil
}

func (ascu *AppStripeCustomerUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := ascu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(appstripecustomer.Table, appstripecustomer.Columns, sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt))
	if ps := ascu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := ascu.mutation.UpdatedAt(); ok {
		_spec.SetField(appstripecustomer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := ascu.mutation.DeletedAt(); ok {
		_spec.SetField(appstripecustomer.FieldDeletedAt, field.TypeTime, value)
	}
	if ascu.mutation.DeletedAtCleared() {
		_spec.ClearField(appstripecustomer.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := ascu.mutation.StripeCustomerID(); ok {
		_spec.SetField(appstripecustomer.FieldStripeCustomerID, field.TypeString, value)
	}
	if ascu.mutation.StripeCustomerIDCleared() {
		_spec.ClearField(appstripecustomer.FieldStripeCustomerID, field.TypeString)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, ascu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appstripecustomer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	ascu.mutation.done = true
	return n, nil
}

// AppStripeCustomerUpdateOne is the builder for updating a single AppStripeCustomer entity.
type AppStripeCustomerUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *AppStripeCustomerMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (ascuo *AppStripeCustomerUpdateOne) SetUpdatedAt(t time.Time) *AppStripeCustomerUpdateOne {
	ascuo.mutation.SetUpdatedAt(t)
	return ascuo
}

// SetDeletedAt sets the "deleted_at" field.
func (ascuo *AppStripeCustomerUpdateOne) SetDeletedAt(t time.Time) *AppStripeCustomerUpdateOne {
	ascuo.mutation.SetDeletedAt(t)
	return ascuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (ascuo *AppStripeCustomerUpdateOne) SetNillableDeletedAt(t *time.Time) *AppStripeCustomerUpdateOne {
	if t != nil {
		ascuo.SetDeletedAt(*t)
	}
	return ascuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (ascuo *AppStripeCustomerUpdateOne) ClearDeletedAt() *AppStripeCustomerUpdateOne {
	ascuo.mutation.ClearDeletedAt()
	return ascuo
}

// SetStripeCustomerID sets the "stripe_customer_id" field.
func (ascuo *AppStripeCustomerUpdateOne) SetStripeCustomerID(s string) *AppStripeCustomerUpdateOne {
	ascuo.mutation.SetStripeCustomerID(s)
	return ascuo
}

// SetNillableStripeCustomerID sets the "stripe_customer_id" field if the given value is not nil.
func (ascuo *AppStripeCustomerUpdateOne) SetNillableStripeCustomerID(s *string) *AppStripeCustomerUpdateOne {
	if s != nil {
		ascuo.SetStripeCustomerID(*s)
	}
	return ascuo
}

// ClearStripeCustomerID clears the value of the "stripe_customer_id" field.
func (ascuo *AppStripeCustomerUpdateOne) ClearStripeCustomerID() *AppStripeCustomerUpdateOne {
	ascuo.mutation.ClearStripeCustomerID()
	return ascuo
}

// Mutation returns the AppStripeCustomerMutation object of the builder.
func (ascuo *AppStripeCustomerUpdateOne) Mutation() *AppStripeCustomerMutation {
	return ascuo.mutation
}

// Where appends a list predicates to the AppStripeCustomerUpdate builder.
func (ascuo *AppStripeCustomerUpdateOne) Where(ps ...predicate.AppStripeCustomer) *AppStripeCustomerUpdateOne {
	ascuo.mutation.Where(ps...)
	return ascuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (ascuo *AppStripeCustomerUpdateOne) Select(field string, fields ...string) *AppStripeCustomerUpdateOne {
	ascuo.fields = append([]string{field}, fields...)
	return ascuo
}

// Save executes the query and returns the updated AppStripeCustomer entity.
func (ascuo *AppStripeCustomerUpdateOne) Save(ctx context.Context) (*AppStripeCustomer, error) {
	ascuo.defaults()
	return withHooks(ctx, ascuo.sqlSave, ascuo.mutation, ascuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (ascuo *AppStripeCustomerUpdateOne) SaveX(ctx context.Context) *AppStripeCustomer {
	node, err := ascuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (ascuo *AppStripeCustomerUpdateOne) Exec(ctx context.Context) error {
	_, err := ascuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ascuo *AppStripeCustomerUpdateOne) ExecX(ctx context.Context) {
	if err := ascuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ascuo *AppStripeCustomerUpdateOne) defaults() {
	if _, ok := ascuo.mutation.UpdatedAt(); !ok {
		v := appstripecustomer.UpdateDefaultUpdatedAt()
		ascuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ascuo *AppStripeCustomerUpdateOne) check() error {
	if ascuo.mutation.AppCleared() && len(ascuo.mutation.AppIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.app"`)
	}
	if ascuo.mutation.AppStripeCleared() && len(ascuo.mutation.AppStripeIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.app_stripe"`)
	}
	if ascuo.mutation.CustomerCleared() && len(ascuo.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AppStripeCustomer.customer"`)
	}
	return nil
}

func (ascuo *AppStripeCustomerUpdateOne) sqlSave(ctx context.Context) (_node *AppStripeCustomer, err error) {
	if err := ascuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(appstripecustomer.Table, appstripecustomer.Columns, sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt))
	id, ok := ascuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "AppStripeCustomer.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := ascuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, appstripecustomer.FieldID)
		for _, f := range fields {
			if !appstripecustomer.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != appstripecustomer.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := ascuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := ascuo.mutation.UpdatedAt(); ok {
		_spec.SetField(appstripecustomer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := ascuo.mutation.DeletedAt(); ok {
		_spec.SetField(appstripecustomer.FieldDeletedAt, field.TypeTime, value)
	}
	if ascuo.mutation.DeletedAtCleared() {
		_spec.ClearField(appstripecustomer.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := ascuo.mutation.StripeCustomerID(); ok {
		_spec.SetField(appstripecustomer.FieldStripeCustomerID, field.TypeString, value)
	}
	if ascuo.mutation.StripeCustomerIDCleared() {
		_spec.ClearField(appstripecustomer.FieldStripeCustomerID, field.TypeString)
	}
	_node = &AppStripeCustomer{config: ascuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, ascuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appstripecustomer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	ascuo.mutation.done = true
	return _node, nil
}
