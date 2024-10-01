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
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppStripeUpdate is the builder for updating AppStripe entities.
type AppStripeUpdate struct {
	config
	hooks    []Hook
	mutation *AppStripeMutation
}

// Where appends a list predicates to the AppStripeUpdate builder.
func (asu *AppStripeUpdate) Where(ps ...predicate.AppStripe) *AppStripeUpdate {
	asu.mutation.Where(ps...)
	return asu
}

// SetUpdatedAt sets the "updated_at" field.
func (asu *AppStripeUpdate) SetUpdatedAt(t time.Time) *AppStripeUpdate {
	asu.mutation.SetUpdatedAt(t)
	return asu
}

// SetDeletedAt sets the "deleted_at" field.
func (asu *AppStripeUpdate) SetDeletedAt(t time.Time) *AppStripeUpdate {
	asu.mutation.SetDeletedAt(t)
	return asu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (asu *AppStripeUpdate) SetNillableDeletedAt(t *time.Time) *AppStripeUpdate {
	if t != nil {
		asu.SetDeletedAt(*t)
	}
	return asu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (asu *AppStripeUpdate) ClearDeletedAt() *AppStripeUpdate {
	asu.mutation.ClearDeletedAt()
	return asu
}

// AddCustomerAppIDs adds the "customer_apps" edge to the AppStripeCustomer entity by IDs.
func (asu *AppStripeUpdate) AddCustomerAppIDs(ids ...int) *AppStripeUpdate {
	asu.mutation.AddCustomerAppIDs(ids...)
	return asu
}

// AddCustomerApps adds the "customer_apps" edges to the AppStripeCustomer entity.
func (asu *AppStripeUpdate) AddCustomerApps(a ...*AppStripeCustomer) *AppStripeUpdate {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return asu.AddCustomerAppIDs(ids...)
}

// Mutation returns the AppStripeMutation object of the builder.
func (asu *AppStripeUpdate) Mutation() *AppStripeMutation {
	return asu.mutation
}

// ClearCustomerApps clears all "customer_apps" edges to the AppStripeCustomer entity.
func (asu *AppStripeUpdate) ClearCustomerApps() *AppStripeUpdate {
	asu.mutation.ClearCustomerApps()
	return asu
}

// RemoveCustomerAppIDs removes the "customer_apps" edge to AppStripeCustomer entities by IDs.
func (asu *AppStripeUpdate) RemoveCustomerAppIDs(ids ...int) *AppStripeUpdate {
	asu.mutation.RemoveCustomerAppIDs(ids...)
	return asu
}

// RemoveCustomerApps removes "customer_apps" edges to AppStripeCustomer entities.
func (asu *AppStripeUpdate) RemoveCustomerApps(a ...*AppStripeCustomer) *AppStripeUpdate {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return asu.RemoveCustomerAppIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (asu *AppStripeUpdate) Save(ctx context.Context) (int, error) {
	asu.defaults()
	return withHooks(ctx, asu.sqlSave, asu.mutation, asu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (asu *AppStripeUpdate) SaveX(ctx context.Context) int {
	affected, err := asu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (asu *AppStripeUpdate) Exec(ctx context.Context) error {
	_, err := asu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (asu *AppStripeUpdate) ExecX(ctx context.Context) {
	if err := asu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (asu *AppStripeUpdate) defaults() {
	if _, ok := asu.mutation.UpdatedAt(); !ok {
		v := appstripe.UpdateDefaultUpdatedAt()
		asu.mutation.SetUpdatedAt(v)
	}
}

func (asu *AppStripeUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(appstripe.Table, appstripe.Columns, sqlgraph.NewFieldSpec(appstripe.FieldID, field.TypeString))
	if ps := asu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := asu.mutation.UpdatedAt(); ok {
		_spec.SetField(appstripe.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := asu.mutation.DeletedAt(); ok {
		_spec.SetField(appstripe.FieldDeletedAt, field.TypeTime, value)
	}
	if asu.mutation.DeletedAtCleared() {
		_spec.ClearField(appstripe.FieldDeletedAt, field.TypeTime)
	}
	if asu.mutation.CustomerAppsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := asu.mutation.RemovedCustomerAppsIDs(); len(nodes) > 0 && !asu.mutation.CustomerAppsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := asu.mutation.CustomerAppsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, asu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appstripe.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	asu.mutation.done = true
	return n, nil
}

// AppStripeUpdateOne is the builder for updating a single AppStripe entity.
type AppStripeUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *AppStripeMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (asuo *AppStripeUpdateOne) SetUpdatedAt(t time.Time) *AppStripeUpdateOne {
	asuo.mutation.SetUpdatedAt(t)
	return asuo
}

// SetDeletedAt sets the "deleted_at" field.
func (asuo *AppStripeUpdateOne) SetDeletedAt(t time.Time) *AppStripeUpdateOne {
	asuo.mutation.SetDeletedAt(t)
	return asuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (asuo *AppStripeUpdateOne) SetNillableDeletedAt(t *time.Time) *AppStripeUpdateOne {
	if t != nil {
		asuo.SetDeletedAt(*t)
	}
	return asuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (asuo *AppStripeUpdateOne) ClearDeletedAt() *AppStripeUpdateOne {
	asuo.mutation.ClearDeletedAt()
	return asuo
}

// AddCustomerAppIDs adds the "customer_apps" edge to the AppStripeCustomer entity by IDs.
func (asuo *AppStripeUpdateOne) AddCustomerAppIDs(ids ...int) *AppStripeUpdateOne {
	asuo.mutation.AddCustomerAppIDs(ids...)
	return asuo
}

// AddCustomerApps adds the "customer_apps" edges to the AppStripeCustomer entity.
func (asuo *AppStripeUpdateOne) AddCustomerApps(a ...*AppStripeCustomer) *AppStripeUpdateOne {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return asuo.AddCustomerAppIDs(ids...)
}

// Mutation returns the AppStripeMutation object of the builder.
func (asuo *AppStripeUpdateOne) Mutation() *AppStripeMutation {
	return asuo.mutation
}

// ClearCustomerApps clears all "customer_apps" edges to the AppStripeCustomer entity.
func (asuo *AppStripeUpdateOne) ClearCustomerApps() *AppStripeUpdateOne {
	asuo.mutation.ClearCustomerApps()
	return asuo
}

// RemoveCustomerAppIDs removes the "customer_apps" edge to AppStripeCustomer entities by IDs.
func (asuo *AppStripeUpdateOne) RemoveCustomerAppIDs(ids ...int) *AppStripeUpdateOne {
	asuo.mutation.RemoveCustomerAppIDs(ids...)
	return asuo
}

// RemoveCustomerApps removes "customer_apps" edges to AppStripeCustomer entities.
func (asuo *AppStripeUpdateOne) RemoveCustomerApps(a ...*AppStripeCustomer) *AppStripeUpdateOne {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return asuo.RemoveCustomerAppIDs(ids...)
}

// Where appends a list predicates to the AppStripeUpdate builder.
func (asuo *AppStripeUpdateOne) Where(ps ...predicate.AppStripe) *AppStripeUpdateOne {
	asuo.mutation.Where(ps...)
	return asuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (asuo *AppStripeUpdateOne) Select(field string, fields ...string) *AppStripeUpdateOne {
	asuo.fields = append([]string{field}, fields...)
	return asuo
}

// Save executes the query and returns the updated AppStripe entity.
func (asuo *AppStripeUpdateOne) Save(ctx context.Context) (*AppStripe, error) {
	asuo.defaults()
	return withHooks(ctx, asuo.sqlSave, asuo.mutation, asuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (asuo *AppStripeUpdateOne) SaveX(ctx context.Context) *AppStripe {
	node, err := asuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (asuo *AppStripeUpdateOne) Exec(ctx context.Context) error {
	_, err := asuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (asuo *AppStripeUpdateOne) ExecX(ctx context.Context) {
	if err := asuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (asuo *AppStripeUpdateOne) defaults() {
	if _, ok := asuo.mutation.UpdatedAt(); !ok {
		v := appstripe.UpdateDefaultUpdatedAt()
		asuo.mutation.SetUpdatedAt(v)
	}
}

func (asuo *AppStripeUpdateOne) sqlSave(ctx context.Context) (_node *AppStripe, err error) {
	_spec := sqlgraph.NewUpdateSpec(appstripe.Table, appstripe.Columns, sqlgraph.NewFieldSpec(appstripe.FieldID, field.TypeString))
	id, ok := asuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "AppStripe.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := asuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, appstripe.FieldID)
		for _, f := range fields {
			if !appstripe.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != appstripe.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := asuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := asuo.mutation.UpdatedAt(); ok {
		_spec.SetField(appstripe.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := asuo.mutation.DeletedAt(); ok {
		_spec.SetField(appstripe.FieldDeletedAt, field.TypeTime, value)
	}
	if asuo.mutation.DeletedAtCleared() {
		_spec.ClearField(appstripe.FieldDeletedAt, field.TypeTime)
	}
	if asuo.mutation.CustomerAppsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := asuo.mutation.RemovedCustomerAppsIDs(); len(nodes) > 0 && !asuo.mutation.CustomerAppsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := asuo.mutation.CustomerAppsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   appstripe.CustomerAppsTable,
			Columns: []string{appstripe.CustomerAppsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &AppStripe{config: asuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, asuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{appstripe.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	asuo.mutation.done = true
	return _node, nil
}