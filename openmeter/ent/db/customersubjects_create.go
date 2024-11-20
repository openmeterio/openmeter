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
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
)

// CustomerSubjectsCreate is the builder for creating a CustomerSubjects entity.
type CustomerSubjectsCreate struct {
	config
	mutation *CustomerSubjectsMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (csc *CustomerSubjectsCreate) SetNamespace(s string) *CustomerSubjectsCreate {
	csc.mutation.SetNamespace(s)
	return csc
}

// SetCustomerID sets the "customer_id" field.
func (csc *CustomerSubjectsCreate) SetCustomerID(s string) *CustomerSubjectsCreate {
	csc.mutation.SetCustomerID(s)
	return csc
}

// SetSubjectKey sets the "subject_key" field.
func (csc *CustomerSubjectsCreate) SetSubjectKey(s string) *CustomerSubjectsCreate {
	csc.mutation.SetSubjectKey(s)
	return csc
}

// SetCreatedAt sets the "created_at" field.
func (csc *CustomerSubjectsCreate) SetCreatedAt(t time.Time) *CustomerSubjectsCreate {
	csc.mutation.SetCreatedAt(t)
	return csc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (csc *CustomerSubjectsCreate) SetNillableCreatedAt(t *time.Time) *CustomerSubjectsCreate {
	if t != nil {
		csc.SetCreatedAt(*t)
	}
	return csc
}

// SetDeletedAt sets the "deleted_at" field.
func (csc *CustomerSubjectsCreate) SetDeletedAt(t time.Time) *CustomerSubjectsCreate {
	csc.mutation.SetDeletedAt(t)
	return csc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (csc *CustomerSubjectsCreate) SetNillableDeletedAt(t *time.Time) *CustomerSubjectsCreate {
	if t != nil {
		csc.SetDeletedAt(*t)
	}
	return csc
}

// SetIsDeleted sets the "is_deleted" field.
func (csc *CustomerSubjectsCreate) SetIsDeleted(b bool) *CustomerSubjectsCreate {
	csc.mutation.SetIsDeleted(b)
	return csc
}

// SetNillableIsDeleted sets the "is_deleted" field if the given value is not nil.
func (csc *CustomerSubjectsCreate) SetNillableIsDeleted(b *bool) *CustomerSubjectsCreate {
	if b != nil {
		csc.SetIsDeleted(*b)
	}
	return csc
}

// SetCustomer sets the "customer" edge to the Customer entity.
func (csc *CustomerSubjectsCreate) SetCustomer(c *Customer) *CustomerSubjectsCreate {
	return csc.SetCustomerID(c.ID)
}

// Mutation returns the CustomerSubjectsMutation object of the builder.
func (csc *CustomerSubjectsCreate) Mutation() *CustomerSubjectsMutation {
	return csc.mutation
}

// Save creates the CustomerSubjects in the database.
func (csc *CustomerSubjectsCreate) Save(ctx context.Context) (*CustomerSubjects, error) {
	csc.defaults()
	return withHooks(ctx, csc.sqlSave, csc.mutation, csc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (csc *CustomerSubjectsCreate) SaveX(ctx context.Context) *CustomerSubjects {
	v, err := csc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (csc *CustomerSubjectsCreate) Exec(ctx context.Context) error {
	_, err := csc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (csc *CustomerSubjectsCreate) ExecX(ctx context.Context) {
	if err := csc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (csc *CustomerSubjectsCreate) defaults() {
	if _, ok := csc.mutation.CreatedAt(); !ok {
		v := customersubjects.DefaultCreatedAt()
		csc.mutation.SetCreatedAt(v)
	}
	if _, ok := csc.mutation.IsDeleted(); !ok {
		v := customersubjects.DefaultIsDeleted
		csc.mutation.SetIsDeleted(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (csc *CustomerSubjectsCreate) check() error {
	if _, ok := csc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "CustomerSubjects.namespace"`)}
	}
	if v, ok := csc.mutation.Namespace(); ok {
		if err := customersubjects.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "CustomerSubjects.namespace": %w`, err)}
		}
	}
	if _, ok := csc.mutation.CustomerID(); !ok {
		return &ValidationError{Name: "customer_id", err: errors.New(`db: missing required field "CustomerSubjects.customer_id"`)}
	}
	if v, ok := csc.mutation.CustomerID(); ok {
		if err := customersubjects.CustomerIDValidator(v); err != nil {
			return &ValidationError{Name: "customer_id", err: fmt.Errorf(`db: validator failed for field "CustomerSubjects.customer_id": %w`, err)}
		}
	}
	if _, ok := csc.mutation.SubjectKey(); !ok {
		return &ValidationError{Name: "subject_key", err: errors.New(`db: missing required field "CustomerSubjects.subject_key"`)}
	}
	if v, ok := csc.mutation.SubjectKey(); ok {
		if err := customersubjects.SubjectKeyValidator(v); err != nil {
			return &ValidationError{Name: "subject_key", err: fmt.Errorf(`db: validator failed for field "CustomerSubjects.subject_key": %w`, err)}
		}
	}
	if _, ok := csc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "CustomerSubjects.created_at"`)}
	}
	if _, ok := csc.mutation.IsDeleted(); !ok {
		return &ValidationError{Name: "is_deleted", err: errors.New(`db: missing required field "CustomerSubjects.is_deleted"`)}
	}
	if len(csc.mutation.CustomerIDs()) == 0 {
		return &ValidationError{Name: "customer", err: errors.New(`db: missing required edge "CustomerSubjects.customer"`)}
	}
	return nil
}

func (csc *CustomerSubjectsCreate) sqlSave(ctx context.Context) (*CustomerSubjects, error) {
	if err := csc.check(); err != nil {
		return nil, err
	}
	_node, _spec := csc.createSpec()
	if err := sqlgraph.CreateNode(ctx, csc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	id := _spec.ID.Value.(int64)
	_node.ID = int(id)
	csc.mutation.id = &_node.ID
	csc.mutation.done = true
	return _node, nil
}

func (csc *CustomerSubjectsCreate) createSpec() (*CustomerSubjects, *sqlgraph.CreateSpec) {
	var (
		_node = &CustomerSubjects{config: csc.config}
		_spec = sqlgraph.NewCreateSpec(customersubjects.Table, sqlgraph.NewFieldSpec(customersubjects.FieldID, field.TypeInt))
	)
	_spec.OnConflict = csc.conflict
	if value, ok := csc.mutation.Namespace(); ok {
		_spec.SetField(customersubjects.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := csc.mutation.SubjectKey(); ok {
		_spec.SetField(customersubjects.FieldSubjectKey, field.TypeString, value)
		_node.SubjectKey = value
	}
	if value, ok := csc.mutation.CreatedAt(); ok {
		_spec.SetField(customersubjects.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := csc.mutation.DeletedAt(); ok {
		_spec.SetField(customersubjects.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := csc.mutation.IsDeleted(); ok {
		_spec.SetField(customersubjects.FieldIsDeleted, field.TypeBool, value)
		_node.IsDeleted = value
	}
	if nodes := csc.mutation.CustomerIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   customersubjects.CustomerTable,
			Columns: []string{customersubjects.CustomerColumn},
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
//	client.CustomerSubjects.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.CustomerSubjectsUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (csc *CustomerSubjectsCreate) OnConflict(opts ...sql.ConflictOption) *CustomerSubjectsUpsertOne {
	csc.conflict = opts
	return &CustomerSubjectsUpsertOne{
		create: csc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (csc *CustomerSubjectsCreate) OnConflictColumns(columns ...string) *CustomerSubjectsUpsertOne {
	csc.conflict = append(csc.conflict, sql.ConflictColumns(columns...))
	return &CustomerSubjectsUpsertOne{
		create: csc,
	}
}

type (
	// CustomerSubjectsUpsertOne is the builder for "upsert"-ing
	//  one CustomerSubjects node.
	CustomerSubjectsUpsertOne struct {
		create *CustomerSubjectsCreate
	}

	// CustomerSubjectsUpsert is the "OnConflict" setter.
	CustomerSubjectsUpsert struct {
		*sql.UpdateSet
	}
)

// SetDeletedAt sets the "deleted_at" field.
func (u *CustomerSubjectsUpsert) SetDeletedAt(v time.Time) *CustomerSubjectsUpsert {
	u.Set(customersubjects.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *CustomerSubjectsUpsert) UpdateDeletedAt() *CustomerSubjectsUpsert {
	u.SetExcluded(customersubjects.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *CustomerSubjectsUpsert) ClearDeletedAt() *CustomerSubjectsUpsert {
	u.SetNull(customersubjects.FieldDeletedAt)
	return u
}

// SetIsDeleted sets the "is_deleted" field.
func (u *CustomerSubjectsUpsert) SetIsDeleted(v bool) *CustomerSubjectsUpsert {
	u.Set(customersubjects.FieldIsDeleted, v)
	return u
}

// UpdateIsDeleted sets the "is_deleted" field to the value that was provided on create.
func (u *CustomerSubjectsUpsert) UpdateIsDeleted() *CustomerSubjectsUpsert {
	u.SetExcluded(customersubjects.FieldIsDeleted)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create.
// Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//		).
//		Exec(ctx)
func (u *CustomerSubjectsUpsertOne) UpdateNewValues() *CustomerSubjectsUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(customersubjects.FieldNamespace)
		}
		if _, exists := u.create.mutation.CustomerID(); exists {
			s.SetIgnore(customersubjects.FieldCustomerID)
		}
		if _, exists := u.create.mutation.SubjectKey(); exists {
			s.SetIgnore(customersubjects.FieldSubjectKey)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(customersubjects.FieldCreatedAt)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *CustomerSubjectsUpsertOne) Ignore() *CustomerSubjectsUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *CustomerSubjectsUpsertOne) DoNothing() *CustomerSubjectsUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the CustomerSubjectsCreate.OnConflict
// documentation for more info.
func (u *CustomerSubjectsUpsertOne) Update(set func(*CustomerSubjectsUpsert)) *CustomerSubjectsUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&CustomerSubjectsUpsert{UpdateSet: update})
	}))
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *CustomerSubjectsUpsertOne) SetDeletedAt(v time.Time) *CustomerSubjectsUpsertOne {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *CustomerSubjectsUpsertOne) UpdateDeletedAt() *CustomerSubjectsUpsertOne {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *CustomerSubjectsUpsertOne) ClearDeletedAt() *CustomerSubjectsUpsertOne {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.ClearDeletedAt()
	})
}

// SetIsDeleted sets the "is_deleted" field.
func (u *CustomerSubjectsUpsertOne) SetIsDeleted(v bool) *CustomerSubjectsUpsertOne {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.SetIsDeleted(v)
	})
}

// UpdateIsDeleted sets the "is_deleted" field to the value that was provided on create.
func (u *CustomerSubjectsUpsertOne) UpdateIsDeleted() *CustomerSubjectsUpsertOne {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.UpdateIsDeleted()
	})
}

// Exec executes the query.
func (u *CustomerSubjectsUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for CustomerSubjectsCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *CustomerSubjectsUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *CustomerSubjectsUpsertOne) ID(ctx context.Context) (id int, err error) {
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *CustomerSubjectsUpsertOne) IDX(ctx context.Context) int {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// CustomerSubjectsCreateBulk is the builder for creating many CustomerSubjects entities in bulk.
type CustomerSubjectsCreateBulk struct {
	config
	err      error
	builders []*CustomerSubjectsCreate
	conflict []sql.ConflictOption
}

// Save creates the CustomerSubjects entities in the database.
func (cscb *CustomerSubjectsCreateBulk) Save(ctx context.Context) ([]*CustomerSubjects, error) {
	if cscb.err != nil {
		return nil, cscb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(cscb.builders))
	nodes := make([]*CustomerSubjects, len(cscb.builders))
	mutators := make([]Mutator, len(cscb.builders))
	for i := range cscb.builders {
		func(i int, root context.Context) {
			builder := cscb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*CustomerSubjectsMutation)
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
					_, err = mutators[i+1].Mutate(root, cscb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = cscb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, cscb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, cscb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (cscb *CustomerSubjectsCreateBulk) SaveX(ctx context.Context) []*CustomerSubjects {
	v, err := cscb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (cscb *CustomerSubjectsCreateBulk) Exec(ctx context.Context) error {
	_, err := cscb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cscb *CustomerSubjectsCreateBulk) ExecX(ctx context.Context) {
	if err := cscb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.CustomerSubjects.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.CustomerSubjectsUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (cscb *CustomerSubjectsCreateBulk) OnConflict(opts ...sql.ConflictOption) *CustomerSubjectsUpsertBulk {
	cscb.conflict = opts
	return &CustomerSubjectsUpsertBulk{
		create: cscb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (cscb *CustomerSubjectsCreateBulk) OnConflictColumns(columns ...string) *CustomerSubjectsUpsertBulk {
	cscb.conflict = append(cscb.conflict, sql.ConflictColumns(columns...))
	return &CustomerSubjectsUpsertBulk{
		create: cscb,
	}
}

// CustomerSubjectsUpsertBulk is the builder for "upsert"-ing
// a bulk of CustomerSubjects nodes.
type CustomerSubjectsUpsertBulk struct {
	create *CustomerSubjectsCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//		).
//		Exec(ctx)
func (u *CustomerSubjectsUpsertBulk) UpdateNewValues() *CustomerSubjectsUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(customersubjects.FieldNamespace)
			}
			if _, exists := b.mutation.CustomerID(); exists {
				s.SetIgnore(customersubjects.FieldCustomerID)
			}
			if _, exists := b.mutation.SubjectKey(); exists {
				s.SetIgnore(customersubjects.FieldSubjectKey)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(customersubjects.FieldCreatedAt)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.CustomerSubjects.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *CustomerSubjectsUpsertBulk) Ignore() *CustomerSubjectsUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *CustomerSubjectsUpsertBulk) DoNothing() *CustomerSubjectsUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the CustomerSubjectsCreateBulk.OnConflict
// documentation for more info.
func (u *CustomerSubjectsUpsertBulk) Update(set func(*CustomerSubjectsUpsert)) *CustomerSubjectsUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&CustomerSubjectsUpsert{UpdateSet: update})
	}))
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *CustomerSubjectsUpsertBulk) SetDeletedAt(v time.Time) *CustomerSubjectsUpsertBulk {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *CustomerSubjectsUpsertBulk) UpdateDeletedAt() *CustomerSubjectsUpsertBulk {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *CustomerSubjectsUpsertBulk) ClearDeletedAt() *CustomerSubjectsUpsertBulk {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.ClearDeletedAt()
	})
}

// SetIsDeleted sets the "is_deleted" field.
func (u *CustomerSubjectsUpsertBulk) SetIsDeleted(v bool) *CustomerSubjectsUpsertBulk {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.SetIsDeleted(v)
	})
}

// UpdateIsDeleted sets the "is_deleted" field to the value that was provided on create.
func (u *CustomerSubjectsUpsertBulk) UpdateIsDeleted() *CustomerSubjectsUpsertBulk {
	return u.Update(func(s *CustomerSubjectsUpsert) {
		s.UpdateIsDeleted()
	})
}

// Exec executes the query.
func (u *CustomerSubjectsUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the CustomerSubjectsCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for CustomerSubjectsCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *CustomerSubjectsUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
