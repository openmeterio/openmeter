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
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// BillingProfileCreate is the builder for creating a BillingProfile entity.
type BillingProfileCreate struct {
	config
	mutation *BillingProfileMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (bpc *BillingProfileCreate) SetNamespace(s string) *BillingProfileCreate {
	bpc.mutation.SetNamespace(s)
	return bpc
}

// SetCreatedAt sets the "created_at" field.
func (bpc *BillingProfileCreate) SetCreatedAt(t time.Time) *BillingProfileCreate {
	bpc.mutation.SetCreatedAt(t)
	return bpc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (bpc *BillingProfileCreate) SetNillableCreatedAt(t *time.Time) *BillingProfileCreate {
	if t != nil {
		bpc.SetCreatedAt(*t)
	}
	return bpc
}

// SetUpdatedAt sets the "updated_at" field.
func (bpc *BillingProfileCreate) SetUpdatedAt(t time.Time) *BillingProfileCreate {
	bpc.mutation.SetUpdatedAt(t)
	return bpc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (bpc *BillingProfileCreate) SetNillableUpdatedAt(t *time.Time) *BillingProfileCreate {
	if t != nil {
		bpc.SetUpdatedAt(*t)
	}
	return bpc
}

// SetDeletedAt sets the "deleted_at" field.
func (bpc *BillingProfileCreate) SetDeletedAt(t time.Time) *BillingProfileCreate {
	bpc.mutation.SetDeletedAt(t)
	return bpc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bpc *BillingProfileCreate) SetNillableDeletedAt(t *time.Time) *BillingProfileCreate {
	if t != nil {
		bpc.SetDeletedAt(*t)
	}
	return bpc
}

// SetKey sets the "key" field.
func (bpc *BillingProfileCreate) SetKey(s string) *BillingProfileCreate {
	bpc.mutation.SetKey(s)
	return bpc
}

// SetProviderConfig sets the "provider_config" field.
func (bpc *BillingProfileCreate) SetProviderConfig(pr provider.Configuration) *BillingProfileCreate {
	bpc.mutation.SetProviderConfig(pr)
	return bpc
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (bpc *BillingProfileCreate) SetWorkflowConfigID(s string) *BillingProfileCreate {
	bpc.mutation.SetWorkflowConfigID(s)
	return bpc
}

// SetTimezone sets the "timezone" field.
func (bpc *BillingProfileCreate) SetTimezone(t timezone.Timezone) *BillingProfileCreate {
	bpc.mutation.SetTimezone(t)
	return bpc
}

// SetDefault sets the "default" field.
func (bpc *BillingProfileCreate) SetDefault(b bool) *BillingProfileCreate {
	bpc.mutation.SetDefault(b)
	return bpc
}

// SetNillableDefault sets the "default" field if the given value is not nil.
func (bpc *BillingProfileCreate) SetNillableDefault(b *bool) *BillingProfileCreate {
	if b != nil {
		bpc.SetDefault(*b)
	}
	return bpc
}

// SetID sets the "id" field.
func (bpc *BillingProfileCreate) SetID(s string) *BillingProfileCreate {
	bpc.mutation.SetID(s)
	return bpc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (bpc *BillingProfileCreate) SetNillableID(s *string) *BillingProfileCreate {
	if s != nil {
		bpc.SetID(*s)
	}
	return bpc
}

// AddBillingInvoiceIDs adds the "billing_invoices" edge to the BillingInvoice entity by IDs.
func (bpc *BillingProfileCreate) AddBillingInvoiceIDs(ids ...string) *BillingProfileCreate {
	bpc.mutation.AddBillingInvoiceIDs(ids...)
	return bpc
}

// AddBillingInvoices adds the "billing_invoices" edges to the BillingInvoice entity.
func (bpc *BillingProfileCreate) AddBillingInvoices(b ...*BillingInvoice) *BillingProfileCreate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bpc.AddBillingInvoiceIDs(ids...)
}

// SetBillingWorkflowConfigID sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity by ID.
func (bpc *BillingProfileCreate) SetBillingWorkflowConfigID(id string) *BillingProfileCreate {
	bpc.mutation.SetBillingWorkflowConfigID(id)
	return bpc
}

// SetBillingWorkflowConfig sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity.
func (bpc *BillingProfileCreate) SetBillingWorkflowConfig(b *BillingWorkflowConfig) *BillingProfileCreate {
	return bpc.SetBillingWorkflowConfigID(b.ID)
}

// Mutation returns the BillingProfileMutation object of the builder.
func (bpc *BillingProfileCreate) Mutation() *BillingProfileMutation {
	return bpc.mutation
}

// Save creates the BillingProfile in the database.
func (bpc *BillingProfileCreate) Save(ctx context.Context) (*BillingProfile, error) {
	bpc.defaults()
	return withHooks(ctx, bpc.sqlSave, bpc.mutation, bpc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (bpc *BillingProfileCreate) SaveX(ctx context.Context) *BillingProfile {
	v, err := bpc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bpc *BillingProfileCreate) Exec(ctx context.Context) error {
	_, err := bpc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpc *BillingProfileCreate) ExecX(ctx context.Context) {
	if err := bpc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bpc *BillingProfileCreate) defaults() {
	if _, ok := bpc.mutation.CreatedAt(); !ok {
		v := billingprofile.DefaultCreatedAt()
		bpc.mutation.SetCreatedAt(v)
	}
	if _, ok := bpc.mutation.UpdatedAt(); !ok {
		v := billingprofile.DefaultUpdatedAt()
		bpc.mutation.SetUpdatedAt(v)
	}
	if _, ok := bpc.mutation.Default(); !ok {
		v := billingprofile.DefaultDefault
		bpc.mutation.SetDefault(v)
	}
	if _, ok := bpc.mutation.ID(); !ok {
		v := billingprofile.DefaultID()
		bpc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bpc *BillingProfileCreate) check() error {
	if _, ok := bpc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "BillingProfile.namespace"`)}
	}
	if v, ok := bpc.mutation.Namespace(); ok {
		if err := billingprofile.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "BillingProfile.namespace": %w`, err)}
		}
	}
	if _, ok := bpc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "BillingProfile.created_at"`)}
	}
	if _, ok := bpc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "BillingProfile.updated_at"`)}
	}
	if _, ok := bpc.mutation.Key(); !ok {
		return &ValidationError{Name: "key", err: errors.New(`db: missing required field "BillingProfile.key"`)}
	}
	if v, ok := bpc.mutation.Key(); ok {
		if err := billingprofile.KeyValidator(v); err != nil {
			return &ValidationError{Name: "key", err: fmt.Errorf(`db: validator failed for field "BillingProfile.key": %w`, err)}
		}
	}
	if _, ok := bpc.mutation.ProviderConfig(); !ok {
		return &ValidationError{Name: "provider_config", err: errors.New(`db: missing required field "BillingProfile.provider_config"`)}
	}
	if v, ok := bpc.mutation.ProviderConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "provider_config", err: fmt.Errorf(`db: validator failed for field "BillingProfile.provider_config": %w`, err)}
		}
	}
	if _, ok := bpc.mutation.WorkflowConfigID(); !ok {
		return &ValidationError{Name: "workflow_config_id", err: errors.New(`db: missing required field "BillingProfile.workflow_config_id"`)}
	}
	if v, ok := bpc.mutation.WorkflowConfigID(); ok {
		if err := billingprofile.WorkflowConfigIDValidator(v); err != nil {
			return &ValidationError{Name: "workflow_config_id", err: fmt.Errorf(`db: validator failed for field "BillingProfile.workflow_config_id": %w`, err)}
		}
	}
	if _, ok := bpc.mutation.Timezone(); !ok {
		return &ValidationError{Name: "timezone", err: errors.New(`db: missing required field "BillingProfile.timezone"`)}
	}
	if _, ok := bpc.mutation.Default(); !ok {
		return &ValidationError{Name: "default", err: errors.New(`db: missing required field "BillingProfile.default"`)}
	}
	if len(bpc.mutation.BillingWorkflowConfigIDs()) == 0 {
		return &ValidationError{Name: "billing_workflow_config", err: errors.New(`db: missing required edge "BillingProfile.billing_workflow_config"`)}
	}
	return nil
}

