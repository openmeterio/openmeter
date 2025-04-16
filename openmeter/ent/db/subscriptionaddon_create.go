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
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonquantity"
)

// SubscriptionAddonCreate is the builder for creating a SubscriptionAddon entity.
type SubscriptionAddonCreate struct {
	config
	mutation *SubscriptionAddonMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (sac *SubscriptionAddonCreate) SetNamespace(s string) *SubscriptionAddonCreate {
	sac.mutation.SetNamespace(s)
	return sac
}

// SetMetadata sets the "metadata" field.
func (sac *SubscriptionAddonCreate) SetMetadata(m map[string]string) *SubscriptionAddonCreate {
	sac.mutation.SetMetadata(m)
	return sac
}

// SetCreatedAt sets the "created_at" field.
func (sac *SubscriptionAddonCreate) SetCreatedAt(t time.Time) *SubscriptionAddonCreate {
	sac.mutation.SetCreatedAt(t)
	return sac
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (sac *SubscriptionAddonCreate) SetNillableCreatedAt(t *time.Time) *SubscriptionAddonCreate {
	if t != nil {
		sac.SetCreatedAt(*t)
	}
	return sac
}

// SetUpdatedAt sets the "updated_at" field.
func (sac *SubscriptionAddonCreate) SetUpdatedAt(t time.Time) *SubscriptionAddonCreate {
	sac.mutation.SetUpdatedAt(t)
	return sac
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (sac *SubscriptionAddonCreate) SetNillableUpdatedAt(t *time.Time) *SubscriptionAddonCreate {
	if t != nil {
		sac.SetUpdatedAt(*t)
	}
	return sac
}

// SetDeletedAt sets the "deleted_at" field.
func (sac *SubscriptionAddonCreate) SetDeletedAt(t time.Time) *SubscriptionAddonCreate {
	sac.mutation.SetDeletedAt(t)
	return sac
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (sac *SubscriptionAddonCreate) SetNillableDeletedAt(t *time.Time) *SubscriptionAddonCreate {
	if t != nil {
		sac.SetDeletedAt(*t)
	}
	return sac
}

// SetAddonID sets the "addon_id" field.
func (sac *SubscriptionAddonCreate) SetAddonID(s string) *SubscriptionAddonCreate {
	sac.mutation.SetAddonID(s)
	return sac
}

// SetSubscriptionID sets the "subscription_id" field.
func (sac *SubscriptionAddonCreate) SetSubscriptionID(s string) *SubscriptionAddonCreate {
	sac.mutation.SetSubscriptionID(s)
	return sac
}

// SetID sets the "id" field.
func (sac *SubscriptionAddonCreate) SetID(s string) *SubscriptionAddonCreate {
	sac.mutation.SetID(s)
	return sac
}

// SetNillableID sets the "id" field if the given value is not nil.
func (sac *SubscriptionAddonCreate) SetNillableID(s *string) *SubscriptionAddonCreate {
	if s != nil {
		sac.SetID(*s)
	}
	return sac
}

// SetSubscription sets the "subscription" edge to the Subscription entity.
func (sac *SubscriptionAddonCreate) SetSubscription(s *Subscription) *SubscriptionAddonCreate {
	return sac.SetSubscriptionID(s.ID)
}

// AddQuantityIDs adds the "quantities" edge to the SubscriptionAddonQuantity entity by IDs.
func (sac *SubscriptionAddonCreate) AddQuantityIDs(ids ...string) *SubscriptionAddonCreate {
	sac.mutation.AddQuantityIDs(ids...)
	return sac
}

// AddQuantities adds the "quantities" edges to the SubscriptionAddonQuantity entity.
func (sac *SubscriptionAddonCreate) AddQuantities(s ...*SubscriptionAddonQuantity) *SubscriptionAddonCreate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return sac.AddQuantityIDs(ids...)
}

// SetAddon sets the "addon" edge to the Addon entity.
func (sac *SubscriptionAddonCreate) SetAddon(a *Addon) *SubscriptionAddonCreate {
	return sac.SetAddonID(a.ID)
}

// Mutation returns the SubscriptionAddonMutation object of the builder.
func (sac *SubscriptionAddonCreate) Mutation() *SubscriptionAddonMutation {
	return sac.mutation
}

// Save creates the SubscriptionAddon in the database.
func (sac *SubscriptionAddonCreate) Save(ctx context.Context) (*SubscriptionAddon, error) {
	sac.defaults()
	return withHooks(ctx, sac.sqlSave, sac.mutation, sac.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (sac *SubscriptionAddonCreate) SaveX(ctx context.Context) *SubscriptionAddon {
	v, err := sac.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sac *SubscriptionAddonCreate) Exec(ctx context.Context) error {
	_, err := sac.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sac *SubscriptionAddonCreate) ExecX(ctx context.Context) {
	if err := sac.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (sac *SubscriptionAddonCreate) defaults() {
	if _, ok := sac.mutation.CreatedAt(); !ok {
		v := subscriptionaddon.DefaultCreatedAt()
		sac.mutation.SetCreatedAt(v)
	}
	if _, ok := sac.mutation.UpdatedAt(); !ok {
		v := subscriptionaddon.DefaultUpdatedAt()
		sac.mutation.SetUpdatedAt(v)
	}
	if _, ok := sac.mutation.ID(); !ok {
		v := subscriptionaddon.DefaultID()
		sac.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (sac *SubscriptionAddonCreate) check() error {
	if _, ok := sac.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "SubscriptionAddon.namespace"`)}
	}
	if v, ok := sac.mutation.Namespace(); ok {
		if err := subscriptionaddon.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "SubscriptionAddon.namespace": %w`, err)}
		}
	}
	if _, ok := sac.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "SubscriptionAddon.created_at"`)}
	}
	if _, ok := sac.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "SubscriptionAddon.updated_at"`)}
	}
	if _, ok := sac.mutation.AddonID(); !ok {
		return &ValidationError{Name: "addon_id", err: errors.New(`db: missing required field "SubscriptionAddon.addon_id"`)}
	}
	if v, ok := sac.mutation.AddonID(); ok {
		if err := subscriptionaddon.AddonIDValidator(v); err != nil {
			return &ValidationError{Name: "addon_id", err: fmt.Errorf(`db: validator failed for field "SubscriptionAddon.addon_id": %w`, err)}
		}
	}
	if _, ok := sac.mutation.SubscriptionID(); !ok {
		return &ValidationError{Name: "subscription_id", err: errors.New(`db: missing required field "SubscriptionAddon.subscription_id"`)}
	}
	if v, ok := sac.mutation.SubscriptionID(); ok {
		if err := subscriptionaddon.SubscriptionIDValidator(v); err != nil {
			return &ValidationError{Name: "subscription_id", err: fmt.Errorf(`db: validator failed for field "SubscriptionAddon.subscription_id": %w`, err)}
		}
	}
	if len(sac.mutation.SubscriptionIDs()) == 0 {
		return &ValidationError{Name: "subscription", err: errors.New(`db: missing required edge "SubscriptionAddon.subscription"`)}
	}
	if len(sac.mutation.AddonIDs()) == 0 {
		return &ValidationError{Name: "addon", err: errors.New(`db: missing required edge "SubscriptionAddon.addon"`)}
	}
	return nil
}

func (sac *SubscriptionAddonCreate) sqlSave(ctx context.Context) (*SubscriptionAddon, error) {
	if err := sac.check(); err != nil {
		return nil, err
	}
	_node, _spec := sac.createSpec()
	if err := sqlgraph.CreateNode(ctx, sac.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected SubscriptionAddon.ID type: %T", _spec.ID.Value)
		}
	}
	sac.mutation.id = &_node.ID
	sac.mutation.done = true
	return _node, nil
}

