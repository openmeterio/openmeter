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
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
)

// PlanCreate is the builder for creating a Plan entity.
type PlanCreate struct {
	config
	mutation *PlanMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (pc *PlanCreate) SetNamespace(s string) *PlanCreate {
	pc.mutation.SetNamespace(s)
	return pc
}

// SetMetadata sets the "metadata" field.
func (pc *PlanCreate) SetMetadata(m map[string]string) *PlanCreate {
	pc.mutation.SetMetadata(m)
	return pc
}

// SetCreatedAt sets the "created_at" field.
func (pc *PlanCreate) SetCreatedAt(t time.Time) *PlanCreate {
	pc.mutation.SetCreatedAt(t)
	return pc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (pc *PlanCreate) SetNillableCreatedAt(t *time.Time) *PlanCreate {
	if t != nil {
		pc.SetCreatedAt(*t)
	}
	return pc
}

// SetUpdatedAt sets the "updated_at" field.
func (pc *PlanCreate) SetUpdatedAt(t time.Time) *PlanCreate {
	pc.mutation.SetUpdatedAt(t)
	return pc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (pc *PlanCreate) SetNillableUpdatedAt(t *time.Time) *PlanCreate {
	if t != nil {
		pc.SetUpdatedAt(*t)
	}
	return pc
}

// SetDeletedAt sets the "deleted_at" field.
func (pc *PlanCreate) SetDeletedAt(t time.Time) *PlanCreate {
	pc.mutation.SetDeletedAt(t)
	return pc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (pc *PlanCreate) SetNillableDeletedAt(t *time.Time) *PlanCreate {
	if t != nil {
		pc.SetDeletedAt(*t)
	}
	return pc
}

// SetName sets the "name" field.
func (pc *PlanCreate) SetName(s string) *PlanCreate {
	pc.mutation.SetName(s)
	return pc
}

// SetDescription sets the "description" field.
func (pc *PlanCreate) SetDescription(s string) *PlanCreate {
	pc.mutation.SetDescription(s)
	return pc
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (pc *PlanCreate) SetNillableDescription(s *string) *PlanCreate {
	if s != nil {
		pc.SetDescription(*s)
	}
	return pc
}

// SetKey sets the "key" field.
func (pc *PlanCreate) SetKey(s string) *PlanCreate {
	pc.mutation.SetKey(s)
	return pc
}

// SetVersion sets the "version" field.
func (pc *PlanCreate) SetVersion(i int) *PlanCreate {
	pc.mutation.SetVersion(i)
	return pc
}

// SetCurrency sets the "currency" field.
func (pc *PlanCreate) SetCurrency(s string) *PlanCreate {
	pc.mutation.SetCurrency(s)
	return pc
}

// SetNillableCurrency sets the "currency" field if the given value is not nil.
func (pc *PlanCreate) SetNillableCurrency(s *string) *PlanCreate {
	if s != nil {
		pc.SetCurrency(*s)
	}
	return pc
}

// SetEffectiveFrom sets the "effective_from" field.
func (pc *PlanCreate) SetEffectiveFrom(t time.Time) *PlanCreate {
	pc.mutation.SetEffectiveFrom(t)
	return pc
}

// SetNillableEffectiveFrom sets the "effective_from" field if the given value is not nil.
func (pc *PlanCreate) SetNillableEffectiveFrom(t *time.Time) *PlanCreate {
	if t != nil {
		pc.SetEffectiveFrom(*t)
	}
	return pc
}

// SetEffectiveTo sets the "effective_to" field.
func (pc *PlanCreate) SetEffectiveTo(t time.Time) *PlanCreate {
	pc.mutation.SetEffectiveTo(t)
	return pc
}

// SetNillableEffectiveTo sets the "effective_to" field if the given value is not nil.
func (pc *PlanCreate) SetNillableEffectiveTo(t *time.Time) *PlanCreate {
	if t != nil {
		pc.SetEffectiveTo(*t)
	}
	return pc
}

// SetID sets the "id" field.
func (pc *PlanCreate) SetID(s string) *PlanCreate {
	pc.mutation.SetID(s)
	return pc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (pc *PlanCreate) SetNillableID(s *string) *PlanCreate {
	if s != nil {
		pc.SetID(*s)
	}
	return pc
}

// AddPhaseIDs adds the "phases" edge to the PlanPhase entity by IDs.
func (pc *PlanCreate) AddPhaseIDs(ids ...string) *PlanCreate {
	pc.mutation.AddPhaseIDs(ids...)
	return pc
}

// AddPhases adds the "phases" edges to the PlanPhase entity.
func (pc *PlanCreate) AddPhases(p ...*PlanPhase) *PlanCreate {
	ids := make([]string, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pc.AddPhaseIDs(ids...)
}

// Mutation returns the PlanMutation object of the builder.
func (pc *PlanCreate) Mutation() *PlanMutation {
	return pc.mutation
}

// Save creates the Plan in the database.
func (pc *PlanCreate) Save(ctx context.Context) (*Plan, error) {
	pc.defaults()
	return withHooks(ctx, pc.sqlSave, pc.mutation, pc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (pc *PlanCreate) SaveX(ctx context.Context) *Plan {
	v, err := pc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pc *PlanCreate) Exec(ctx context.Context) error {
	_, err := pc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pc *PlanCreate) ExecX(ctx context.Context) {
	if err := pc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (pc *PlanCreate) defaults() {
	if _, ok := pc.mutation.CreatedAt(); !ok {
		v := plan.DefaultCreatedAt()
		pc.mutation.SetCreatedAt(v)
	}
	if _, ok := pc.mutation.UpdatedAt(); !ok {
		v := plan.DefaultUpdatedAt()
		pc.mutation.SetUpdatedAt(v)
	}
	if _, ok := pc.mutation.Currency(); !ok {
		v := plan.DefaultCurrency
		pc.mutation.SetCurrency(v)
	}
	if _, ok := pc.mutation.ID(); !ok {
		v := plan.DefaultID()
		pc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pc *PlanCreate) check() error {
	if _, ok := pc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "Plan.namespace"`)}
	}
	if v, ok := pc.mutation.Namespace(); ok {
		if err := plan.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "Plan.namespace": %w`, err)}
		}
	}
	if _, ok := pc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "Plan.created_at"`)}
	}
	if _, ok := pc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "Plan.updated_at"`)}
	}
	if _, ok := pc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`db: missing required field "Plan.name"`)}
	}
	if _, ok := pc.mutation.Key(); !ok {
		return &ValidationError{Name: "key", err: errors.New(`db: missing required field "Plan.key"`)}
	}
	if v, ok := pc.mutation.Key(); ok {
		if err := plan.KeyValidator(v); err != nil {
			return &ValidationError{Name: "key", err: fmt.Errorf(`db: validator failed for field "Plan.key": %w`, err)}
		}
	}
	if _, ok := pc.mutation.Version(); !ok {
		return &ValidationError{Name: "version", err: errors.New(`db: missing required field "Plan.version"`)}
	}
	if v, ok := pc.mutation.Version(); ok {
		if err := plan.VersionValidator(v); err != nil {
			return &ValidationError{Name: "version", err: fmt.Errorf(`db: validator failed for field "Plan.version": %w`, err)}
		}
	}
	if _, ok := pc.mutation.Currency(); !ok {
		return &ValidationError{Name: "currency", err: errors.New(`db: missing required field "Plan.currency"`)}
	}
	if v, ok := pc.mutation.Currency(); ok {
		if err := plan.CurrencyValidator(v); err != nil {
			return &ValidationError{Name: "currency", err: fmt.Errorf(`db: validator failed for field "Plan.currency": %w`, err)}
		}
	}
	return nil
}

func (pc *PlanCreate) sqlSave(ctx context.Context) (*Plan, error) {
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
			return nil, fmt.Errorf("unexpected Plan.ID type: %T", _spec.ID.Value)
		}
	}
	pc.mutation.id = &_node.ID
	pc.mutation.done = true
	return _node, nil
}

