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
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
)

// PlanAddonCreate is the builder for creating a PlanAddon entity.
type PlanAddonCreate struct {
	config
	mutation *PlanAddonMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (_c *PlanAddonCreate) SetNamespace(v string) *PlanAddonCreate {
	_c.mutation.SetNamespace(v)
	return _c
}

// SetMetadata sets the "metadata" field.
func (_c *PlanAddonCreate) SetMetadata(v map[string]string) *PlanAddonCreate {
	_c.mutation.SetMetadata(v)
	return _c
}

// SetAnnotations sets the "annotations" field.
func (_c *PlanAddonCreate) SetAnnotations(v map[string]interface{}) *PlanAddonCreate {
	_c.mutation.SetAnnotations(v)
	return _c
}

// SetCreatedAt sets the "created_at" field.
func (_c *PlanAddonCreate) SetCreatedAt(v time.Time) *PlanAddonCreate {
	_c.mutation.SetCreatedAt(v)
	return _c
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (_c *PlanAddonCreate) SetNillableCreatedAt(v *time.Time) *PlanAddonCreate {
	if v != nil {
		_c.SetCreatedAt(*v)
	}
	return _c
}

// SetUpdatedAt sets the "updated_at" field.
func (_c *PlanAddonCreate) SetUpdatedAt(v time.Time) *PlanAddonCreate {
	_c.mutation.SetUpdatedAt(v)
	return _c
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (_c *PlanAddonCreate) SetNillableUpdatedAt(v *time.Time) *PlanAddonCreate {
	if v != nil {
		_c.SetUpdatedAt(*v)
	}
	return _c
}

// SetDeletedAt sets the "deleted_at" field.
func (_c *PlanAddonCreate) SetDeletedAt(v time.Time) *PlanAddonCreate {
	_c.mutation.SetDeletedAt(v)
	return _c
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_c *PlanAddonCreate) SetNillableDeletedAt(v *time.Time) *PlanAddonCreate {
	if v != nil {
		_c.SetDeletedAt(*v)
	}
	return _c
}

// SetPlanID sets the "plan_id" field.
func (_c *PlanAddonCreate) SetPlanID(v string) *PlanAddonCreate {
	_c.mutation.SetPlanID(v)
	return _c
}

// SetAddonID sets the "addon_id" field.
func (_c *PlanAddonCreate) SetAddonID(v string) *PlanAddonCreate {
	_c.mutation.SetAddonID(v)
	return _c
}

// SetFromPlanPhase sets the "from_plan_phase" field.
func (_c *PlanAddonCreate) SetFromPlanPhase(v string) *PlanAddonCreate {
	_c.mutation.SetFromPlanPhase(v)
	return _c
}

// SetMaxQuantity sets the "max_quantity" field.
func (_c *PlanAddonCreate) SetMaxQuantity(v int) *PlanAddonCreate {
	_c.mutation.SetMaxQuantity(v)
	return _c
}

// SetNillableMaxQuantity sets the "max_quantity" field if the given value is not nil.
func (_c *PlanAddonCreate) SetNillableMaxQuantity(v *int) *PlanAddonCreate {
	if v != nil {
		_c.SetMaxQuantity(*v)
	}
	return _c
}

// SetID sets the "id" field.
func (_c *PlanAddonCreate) SetID(v string) *PlanAddonCreate {
	_c.mutation.SetID(v)
	return _c
}

// SetNillableID sets the "id" field if the given value is not nil.
func (_c *PlanAddonCreate) SetNillableID(v *string) *PlanAddonCreate {
	if v != nil {
		_c.SetID(*v)
	}
	return _c
}

// SetPlan sets the "plan" edge to the Plan entity.
func (_c *PlanAddonCreate) SetPlan(v *Plan) *PlanAddonCreate {
	return _c.SetPlanID(v.ID)
}

// SetAddon sets the "addon" edge to the Addon entity.
func (_c *PlanAddonCreate) SetAddon(v *Addon) *PlanAddonCreate {
	return _c.SetAddonID(v.ID)
}

// Mutation returns the PlanAddonMutation object of the builder.
func (_c *PlanAddonCreate) Mutation() *PlanAddonMutation {
	return _c.mutation
}

// Save creates the PlanAddon in the database.
func (_c *PlanAddonCreate) Save(ctx context.Context) (*PlanAddon, error) {
	_c.defaults()
	return withHooks(ctx, _c.sqlSave, _c.mutation, _c.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (_c *PlanAddonCreate) SaveX(ctx context.Context) *PlanAddon {
	v, err := _c.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (_c *PlanAddonCreate) Exec(ctx context.Context) error {
	_, err := _c.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_c *PlanAddonCreate) ExecX(ctx context.Context) {
	if err := _c.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_c *PlanAddonCreate) defaults() {
	if _, ok := _c.mutation.CreatedAt(); !ok {
		v := planaddon.DefaultCreatedAt()
		_c.mutation.SetCreatedAt(v)
	}
	if _, ok := _c.mutation.UpdatedAt(); !ok {
		v := planaddon.DefaultUpdatedAt()
		_c.mutation.SetUpdatedAt(v)
	}
	if _, ok := _c.mutation.ID(); !ok {
		v := planaddon.DefaultID()
		_c.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_c *PlanAddonCreate) check() error {
	if _, ok := _c.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "PlanAddon.namespace"`)}
	}
	if v, ok := _c.mutation.Namespace(); ok {
		if err := planaddon.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "PlanAddon.namespace": %w`, err)}
		}
	}
	if _, ok := _c.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "PlanAddon.created_at"`)}
	}
	if _, ok := _c.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "PlanAddon.updated_at"`)}
	}
	if _, ok := _c.mutation.PlanID(); !ok {
		return &ValidationError{Name: "plan_id", err: errors.New(`db: missing required field "PlanAddon.plan_id"`)}
	}
	if v, ok := _c.mutation.PlanID(); ok {
		if err := planaddon.PlanIDValidator(v); err != nil {
			return &ValidationError{Name: "plan_id", err: fmt.Errorf(`db: validator failed for field "PlanAddon.plan_id": %w`, err)}
		}
	}
	if _, ok := _c.mutation.AddonID(); !ok {
		return &ValidationError{Name: "addon_id", err: errors.New(`db: missing required field "PlanAddon.addon_id"`)}
	}
	if v, ok := _c.mutation.AddonID(); ok {
		if err := planaddon.AddonIDValidator(v); err != nil {
			return &ValidationError{Name: "addon_id", err: fmt.Errorf(`db: validator failed for field "PlanAddon.addon_id": %w`, err)}
		}
	}
	if _, ok := _c.mutation.FromPlanPhase(); !ok {
		return &ValidationError{Name: "from_plan_phase", err: errors.New(`db: missing required field "PlanAddon.from_plan_phase"`)}
	}
	if len(_c.mutation.PlanIDs()) == 0 {
		return &ValidationError{Name: "plan", err: errors.New(`db: missing required edge "PlanAddon.plan"`)}
	}
	if len(_c.mutation.AddonIDs()) == 0 {
		return &ValidationError{Name: "addon", err: errors.New(`db: missing required edge "PlanAddon.addon"`)}
	}
	return nil
}

