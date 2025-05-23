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
	dbapp "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
)

// AppCustomerCreate is the builder for creating a AppCustomer entity.
type AppCustomerCreate struct {
	config
	mutation *AppCustomerMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (_c *AppCustomerCreate) SetNamespace(v string) *AppCustomerCreate {
	_c.mutation.SetNamespace(v)
	return _c
}

// SetCreatedAt sets the "created_at" field.
func (_c *AppCustomerCreate) SetCreatedAt(v time.Time) *AppCustomerCreate {
	_c.mutation.SetCreatedAt(v)
	return _c
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (_c *AppCustomerCreate) SetNillableCreatedAt(v *time.Time) *AppCustomerCreate {
	if v != nil {
		_c.SetCreatedAt(*v)
	}
	return _c
}

// SetUpdatedAt sets the "updated_at" field.
func (_c *AppCustomerCreate) SetUpdatedAt(v time.Time) *AppCustomerCreate {
	_c.mutation.SetUpdatedAt(v)
	return _c
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (_c *AppCustomerCreate) SetNillableUpdatedAt(v *time.Time) *AppCustomerCreate {
	if v != nil {
		_c.SetUpdatedAt(*v)
	}
	return _c
}

// SetDeletedAt sets the "deleted_at" field.
func (_c *AppCustomerCreate) SetDeletedAt(v time.Time) *AppCustomerCreate {
	_c.mutation.SetDeletedAt(v)
	return _c
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_c *AppCustomerCreate) SetNillableDeletedAt(v *time.Time) *AppCustomerCreate {
	if v != nil {
		_c.SetDeletedAt(*v)
	}
	return _c
}

// SetAppID sets the "app_id" field.
func (_c *AppCustomerCreate) SetAppID(v string) *AppCustomerCreate {
	_c.mutation.SetAppID(v)
	return _c
}

// SetCustomerID sets the "customer_id" field.
func (_c *AppCustomerCreate) SetCustomerID(v string) *AppCustomerCreate {
	_c.mutation.SetCustomerID(v)
	return _c
}

// SetApp sets the "app" edge to the App entity.
func (_c *AppCustomerCreate) SetApp(v *App) *AppCustomerCreate {
	return _c.SetAppID(v.ID)
}

// SetCustomer sets the "customer" edge to the Customer entity.
func (_c *AppCustomerCreate) SetCustomer(v *Customer) *AppCustomerCreate {
	return _c.SetCustomerID(v.ID)
}

// Mutation returns the AppCustomerMutation object of the builder.
func (_c *AppCustomerCreate) Mutation() *AppCustomerMutation {
	return _c.mutation
}

// Save creates the AppCustomer in the database.
func (_c *AppCustomerCreate) Save(ctx context.Context) (*AppCustomer, error) {
	_c.defaults()
	return withHooks(ctx, _c.sqlSave, _c.mutation, _c.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (_c *AppCustomerCreate) SaveX(ctx context.Context) *AppCustomer {
	v, err := _c.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (_c *AppCustomerCreate) Exec(ctx context.Context) error {
	_, err := _c.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_c *AppCustomerCreate) ExecX(ctx context.Context) {
	if err := _c.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_c *AppCustomerCreate) defaults() {
	if _, ok := _c.mutation.CreatedAt(); !ok {
		v := appcustomer.DefaultCreatedAt()
		_c.mutation.SetCreatedAt(v)
	}
	if _, ok := _c.mutation.UpdatedAt(); !ok {
		v := appcustomer.DefaultUpdatedAt()
		_c.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_c *AppCustomerCreate) check() error {
	if _, ok := _c.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "AppCustomer.namespace"`)}
	}
	if v, ok := _c.mutation.Namespace(); ok {
		if err := appcustomer.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "AppCustomer.namespace": %w`, err)}
		}
	}
	if _, ok := _c.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "AppCustomer.created_at"`)}
	}
	if _, ok := _c.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "AppCustomer.updated_at"`)}
	}
	if _, ok := _c.mutation.AppID(); !ok {
		return &ValidationError{Name: "app_id", err: errors.New(`db: missing required field "AppCustomer.app_id"`)}
	}
	if v, ok := _c.mutation.AppID(); ok {
		if err := appcustomer.AppIDValidator(v); err != nil {
			return &ValidationError{Name: "app_id", err: fmt.Errorf(`db: validator failed for field "AppCustomer.app_id": %w`, err)}
		}
	}
	if _, ok := _c.mutation.CustomerID(); !ok {
		return &ValidationError{Name: "customer_id", err: errors.New(`db: missing required field "AppCustomer.customer_id"`)}
	}
	if v, ok := _c.mutation.CustomerID(); ok {
		if err := appcustomer.CustomerIDValidator(v); err != nil {
			return &ValidationError{Name: "customer_id", err: fmt.Errorf(`db: validator failed for field "AppCustomer.customer_id": %w`, err)}
		}
	}
	if len(_c.mutation.AppIDs()) == 0 {
		return &ValidationError{Name: "app", err: errors.New(`db: missing required edge "AppCustomer.app"`)}
	}
	if len(_c.mutation.CustomerIDs()) == 0 {
		return &ValidationError{Name: "customer", err: errors.New(`db: missing required edge "AppCustomer.customer"`)}
	}
	return nil
}