func (pc *PlanCreate) createSpec() (*Plan, *sqlgraph.CreateSpec) {
	var (
		_node = &Plan{config: pc.config}
		_spec = sqlgraph.NewCreateSpec(plan.Table, sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString))
	)
	_spec.OnConflict = pc.conflict
	if id, ok := pc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := pc.mutation.Namespace(); ok {
		_spec.SetField(plan.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := pc.mutation.Metadata(); ok {
		_spec.SetField(plan.FieldMetadata, field.TypeJSON, value)
		_node.Metadata = value
	}
	if value, ok := pc.mutation.CreatedAt(); ok {
		_spec.SetField(plan.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := pc.mutation.UpdatedAt(); ok {
		_spec.SetField(plan.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := pc.mutation.DeletedAt(); ok {
		_spec.SetField(plan.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := pc.mutation.Name(); ok {
		_spec.SetField(plan.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := pc.mutation.Description(); ok {
		_spec.SetField(plan.FieldDescription, field.TypeString, value)
		_node.Description = &value
	}
	if value, ok := pc.mutation.Key(); ok {
		_spec.SetField(plan.FieldKey, field.TypeString, value)
		_node.Key = value
	}
	if value, ok := pc.mutation.Version(); ok {
		_spec.SetField(plan.FieldVersion, field.TypeInt, value)
		_node.Version = value
	}
	if value, ok := pc.mutation.Currency(); ok {
		_spec.SetField(plan.FieldCurrency, field.TypeString, value)
		_node.Currency = value
	}
	if value, ok := pc.mutation.EffectiveFrom(); ok {
		_spec.SetField(plan.FieldEffectiveFrom, field.TypeTime, value)
		_node.EffectiveFrom = &value
	}
	if value, ok := pc.mutation.EffectiveTo(); ok {
		_spec.SetField(plan.FieldEffectiveTo, field.TypeTime, value)
		_node.EffectiveTo = &value
	}
	if nodes := pc.mutation.PhasesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   plan.PhasesTable,
			Columns: []string{plan.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Plan.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlanUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (pc *PlanCreate) OnConflict(opts ...sql.ConflictOption) *PlanUpsertOne {
	pc.conflict = opts
	return &PlanUpsertOne{
		create: pc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Plan.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (pc *PlanCreate) OnConflictColumns(columns ...string) *PlanUpsertOne {
	pc.conflict = append(pc.conflict, sql.ConflictColumns(columns...))
	return &PlanUpsertOne{
		create: pc,
	}
}

type (
	// PlanUpsertOne is the builder for "upsert"-ing
	//  one Plan node.
	PlanUpsertOne struct {
		create *PlanCreate
	}

	// PlanUpsert is the "OnConflict" setter.
	PlanUpsert struct {
		*sql.UpdateSet
	}
)

// SetMetadata sets the "metadata" field.
func (u *PlanUpsert) SetMetadata(v map[string]string) *PlanUpsert {
	u.Set(plan.FieldMetadata, v)
	return u
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanUpsert) UpdateMetadata() *PlanUpsert {
	u.SetExcluded(plan.FieldMetadata)
	return u
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanUpsert) ClearMetadata() *PlanUpsert {
	u.SetNull(plan.FieldMetadata)
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanUpsert) SetUpdatedAt(v time.Time) *PlanUpsert {
	u.Set(plan.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanUpsert) UpdateUpdatedAt() *PlanUpsert {
	u.SetExcluded(plan.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanUpsert) SetDeletedAt(v time.Time) *PlanUpsert {
	u.Set(plan.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanUpsert) UpdateDeletedAt() *PlanUpsert {
	u.SetExcluded(plan.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanUpsert) ClearDeletedAt() *PlanUpsert {
	u.SetNull(plan.FieldDeletedAt)
	return u
}

// SetName sets the "name" field.
func (u *PlanUpsert) SetName(v string) *PlanUpsert {
	u.Set(plan.FieldName, v)
	return u
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlanUpsert) UpdateName() *PlanUpsert {
	u.SetExcluded(plan.FieldName)
	return u
}

// SetDescription sets the "description" field.
func (u *PlanUpsert) SetDescription(v string) *PlanUpsert {
	u.Set(plan.FieldDescription, v)
	return u
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *PlanUpsert) UpdateDescription() *PlanUpsert {
	u.SetExcluded(plan.FieldDescription)
	return u
}

// ClearDescription clears the value of the "description" field.
func (u *PlanUpsert) ClearDescription() *PlanUpsert {
	u.SetNull(plan.FieldDescription)
	return u
}

// SetVersion sets the "version" field.
func (u *PlanUpsert) SetVersion(v int) *PlanUpsert {
	u.Set(plan.FieldVersion, v)
	return u
}

// UpdateVersion sets the "version" field to the value that was provided on create.
func (u *PlanUpsert) UpdateVersion() *PlanUpsert {
	u.SetExcluded(plan.FieldVersion)
	return u
}

// AddVersion adds v to the "version" field.
func (u *PlanUpsert) AddVersion(v int) *PlanUpsert {
	u.Add(plan.FieldVersion, v)
	return u
}

// SetEffectiveFrom sets the "effective_from" field.
func (u *PlanUpsert) SetEffectiveFrom(v time.Time) *PlanUpsert {
	u.Set(plan.FieldEffectiveFrom, v)
	return u
}

// UpdateEffectiveFrom sets the "effective_from" field to the value that was provided on create.
func (u *PlanUpsert) UpdateEffectiveFrom() *PlanUpsert {
	u.SetExcluded(plan.FieldEffectiveFrom)
	return u
}

// ClearEffectiveFrom clears the value of the "effective_from" field.
func (u *PlanUpsert) ClearEffectiveFrom() *PlanUpsert {
	u.SetNull(plan.FieldEffectiveFrom)
	return u
}

// SetEffectiveTo sets the "effective_to" field.
func (u *PlanUpsert) SetEffectiveTo(v time.Time) *PlanUpsert {
	u.Set(plan.FieldEffectiveTo, v)
	return u
}

// UpdateEffectiveTo sets the "effective_to" field to the value that was provided on create.
func (u *PlanUpsert) UpdateEffectiveTo() *PlanUpsert {
	u.SetExcluded(plan.FieldEffectiveTo)
	return u
}

// ClearEffectiveTo clears the value of the "effective_to" field.
func (u *PlanUpsert) ClearEffectiveTo() *PlanUpsert {
	u.SetNull(plan.FieldEffectiveTo)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Plan.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(plan.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlanUpsertOne) UpdateNewValues() *PlanUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(plan.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(plan.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(plan.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.Key(); exists {
			s.SetIgnore(plan.FieldKey)
		}
		if _, exists := u.create.mutation.Currency(); exists {
			s.SetIgnore(plan.FieldCurrency)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Plan.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *PlanUpsertOne) Ignore() *PlanUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlanUpsertOne) DoNothing() *PlanUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlanCreate.OnConflict
// documentation for more info.
func (u *PlanUpsertOne) Update(set func(*PlanUpsert)) *PlanUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlanUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *PlanUpsertOne) SetMetadata(v map[string]string) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateMetadata() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanUpsertOne) ClearMetadata() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanUpsertOne) SetUpdatedAt(v time.Time) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateUpdatedAt() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanUpsertOne) SetDeletedAt(v time.Time) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateDeletedAt() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanUpsertOne) ClearDeletedAt() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.ClearDeletedAt()
	})
}

// SetName sets the "name" field.
func (u *PlanUpsertOne) SetName(v string) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetName(v)
	})
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateName() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateName()
	})
}

// SetDescription sets the "description" field.
func (u *PlanUpsertOne) SetDescription(v string) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetDescription(v)
	})
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateDescription() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateDescription()
	})
}

// ClearDescription clears the value of the "description" field.
func (u *PlanUpsertOne) ClearDescription() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.ClearDescription()
	})
}