func (_c *PlanAddonCreate) sqlSave(ctx context.Context) (*PlanAddon, error) {
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
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected PlanAddon.ID type: %T", _spec.ID.Value)
		}
	}
	_c.mutation.id = &_node.ID
	_c.mutation.done = true
	return _node, nil
}

func (_c *PlanAddonCreate) createSpec() (*PlanAddon, *sqlgraph.CreateSpec) {
	var (
		_node = &PlanAddon{config: _c.config}
		_spec = sqlgraph.NewCreateSpec(planaddon.Table, sqlgraph.NewFieldSpec(planaddon.FieldID, field.TypeString))
	)
	_spec.OnConflict = _c.conflict
	if id, ok := _c.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := _c.mutation.Namespace(); ok {
		_spec.SetField(planaddon.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := _c.mutation.Metadata(); ok {
		_spec.SetField(planaddon.FieldMetadata, field.TypeJSON, value)
		_node.Metadata = value
	}
	if value, ok := _c.mutation.Annotations(); ok {
		_spec.SetField(planaddon.FieldAnnotations, field.TypeJSON, value)
		_node.Annotations = value
	}
	if value, ok := _c.mutation.CreatedAt(); ok {
		_spec.SetField(planaddon.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := _c.mutation.UpdatedAt(); ok {
		_spec.SetField(planaddon.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := _c.mutation.DeletedAt(); ok {
		_spec.SetField(planaddon.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := _c.mutation.FromPlanPhase(); ok {
		_spec.SetField(planaddon.FieldFromPlanPhase, field.TypeString, value)
		_node.FromPlanPhase = value
	}
	if value, ok := _c.mutation.MaxQuantity(); ok {
		_spec.SetField(planaddon.FieldMaxQuantity, field.TypeInt, value)
		_node.MaxQuantity = &value
	}
	if nodes := _c.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planaddon.PlanTable,
			Columns: []string{planaddon.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.PlanID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := _c.mutation.AddonIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planaddon.AddonTable,
			Columns: []string{planaddon.AddonColumn},
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
//	client.PlanAddon.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlanAddonUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (_c *PlanAddonCreate) OnConflict(opts ...sql.ConflictOption) *PlanAddonUpsertOne {
	_c.conflict = opts
	return &PlanAddonUpsertOne{
		create: _c,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (_c *PlanAddonCreate) OnConflictColumns(columns ...string) *PlanAddonUpsertOne {
	_c.conflict = append(_c.conflict, sql.ConflictColumns(columns...))
	return &PlanAddonUpsertOne{
		create: _c,
	}
}

type (
	// PlanAddonUpsertOne is the builder for "upsert"-ing
	//  one PlanAddon node.
	PlanAddonUpsertOne struct {
		create *PlanAddonCreate
	}

	// PlanAddonUpsert is the "OnConflict" setter.
	PlanAddonUpsert struct {
		*sql.UpdateSet
	}
)

// SetMetadata sets the "metadata" field.
func (u *PlanAddonUpsert) SetMetadata(v map[string]string) *PlanAddonUpsert {
	u.Set(planaddon.FieldMetadata, v)
	return u
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateMetadata() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldMetadata)
	return u
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanAddonUpsert) ClearMetadata() *PlanAddonUpsert {
	u.SetNull(planaddon.FieldMetadata)
	return u
}

// SetAnnotations sets the "annotations" field.
func (u *PlanAddonUpsert) SetAnnotations(v map[string]interface{}) *PlanAddonUpsert {
	u.Set(planaddon.FieldAnnotations, v)
	return u
}

// UpdateAnnotations sets the "annotations" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateAnnotations() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldAnnotations)
	return u
}

// ClearAnnotations clears the value of the "annotations" field.
func (u *PlanAddonUpsert) ClearAnnotations() *PlanAddonUpsert {
	u.SetNull(planaddon.FieldAnnotations)
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanAddonUpsert) SetUpdatedAt(v time.Time) *PlanAddonUpsert {
	u.Set(planaddon.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateUpdatedAt() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanAddonUpsert) SetDeletedAt(v time.Time) *PlanAddonUpsert {
	u.Set(planaddon.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateDeletedAt() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanAddonUpsert) ClearDeletedAt() *PlanAddonUpsert {
	u.SetNull(planaddon.FieldDeletedAt)
	return u
}

// SetFromPlanPhase sets the "from_plan_phase" field.
func (u *PlanAddonUpsert) SetFromPlanPhase(v string) *PlanAddonUpsert {
	u.Set(planaddon.FieldFromPlanPhase, v)
	return u
}

// UpdateFromPlanPhase sets the "from_plan_phase" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateFromPlanPhase() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldFromPlanPhase)
	return u
}

// SetMaxQuantity sets the "max_quantity" field.
func (u *PlanAddonUpsert) SetMaxQuantity(v int) *PlanAddonUpsert {
	u.Set(planaddon.FieldMaxQuantity, v)
	return u
}

// UpdateMaxQuantity sets the "max_quantity" field to the value that was provided on create.
func (u *PlanAddonUpsert) UpdateMaxQuantity() *PlanAddonUpsert {
	u.SetExcluded(planaddon.FieldMaxQuantity)
	return u
}

// AddMaxQuantity adds v to the "max_quantity" field.
func (u *PlanAddonUpsert) AddMaxQuantity(v int) *PlanAddonUpsert {
	u.Add(planaddon.FieldMaxQuantity, v)
	return u
}

// ClearMaxQuantity clears the value of the "max_quantity" field.
func (u *PlanAddonUpsert) ClearMaxQuantity() *PlanAddonUpsert {
	u.SetNull(planaddon.FieldMaxQuantity)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(planaddon.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlanAddonUpsertOne) UpdateNewValues() *PlanAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(planaddon.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(planaddon.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(planaddon.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.PlanID(); exists {
			s.SetIgnore(planaddon.FieldPlanID)
		}
		if _, exists := u.create.mutation.AddonID(); exists {
			s.SetIgnore(planaddon.FieldAddonID)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *PlanAddonUpsertOne) Ignore() *PlanAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlanAddonUpsertOne) DoNothing() *PlanAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlanAddonCreate.OnConflict
// documentation for more info.
func (u *PlanAddonUpsertOne) Update(set func(*PlanAddonUpsert)) *PlanAddonUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlanAddonUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *PlanAddonUpsertOne) SetMetadata(v map[string]string) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateMetadata() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanAddonUpsertOne) ClearMetadata() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearMetadata()
	})
}

// SetAnnotations sets the "annotations" field.
func (u *PlanAddonUpsertOne) SetAnnotations(v map[string]interface{}) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetAnnotations(v)
	})
}

// UpdateAnnotations sets the "annotations" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateAnnotations() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateAnnotations()
	})
}

// ClearAnnotations clears the value of the "annotations" field.
func (u *PlanAddonUpsertOne) ClearAnnotations() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearAnnotations()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanAddonUpsertOne) SetUpdatedAt(v time.Time) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateUpdatedAt() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanAddonUpsertOne) SetDeletedAt(v time.Time) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateDeletedAt() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanAddonUpsertOne) ClearDeletedAt() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearDeletedAt()
	})
}