func (bpc *BillingProfileCreate) sqlSave(ctx context.Context) (*BillingProfile, error) {
	if err := bpc.check(); err != nil {
		return nil, err
	}
	_node, _spec, err := bpc.createSpec()
	if err != nil {
		return nil, err
	}
	if err := sqlgraph.CreateNode(ctx, bpc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected BillingProfile.ID type: %T", _spec.ID.Value)
		}
	}
	bpc.mutation.id = &_node.ID
	bpc.mutation.done = true
	return _node, nil
}

func (bpc *BillingProfileCreate) createSpec() (*BillingProfile, *sqlgraph.CreateSpec, error) {
	var (
		_node = &BillingProfile{config: bpc.config}
		_spec = sqlgraph.NewCreateSpec(billingprofile.Table, sqlgraph.NewFieldSpec(billingprofile.FieldID, field.TypeString))
	)
	_spec.OnConflict = bpc.conflict
	if id, ok := bpc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := bpc.mutation.Namespace(); ok {
		_spec.SetField(billingprofile.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := bpc.mutation.CreatedAt(); ok {
		_spec.SetField(billingprofile.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := bpc.mutation.UpdatedAt(); ok {
		_spec.SetField(billingprofile.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := bpc.mutation.DeletedAt(); ok {
		_spec.SetField(billingprofile.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := bpc.mutation.Key(); ok {
		_spec.SetField(billingprofile.FieldKey, field.TypeString, value)
		_node.Key = value
	}
	if value, ok := bpc.mutation.ProviderConfig(); ok {
		vv, err := billingprofile.ValueScanner.ProviderConfig.Value(value)
		if err != nil {
			return nil, nil, err
		}
		_spec.SetField(billingprofile.FieldProviderConfig, field.TypeString, vv)
		_node.ProviderConfig = value
	}
	if value, ok := bpc.mutation.Timezone(); ok {
		_spec.SetField(billingprofile.FieldTimezone, field.TypeString, value)
		_node.Timezone = value
	}
	if value, ok := bpc.mutation.Default(); ok {
		_spec.SetField(billingprofile.FieldDefault, field.TypeBool, value)
		_node.Default = value
	}
	if nodes := bpc.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   billingprofile.BillingInvoicesTable,
			Columns: []string{billingprofile.BillingInvoicesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoice.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bpc.mutation.BillingWorkflowConfigIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billingprofile.BillingWorkflowConfigTable,
			Columns: []string{billingprofile.BillingWorkflowConfigColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billingworkflowconfig.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.WorkflowConfigID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec, nil
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.BillingProfile.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.BillingProfileUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (bpc *BillingProfileCreate) OnConflict(opts ...sql.ConflictOption) *BillingProfileUpsertOne {
	bpc.conflict = opts
	return &BillingProfileUpsertOne{
		create: bpc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (bpc *BillingProfileCreate) OnConflictColumns(columns ...string) *BillingProfileUpsertOne {
	bpc.conflict = append(bpc.conflict, sql.ConflictColumns(columns...))
	return &BillingProfileUpsertOne{
		create: bpc,
	}
}

type (
	// BillingProfileUpsertOne is the builder for "upsert"-ing
	//  one BillingProfile node.
	BillingProfileUpsertOne struct {
		create *BillingProfileCreate
	}

	// BillingProfileUpsert is the "OnConflict" setter.
	BillingProfileUpsert struct {
		*sql.UpdateSet
	}
)

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingProfileUpsert) SetUpdatedAt(v time.Time) *BillingProfileUpsert {
	u.Set(billingprofile.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateUpdatedAt() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingProfileUpsert) SetDeletedAt(v time.Time) *BillingProfileUpsert {
	u.Set(billingprofile.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateDeletedAt() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingProfileUpsert) ClearDeletedAt() *BillingProfileUpsert {
	u.SetNull(billingprofile.FieldDeletedAt)
	return u
}

// SetProviderConfig sets the "provider_config" field.
func (u *BillingProfileUpsert) SetProviderConfig(v provider.Configuration) *BillingProfileUpsert {
	u.Set(billingprofile.FieldProviderConfig, v)
	return u
}

// UpdateProviderConfig sets the "provider_config" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateProviderConfig() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldProviderConfig)
	return u
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (u *BillingProfileUpsert) SetWorkflowConfigID(v string) *BillingProfileUpsert {
	u.Set(billingprofile.FieldWorkflowConfigID, v)
	return u
}

// UpdateWorkflowConfigID sets the "workflow_config_id" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateWorkflowConfigID() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldWorkflowConfigID)
	return u
}

// SetTimezone sets the "timezone" field.
func (u *BillingProfileUpsert) SetTimezone(v timezone.Timezone) *BillingProfileUpsert {
	u.Set(billingprofile.FieldTimezone, v)
	return u
}

// UpdateTimezone sets the "timezone" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateTimezone() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldTimezone)
	return u
}

// SetDefault sets the "default" field.
func (u *BillingProfileUpsert) SetDefault(v bool) *BillingProfileUpsert {
	u.Set(billingprofile.FieldDefault, v)
	return u
}

// UpdateDefault sets the "default" field to the value that was provided on create.
func (u *BillingProfileUpsert) UpdateDefault() *BillingProfileUpsert {
	u.SetExcluded(billingprofile.FieldDefault)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(billingprofile.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *BillingProfileUpsertOne) UpdateNewValues() *BillingProfileUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(billingprofile.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(billingprofile.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(billingprofile.FieldCreatedAt)
		}
		if _, exists := u.create.mutation.Key(); exists {
			s.SetIgnore(billingprofile.FieldKey)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *BillingProfileUpsertOne) Ignore() *BillingProfileUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *BillingProfileUpsertOne) DoNothing() *BillingProfileUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the BillingProfileCreate.OnConflict
// documentation for more info.
func (u *BillingProfileUpsertOne) Update(set func(*BillingProfileUpsert)) *BillingProfileUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&BillingProfileUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingProfileUpsertOne) SetUpdatedAt(v time.Time) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateUpdatedAt() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingProfileUpsertOne) SetDeletedAt(v time.Time) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateDeletedAt() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingProfileUpsertOne) ClearDeletedAt() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.ClearDeletedAt()
	})
}

// SetProviderConfig sets the "provider_config" field.
func (u *BillingProfileUpsertOne) SetProviderConfig(v provider.Configuration) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetProviderConfig(v)
	})
}

// UpdateProviderConfig sets the "provider_config" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateProviderConfig() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateProviderConfig()
	})
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (u *BillingProfileUpsertOne) SetWorkflowConfigID(v string) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetWorkflowConfigID(v)
	})
}