// SetVersion sets the "version" field.
func (u *PlanUpsertOne) SetVersion(v int) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetVersion(v)
	})
}

// AddVersion adds v to the "version" field.
func (u *PlanUpsertOne) AddVersion(v int) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.AddVersion(v)
	})
}

// UpdateVersion sets the "version" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateVersion() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateVersion()
	})
}

// SetEffectiveFrom sets the "effective_from" field.
func (u *PlanUpsertOne) SetEffectiveFrom(v time.Time) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetEffectiveFrom(v)
	})
}

// UpdateEffectiveFrom sets the "effective_from" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateEffectiveFrom() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateEffectiveFrom()
	})
}

// ClearEffectiveFrom clears the value of the "effective_from" field.
func (u *PlanUpsertOne) ClearEffectiveFrom() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.ClearEffectiveFrom()
	})
}

// SetEffectiveTo sets the "effective_to" field.
func (u *PlanUpsertOne) SetEffectiveTo(v time.Time) *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.SetEffectiveTo(v)
	})
}

// UpdateEffectiveTo sets the "effective_to" field to the value that was provided on create.
func (u *PlanUpsertOne) UpdateEffectiveTo() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateEffectiveTo()
	})
}

// ClearEffectiveTo clears the value of the "effective_to" field.
func (u *PlanUpsertOne) ClearEffectiveTo() *PlanUpsertOne {
	return u.Update(func(s *PlanUpsert) {
		s.ClearEffectiveTo()
	})
}