// SetFromPlanPhase sets the "from_plan_phase" field.
func (u *PlanAddonUpsertOne) SetFromPlanPhase(v string) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetFromPlanPhase(v)
	})
}

// UpdateFromPlanPhase sets the "from_plan_phase" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateFromPlanPhase() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateFromPlanPhase()
	})
}

// SetMaxQuantity sets the "max_quantity" field.
func (u *PlanAddonUpsertOne) SetMaxQuantity(v int) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetMaxQuantity(v)
	})
}

// AddMaxQuantity adds v to the "max_quantity" field.
func (u *PlanAddonUpsertOne) AddMaxQuantity(v int) *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.AddMaxQuantity(v)
	})
}

// UpdateMaxQuantity sets the "max_quantity" field to the value that was provided on create.
func (u *PlanAddonUpsertOne) UpdateMaxQuantity() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateMaxQuantity()
	})
}

// ClearMaxQuantity clears the value of the "max_quantity" field.
func (u *PlanAddonUpsertOne) ClearMaxQuantity() *PlanAddonUpsertOne {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearMaxQuantity()
	})
}

// Exec executes the query.
func (u *PlanAddonUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PlanAddonCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlanAddonUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *PlanAddonUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: PlanAddonUpsertOne.ID is not supported by MySQL driver. Use PlanAddonUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *PlanAddonUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// PlanAddonCreateBulk is the builder for creating many PlanAddon entities in bulk.
type PlanAddonCreateBulk struct {
	config
	err      error
	builders []*PlanAddonCreate
	conflict []sql.ConflictOption
}

// Save creates the PlanAddon entities in the database.
func (_c *PlanAddonCreateBulk) Save(ctx context.Context) ([]*PlanAddon, error) {
	if _c.err != nil {
		return nil, _c.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(_c.builders))
	nodes := make([]*PlanAddon, len(_c.builders))
	mutators := make([]Mutator, len(_c.builders))
	for i := range _c.builders {
		func(i int, root context.Context) {
			builder := _c.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PlanAddonMutation)
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
func (_c *PlanAddonCreateBulk) SaveX(ctx context.Context) []*PlanAddon {
	v, err := _c.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (_c *PlanAddonCreateBulk) Exec(ctx context.Context) error {
	_, err := _c.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_c *PlanAddonCreateBulk) ExecX(ctx context.Context) {
	if err := _c.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.PlanAddon.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlanAddonUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (_c *PlanAddonCreateBulk) OnConflict(opts ...sql.ConflictOption) *PlanAddonUpsertBulk {
	_c.conflict = opts
	return &PlanAddonUpsertBulk{
		create: _c,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (_c *PlanAddonCreateBulk) OnConflictColumns(columns ...string) *PlanAddonUpsertBulk {
	_c.conflict = append(_c.conflict, sql.ConflictColumns(columns...))
	return &PlanAddonUpsertBulk{
		create: _c,
	}
}

// PlanAddonUpsertBulk is the builder for "upsert"-ing
// a bulk of PlanAddon nodes.
type PlanAddonUpsertBulk struct {
	create *PlanAddonCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(planaddon.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlanAddonUpsertBulk) UpdateNewValues() *PlanAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(planaddon.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(planaddon.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(planaddon.FieldCreatedAt)
			}
			if _, exists := b.mutation.PlanID(); exists {
				s.SetIgnore(planaddon.FieldPlanID)
			}
			if _, exists := b.mutation.AddonID(); exists {
				s.SetIgnore(planaddon.FieldAddonID)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.PlanAddon.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *PlanAddonUpsertBulk) Ignore() *PlanAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlanAddonUpsertBulk) DoNothing() *PlanAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlanAddonCreateBulk.OnConflict
// documentation for more info.
func (u *PlanAddonUpsertBulk) Update(set func(*PlanAddonUpsert)) *PlanAddonUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlanAddonUpsert{UpdateSet: update})
	}))
	return u
}

// SetMetadata sets the "metadata" field.
func (u *PlanAddonUpsertBulk) SetMetadata(v map[string]string) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetMetadata(v)
	})
}

