// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db/example1"
)

// Example1Create is the builder for creating a Example1 entity.
type Example1Create struct {
	config
	mutation *Example1Mutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetCreatedAt sets the "created_at" field.
func (e *Example1Create) SetCreatedAt(t time.Time) *Example1Create {
	e.mutation.SetCreatedAt(t)
	return e
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (e *Example1Create) SetNillableCreatedAt(t *time.Time) *Example1Create {
	if t != nil {
		e.SetCreatedAt(*t)
	}
	return e
}

// SetUpdatedAt sets the "updated_at" field.
func (e *Example1Create) SetUpdatedAt(t time.Time) *Example1Create {
	e.mutation.SetUpdatedAt(t)
	return e
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (e *Example1Create) SetNillableUpdatedAt(t *time.Time) *Example1Create {
	if t != nil {
		e.SetUpdatedAt(*t)
	}
	return e
}

// SetDeletedAt sets the "deleted_at" field.
func (e *Example1Create) SetDeletedAt(t time.Time) *Example1Create {
	e.mutation.SetDeletedAt(t)
	return e
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (e *Example1Create) SetNillableDeletedAt(t *time.Time) *Example1Create {
	if t != nil {
		e.SetDeletedAt(*t)
	}
	return e
}

// SetExampleValue1 sets the "example_value_1" field.
func (e *Example1Create) SetExampleValue1(s string) *Example1Create {
	e.mutation.SetExampleValue1(s)
	return e
}

// SetID sets the "id" field.
func (e *Example1Create) SetID(s string) *Example1Create {
	e.mutation.SetID(s)
	return e
}

// Mutation returns the Example1Mutation object of the builder.
func (e *Example1Create) Mutation() *Example1Mutation {
	return e.mutation
}

// Save creates the Example1 in the database.
func (e *Example1Create) Save(ctx context.Context) (*Example1, error) {
	e.defaults()
	return withHooks(ctx, e.sqlSave, e.mutation, e.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (e *Example1Create) SaveX(ctx context.Context) *Example1 {
	v, err := e.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (e *Example1Create) Exec(ctx context.Context) error {
	_, err := e.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (e *Example1Create) ExecX(ctx context.Context) {
	if err := e.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (e *Example1Create) defaults() {
	if _, ok := e.mutation.CreatedAt(); !ok {
		v := example1.DefaultCreatedAt()
		e.mutation.SetCreatedAt(v)
	}
	if _, ok := e.mutation.UpdatedAt(); !ok {
		v := example1.DefaultUpdatedAt()
		e.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (e *Example1Create) check() error {
	if _, ok := e.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "Example1.created_at"`)}
	}
	if _, ok := e.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "Example1.updated_at"`)}
	}
	if _, ok := e.mutation.ExampleValue1(); !ok {
		return &ValidationError{Name: "example_value_1", err: errors.New(`db: missing required field "Example1.example_value_1"`)}
	}
	return nil
}

func (e *Example1Create) sqlSave(ctx context.Context) (*Example1, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	_node, _spec := e.createSpec()
	if err := sqlgraph.CreateNode(ctx, e.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected Example1.ID type: %T", _spec.ID.Value)
		}
	}
	e.mutation.id = &_node.ID
	e.mutation.done = true
	return _node, nil
}

func (e *Example1Create) createSpec() (*Example1, *sqlgraph.CreateSpec) {
	var (
		_node = &Example1{config: e.config}
		_spec = sqlgraph.NewCreateSpec(example1.Table, sqlgraph.NewFieldSpec(example1.FieldID, field.TypeString))
	)
	_spec.OnConflict = e.conflict
	if id, ok := e.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := e.mutation.CreatedAt(); ok {
		_spec.SetField(example1.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := e.mutation.UpdatedAt(); ok {
		_spec.SetField(example1.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := e.mutation.DeletedAt(); ok {
		_spec.SetField(example1.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := e.mutation.ExampleValue1(); ok {
		_spec.SetField(example1.FieldExampleValue1, field.TypeString, value)
		_node.ExampleValue1 = value
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Example1.Create().
//		SetCreatedAt(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.Example1Upsert) {
//			SetCreatedAt(v+v).
//		}).
//		Exec(ctx)
func (e *Example1Create) OnConflict(opts ...sql.ConflictOption) *Example1UpsertOne {
	e.conflict = opts
	return &Example1UpsertOne{
		create: e,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Example1.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (e *Example1Create) OnConflictColumns(columns ...string) *Example1UpsertOne {
	e.conflict = append(e.conflict, sql.ConflictColumns(columns...))
	return &Example1UpsertOne{
		create: e,
	}
}

type (
	// Example1UpsertOne is the builder for "upsert"-ing
	//  one Example1 node.
	Example1UpsertOne struct {
		create *Example1Create
	}

	// Example1Upsert is the "OnConflict" setter.
	Example1Upsert struct {
		*sql.UpdateSet
	}
)

// SetUpdatedAt sets the "updated_at" field.
func (u *Example1Upsert) SetUpdatedAt(v time.Time) *Example1Upsert {
	u.Set(example1.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *Example1Upsert) UpdateUpdatedAt() *Example1Upsert {
	u.SetExcluded(example1.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *Example1Upsert) SetDeletedAt(v time.Time) *Example1Upsert {
	u.Set(example1.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *Example1Upsert) UpdateDeletedAt() *Example1Upsert {
	u.SetExcluded(example1.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *Example1Upsert) ClearDeletedAt() *Example1Upsert {
	u.SetNull(example1.FieldDeletedAt)
	return u
}

// SetExampleValue1 sets the "example_value_1" field.
func (u *Example1Upsert) SetExampleValue1(v string) *Example1Upsert {
	u.Set(example1.FieldExampleValue1, v)
	return u
}

// UpdateExampleValue1 sets the "example_value_1" field to the value that was provided on create.
func (u *Example1Upsert) UpdateExampleValue1() *Example1Upsert {
	u.SetExcluded(example1.FieldExampleValue1)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Example1.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(example1.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *Example1UpsertOne) UpdateNewValues() *Example1UpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(example1.FieldID)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(example1.FieldCreatedAt)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Example1.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *Example1UpsertOne) Ignore() *Example1UpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *Example1UpsertOne) DoNothing() *Example1UpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the Example1Create.OnConflict
// documentation for more info.
func (u *Example1UpsertOne) Update(set func(*Example1Upsert)) *Example1UpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&Example1Upsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *Example1UpsertOne) SetUpdatedAt(v time.Time) *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *Example1UpsertOne) UpdateUpdatedAt() *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *Example1UpsertOne) SetDeletedAt(v time.Time) *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *Example1UpsertOne) UpdateDeletedAt() *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *Example1UpsertOne) ClearDeletedAt() *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.ClearDeletedAt()
	})
}

// SetExampleValue1 sets the "example_value_1" field.
func (u *Example1UpsertOne) SetExampleValue1(v string) *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.SetExampleValue1(v)
	})
}

// UpdateExampleValue1 sets the "example_value_1" field to the value that was provided on create.
func (u *Example1UpsertOne) UpdateExampleValue1() *Example1UpsertOne {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateExampleValue1()
	})
}

// Exec executes the query.
func (u *Example1UpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for Example1Create.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *Example1UpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *Example1UpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: Example1UpsertOne.ID is not supported by MySQL driver. Use Example1UpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *Example1UpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// Example1CreateBulk is the builder for creating many Example1 entities in bulk.
type Example1CreateBulk struct {
	config
	err      error
	builders []*Example1Create
	conflict []sql.ConflictOption
}

// Save creates the Example1 entities in the database.
func (eb *Example1CreateBulk) Save(ctx context.Context) ([]*Example1, error) {
	if eb.err != nil {
		return nil, eb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(eb.builders))
	nodes := make([]*Example1, len(eb.builders))
	mutators := make([]Mutator, len(eb.builders))
	for i := range eb.builders {
		func(i int, root context.Context) {
			builder := eb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*Example1Mutation)
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
					_, err = mutators[i+1].Mutate(root, eb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = eb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, eb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
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
		if _, err := mutators[0].Mutate(ctx, eb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (eb *Example1CreateBulk) SaveX(ctx context.Context) []*Example1 {
	v, err := eb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (eb *Example1CreateBulk) Exec(ctx context.Context) error {
	_, err := eb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (eb *Example1CreateBulk) ExecX(ctx context.Context) {
	if err := eb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Example1.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.Example1Upsert) {
//			SetCreatedAt(v+v).
//		}).
//		Exec(ctx)
func (eb *Example1CreateBulk) OnConflict(opts ...sql.ConflictOption) *Example1UpsertBulk {
	eb.conflict = opts
	return &Example1UpsertBulk{
		create: eb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Example1.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (eb *Example1CreateBulk) OnConflictColumns(columns ...string) *Example1UpsertBulk {
	eb.conflict = append(eb.conflict, sql.ConflictColumns(columns...))
	return &Example1UpsertBulk{
		create: eb,
	}
}

// Example1UpsertBulk is the builder for "upsert"-ing
// a bulk of Example1 nodes.
type Example1UpsertBulk struct {
	create *Example1CreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Example1.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(example1.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *Example1UpsertBulk) UpdateNewValues() *Example1UpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(example1.FieldID)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(example1.FieldCreatedAt)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Example1.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *Example1UpsertBulk) Ignore() *Example1UpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *Example1UpsertBulk) DoNothing() *Example1UpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the Example1CreateBulk.OnConflict
// documentation for more info.
func (u *Example1UpsertBulk) Update(set func(*Example1Upsert)) *Example1UpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&Example1Upsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *Example1UpsertBulk) SetUpdatedAt(v time.Time) *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *Example1UpsertBulk) UpdateUpdatedAt() *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *Example1UpsertBulk) SetDeletedAt(v time.Time) *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *Example1UpsertBulk) UpdateDeletedAt() *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *Example1UpsertBulk) ClearDeletedAt() *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.ClearDeletedAt()
	})
}

// SetExampleValue1 sets the "example_value_1" field.
func (u *Example1UpsertBulk) SetExampleValue1(v string) *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.SetExampleValue1(v)
	})
}

// UpdateExampleValue1 sets the "example_value_1" field to the value that was provided on create.
func (u *Example1UpsertBulk) UpdateExampleValue1() *Example1UpsertBulk {
	return u.Update(func(s *Example1Upsert) {
		s.UpdateExampleValue1()
	})
}

// Exec executes the query.
func (u *Example1UpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the Example1CreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for Example1CreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *Example1UpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}