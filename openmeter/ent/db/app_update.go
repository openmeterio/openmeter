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
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppUpdate is the builder for updating App entities.
type AppUpdate struct {
	config
	hooks    []Hook
	mutation *AppMutation
}

// Where appends a list predicates to the AppUpdate builder.
func (au *AppUpdate) Where(ps ...predicate.App) *AppUpdate {
	au.mutation.Where(ps...)
	return au
}

// SetMetadata sets the "metadata" field.
func (au *AppUpdate) SetMetadata(m map[string]string) *AppUpdate {
	au.mutation.SetMetadata(m)
	return au
}

// ClearMetadata clears the value of the "metadata" field.
func (au *AppUpdate) ClearMetadata() *AppUpdate {
	au.mutation.ClearMetadata()
	return au
}

// SetUpdatedAt sets the "updated_at" field.
func (au *AppUpdate) SetUpdatedAt(t time.Time) *AppUpdate {
	au.mutation.SetUpdatedAt(t)
	return au
}

// SetDeletedAt sets the "deleted_at" field.
func (au *AppUpdate) SetDeletedAt(t time.Time) *AppUpdate {
	au.mutation.SetDeletedAt(t)
	return au
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (au *AppUpdate) SetNillableDeletedAt(t *time.Time) *AppUpdate {
	if t != nil {
		au.SetDeletedAt(*t)
	}
	return au
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (au *AppUpdate) ClearDeletedAt() *AppUpdate {
	au.mutation.ClearDeletedAt()
	return au
}

// SetName sets the "name" field.
func (au *AppUpdate) SetName(s string) *AppUpdate {
	au.mutation.SetName(s)
	return au
}

// SetNillableName sets the "name" field if the given value is not nil.
func (au *AppUpdate) SetNillableName(s *string) *AppUpdate {
	if s != nil {
		au.SetName(*s)
	}
	return au
}

// SetDescription sets the "description" field.
func (au *AppUpdate) SetDescription(s string) *AppUpdate {
	au.mutation.SetDescription(s)
	return au
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (au *AppUpdate) SetNillableDescription(s *string) *AppUpdate {
	if s != nil {
		au.SetDescription(*s)
	}
	return au
}

// SetStatus sets the "status" field.
func (au *AppUpdate) SetStatus(as appentity.AppStatus) *AppUpdate {
	au.mutation.SetStatus(as)
	return au
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (au *AppUpdate) SetNillableStatus(as *appentity.AppStatus) *AppUpdate {
	if as != nil {
		au.SetStatus(*as)
	}
	return au
}

// SetIsDefault sets the "is_default" field.
func (au *AppUpdate) SetIsDefault(b bool) *AppUpdate {
	au.mutation.SetIsDefault(b)
	return au
}

// SetNillableIsDefault sets the "is_default" field if the given value is not nil.
func (au *AppUpdate) SetNillableIsDefault(b *bool) *AppUpdate {
	if b != nil {
		au.SetIsDefault(*b)
	}
	return au
}

// AddAppCustomerIDs adds the "app_customers" edge to the AppStripeCustomer entity by IDs.
func (au *AppUpdate) AddAppCustomerIDs(ids ...int) *AppUpdate {
	au.mutation.AddAppCustomerIDs(ids...)
	return au
}

// AddAppCustomers adds the "app_customers" edges to the AppStripeCustomer entity.
func (au *AppUpdate) AddAppCustomers(a ...*AppStripeCustomer) *AppUpdate {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return au.AddAppCustomerIDs(ids...)
}

// Mutation returns the AppMutation object of the builder.
func (au *AppUpdate) Mutation() *AppMutation {
	return au.mutation
}

// ClearAppCustomers clears all "app_customers" edges to the AppStripeCustomer entity.
func (au *AppUpdate) ClearAppCustomers() *AppUpdate {
	au.mutation.ClearAppCustomers()
	return au
}

// RemoveAppCustomerIDs removes the "app_customers" edge to AppStripeCustomer entities by IDs.
func (au *AppUpdate) RemoveAppCustomerIDs(ids ...int) *AppUpdate {
	au.mutation.RemoveAppCustomerIDs(ids...)
	return au
}

// RemoveAppCustomers removes "app_customers" edges to AppStripeCustomer entities.
func (au *AppUpdate) RemoveAppCustomers(a ...*AppStripeCustomer) *AppUpdate {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return au.RemoveAppCustomerIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (au *AppUpdate) Save(ctx context.Context) (int, error) {
	au.defaults()
	return withHooks(ctx, au.sqlSave, au.mutation, au.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (au *AppUpdate) SaveX(ctx context.Context) int {
	affected, err := au.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (au *AppUpdate) Exec(ctx context.Context) error {
	_, err := au.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (au *AppUpdate) ExecX(ctx context.Context) {
	if err := au.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (au *AppUpdate) defaults() {
	if _, ok := au.mutation.UpdatedAt(); !ok {
		v := app.UpdateDefaultUpdatedAt()
		au.mutation.SetUpdatedAt(v)
	}
}

func (au *AppUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(app.Table, app.Columns, sqlgraph.NewFieldSpec(app.FieldID, field.TypeString))
	if ps := au.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := au.mutation.Metadata(); ok {
		_spec.SetField(app.FieldMetadata, field.TypeJSON, value)
	}
	if au.mutation.MetadataCleared() {
		_spec.ClearField(app.FieldMetadata, field.TypeJSON)
	}
	if value, ok := au.mutation.UpdatedAt(); ok {
		_spec.SetField(app.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := au.mutation.DeletedAt(); ok {
		_spec.SetField(app.FieldDeletedAt, field.TypeTime, value)
	}
	if au.mutation.DeletedAtCleared() {
		_spec.ClearField(app.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := au.mutation.Name(); ok {
		_spec.SetField(app.FieldName, field.TypeString, value)
	}
	if value, ok := au.mutation.Description(); ok {
		_spec.SetField(app.FieldDescription, field.TypeString, value)
	}
	if value, ok := au.mutation.Status(); ok {
		_spec.SetField(app.FieldStatus, field.TypeString, value)
	}
	if value, ok := au.mutation.IsDefault(); ok {
		_spec.SetField(app.FieldIsDefault, field.TypeBool, value)
	}
	if au.mutation.AppCustomersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := au.mutation.RemovedAppCustomersIDs(); len(nodes) > 0 && !au.mutation.AppCustomersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
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
	if nodes := au.mutation.AppCustomersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, au.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{app.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	au.mutation.done = true
	return n, nil
}

// AppUpdateOne is the builder for updating a single App entity.
type AppUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *AppMutation
}

// SetMetadata sets the "metadata" field.
func (auo *AppUpdateOne) SetMetadata(m map[string]string) *AppUpdateOne {
	auo.mutation.SetMetadata(m)
	return auo
}

// ClearMetadata clears the value of the "metadata" field.
func (auo *AppUpdateOne) ClearMetadata() *AppUpdateOne {
	auo.mutation.ClearMetadata()
	return auo
}

// SetUpdatedAt sets the "updated_at" field.
func (auo *AppUpdateOne) SetUpdatedAt(t time.Time) *AppUpdateOne {
	auo.mutation.SetUpdatedAt(t)
	return auo
}

// SetDeletedAt sets the "deleted_at" field.
func (auo *AppUpdateOne) SetDeletedAt(t time.Time) *AppUpdateOne {
	auo.mutation.SetDeletedAt(t)
	return auo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (auo *AppUpdateOne) SetNillableDeletedAt(t *time.Time) *AppUpdateOne {
	if t != nil {
		auo.SetDeletedAt(*t)
	}
	return auo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (auo *AppUpdateOne) ClearDeletedAt() *AppUpdateOne {
	auo.mutation.ClearDeletedAt()
	return auo
}

// SetName sets the "name" field.
func (auo *AppUpdateOne) SetName(s string) *AppUpdateOne {
	auo.mutation.SetName(s)
	return auo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (auo *AppUpdateOne) SetNillableName(s *string) *AppUpdateOne {
	if s != nil {
		auo.SetName(*s)
	}
	return auo
}

// SetDescription sets the "description" field.
func (auo *AppUpdateOne) SetDescription(s string) *AppUpdateOne {
	auo.mutation.SetDescription(s)
	return auo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (auo *AppUpdateOne) SetNillableDescription(s *string) *AppUpdateOne {
	if s != nil {
		auo.SetDescription(*s)
	}
	return auo
}

// SetStatus sets the "status" field.
func (auo *AppUpdateOne) SetStatus(as appentity.AppStatus) *AppUpdateOne {
	auo.mutation.SetStatus(as)
	return auo
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (auo *AppUpdateOne) SetNillableStatus(as *appentity.AppStatus) *AppUpdateOne {
	if as != nil {
		auo.SetStatus(*as)
	}
	return auo
}

// SetIsDefault sets the "is_default" field.
func (auo *AppUpdateOne) SetIsDefault(b bool) *AppUpdateOne {
	auo.mutation.SetIsDefault(b)
	return auo
}

// SetNillableIsDefault sets the "is_default" field if the given value is not nil.
func (auo *AppUpdateOne) SetNillableIsDefault(b *bool) *AppUpdateOne {
	if b != nil {
		auo.SetIsDefault(*b)
	}
	return auo
}

// AddAppCustomerIDs adds the "app_customers" edge to the AppStripeCustomer entity by IDs.
func (auo *AppUpdateOne) AddAppCustomerIDs(ids ...int) *AppUpdateOne {
	auo.mutation.AddAppCustomerIDs(ids...)
	return auo
}

// AddAppCustomers adds the "app_customers" edges to the AppStripeCustomer entity.
func (auo *AppUpdateOne) AddAppCustomers(a ...*AppStripeCustomer) *AppUpdateOne {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return auo.AddAppCustomerIDs(ids...)
}

// Mutation returns the AppMutation object of the builder.
func (auo *AppUpdateOne) Mutation() *AppMutation {
	return auo.mutation
}

// ClearAppCustomers clears all "app_customers" edges to the AppStripeCustomer entity.
func (auo *AppUpdateOne) ClearAppCustomers() *AppUpdateOne {
	auo.mutation.ClearAppCustomers()
	return auo
}

// RemoveAppCustomerIDs removes the "app_customers" edge to AppStripeCustomer entities by IDs.
func (auo *AppUpdateOne) RemoveAppCustomerIDs(ids ...int) *AppUpdateOne {
	auo.mutation.RemoveAppCustomerIDs(ids...)
	return auo
}

// RemoveAppCustomers removes "app_customers" edges to AppStripeCustomer entities.
func (auo *AppUpdateOne) RemoveAppCustomers(a ...*AppStripeCustomer) *AppUpdateOne {
	ids := make([]int, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return auo.RemoveAppCustomerIDs(ids...)
}

// Where appends a list predicates to the AppUpdate builder.
func (auo *AppUpdateOne) Where(ps ...predicate.App) *AppUpdateOne {
	auo.mutation.Where(ps...)
	return auo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (auo *AppUpdateOne) Select(field string, fields ...string) *AppUpdateOne {
	auo.fields = append([]string{field}, fields...)
	return auo
}

// Save executes the query and returns the updated App entity.
func (auo *AppUpdateOne) Save(ctx context.Context) (*App, error) {
	auo.defaults()
	return withHooks(ctx, auo.sqlSave, auo.mutation, auo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (auo *AppUpdateOne) SaveX(ctx context.Context) *App {
	node, err := auo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (auo *AppUpdateOne) Exec(ctx context.Context) error {
	_, err := auo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (auo *AppUpdateOne) ExecX(ctx context.Context) {
	if err := auo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (auo *AppUpdateOne) defaults() {
	if _, ok := auo.mutation.UpdatedAt(); !ok {
		v := app.UpdateDefaultUpdatedAt()
		auo.mutation.SetUpdatedAt(v)
	}
}

func (auo *AppUpdateOne) sqlSave(ctx context.Context) (_node *App, err error) {
	_spec := sqlgraph.NewUpdateSpec(app.Table, app.Columns, sqlgraph.NewFieldSpec(app.FieldID, field.TypeString))
	id, ok := auo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "App.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := auo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, app.FieldID)
		for _, f := range fields {
			if !app.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != app.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := auo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := auo.mutation.Metadata(); ok {
		_spec.SetField(app.FieldMetadata, field.TypeJSON, value)
	}
	if auo.mutation.MetadataCleared() {
		_spec.ClearField(app.FieldMetadata, field.TypeJSON)
	}
	if value, ok := auo.mutation.UpdatedAt(); ok {
		_spec.SetField(app.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := auo.mutation.DeletedAt(); ok {
		_spec.SetField(app.FieldDeletedAt, field.TypeTime, value)
	}
	if auo.mutation.DeletedAtCleared() {
		_spec.ClearField(app.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := auo.mutation.Name(); ok {
		_spec.SetField(app.FieldName, field.TypeString, value)
	}
	if value, ok := auo.mutation.Description(); ok {
		_spec.SetField(app.FieldDescription, field.TypeString, value)
	}
	if value, ok := auo.mutation.Status(); ok {
		_spec.SetField(app.FieldStatus, field.TypeString, value)
	}
	if value, ok := auo.mutation.IsDefault(); ok {
		_spec.SetField(app.FieldIsDefault, field.TypeBool, value)
	}
	if auo.mutation.AppCustomersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(appstripecustomer.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := auo.mutation.RemovedAppCustomersIDs(); len(nodes) > 0 && !auo.mutation.AppCustomersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
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
	if nodes := auo.mutation.AppCustomersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   app.AppCustomersTable,
			Columns: []string{app.AppCustomersColumn},
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
	_node = &App{config: auo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, auo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{app.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	auo.mutation.done = true
	return _node, nil
}