// UpdateMetadata sets the "metadata" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateMetadata() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateMetadata()
	})
}

// ClearMetadata clears the value of the "metadata" field.
func (u *PlanAddonUpsertBulk) ClearMetadata() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearMetadata()
	})
}

// SetAnnotations sets the "annotations" field.
func (u *PlanAddonUpsertBulk) SetAnnotations(v map[string]interface{}) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetAnnotations(v)
	})
}

// UpdateAnnotations sets the "annotations" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateAnnotations() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateAnnotations()
	})
}

// ClearAnnotations clears the value of the "annotations" field.
func (u *PlanAddonUpsertBulk) ClearAnnotations() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearAnnotations()
	})
}

// SetUpdatedAt sets the "updated_at" field.
func (u *PlanAddonUpsertBulk) SetUpdatedAt(v time.Time) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateUpdatedAt() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *PlanAddonUpsertBulk) SetDeletedAt(v time.Time) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateDeletedAt() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *PlanAddonUpsertBulk) ClearDeletedAt() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearDeletedAt()
	})
}

// SetFromPlanPhase sets the "from_plan_phase" field.
func (u *PlanAddonUpsertBulk) SetFromPlanPhase(v string) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetFromPlanPhase(v)
	})
}

// UpdateFromPlanPhase sets the "from_plan_phase" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateFromPlanPhase() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateFromPlanPhase()
	})
}

// SetMaxQuantity sets the "max_quantity" field.
func (u *PlanAddonUpsertBulk) SetMaxQuantity(v int) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.SetMaxQuantity(v)
	})
}

// AddMaxQuantity adds v to the "max_quantity" field.
func (u *PlanAddonUpsertBulk) AddMaxQuantity(v int) *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.AddMaxQuantity(v)
	})
}

// UpdateMaxQuantity sets the "max_quantity" field to the value that was provided on create.
func (u *PlanAddonUpsertBulk) UpdateMaxQuantity() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.UpdateMaxQuantity()
	})
}

// ClearMaxQuantity clears the value of the "max_quantity" field.
func (u *PlanAddonUpsertBulk) ClearMaxQuantity() *PlanAddonUpsertBulk {
	return u.Update(func(s *PlanAddonUpsert) {
		s.ClearMaxQuantity()
	})
}

// Exec executes the query.
func (u *PlanAddonUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the PlanAddonCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for PlanAddonCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlanAddonUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
