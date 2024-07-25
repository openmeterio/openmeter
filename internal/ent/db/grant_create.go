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
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/ent/db/entitlement"
	"github.com/openmeterio/openmeter/internal/ent/db/grant"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

// GrantCreate is the builder for creating a Grant entity.
type GrantCreate struct {
	config
	mutation *GrantMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (gc *GrantCreate) SetNamespace(s string) *GrantCreate {
	gc.mutation.SetNamespace(s)
	return gc
}

// SetMetadata sets the "metadata" field.
func (gc *GrantCreate) SetMetadata(m map[string]string) *GrantCreate {
	gc.mutation.SetMetadata(m)
	return gc
}

// SetCreatedAt sets the "created_at" field.
func (gc *GrantCreate) SetCreatedAt(t time.Time) *GrantCreate {
	gc.mutation.SetCreatedAt(t)
	return gc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (gc *GrantCreate) SetNillableCreatedAt(t *time.Time) *GrantCreate {
	if t != nil {
		gc.SetCreatedAt(*t)
	}
	return gc
}

// SetUpdatedAt sets the "updated_at" field.
func (gc *GrantCreate) SetUpdatedAt(t time.Time) *GrantCreate {
	gc.mutation.SetUpdatedAt(t)
	return gc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (gc *GrantCreate) SetNillableUpdatedAt(t *time.Time) *GrantCreate {
	if t != nil {
		gc.SetUpdatedAt(*t)
	}
	return gc
}

// SetDeletedAt sets the "deleted_at" field.
func (gc *GrantCreate) SetDeletedAt(t time.Time) *GrantCreate {
	gc.mutation.SetDeletedAt(t)
	return gc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (gc *GrantCreate) SetNillableDeletedAt(t *time.Time) *GrantCreate {
	if t != nil {
		gc.SetDeletedAt(*t)
	}
	return gc
}

// SetOwnerID sets the "owner_id" field.
func (gc *GrantCreate) SetOwnerID(co credit.GrantOwner) *GrantCreate {
	gc.mutation.SetOwnerID(co)
	return gc
}

// SetAmount sets the "amount" field.
func (gc *GrantCreate) SetAmount(f float64) *GrantCreate {
	gc.mutation.SetAmount(f)
	return gc
}

// SetPriority sets the "priority" field.
func (gc *GrantCreate) SetPriority(u uint8) *GrantCreate {
	gc.mutation.SetPriority(u)
	return gc
}

// SetNillablePriority sets the "priority" field if the given value is not nil.
func (gc *GrantCreate) SetNillablePriority(u *uint8) *GrantCreate {
	if u != nil {
		gc.SetPriority(*u)
	}
	return gc
}

// SetEffectiveAt sets the "effective_at" field.
func (gc *GrantCreate) SetEffectiveAt(t time.Time) *GrantCreate {
	gc.mutation.SetEffectiveAt(t)
	return gc
}

// SetExpiration sets the "expiration" field.
func (gc *GrantCreate) SetExpiration(cp credit.ExpirationPeriod) *GrantCreate {
	gc.mutation.SetExpiration(cp)
	return gc
}

// SetExpiresAt sets the "expires_at" field.
func (gc *GrantCreate) SetExpiresAt(t time.Time) *GrantCreate {
	gc.mutation.SetExpiresAt(t)
	return gc
}

// SetVoidedAt sets the "voided_at" field.
func (gc *GrantCreate) SetVoidedAt(t time.Time) *GrantCreate {
	gc.mutation.SetVoidedAt(t)
	return gc
}

// SetNillableVoidedAt sets the "voided_at" field if the given value is not nil.
func (gc *GrantCreate) SetNillableVoidedAt(t *time.Time) *GrantCreate {
	if t != nil {
		gc.SetVoidedAt(*t)
	}
	return gc
}

// SetResetMaxRollover sets the "reset_max_rollover" field.
func (gc *GrantCreate) SetResetMaxRollover(f float64) *GrantCreate {
	gc.mutation.SetResetMaxRollover(f)
	return gc
}

// SetResetMinRollover sets the "reset_min_rollover" field.
func (gc *GrantCreate) SetResetMinRollover(f float64) *GrantCreate {
	gc.mutation.SetResetMinRollover(f)
	return gc
}

// SetRecurrencePeriod sets the "recurrence_period" field.
func (gc *GrantCreate) SetRecurrencePeriod(ri recurrence.RecurrenceInterval) *GrantCreate {
	gc.mutation.SetRecurrencePeriod(ri)
	return gc
}

// SetNillableRecurrencePeriod sets the "recurrence_period" field if the given value is not nil.
func (gc *GrantCreate) SetNillableRecurrencePeriod(ri *recurrence.RecurrenceInterval) *GrantCreate {
	if ri != nil {
		gc.SetRecurrencePeriod(*ri)
	}
	return gc
}

// SetRecurrenceAnchor sets the "recurrence_anchor" field.
func (gc *GrantCreate) SetRecurrenceAnchor(t time.Time) *GrantCreate {
	gc.mutation.SetRecurrenceAnchor(t)
	return gc
}

// SetNillableRecurrenceAnchor sets the "recurrence_anchor" field if the given value is not nil.
func (gc *GrantCreate) SetNillableRecurrenceAnchor(t *time.Time) *GrantCreate {
	if t != nil {
		gc.SetRecurrenceAnchor(*t)
	}
	return gc
}

// SetID sets the "id" field.
func (gc *GrantCreate) SetID(s string) *GrantCreate {
	gc.mutation.SetID(s)
	return gc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (gc *GrantCreate) SetNillableID(s *string) *GrantCreate {
	if s != nil {
		gc.SetID(*s)
	}
	return gc
}

// SetEntitlementID sets the "entitlement" edge to the Entitlement entity by ID.
func (gc *GrantCreate) SetEntitlementID(id string) *GrantCreate {
	gc.mutation.SetEntitlementID(id)
	return gc
}

// SetEntitlement sets the "entitlement" edge to the Entitlement entity.
func (gc *GrantCreate) SetEntitlement(e *Entitlement) *GrantCreate {
	return gc.SetEntitlementID(e.ID)
}

// Mutation returns the GrantMutation object of the builder.
func (gc *GrantCreate) Mutation() *GrantMutation {
	return gc.mutation
}

// Save creates the Grant in the database.
func (gc *GrantCreate) Save(ctx context.Context) (*Grant, error) {
	gc.defaults()
	return withHooks(ctx, gc.sqlSave, gc.mutation, gc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (gc *GrantCreate) SaveX(ctx context.Context) *Grant {
	v, err := gc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gc *GrantCreate) Exec(ctx context.Context) error {
	_, err := gc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gc *GrantCreate) ExecX(ctx context.Context) {
	if err := gc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (gc *GrantCreate) defaults() {
	if _, ok := gc.mutation.CreatedAt(); !ok {
		v := grant.DefaultCreatedAt()
		gc.mutation.SetCreatedAt(v)
	}
	if _, ok := gc.mutation.UpdatedAt(); !ok {
		v := grant.DefaultUpdatedAt()
		gc.mutation.SetUpdatedAt(v)
	}
	if _, ok := gc.mutation.Priority(); !ok {
		v := grant.DefaultPriority
		gc.mutation.SetPriority(v)
	}
	if _, ok := gc.mutation.ID(); !ok {
		v := grant.DefaultID()
		gc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (gc *GrantCreate) check() error {
	if _, ok := gc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "Grant.namespace"`)}
	}
	if v, ok := gc.mutation.Namespace(); ok {
		if err := grant.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "Grant.namespace": %w`, err)}
		}
	}
	if _, ok := gc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "Grant.created_at"`)}
	}
	if _, ok := gc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "Grant.updated_at"`)}
	}
	if _, ok := gc.mutation.OwnerID(); !ok {
		return &ValidationError{Name: "owner_id", err: errors.New(`db: missing required field "Grant.owner_id"`)}
	}
	if _, ok := gc.mutation.Amount(); !ok {
		return &ValidationError{Name: "amount", err: errors.New(`db: missing required field "Grant.amount"`)}
	}
	if _, ok := gc.mutation.Priority(); !ok {
		return &ValidationError{Name: "priority", err: errors.New(`db: missing required field "Grant.priority"`)}
	}
	if _, ok := gc.mutation.EffectiveAt(); !ok {
		return &ValidationError{Name: "effective_at", err: errors.New(`db: missing required field "Grant.effective_at"`)}
	}
	if _, ok := gc.mutation.Expiration(); !ok {
		return &ValidationError{Name: "expiration", err: errors.New(`db: missing required field "Grant.expiration"`)}
	}
	if _, ok := gc.mutation.ExpiresAt(); !ok {
		return &ValidationError{Name: "expires_at", err: errors.New(`db: missing required field "Grant.expires_at"`)}
	}
	if _, ok := gc.mutation.ResetMaxRollover(); !ok {
		return &ValidationError{Name: "reset_max_rollover", err: errors.New(`db: missing required field "Grant.reset_max_rollover"`)}
	}
	if _, ok := gc.mutation.ResetMinRollover(); !ok {
		return &ValidationError{Name: "reset_min_rollover", err: errors.New(`db: missing required field "Grant.reset_min_rollover"`)}
	}
	if v, ok := gc.mutation.RecurrencePeriod(); ok {
		if err := grant.RecurrencePeriodValidator(v); err != nil {
			return &ValidationError{Name: "recurrence_period", err: fmt.Errorf(`db: validator failed for field "Grant.recurrence_period": %w`, err)}
		}
	}
	if _, ok := gc.mutation.EntitlementID(); !ok {
		return &ValidationError{Name: "entitlement", err: errors.New(`db: missing required edge "Grant.entitlement"`)}
	}
	return nil
}

func (gc *GrantCreate) sqlSave(ctx context.Context) (*Grant, error) {
	if err := gc.check(); err != nil {
		return nil, err
	}
	_node, _spec := gc.createSpec()
	if err := sqlgraph.CreateNode(ctx, gc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected Grant.ID type: %T", _spec.ID.Value)
		}
	}
	gc.mutation.id = &_node.ID
	gc.mutation.done = true
	return _node, nil
}

func (gc *GrantCreate) createSpec() (*Grant, *sqlgraph.CreateSpec) {
	var (
		_node = &Grant{config: gc.config}
		_spec = sqlgraph.NewCreateSpec(grant.Table, sqlgraph.NewFieldSpec(grant.FieldID, field.TypeString))
	)
	_spec.OnConflict = gc.conflict
	if id, ok := gc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := gc.mutation.Namespace(); ok {
		_spec.SetField(grant.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := gc.mutation.Metadata(); ok {
		_spec.SetField(grant.FieldMetadata, field.TypeJSON, value)
		_node.Metadata = value
	}
	if value, ok := gc.mutation.CreatedAt(); ok {
		_spec.SetField(grant.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := gc.mutation.UpdatedAt(); ok {
		_spec.SetField(grant.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := gc.mutation.DeletedAt(); ok {
		_spec.SetField(grant.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := gc.mutation.Amount(); ok {
		_spec.SetField(grant.FieldAmount, field.TypeFloat64, value)
		_node.Amount = value
	}
	if value, ok := gc.mutation.Priority(); ok {
		_spec.SetField(grant.FieldPriority, field.TypeUint8, value)
		_node.Priority = value
	}
	if value, ok := gc.mutation.EffectiveAt(); ok {
		_spec.SetField(grant.FieldEffectiveAt, field.TypeTime, value)
		_node.EffectiveAt = value
	}
	if value, ok := gc.mutation.Expiration(); ok {
		_spec.SetField(grant.FieldExpiration, field.TypeJSON, value)
		_node.Expiration = value
	}
	if value, ok := gc.mutation.ExpiresAt(); ok {
		_spec.SetField(grant.FieldExpiresAt, field.TypeTime, value)
		_node.ExpiresAt = value
	}
	if value, ok := gc.mutation.VoidedAt(); ok {
		_spec.SetField(grant.FieldVoidedAt, field.TypeTime, value)
		_node.VoidedAt = &value
	}
	if value, ok := gc.mutation.ResetMaxRollover(); ok {
		_spec.SetField(grant.FieldResetMaxRollover, field.TypeFloat64, value)
		_node.ResetMaxRollover = value
	}
	if value, ok := gc.mutation.ResetMinRollover(); ok {
		_spec.SetField(grant.FieldResetMinRollover, field.TypeFloat64, value)
		_node.ResetMinRollover = value
	}
	if value, ok := gc.mutation.RecurrencePeriod(); ok {
		_spec.SetField(grant.FieldRecurrencePeriod, field.TypeEnum, value)
		_node.RecurrencePeriod = &value
	}
	if value, ok := gc.mutation.RecurrenceAnchor(); ok {
		_spec.SetField(grant.FieldRecurrenceAnchor, field.TypeTime, value)
		_node.RecurrenceAnchor = &value
	}
	if nodes := gc.mutation.EntitlementIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   grant.EntitlementTable,
			Columns: []string{grant.EntitlementColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(entitlement.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.OwnerID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Grant.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.GrantUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (gc *GrantCreate) OnConflict(opts ...sql.ConflictOption) *GrantUpsertOne {
	gc.conflict = opts
	return &GrantUpsertOne{
		create: gc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Grant.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (gc *GrantCreate) OnConflictColumns(columns ...string) *GrantUpsertOne {
	gc.conflict = append(gc.conflict, sql.ConflictColumns(columns...))
	return &GrantUpsertOne{
		create: gc,
	}
}

type (
	// GrantUpsertOne is the builder for "upsert"-ing
	//  one Grant node.
	GrantUpsertOne struct {
		create *GrantCreate
	}

	// GrantUpsert is the "OnConflict" setter.
	GrantUpsert struct {
		*sql.UpdateSet
	}
)

// SetMetadata sets the "metadata" field.
func (u *GrantUpsert) SetMetadata(v map[string]string) *GrantUpsert {
	u.Set(grant.FieldMetadata, v)
	return u
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *GrantUpsert) UpdateMetadata() *GrantUpsert {
	u.SetExcluded(grant.FieldMetadata)
	return u
}

// ClearMetadata clears the value of the "metadata" field.
func (u *GrantUpsert) ClearMetadata() *GrantUpsert {
	u.SetNull(grant.FieldMetadata)
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *GrantUpsert) SetUpdatedAt(v time.Time) *GrantUpsert {
	u.Set(grant.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *GrantUpsert) UpdateUpdatedAt() *GrantUpsert {
	u.SetExcluded(grant.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *GrantUpsert) SetDeletedAt(v time.Time) *GrantUpsert {
	u.Set(grant.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *GrantUpsert) UpdateDeletedAt() *GrantUpsert {
	u.SetExcluded(grant.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *GrantUpsert) ClearDeletedAt() *GrantUpsert {
	u.SetNull(grant.FieldDeletedAt)
	return u
}

// SetVoidedAt sets the "voided_at" field.
func (u *GrantUpsert) SetVoidedAt(v time.Time) *GrantUpsert {
	u.Set(grant.FieldVoidedAt, v)
	return u
}

// UpdateVoidedAt sets the "voided_at" field to the value that was provided on create.
func (u *GrantUpsert) UpdateVoidedAt() *GrantUpsert {
	u.SetExcluded(grant.FieldVoidedAt)
	return u
}

// ClearVoidedAt clears the value of the "voided_at" field.
func (u *GrantUpsert) ClearVoidedAt() *GrantUpsert {
	u.SetNull(grant.FieldVoidedAt)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Grant.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(grant.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *GrantUpsertOne) UpdateNewValues() *GrantUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(grant.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(grant.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(grant.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.OwnerID(); exists {
			s.SetIgnore(grant.FieldOwnerID)
		}
		if _, exists := u.create.mutation.Amount(); exists {
			s.SetIgnore(grant.FieldAmount)
		}
		if _, exists := u.create.mutation.Priority(); exists {
			s.SetIgnore(grant.FieldPriority)
		}
		if _, exists := u.create.mutation.EffectiveAt(); exists {
			s.SetIgnore(grant.FieldEffectiveAt)
		}
		if _, exists := u.create.mutation.Expiration(); exists {
			s.SetIgnore(grant.FieldExpiration)
		}
		if _, exists := u.create.mutation.ExpiresAt(); exists {
			s.SetIgnore(grant.FieldExpiresAt)
		}
		if _, exists := u.create.mutation.ResetMaxRollover(); exists {
			s.SetIgnore(grant.FieldResetMaxRollover)
		}
		if _, exists := u.create.mutation.ResetMinRollover(); exists {
			s.SetIgnore(grant.FieldResetMinRollover)
		}
		if _, exists := u.create.mutation.RecurrencePeriod(); exists {
			s.SetIgnore(grant.FieldRecurrencePeriod)
		}
		if _, exists := u.create.mutation.RecurrenceAnchor(); exists {
			s.SetIgnore(grant.FieldRecurrenceAnchor)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Grant.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *GrantUpsertOne) Ignore() *GrantUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *GrantUpsertOne) DoNothing() *GrantUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the GrantCreate.OnConflict
// documentation for more info.
func (u *GrantUpsertOne) Update(set func(*GrantUpsert)) *GrantUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&GrantUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *GrantUpsertOne) SetMetadata(v map[string]string) *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *GrantUpsertOne) UpdateMetadata() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *GrantUpsertOne) ClearMetadata() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *GrantUpsertOne) SetUpdatedAt(v time.Time) *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *GrantUpsertOne) UpdateUpdatedAt() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *GrantUpsertOne) SetDeletedAt(v time.Time) *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *GrantUpsertOne) UpdateDeletedAt() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *GrantUpsertOne) ClearDeletedAt() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.ClearDeletedAt()
	})
}

// SetVoidedAt sets the "voided_at" field.
func (u *GrantUpsertOne) SetVoidedAt(v time.Time) *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.SetVoidedAt(v)
	})
}

// UpdateVoidedAt sets the "voided_at" field to the value that was provided on create.
func (u *GrantUpsertOne) UpdateVoidedAt() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateVoidedAt()
	})
}

// ClearVoidedAt clears the value of the "voided_at" field.
func (u *GrantUpsertOne) ClearVoidedAt() *GrantUpsertOne {
	return u.Update(func(s *GrantUpsert) {
		s.ClearVoidedAt()
	})
}

// Exec executes the query.
func (u *GrantUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for GrantCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *GrantUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *GrantUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: GrantUpsertOne.ID is not supported by MySQL driver. Use GrantUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *GrantUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// GrantCreateBulk is the builder for creating many Grant entities in bulk.
type GrantCreateBulk struct {
	config
	err      error
	builders []*GrantCreate
	conflict []sql.ConflictOption
}

// Save creates the Grant entities in the database.
func (gcb *GrantCreateBulk) Save(ctx context.Context) ([]*Grant, error) {
	if gcb.err != nil {
		return nil, gcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(gcb.builders))
	nodes := make([]*Grant, len(gcb.builders))
	mutators := make([]Mutator, len(gcb.builders))
	for i := range gcb.builders {
		func(i int, root context.Context) {
			builder := gcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*GrantMutation)
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
					_, err = mutators[i+1].Mutate(root, gcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = gcb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, gcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, gcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (gcb *GrantCreateBulk) SaveX(ctx context.Context) []*Grant {
	v, err := gcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gcb *GrantCreateBulk) Exec(ctx context.Context) error {
	_, err := gcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gcb *GrantCreateBulk) ExecX(ctx context.Context) {
	if err := gcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Grant.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.GrantUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (gcb *GrantCreateBulk) OnConflict(opts ...sql.ConflictOption) *GrantUpsertBulk {
	gcb.conflict = opts
	return &GrantUpsertBulk{
		create: gcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Grant.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (gcb *GrantCreateBulk) OnConflictColumns(columns ...string) *GrantUpsertBulk {
	gcb.conflict = append(gcb.conflict, sql.ConflictColumns(columns...))
	return &GrantUpsertBulk{
		create: gcb,
	}
}

// GrantUpsertBulk is the builder for "upsert"-ing
// a bulk of Grant nodes.
type GrantUpsertBulk struct {
	create *GrantCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Grant.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(grant.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *GrantUpsertBulk) UpdateNewValues() *GrantUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(grant.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(grant.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(grant.FieldCreatedAt)
			}
			if _, exists := b.mutation.OwnerID(); exists {
				s.SetIgnore(grant.FieldOwnerID)
			}
			if _, exists := b.mutation.Amount(); exists {
				s.SetIgnore(grant.FieldAmount)
			}
			if _, exists := b.mutation.Priority(); exists {
				s.SetIgnore(grant.FieldPriority)
			}
			if _, exists := b.mutation.EffectiveAt(); exists {
				s.SetIgnore(grant.FieldEffectiveAt)
			}
			if _, exists := b.mutation.Expiration(); exists {
				s.SetIgnore(grant.FieldExpiration)
			}
			if _, exists := b.mutation.ExpiresAt(); exists {
				s.SetIgnore(grant.FieldExpiresAt)
			}
			if _, exists := b.mutation.ResetMaxRollover(); exists {
				s.SetIgnore(grant.FieldResetMaxRollover)
			}
			if _, exists := b.mutation.ResetMinRollover(); exists {
				s.SetIgnore(grant.FieldResetMinRollover)
			}
			if _, exists := b.mutation.RecurrencePeriod(); exists {
				s.SetIgnore(grant.FieldRecurrencePeriod)
			}
			if _, exists := b.mutation.RecurrenceAnchor(); exists {
				s.SetIgnore(grant.FieldRecurrenceAnchor)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Grant.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *GrantUpsertBulk) Ignore() *GrantUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *GrantUpsertBulk) DoNothing() *GrantUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the GrantCreateBulk.OnConflict
// documentation for more info.
func (u *GrantUpsertBulk) Update(set func(*GrantUpsert)) *GrantUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&GrantUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *GrantUpsertBulk) SetMetadata(v map[string]string) *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *GrantUpsertBulk) UpdateMetadata() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *GrantUpsertBulk) ClearMetadata() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *GrantUpsertBulk) SetUpdatedAt(v time.Time) *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *GrantUpsertBulk) UpdateUpdatedAt() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *GrantUpsertBulk) SetDeletedAt(v time.Time) *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *GrantUpsertBulk) UpdateDeletedAt() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *GrantUpsertBulk) ClearDeletedAt() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.ClearDeletedAt()
	})
}

// SetVoidedAt sets the "voided_at" field.
func (u *GrantUpsertBulk) SetVoidedAt(v time.Time) *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.SetVoidedAt(v)
	})
}

// UpdateVoidedAt sets the "voided_at" field to the value that was provided on create.
func (u *GrantUpsertBulk) UpdateVoidedAt() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.UpdateVoidedAt()
	})
}

// ClearVoidedAt clears the value of the "voided_at" field.
func (u *GrantUpsertBulk) ClearVoidedAt() *GrantUpsertBulk {
	return u.Update(func(s *GrantUpsert) {
		s.ClearVoidedAt()
	})
}

// Exec executes the query.
func (u *GrantUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the GrantCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for GrantCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *GrantUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