// Exec executes the query.
func (u *PlanUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PlanCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlanUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *PlanUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: PlanUpsertOne.ID is not supported by MySQL driver. Use PlanUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *PlanUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// PlanCreateBulk is the builder for creating many Plan entities in bulk.
type PlanCreateBulk struct {
	config
	err      error
	builders []*PlanCreate
	conflict []sql.ConflictOption
}

// Save creates the Plan entities in the database.
func (pcb *PlanCreateBulk) Save(ctx context.Context) ([]*Plan, error) {
	if pcb.err != nil {
		return nil, pcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(pcb.builders))
	nodes := make([]*Plan, len(pcb.builders))
	mutators := make([]Mutator, len(pcb.builders))
	for i := range pcb.builders {
		func(i int, root context.Context) {
			builder := pcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PlanMutation)
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
func (pcb *PlanCreateBulk) SaveX(ctx context.Context) []*Plan {
	v, err := pcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pcb *PlanCreateBulk) Exec(ctx context.Context) error {
	_, err := pcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pcb *PlanCreateBulk) ExecX(ctx context.Context) {
	if err := pcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Plan.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlanUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (pcb *PlanCreateBulk) OnConflict(opts ...sql.ConflictOption) *PlanUpsertBulk {
	pcb.conflict = opts
	return &PlanUpsertBulk{
		create: pcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Plan.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (pcb *PlanCreateBulk) OnConflictColumns(columns ...string) *PlanUpsertBulk {
	pcb.conflict = append(pcb.conflict, sql.ConflictColumns(columns...))
	return &PlanUpsertBulk{
		create: pcb,
	}
}

// PlanUpsertBulk is the builder for "upsert"-ing
// a bulk of Plan nodes.
type PlanUpsertBulk struct {
	create *PlanCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Plan.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(plan.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlanUpsertBulk) UpdateNewValues() *PlanUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(plan.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(plan.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(plan.FieldCreatedAt)
			}
			if _, exists := b.mutation.Key(); exists {
				s.SetIgnore(plan.FieldKey)
			}
			if _, exists := b.mutation.Currency(); exists {
				s.SetIgnore(plan.FieldCurrency)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Plan.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *PlanUpsertBulk) Ignore() *PlanUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlanUpsertBulk) DoNothing() *PlanUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlanCreateBulk.OnConflict
// documentation for more info.
func (u *PlanUpsertBulk) Update(set func(*PlanUpsert)) *PlanUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlanUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *PlanUpsertBulk) SetMetadata(v map[string]string) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateMetadata() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanUpsertBulk) ClearMetadata() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.ClearMetadata()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanUpsertBulk) SetUpdatedAt(v time.Time) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateUpdatedAt() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanUpsertBulk) SetDeletedAt(v time.Time) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateDeletedAt() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanUpsertBulk) ClearDeletedAt() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.ClearDeletedAt()
	})
}

