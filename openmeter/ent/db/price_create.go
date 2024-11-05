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
	"github.com/openmeterio/openmeter/openmeter/ent/db/price"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
)

// PriceCreate is the builder for creating a Price entity.
type PriceCreate struct {
	config
	mutation *PriceMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (pc *PriceCreate) SetNamespace(s string) *PriceCreate {
	pc.mutation.SetNamespace(s)
	return pc
}

// SetCreatedAt sets the "created_at" field.
func (pc *PriceCreate) SetCreatedAt(t time.Time) *PriceCreate {
	pc.mutation.SetCreatedAt(t)
	return pc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (pc *PriceCreate) SetNillableCreatedAt(t *time.Time) *PriceCreate {
	if t != nil {
		pc.SetCreatedAt(*t)
	}
	return pc
}

// SetUpdatedAt sets the "updated_at" field.
func (pc *PriceCreate) SetUpdatedAt(t time.Time) *PriceCreate {
	pc.mutation.SetUpdatedAt(t)
	return pc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (pc *PriceCreate) SetNillableUpdatedAt(t *time.Time) *PriceCreate {
	if t != nil {
		pc.SetUpdatedAt(*t)
	}
	return pc
}

// SetDeletedAt sets the "deleted_at" field.
func (pc *PriceCreate) SetDeletedAt(t time.Time) *PriceCreate {
	pc.mutation.SetDeletedAt(t)
	return pc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (pc *PriceCreate) SetNillableDeletedAt(t *time.Time) *PriceCreate {
	if t != nil {
		pc.SetDeletedAt(*t)
	}
	return pc
}

// SetActiveFrom sets the "active_from" field.
func (pc *PriceCreate) SetActiveFrom(t time.Time) *PriceCreate {
	pc.mutation.SetActiveFrom(t)
	return pc
}

// SetActiveTo sets the "active_to" field.
func (pc *PriceCreate) SetActiveTo(t time.Time) *PriceCreate {
	pc.mutation.SetActiveTo(t)
	return pc
}

// SetNillableActiveTo sets the "active_to" field if the given value is not nil.
func (pc *PriceCreate) SetNillableActiveTo(t *time.Time) *PriceCreate {
	if t != nil {
		pc.SetActiveTo(*t)
	}
	return pc
}

// SetKey sets the "key" field.
func (pc *PriceCreate) SetKey(s string) *PriceCreate {
	pc.mutation.SetKey(s)
	return pc
}

// SetSubscriptionID sets the "subscription_id" field.
func (pc *PriceCreate) SetSubscriptionID(s string) *PriceCreate {
	pc.mutation.SetSubscriptionID(s)
	return pc
}

// SetPhaseKey sets the "phase_key" field.
func (pc *PriceCreate) SetPhaseKey(s string) *PriceCreate {
	pc.mutation.SetPhaseKey(s)
	return pc
}

// SetItemKey sets the "item_key" field.
func (pc *PriceCreate) SetItemKey(s string) *PriceCreate {
	pc.mutation.SetItemKey(s)
	return pc
}

// SetValue sets the "value" field.
func (pc *PriceCreate) SetValue(s string) *PriceCreate {
	pc.mutation.SetValue(s)
	return pc
}

// SetID sets the "id" field.
func (pc *PriceCreate) SetID(s string) *PriceCreate {
	pc.mutation.SetID(s)
	return pc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (pc *PriceCreate) SetNillableID(s *string) *PriceCreate {
	if s != nil {
		pc.SetID(*s)
	}
	return pc
}

// SetSubscription sets the "subscription" edge to the Subscription entity.
func (pc *PriceCreate) SetSubscription(s *Subscription) *PriceCreate {
	return pc.SetSubscriptionID(s.ID)
}

// Mutation returns the PriceMutation object of the builder.
func (pc *PriceCreate) Mutation() *PriceMutation {
	return pc.mutation
}

// Save creates the Price in the database.
func (pc *PriceCreate) Save(ctx context.Context) (*Price, error) {
	pc.defaults()
	return withHooks(ctx, pc.sqlSave, pc.mutation, pc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (pc *PriceCreate) SaveX(ctx context.Context) *Price {
	v, err := pc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pc *PriceCreate) Exec(ctx context.Context) error {
	_, err := pc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pc *PriceCreate) ExecX(ctx context.Context) {
	if err := pc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (pc *PriceCreate) defaults() {
	if _, ok := pc.mutation.CreatedAt(); !ok {
		v := price.DefaultCreatedAt()
		pc.mutation.SetCreatedAt(v)
	}
	if _, ok := pc.mutation.UpdatedAt(); !ok {
		v := price.DefaultUpdatedAt()
		pc.mutation.SetUpdatedAt(v)
	}
	if _, ok := pc.mutation.ID(); !ok {
		v := price.DefaultID()
		pc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pc *PriceCreate) check() error {
	if _, ok := pc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "Price.namespace"`)}
	}
	if v, ok := pc.mutation.Namespace(); ok {
		if err := price.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "Price.namespace": %w`, err)}
		}
	}
	if _, ok := pc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "Price.created_at"`)}
	}
	if _, ok := pc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "Price.updated_at"`)}
	}
	if _, ok := pc.mutation.ActiveFrom(); !ok {
		return &ValidationError{Name: "active_from", err: errors.New(`db: missing required field "Price.active_from"`)}
	}
	if _, ok := pc.mutation.Key(); !ok {
		return &ValidationError{Name: "key", err: errors.New(`db: missing required field "Price.key"`)}
	}
	if v, ok := pc.mutation.Key(); ok {
		if err := price.KeyValidator(v); err != nil {
			return &ValidationError{Name: "key", err: fmt.Errorf(`db: validator failed for field "Price.key": %w`, err)}
		}
	}
	if _, ok := pc.mutation.SubscriptionID(); !ok {
		return &ValidationError{Name: "subscription_id", err: errors.New(`db: missing required field "Price.subscription_id"`)}
	}
	if v, ok := pc.mutation.SubscriptionID(); ok {
		if err := price.SubscriptionIDValidator(v); err != nil {
			return &ValidationError{Name: "subscription_id", err: fmt.Errorf(`db: validator failed for field "Price.subscription_id": %w`, err)}
		}
	}
	if _, ok := pc.mutation.PhaseKey(); !ok {
		return &ValidationError{Name: "phase_key", err: errors.New(`db: missing required field "Price.phase_key"`)}
	}
	if v, ok := pc.mutation.PhaseKey(); ok {
		if err := price.PhaseKeyValidator(v); err != nil {
			return &ValidationError{Name: "phase_key", err: fmt.Errorf(`db: validator failed for field "Price.phase_key": %w`, err)}
		}
	}
	if _, ok := pc.mutation.ItemKey(); !ok {
		return &ValidationError{Name: "item_key", err: errors.New(`db: missing required field "Price.item_key"`)}
	}
	if v, ok := pc.mutation.ItemKey(); ok {
		if err := price.ItemKeyValidator(v); err != nil {
			return &ValidationError{Name: "item_key", err: fmt.Errorf(`db: validator failed for field "Price.item_key": %w`, err)}
		}
	}
	if _, ok := pc.mutation.Value(); !ok {
		return &ValidationError{Name: "value", err: errors.New(`db: missing required field "Price.value"`)}
	}
	if v, ok := pc.mutation.Value(); ok {
		if err := price.ValueValidator(v); err != nil {
			return &ValidationError{Name: "value", err: fmt.Errorf(`db: validator failed for field "Price.value": %w`, err)}
		}
	}
	if len(pc.mutation.SubscriptionIDs()) == 0 {
		return &ValidationError{Name: "subscription", err: errors.New(`db: missing required edge "Price.subscription"`)}
	}
	return nil
}

func (pc *PriceCreate) sqlSave(ctx context.Context) (*Price, error) {
	if err := pc.check(); err != nil {
		return nil, err
	}
	_node, _spec := pc.createSpec()
	if err := sqlgraph.CreateNode(ctx, pc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected Price.ID type: %T", _spec.ID.Value)
		}
	}
	pc.mutation.id = &_node.ID
	pc.mutation.done = true
	return _node, nil
}

func (pc *PriceCreate) createSpec() (*Price, *sqlgraph.CreateSpec) {
	var (
		_node = &Price{config: pc.config}
		_spec = sqlgraph.NewCreateSpec(price.Table, sqlgraph.NewFieldSpec(price.FieldID, field.TypeString))
	)
	_spec.OnConflict = pc.conflict
	if id, ok := pc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := pc.mutation.Namespace(); ok {
		_spec.SetField(price.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := pc.mutation.CreatedAt(); ok {
		_spec.SetField(price.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := pc.mutation.UpdatedAt(); ok {
		_spec.SetField(price.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := pc.mutation.DeletedAt(); ok {
		_spec.SetField(price.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := pc.mutation.ActiveFrom(); ok {
		_spec.SetField(price.FieldActiveFrom, field.TypeTime, value)
		_node.ActiveFrom = value
	}
	if value, ok := pc.mutation.ActiveTo(); ok {
		_spec.SetField(price.FieldActiveTo, field.TypeTime, value)
		_node.ActiveTo = &value
	}
	if value, ok := pc.mutation.Key(); ok {
		_spec.SetField(price.FieldKey, field.TypeString, value)
		_node.Key = value
	}
	if value, ok := pc.mutation.PhaseKey(); ok {
		_spec.SetField(price.FieldPhaseKey, field.TypeString, value)
		_node.PhaseKey = value
	}
	if value, ok := pc.mutation.ItemKey(); ok {
		_spec.SetField(price.FieldItemKey, field.TypeString, value)
		_node.ItemKey = value
	}
	if value, ok := pc.mutation.Value(); ok {
		_spec.SetField(price.FieldValue, field.TypeString, value)
		_node.Value = value
	}
	if nodes := pc.mutation.SubscriptionIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   price.SubscriptionTable,
			Columns: []string{price.SubscriptionColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscription.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.SubscriptionID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Price.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PriceUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (pc *PriceCreate) OnConflict(opts ...sql.ConflictOption) *PriceUpsertOne {
	pc.conflict = opts
	return &PriceUpsertOne{
		create: pc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Price.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (pc *PriceCreate) OnConflictColumns(columns ...string) *PriceUpsertOne {
	pc.conflict = append(pc.conflict, sql.ConflictColumns(columns...))
	return &PriceUpsertOne{
		create: pc,
	}
}

type (
	// PriceUpsertOne is the builder for "upsert"-ing
	//  one Price node.
	PriceUpsertOne struct {
		create *PriceCreate
	}

	// PriceUpsert is the "OnConflict" setter.
	PriceUpsert struct {
		*sql.UpdateSet
	}
)

// SetUpdatedAt sets the "updated_at" field.
func (u *PriceUpsert) SetUpdatedAt(v time.Time) *PriceUpsert {
	u.Set(price.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PriceUpsert) UpdateUpdatedAt() *PriceUpsert {
	u.SetExcluded(price.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PriceUpsert) SetDeletedAt(v time.Time) *PriceUpsert {
	u.Set(price.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PriceUpsert) UpdateDeletedAt() *PriceUpsert {
	u.SetExcluded(price.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PriceUpsert) ClearDeletedAt() *PriceUpsert {
	u.SetNull(price.FieldDeletedAt)
	return u
}

// SetActiveTo sets the "active_to" field.
func (u *PriceUpsert) SetActiveTo(v time.Time) *PriceUpsert {
	u.Set(price.FieldActiveTo, v)
	return u
}

// UpdateActiveTo sets the "active_to" field to the value that was provided on create.
func (u *PriceUpsert) UpdateActiveTo() *PriceUpsert {
	u.SetExcluded(price.FieldActiveTo)
	return u
}

// ClearActiveTo clears the value of the "active_to" field.
func (u *PriceUpsert) ClearActiveTo() *PriceUpsert {
	u.SetNull(price.FieldActiveTo)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Price.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(price.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PriceUpsertOne) UpdateNewValues() *PriceUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(price.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(price.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(price.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.ActiveFrom(); exists {
			s.SetIgnore(price.FieldActiveFrom)
		}
		if _, exists := u.create.mutation.Key(); exists {
			s.SetIgnore(price.FieldKey)
		}
		if _, exists := u.create.mutation.SubscriptionID(); exists {
			s.SetIgnore(price.FieldSubscriptionID)
		}
		if _, exists := u.create.mutation.PhaseKey(); exists {
			s.SetIgnore(price.FieldPhaseKey)
		}
		if _, exists := u.create.mutation.ItemKey(); exists {
			s.SetIgnore(price.FieldItemKey)
		}
		if _, exists := u.create.mutation.Value(); exists {
			s.SetIgnore(price.FieldValue)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Price.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *PriceUpsertOne) Ignore() *PriceUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PriceUpsertOne) DoNothing() *PriceUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PriceCreate.OnConflict
// documentation for more info.
func (u *PriceUpsertOne) Update(set func(*PriceUpsert)) *PriceUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PriceUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PriceUpsertOne) SetUpdatedAt(v time.Time) *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PriceUpsertOne) UpdateUpdatedAt() *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PriceUpsertOne) SetDeletedAt(v time.Time) *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PriceUpsertOne) UpdateDeletedAt() *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PriceUpsertOne) ClearDeletedAt() *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.ClearDeletedAt()
	})
}

// SetActiveTo sets the "active_to" field.
func (u *PriceUpsertOne) SetActiveTo(v time.Time) *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.SetActiveTo(v)
	})
}

// UpdateActiveTo sets the "active_to" field to the value that was provided on create.
func (u *PriceUpsertOne) UpdateActiveTo() *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateActiveTo()
	})
}

// ClearActiveTo clears the value of the "active_to" field.
func (u *PriceUpsertOne) ClearActiveTo() *PriceUpsertOne {
	return u.Update(func(s *PriceUpsert) {
		s.ClearActiveTo()
	})
}

// Exec executes the query.
func (u *PriceUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PriceCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PriceUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *PriceUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: PriceUpsertOne.ID is not supported by MySQL driver. Use PriceUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *PriceUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// PriceCreateBulk is the builder for creating many Price entities in bulk.
type PriceCreateBulk struct {
	config
	err      error
	builders []*PriceCreate
	conflict []sql.ConflictOption
}

// Save creates the Price entities in the database.
func (pcb *PriceCreateBulk) Save(ctx context.Context) ([]*Price, error) {
	if pcb.err != nil {
		return nil, pcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(pcb.builders))
	nodes := make([]*Price, len(pcb.builders))
	mutators := make([]Mutator, len(pcb.builders))
	for i := range pcb.builders {
		func(i int, root context.Context) {
			builder := pcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PriceMutation)
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
					_, err = mutators[i+1].Mutate(root, pcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = pcb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, pcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, pcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (pcb *PriceCreateBulk) SaveX(ctx context.Context) []*Price {
	v, err := pcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pcb *PriceCreateBulk) Exec(ctx context.Context) error {
	_, err := pcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pcb *PriceCreateBulk) ExecX(ctx context.Context) {
	if err := pcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Price.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PriceUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (pcb *PriceCreateBulk) OnConflict(opts ...sql.ConflictOption) *PriceUpsertBulk {
	pcb.conflict = opts
	return &PriceUpsertBulk{
		create: pcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Price.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (pcb *PriceCreateBulk) OnConflictColumns(columns ...string) *PriceUpsertBulk {
	pcb.conflict = append(pcb.conflict, sql.ConflictColumns(columns...))
	return &PriceUpsertBulk{
		create: pcb,
	}
}

// PriceUpsertBulk is the builder for "upsert"-ing
// a bulk of Price nodes.
type PriceUpsertBulk struct {
	create *PriceCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Price.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(price.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PriceUpsertBulk) UpdateNewValues() *PriceUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(price.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(price.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(price.FieldCreatedAt)
			}
			if _, exists := b.mutation.ActiveFrom(); exists {
				s.SetIgnore(price.FieldActiveFrom)
			}
			if _, exists := b.mutation.Key(); exists {
				s.SetIgnore(price.FieldKey)
			}
			if _, exists := b.mutation.SubscriptionID(); exists {
				s.SetIgnore(price.FieldSubscriptionID)
			}
			if _, exists := b.mutation.PhaseKey(); exists {
				s.SetIgnore(price.FieldPhaseKey)
			}
			if _, exists := b.mutation.ItemKey(); exists {
				s.SetIgnore(price.FieldItemKey)
			}
			if _, exists := b.mutation.Value(); exists {
				s.SetIgnore(price.FieldValue)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Price.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *PriceUpsertBulk) Ignore() *PriceUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PriceUpsertBulk) DoNothing() *PriceUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PriceCreateBulk.OnConflict
// documentation for more info.
func (u *PriceUpsertBulk) Update(set func(*PriceUpsert)) *PriceUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PriceUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PriceUpsertBulk) SetUpdatedAt(v time.Time) *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PriceUpsertBulk) UpdateUpdatedAt() *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PriceUpsertBulk) SetDeletedAt(v time.Time) *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PriceUpsertBulk) UpdateDeletedAt() *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PriceUpsertBulk) ClearDeletedAt() *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.ClearDeletedAt()
	})
}

// SetActiveTo sets the "active_to" field.
func (u *PriceUpsertBulk) SetActiveTo(v time.Time) *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.SetActiveTo(v)
	})
}

// UpdateActiveTo sets the "active_to" field to the value that was provided on create.
func (u *PriceUpsertBulk) UpdateActiveTo() *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.UpdateActiveTo()
	})
}

// ClearActiveTo clears the value of the "active_to" field.
func (u *PriceUpsertBulk) ClearActiveTo() *PriceUpsertBulk {
	return u.Update(func(s *PriceUpsert) {
		s.ClearActiveTo()
	})
}

// Exec executes the query.
func (u *PriceUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the PriceCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PriceCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PriceUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}