// UpdateWorkflowConfigID sets the "workflow_config_id" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateWorkflowConfigID() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateWorkflowConfigID()
	})
}

// SetTimezone sets the "timezone" field.
func (u *BillingProfileUpsertOne) SetTimezone(v timezone.Timezone) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetTimezone(v)
	})
}

// UpdateTimezone sets the "timezone" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateTimezone() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateTimezone()
	})
}

// SetDefault sets the "default" field.
func (u *BillingProfileUpsertOne) SetDefault(v bool) *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetDefault(v)
	})
}

// UpdateDefault sets the "default" field to the value that was provided on create.
func (u *BillingProfileUpsertOne) UpdateDefault() *BillingProfileUpsertOne {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateDefault()
	})
}

// Exec executes the query.
func (u *BillingProfileUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for BillingProfileCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *BillingProfileUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *BillingProfileUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: BillingProfileUpsertOne.ID is not supported by MySQL driver. Use BillingProfileUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *BillingProfileUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// BillingProfileCreateBulk is the builder for creating many BillingProfile entities in bulk.
type BillingProfileCreateBulk struct {
	config
	err      error
	builders []*BillingProfileCreate
	conflict []sql.ConflictOption
}

// Save creates the BillingProfile entities in the database.
func (bpcb *BillingProfileCreateBulk) Save(ctx context.Context) ([]*BillingProfile, error) {
	if bpcb.err != nil {
		return nil, bpcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(bpcb.builders))
	nodes := make([]*BillingProfile, len(bpcb.builders))
	mutators := make([]Mutator, len(bpcb.builders))
	for i := range bpcb.builders {
		func(i int, root context.Context) {
			builder := bpcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*BillingProfileMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i], err = builder.createSpec()
				if err != nil {
					return nil, err
				}
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, bpcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = bpcb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, bpcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, bpcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (bpcb *BillingProfileCreateBulk) SaveX(ctx context.Context) []*BillingProfile {
	v, err := bpcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bpcb *BillingProfileCreateBulk) Exec(ctx context.Context) error {
	_, err := bpcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpcb *BillingProfileCreateBulk) ExecX(ctx context.Context) {
	if err := bpcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.BillingProfile.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.BillingProfileUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (bpcb *BillingProfileCreateBulk) OnConflict(opts ...sql.ConflictOption) *BillingProfileUpsertBulk {
	bpcb.conflict = opts
	return &BillingProfileUpsertBulk{
		create: bpcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (bpcb *BillingProfileCreateBulk) OnConflictColumns(columns ...string) *BillingProfileUpsertBulk {
	bpcb.conflict = append(bpcb.conflict, sql.ConflictColumns(columns...))
	return &BillingProfileUpsertBulk{
		create: bpcb,
	}
}

// BillingProfileUpsertBulk is the builder for "upsert"-ing
// a bulk of BillingProfile nodes.
type BillingProfileUpsertBulk struct {
	create *BillingProfileCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(billingprofile.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *BillingProfileUpsertBulk) UpdateNewValues() *BillingProfileUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(billingprofile.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(billingprofile.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(billingprofile.FieldCreatedAt)
			}
			if _, exists := b.mutation.Key(); exists {
				s.SetIgnore(billingprofile.FieldKey)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.BillingProfile.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *BillingProfileUpsertBulk) Ignore() *BillingProfileUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *BillingProfileUpsertBulk) DoNothing() *BillingProfileUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the BillingProfileCreateBulk.OnConflict
// documentation for more info.
func (u *BillingProfileUpsertBulk) Update(set func(*BillingProfileUpsert)) *BillingProfileUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&BillingProfileUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingProfileUpsertBulk) SetUpdatedAt(v time.Time) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateUpdatedAt() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingProfileUpsertBulk) SetDeletedAt(v time.Time) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateDeletedAt() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingProfileUpsertBulk) ClearDeletedAt() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.ClearDeletedAt()
	})
}

// SetProviderConfig sets the "provider_config" field.
func (u *BillingProfileUpsertBulk) SetProviderConfig(v provider.Configuration) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetProviderConfig(v)
	})
}

// UpdateProviderConfig sets the "provider_config" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateProviderConfig() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateProviderConfig()
	})
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (u *BillingProfileUpsertBulk) SetWorkflowConfigID(v string) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetWorkflowConfigID(v)
	})
}

// UpdateWorkflowConfigID sets the "workflow_config_id" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateWorkflowConfigID() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateWorkflowConfigID()
	})
}

// SetTimezone sets the "timezone" field.
func (u *BillingProfileUpsertBulk) SetTimezone(v timezone.Timezone) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetTimezone(v)
	})
}

// UpdateTimezone sets the "timezone" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateTimezone() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateTimezone()
	})
}

// SetDefault sets the "default" field.
func (u *BillingProfileUpsertBulk) SetDefault(v bool) *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.SetDefault(v)
	})
}

// UpdateDefault sets the "default" field to the value that was provided on create.
func (u *BillingProfileUpsertBulk) UpdateDefault() *BillingProfileUpsertBulk {
	return u.Update(func(s *BillingProfileUpsert) {
		s.UpdateDefault()
	})
}

// Exec executes the query.
func (u *BillingProfileUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the BillingProfileCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for BillingProfileCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *BillingProfileUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