func (sac *SubscriptionAddonCreate) createSpec() (*SubscriptionAddon, *sqlgraph.CreateSpec) {
	var (
		_node = &SubscriptionAddon{config: sac.config}
		_spec = sqlgraph.NewCreateSpec(subscriptionaddon.Table, sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString))
	)
	_spec.OnConflict = sac.conflict
	if id, ok := sac.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := sac.mutation.Namespace(); ok {
		_spec.SetField(subscriptionaddon.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := sac.mutation.Metadata(); ok {
		_spec.SetField(subscriptionaddon.FieldMetadata, field.TypeJSON, value)
		_node.Metadata = value
	}
	if value, ok := sac.mutation.CreatedAt(); ok {
		_spec.SetField(subscriptionaddon.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := sac.mutation.UpdatedAt(); ok {
		_spec.SetField(subscriptionaddon.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := sac.mutation.DeletedAt(); ok {
		_spec.SetField(subscriptionaddon.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if nodes := sac.mutation.SubscriptionIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscriptionaddon.SubscriptionTable,
			Columns: []string{subscriptionaddon.SubscriptionColumn},
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
	if nodes := sac.mutation.QuantitiesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionaddon.QuantitiesTable,
			Columns: []string{subscriptionaddon.QuantitiesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonquantity.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := sac.mutation.AddonIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscriptionaddon.AddonTable,
			Columns: []string{subscriptionaddon.AddonColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.AddonID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.SubscriptionAddon.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.SubscriptionAddonUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (sac *SubscriptionAddonCreate) OnConflict(opts ...sql.ConflictOption) *SubscriptionAddonUpsertOne {
	sac.conflict = opts
	return &SubscriptionAddonUpsertOne{
		create: sac,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (sac *SubscriptionAddonCreate) OnConflictColumns(columns ...string) *SubscriptionAddonUpsertOne {
	sac.conflict = append(sac.conflict, sql.ConflictColumns(columns...))
	return &SubscriptionAddonUpsertOne{
		create: sac,
	}
}

type (
	// SubscriptionAddonUpsertOne is the builder for "upsert"-ing
	//  one SubscriptionAddon node.
	SubscriptionAddonUpsertOne struct {
		create *SubscriptionAddonCreate
	}

	// SubscriptionAddonUpsert is the "OnConflict" setter.
	SubscriptionAddonUpsert struct {
		*sql.UpdateSet
	}
)

// SetMetadata sets the "metadata" field.
func (u *SubscriptionAddonUpsert) SetMetadata(v map[string]string) *SubscriptionAddonUpsert {
	u.Set(subscriptionaddon.FieldMetadata, v)
	return u
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *SubscriptionAddonUpsert) UpdateMetadata() *SubscriptionAddonUpsert {
	u.SetExcluded(subscriptionaddon.FieldMetadata)
	return u
}

// ClearMetadata clears the value of the "metadata" field.
func (u *SubscriptionAddonUpsert) ClearMetadata() *SubscriptionAddonUpsert {
	u.SetNull(subscriptionaddon.FieldMetadata)
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *SubscriptionAddonUpsert) SetUpdatedAt(v time.Time) *SubscriptionAddonUpsert {
	u.Set(subscriptionaddon.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsert) UpdateUpdatedAt() *SubscriptionAddonUpsert {
	u.SetExcluded(subscriptionaddon.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *SubscriptionAddonUpsert) SetDeletedAt(v time.Time) *SubscriptionAddonUpsert {
	u.Set(subscriptionaddon.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsert) UpdateDeletedAt() *SubscriptionAddonUpsert {
	u.SetExcluded(subscriptionaddon.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *SubscriptionAddonUpsert) ClearDeletedAt() *SubscriptionAddonUpsert {
	u.SetNull(subscriptionaddon.FieldDeletedAt)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(subscriptionaddon.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *SubscriptionAddonUpsertOne) UpdateNewValues() *SubscriptionAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(subscriptionaddon.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(subscriptionaddon.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(subscriptionaddon.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.AddonID(); exists {
			s.SetIgnore(subscriptionaddon.FieldAddonID)
		}
		if _, exists := u.create.mutation.SubscriptionID(); exists {
			s.SetIgnore(subscriptionaddon.FieldSubscriptionID)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *SubscriptionAddonUpsertOne) Ignore() *SubscriptionAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *SubscriptionAddonUpsertOne) DoNothing() *SubscriptionAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the SubscriptionAddonCreate.OnConflict
// documentation for more info.
func (u *SubscriptionAddonUpsertOne) Update(set func(*SubscriptionAddonUpsert)) *SubscriptionAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&SubscriptionAddonUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *SubscriptionAddonUpsertOne) SetMetadata(v map[string]string) *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertOne) UpdateMetadata() *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *SubscriptionAddonUpsertOne) ClearMetadata() *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *SubscriptionAddonUpsertOne) SetUpdatedAt(v time.Time) *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertOne) UpdateUpdatedAt() *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *SubscriptionAddonUpsertOne) SetDeletedAt(v time.Time) *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertOne) UpdateDeletedAt() *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *SubscriptionAddonUpsertOne) ClearDeletedAt() *SubscriptionAddonUpsertOne {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.ClearDeletedAt()
	})
}

// Exec executes the query.
func (u *SubscriptionAddonUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for SubscriptionAddonCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *SubscriptionAddonUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *SubscriptionAddonUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: SubscriptionAddonUpsertOne.ID is not supported by MySQL driver. Use SubscriptionAddonUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *SubscriptionAddonUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// SubscriptionAddonCreateBulk is the builder for creating many SubscriptionAddon entities in bulk.
type SubscriptionAddonCreateBulk struct {
	config
	err      error
	builders []*SubscriptionAddonCreate
	conflict []sql.ConflictOption
}

// Save creates the SubscriptionAddon entities in the database.
func (sacb *SubscriptionAddonCreateBulk) Save(ctx context.Context) ([]*SubscriptionAddon, error) {
	if sacb.err != nil {
		return nil, sacb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(sacb.builders))
	nodes := make([]*SubscriptionAddon, len(sacb.builders))
	mutators := make([]Mutator, len(sacb.builders))
	for i := range sacb.builders {
		func(i int, root context.Context) {
			builder := sacb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*SubscriptionAddonMutation)
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
					_, err = mutators[i+1].Mutate(root, sacb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = sacb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, sacb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, sacb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (sacb *SubscriptionAddonCreateBulk) SaveX(ctx context.Context) []*SubscriptionAddon {
	v, err := sacb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sacb *SubscriptionAddonCreateBulk) Exec(ctx context.Context) error {
	_, err := sacb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sacb *SubscriptionAddonCreateBulk) ExecX(ctx context.Context) {
	if err := sacb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.SubscriptionAddon.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.SubscriptionAddonUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (sacb *SubscriptionAddonCreateBulk) OnConflict(opts ...sql.ConflictOption) *SubscriptionAddonUpsertBulk {
	sacb.conflict = opts
	return &SubscriptionAddonUpsertBulk{
		create: sacb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (sacb *SubscriptionAddonCreateBulk) OnConflictColumns(columns ...string) *SubscriptionAddonUpsertBulk {
	sacb.conflict = append(sacb.conflict, sql.ConflictColumns(columns...))
	return &SubscriptionAddonUpsertBulk{
		create: sacb,
	}
}

// SubscriptionAddonUpsertBulk is the builder for "upsert"-ing
// a bulk of SubscriptionAddon nodes.
type SubscriptionAddonUpsertBulk struct {
	create *SubscriptionAddonCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(subscriptionaddon.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *SubscriptionAddonUpsertBulk) UpdateNewValues() *SubscriptionAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(subscriptionaddon.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(subscriptionaddon.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(subscriptionaddon.FieldCreatedAt)
			}
			if _, exists := b.mutation.AddonID(); exists {
				s.SetIgnore(subscriptionaddon.FieldAddonID)
			}
			if _, exists := b.mutation.SubscriptionID(); exists {
				s.SetIgnore(subscriptionaddon.FieldSubscriptionID)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.SubscriptionAddon.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *SubscriptionAddonUpsertBulk) Ignore() *SubscriptionAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *SubscriptionAddonUpsertBulk) DoNothing() *SubscriptionAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the SubscriptionAddonCreateBulk.OnConflict
// documentation for more info.
func (u *SubscriptionAddonUpsertBulk) Update(set func(*SubscriptionAddonUpsert)) *SubscriptionAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&SubscriptionAddonUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *SubscriptionAddonUpsertBulk) SetMetadata(v map[string]string) *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertBulk) UpdateMetadata() *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *SubscriptionAddonUpsertBulk) ClearMetadata() *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *SubscriptionAddonUpsertBulk) SetUpdatedAt(v time.Time) *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertBulk) UpdateUpdatedAt() *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *SubscriptionAddonUpsertBulk) SetDeletedAt(v time.Time) *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *SubscriptionAddonUpsertBulk) UpdateDeletedAt() *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *SubscriptionAddonUpsertBulk) ClearDeletedAt() *SubscriptionAddonUpsertBulk {
	return u.Update(func(s *SubscriptionAddonUpsert) {
		s.ClearDeletedAt()
	})
}

// Exec executes the query.
func (u *SubscriptionAddonUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the SubscriptionAddonCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for SubscriptionAddonCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *SubscriptionAddonUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