func (_c *AppCustomerCreate) sqlSave(ctx context.Context) (*AppCustomer, error) {
	if err := _c.check(); err != nil {
		return nil, err
	}
	_node, _spec := _c.createSpec()
	if err := sqlgraph.CreateNode(ctx, _c.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	id := _spec.ID.Value.(int64)
	_node.ID = int(id)
	_c.mutation.id = &_node.ID
	_c.mutation.done = true
	return _node, nil
}

func (_c *AppCustomerCreate) createSpec() (*AppCustomer, *sqlgraph.CreateSpec) {
	var (
		_node = &AppCustomer{config: _c.config}
		_spec = sqlgraph.NewCreateSpec(appcustomer.Table, sqlgraph.NewFieldSpec(appcustomer.FieldID, field.TypeInt))
	)
	_spec.OnConflict = _c.conflict
	if value, ok := _c.mutation.Namespace(); ok {
		_spec.SetField(appcustomer.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := _c.mutation.CreatedAt(); ok {
		_spec.SetField(appcustomer.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := _c.mutation.UpdatedAt(); ok {
		_spec.SetField(appcustomer.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := _c.mutation.DeletedAt(); ok {
		_spec.SetField(appcustomer.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if nodes := _c.mutation.AppIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   appcustomer.AppTable,
			Columns: []string{appcustomer.AppColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(dbapp.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.AppID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := _c.mutation.CustomerIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   appcustomer.CustomerTable,
			Columns: []string{appcustomer.CustomerColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(customer.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.CustomerID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.AppCustomer.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.AppCustomerUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (_c *AppCustomerCreate) OnConflict(opts ...sql.ConflictOption) *AppCustomerUpsertOne {
	_c.conflict = opts
	return &AppCustomerUpsertOne{
		create: _c,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (_c *AppCustomerCreate) OnConflictColumns(columns ...string) *AppCustomerUpsertOne {
	_c.conflict = append(_c.conflict, sql.ConflictColumns(columns...))
	return &AppCustomerUpsertOne{
		create: _c,
	}
}

type (
	// AppCustomerUpsertOne is the builder for "upsert"-ing
	//  one AppCustomer node.
	AppCustomerUpsertOne struct {
		create *AppCustomerCreate
	}

	// AppCustomerUpsert is the "OnConflict" setter.
	AppCustomerUpsert struct {
		*sql.UpdateSet
	}
)

// SetUpdatedAt sets the "updated_at" field.
func (u *AppCustomerUpsert) SetUpdatedAt(v time.Time) *AppCustomerUpsert {
	u.Set(appcustomer.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *AppCustomerUpsert) UpdateUpdatedAt() *AppCustomerUpsert {
	u.SetExcluded(appcustomer.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *AppCustomerUpsert) SetDeletedAt(v time.Time) *AppCustomerUpsert {
	u.Set(appcustomer.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *AppCustomerUpsert) UpdateDeletedAt() *AppCustomerUpsert {
	u.SetExcluded(appcustomer.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *AppCustomerUpsert) ClearDeletedAt() *AppCustomerUpsert {
	u.SetNull(appcustomer.FieldDeletedAt)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create.
// Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//		).
//		Exec(ctx)
func (u *AppCustomerUpsertOne) UpdateNewValues() *AppCustomerUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(appcustomer.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(appcustomer.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.AppID(); exists {
			s.SetIgnore(appcustomer.FieldAppID)
		}
		if _, exists := u.create.mutation.CustomerID(); exists {
			s.SetIgnore(appcustomer.FieldCustomerID)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *AppCustomerUpsertOne) Ignore() *AppCustomerUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *AppCustomerUpsertOne) DoNothing() *AppCustomerUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the AppCustomerCreate.OnConflict
// documentation for more info.
func (u *AppCustomerUpsertOne) Update(set func(*AppCustomerUpsert)) *AppCustomerUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&AppCustomerUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *AppCustomerUpsertOne) SetUpdatedAt(v time.Time) *AppCustomerUpsertOne {
	return u.Update(func(s *AppCustomerUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *AppCustomerUpsertOne) UpdateUpdatedAt() *AppCustomerUpsertOne {
	return u.Update(func(s *AppCustomerUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *AppCustomerUpsertOne) SetDeletedAt(v time.Time) *AppCustomerUpsertOne {
	return u.Update(func(s *AppCustomerUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *AppCustomerUpsertOne) UpdateDeletedAt() *AppCustomerUpsertOne {
	return u.Update(func(s *AppCustomerUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *AppCustomerUpsertOne) ClearDeletedAt() *AppCustomerUpsertOne {
	return u.Update(func(s *AppCustomerUpsert) {
		s.ClearDeletedAt()
	})
}

// Exec executes the query.
func (u *AppCustomerUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for AppCustomerCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *AppCustomerUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *AppCustomerUpsertOne) ID(ctx context.Context) (id int, err error) {
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *AppCustomerUpsertOne) IDX(ctx context.Context) int {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// AppCustomerCreateBulk is the builder for creating many AppCustomer entities in bulk.
type AppCustomerCreateBulk struct {
	config
	err      error
	builders []*AppCustomerCreate
	conflict []sql.ConflictOption
}

// Save creates the AppCustomer entities in the database.
func (_c *AppCustomerCreateBulk) Save(ctx context.Context) ([]*AppCustomer, error) {
	if _c.err != nil {
		return nil, _c.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(_c.builders))
	nodes := make([]*AppCustomer, len(_c.builders))
	mutators := make([]Mutator, len(_c.builders))
	for i := range _c.builders {
		func(i int, root context.Context) {
			builder := _c.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*AppCustomerMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, _c.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = _c.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, _c.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				if specs[i].ID.Value != nil {
					id := specs[i].ID.Value.(int64)
					nodes[i].ID = int(id)
				}
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, _c.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (_c *AppCustomerCreateBulk) SaveX(ctx context.Context) []*AppCustomer {
	v, err := _c.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (_c *AppCustomerCreateBulk) Exec(ctx context.Context) error {
	_, err := _c.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_c *AppCustomerCreateBulk) ExecX(ctx context.Context) {
	if err := _c.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.AppCustomer.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.AppCustomerUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (_c *AppCustomerCreateBulk) OnConflict(opts ...sql.ConflictOption) *AppCustomerUpsertBulk {
	_c.conflict = opts
	return &AppCustomerUpsertBulk{
		create: _c,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (_c *AppCustomerCreateBulk) OnConflictColumns(columns ...string) *AppCustomerUpsertBulk {
	_c.conflict = append(_c.conflict, sql.ConflictColumns(columns...))
	return &AppCustomerUpsertBulk{
		create: _c,
	}
}

// AppCustomerUpsertBulk is the builder for "upsert"-ing
// a bulk of AppCustomer nodes.
type AppCustomerUpsertBulk struct {
	create *AppCustomerCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//		).
//		Exec(ctx)
func (u *AppCustomerUpsertBulk) UpdateNewValues() *AppCustomerUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(appcustomer.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(appcustomer.FieldCreatedAt)
			}
			if _, exists := b.mutation.AppID(); exists {
				s.SetIgnore(appcustomer.FieldAppID)
			}
			if _, exists := b.mutation.CustomerID(); exists {
				s.SetIgnore(appcustomer.FieldCustomerID)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.AppCustomer.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *AppCustomerUpsertBulk) Ignore() *AppCustomerUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *AppCustomerUpsertBulk) DoNothing() *AppCustomerUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the AppCustomerCreateBulk.OnConflict
// documentation for more info.
func (u *AppCustomerUpsertBulk) Update(set func(*AppCustomerUpsert)) *AppCustomerUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&AppCustomerUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *AppCustomerUpsertBulk) SetUpdatedAt(v time.Time) *AppCustomerUpsertBulk {
	return u.Update(func(s *AppCustomerUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *AppCustomerUpsertBulk) UpdateUpdatedAt() *AppCustomerUpsertBulk {
	return u.Update(func(s *AppCustomerUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *AppCustomerUpsertBulk) SetDeletedAt(v time.Time) *AppCustomerUpsertBulk {
	return u.Update(func(s *AppCustomerUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *AppCustomerUpsertBulk) UpdateDeletedAt() *AppCustomerUpsertBulk {
	return u.Update(func(s *AppCustomerUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *AppCustomerUpsertBulk) ClearDeletedAt() *AppCustomerUpsertBulk {
	return u.Update(func(s *AppCustomerUpsert) {
		s.ClearDeletedAt()
	})
}

// Exec executes the query.
func (u *AppCustomerUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the AppCustomerCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for AppCustomerCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *AppCustomerUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