// SetName sets the "name" field.
func (u *PlanUpsertBulk) SetName(v string) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetName(v)
	})
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateName() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateName()
	})
}

// SetDescription sets the "description" field.
func (u *PlanUpsertBulk) SetDescription(v string) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetDescription(v)
	})
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateDescription() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateDescription()
	})
}

// ClearDescription clears the value of the "description" field.
func (u *PlanUpsertBulk) ClearDescription() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.ClearDescription()
	})
}

// SetVersion sets the "version" field.
func (u *PlanUpsertBulk) SetVersion(v int) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetVersion(v)
	})
}

// AddVersion adds v to the "version" field.
func (u *PlanUpsertBulk) AddVersion(v int) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.AddVersion(v)
	})
}

// UpdateVersion sets the "version" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateVersion() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateVersion()
	})
}

// SetEffectiveFrom sets the "effective_from" field.
func (u *PlanUpsertBulk) SetEffectiveFrom(v time.Time) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetEffectiveFrom(v)
	})
}

// UpdateEffectiveFrom sets the "effective_from" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateEffectiveFrom() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateEffectiveFrom()
	})
}

// ClearEffectiveFrom clears the value of the "effective_from" field.
func (u *PlanUpsertBulk) ClearEffectiveFrom() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.ClearEffectiveFrom()
	})
}

// SetEffectiveTo sets the "effective_to" field.
func (u *PlanUpsertBulk) SetEffectiveTo(v time.Time) *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.SetEffectiveTo(v)
	})
}

// UpdateEffectiveTo sets the "effective_to" field to the value that was provided on create.
func (u *PlanUpsertBulk) UpdateEffectiveTo() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.UpdateEffectiveTo()
	})
}

// ClearEffectiveTo clears the value of the "effective_to" field.
func (u *PlanUpsertBulk) ClearEffectiveTo() *PlanUpsertBulk {
	return u.Update(func(s *PlanUpsert) {
		s.ClearEffectiveTo()
	})
}

// Exec executes the query.
func (u *PlanUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the PlanCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PlanCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlanUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
