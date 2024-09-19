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
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// BillingProfileUpdate is the builder for updating BillingProfile entities.
type BillingProfileUpdate struct {
	config
	hooks    []Hook
	mutation *BillingProfileMutation
}

// Where appends a list predicates to the BillingProfileUpdate builder.
func (bpu *BillingProfileUpdate) Where(ps ...predicate.BillingProfile) *BillingProfileUpdate {
	bpu.mutation.Where(ps...)
	return bpu
}

// SetUpdatedAt sets the "updated_at" field.
func (bpu *BillingProfileUpdate) SetUpdatedAt(t time.Time) *BillingProfileUpdate {
	bpu.mutation.SetUpdatedAt(t)
	return bpu
}

// SetDeletedAt sets the "deleted_at" field.
func (bpu *BillingProfileUpdate) SetDeletedAt(t time.Time) *BillingProfileUpdate {
	bpu.mutation.SetDeletedAt(t)
	return bpu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bpu *BillingProfileUpdate) SetNillableDeletedAt(t *time.Time) *BillingProfileUpdate {
	if t != nil {
		bpu.SetDeletedAt(*t)
	}
	return bpu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bpu *BillingProfileUpdate) ClearDeletedAt() *BillingProfileUpdate {
	bpu.mutation.ClearDeletedAt()
	return bpu
}

// SetProviderConfig sets the "provider_config" field.
func (bpu *BillingProfileUpdate) SetProviderConfig(pr provider.Configuration) *BillingProfileUpdate {
	bpu.mutation.SetProviderConfig(pr)
	return bpu
}

// SetNillableProviderConfig sets the "provider_config" field if the given value is not nil.
func (bpu *BillingProfileUpdate) SetNillableProviderConfig(pr *provider.Configuration) *BillingProfileUpdate {
	if pr != nil {
		bpu.SetProviderConfig(*pr)
	}
	return bpu
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (bpu *BillingProfileUpdate) SetWorkflowConfigID(s string) *BillingProfileUpdate {
	bpu.mutation.SetWorkflowConfigID(s)
	return bpu
}

// SetNillableWorkflowConfigID sets the "workflow_config_id" field if the given value is not nil.
func (bpu *BillingProfileUpdate) SetNillableWorkflowConfigID(s *string) *BillingProfileUpdate {
	if s != nil {
		bpu.SetWorkflowConfigID(*s)
	}
	return bpu
}

// SetTimezone sets the "timezone" field.
func (bpu *BillingProfileUpdate) SetTimezone(t timezone.Timezone) *BillingProfileUpdate {
	bpu.mutation.SetTimezone(t)
	return bpu
}

// SetNillableTimezone sets the "timezone" field if the given value is not nil.
func (bpu *BillingProfileUpdate) SetNillableTimezone(t *timezone.Timezone) *BillingProfileUpdate {
	if t != nil {
		bpu.SetTimezone(*t)
	}
	return bpu
}

// SetDefault sets the "default" field.
func (bpu *BillingProfileUpdate) SetDefault(b bool) *BillingProfileUpdate {
	bpu.mutation.SetDefault(b)
	return bpu
}

// SetNillableDefault sets the "default" field if the given value is not nil.
func (bpu *BillingProfileUpdate) SetNillableDefault(b *bool) *BillingProfileUpdate {
	if b != nil {
		bpu.SetDefault(*b)
	}
	return bpu
}

// AddBillingInvoiceIDs adds the "billing_invoices" edge to the BillingInvoice entity by IDs.
func (bpu *BillingProfileUpdate) AddBillingInvoiceIDs(ids ...string) *BillingProfileUpdate {
	bpu.mutation.AddBillingInvoiceIDs(ids...)
	return bpu
}

// AddBillingInvoices adds the "billing_invoices" edges to the BillingInvoice entity.
func (bpu *BillingProfileUpdate) AddBillingInvoices(b ...*BillingInvoice) *BillingProfileUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bpu.AddBillingInvoiceIDs(ids...)
}

// SetBillingWorkflowConfigID sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity by ID.
func (bpu *BillingProfileUpdate) SetBillingWorkflowConfigID(id string) *BillingProfileUpdate {
	bpu.mutation.SetBillingWorkflowConfigID(id)
	return bpu
}

// SetBillingWorkflowConfig sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity.
func (bpu *BillingProfileUpdate) SetBillingWorkflowConfig(b *BillingWorkflowConfig) *BillingProfileUpdate {
	return bpu.SetBillingWorkflowConfigID(b.ID)
}

// Mutation returns the BillingProfileMutation object of the builder.
func (bpu *BillingProfileUpdate) Mutation() *BillingProfileMutation {
	return bpu.mutation
}

// ClearBillingInvoices clears all "billing_invoices" edges to the BillingInvoice entity.
func (bpu *BillingProfileUpdate) ClearBillingInvoices() *BillingProfileUpdate {
	bpu.mutation.ClearBillingInvoices()
	return bpu
}

// RemoveBillingInvoiceIDs removes the "billing_invoices" edge to BillingInvoice entities by IDs.
func (bpu *BillingProfileUpdate) RemoveBillingInvoiceIDs(ids ...string) *BillingProfileUpdate {
	bpu.mutation.RemoveBillingInvoiceIDs(ids...)
	return bpu
}

// RemoveBillingInvoices removes "billing_invoices" edges to BillingInvoice entities.
func (bpu *BillingProfileUpdate) RemoveBillingInvoices(b ...*BillingInvoice) *BillingProfileUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bpu.RemoveBillingInvoiceIDs(ids...)
}

// ClearBillingWorkflowConfig clears the "billing_workflow_config" edge to the BillingWorkflowConfig entity.
func (bpu *BillingProfileUpdate) ClearBillingWorkflowConfig() *BillingProfileUpdate {
	bpu.mutation.ClearBillingWorkflowConfig()
	return bpu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (bpu *BillingProfileUpdate) Save(ctx context.Context) (int, error) {
	bpu.defaults()
	return withHooks(ctx, bpu.sqlSave, bpu.mutation, bpu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bpu *BillingProfileUpdate) SaveX(ctx context.Context) int {
	affected, err := bpu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (bpu *BillingProfileUpdate) Exec(ctx context.Context) error {
	_, err := bpu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpu *BillingProfileUpdate) ExecX(ctx context.Context) {
	if err := bpu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bpu *BillingProfileUpdate) defaults() {
	if _, ok := bpu.mutation.UpdatedAt(); !ok {
		v := billingprofile.UpdateDefaultUpdatedAt()
		bpu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bpu *BillingProfileUpdate) check() error {
	if v, ok := bpu.mutation.ProviderConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "provider_config", err: fmt.Errorf(`db: validator failed for field "BillingProfile.provider_config": %w`, err)}
		}
	}
	if v, ok := bpu.mutation.WorkflowConfigID(); ok {
		if err := billingprofile.WorkflowConfigIDValidator(v); err != nil {
			return &ValidationError{Name: "workflow_config_id", err: fmt.Errorf(`db: validator failed for field "BillingProfile.workflow_config_id": %w`, err)}
		}
	}
	if bpu.mutation.BillingWorkflowConfigCleared() && len(bpu.mutation.BillingWorkflowConfigIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingProfile.billing_workflow_config"`)
	}
	return nil
}

func (bpu *BillingProfileUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := bpu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(billingprofile.Table, billingprofile.Columns, sqlgraph.NewFieldSpec(billingprofile.FieldID, field.TypeString))
	if ps := bpu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bpu.mutation.UpdatedAt(); ok {
		_spec.SetField(billingprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bpu.mutation.DeletedAt(); ok {
		_spec.SetField(billingprofile.FieldDeletedAt, field.TypeTime, value)
	}
	if bpu.mutation.DeletedAtCleared() {
		_spec.ClearField(billingprofile.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := bpu.mutation.ProviderConfig(); ok {
		vv, err := billingprofile.ValueScanner.ProviderConfig.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(billingprofile.FieldProviderConfig, field.TypeString, vv)
	}
	if value, ok := bpu.mutation.Timezone(); ok {
		_spec.SetField(billingprofile.FieldTimezone, field.TypeString, value)
	}
	if value, ok := bpu.mutation.Default(); ok {
		_spec.SetField(billingprofile.FieldDefault, field.TypeBool, value)
	}
	if bpu.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpu.mutation.RemovedBillingInvoicesIDs(); len(nodes) > 0 && !bpu.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpu.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if bpu.mutation.BillingWorkflowConfigCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpu.mutation.BillingWorkflowConfigIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, bpu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billingprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	bpu.mutation.done = true
	return n, nil
}

// BillingProfileUpdateOne is the builder for updating a single BillingProfile entity.
type BillingProfileUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingProfileMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (bpuo *BillingProfileUpdateOne) SetUpdatedAt(t time.Time) *BillingProfileUpdateOne {
	bpuo.mutation.SetUpdatedAt(t)
	return bpuo
}

// SetDeletedAt sets the "deleted_at" field.
func (bpuo *BillingProfileUpdateOne) SetDeletedAt(t time.Time) *BillingProfileUpdateOne {
	bpuo.mutation.SetDeletedAt(t)
	return bpuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bpuo *BillingProfileUpdateOne) SetNillableDeletedAt(t *time.Time) *BillingProfileUpdateOne {
	if t != nil {
		bpuo.SetDeletedAt(*t)
	}
	return bpuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bpuo *BillingProfileUpdateOne) ClearDeletedAt() *BillingProfileUpdateOne {
	bpuo.mutation.ClearDeletedAt()
	return bpuo
}

// SetProviderConfig sets the "provider_config" field.
func (bpuo *BillingProfileUpdateOne) SetProviderConfig(pr provider.Configuration) *BillingProfileUpdateOne {
	bpuo.mutation.SetProviderConfig(pr)
	return bpuo
}

// SetNillableProviderConfig sets the "provider_config" field if the given value is not nil.
func (bpuo *BillingProfileUpdateOne) SetNillableProviderConfig(pr *provider.Configuration) *BillingProfileUpdateOne {
	if pr != nil {
		bpuo.SetProviderConfig(*pr)
	}
	return bpuo
}

// SetWorkflowConfigID sets the "workflow_config_id" field.
func (bpuo *BillingProfileUpdateOne) SetWorkflowConfigID(s string) *BillingProfileUpdateOne {
	bpuo.mutation.SetWorkflowConfigID(s)
	return bpuo
}

// SetNillableWorkflowConfigID sets the "workflow_config_id" field if the given value is not nil.
func (bpuo *BillingProfileUpdateOne) SetNillableWorkflowConfigID(s *string) *BillingProfileUpdateOne {
	if s != nil {
		bpuo.SetWorkflowConfigID(*s)
	}
	return bpuo
}

// SetTimezone sets the "timezone" field.
func (bpuo *BillingProfileUpdateOne) SetTimezone(t timezone.Timezone) *BillingProfileUpdateOne {
	bpuo.mutation.SetTimezone(t)
	return bpuo
}

// SetNillableTimezone sets the "timezone" field if the given value is not nil.
func (bpuo *BillingProfileUpdateOne) SetNillableTimezone(t *timezone.Timezone) *BillingProfileUpdateOne {
	if t != nil {
		bpuo.SetTimezone(*t)
	}
	return bpuo
}

// SetDefault sets the "default" field.
func (bpuo *BillingProfileUpdateOne) SetDefault(b bool) *BillingProfileUpdateOne {
	bpuo.mutation.SetDefault(b)
	return bpuo
}

// SetNillableDefault sets the "default" field if the given value is not nil.
func (bpuo *BillingProfileUpdateOne) SetNillableDefault(b *bool) *BillingProfileUpdateOne {
	if b != nil {
		bpuo.SetDefault(*b)
	}
	return bpuo
}

// AddBillingInvoiceIDs adds the "billing_invoices" edge to the BillingInvoice entity by IDs.
func (bpuo *BillingProfileUpdateOne) AddBillingInvoiceIDs(ids ...string) *BillingProfileUpdateOne {
	bpuo.mutation.AddBillingInvoiceIDs(ids...)
	return bpuo
}

// AddBillingInvoices adds the "billing_invoices" edges to the BillingInvoice entity.
func (bpuo *BillingProfileUpdateOne) AddBillingInvoices(b ...*BillingInvoice) *BillingProfileUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bpuo.AddBillingInvoiceIDs(ids...)
}

// SetBillingWorkflowConfigID sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity by ID.
func (bpuo *BillingProfileUpdateOne) SetBillingWorkflowConfigID(id string) *BillingProfileUpdateOne {
	bpuo.mutation.SetBillingWorkflowConfigID(id)
	return bpuo
}

// SetBillingWorkflowConfig sets the "billing_workflow_config" edge to the BillingWorkflowConfig entity.
func (bpuo *BillingProfileUpdateOne) SetBillingWorkflowConfig(b *BillingWorkflowConfig) *BillingProfileUpdateOne {
	return bpuo.SetBillingWorkflowConfigID(b.ID)
}

// Mutation returns the BillingProfileMutation object of the builder.
func (bpuo *BillingProfileUpdateOne) Mutation() *BillingProfileMutation {
	return bpuo.mutation
}

// ClearBillingInvoices clears all "billing_invoices" edges to the BillingInvoice entity.
func (bpuo *BillingProfileUpdateOne) ClearBillingInvoices() *BillingProfileUpdateOne {
	bpuo.mutation.ClearBillingInvoices()
	return bpuo
}

// RemoveBillingInvoiceIDs removes the "billing_invoices" edge to BillingInvoice entities by IDs.
func (bpuo *BillingProfileUpdateOne) RemoveBillingInvoiceIDs(ids ...string) *BillingProfileUpdateOne {
	bpuo.mutation.RemoveBillingInvoiceIDs(ids...)
	return bpuo
}

// RemoveBillingInvoices removes "billing_invoices" edges to BillingInvoice entities.
func (bpuo *BillingProfileUpdateOne) RemoveBillingInvoices(b ...*BillingInvoice) *BillingProfileUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bpuo.RemoveBillingInvoiceIDs(ids...)
}

// ClearBillingWorkflowConfig clears the "billing_workflow_config" edge to the BillingWorkflowConfig entity.
func (bpuo *BillingProfileUpdateOne) ClearBillingWorkflowConfig() *BillingProfileUpdateOne {
	bpuo.mutation.ClearBillingWorkflowConfig()
	return bpuo
}

// Where appends a list predicates to the BillingProfileUpdate builder.
func (bpuo *BillingProfileUpdateOne) Where(ps ...predicate.BillingProfile) *BillingProfileUpdateOne {
	bpuo.mutation.Where(ps...)
	return bpuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (bpuo *BillingProfileUpdateOne) Select(field string, fields ...string) *BillingProfileUpdateOne {
	bpuo.fields = append([]string{field}, fields...)
	return bpuo
}

// Save executes the query and returns the updated BillingProfile entity.
func (bpuo *BillingProfileUpdateOne) Save(ctx context.Context) (*BillingProfile, error) {
	bpuo.defaults()
	return withHooks(ctx, bpuo.sqlSave, bpuo.mutation, bpuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bpuo *BillingProfileUpdateOne) SaveX(ctx context.Context) *BillingProfile {
	node, err := bpuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (bpuo *BillingProfileUpdateOne) Exec(ctx context.Context) error {
	_, err := bpuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpuo *BillingProfileUpdateOne) ExecX(ctx context.Context) {
	if err := bpuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bpuo *BillingProfileUpdateOne) defaults() {
	if _, ok := bpuo.mutation.UpdatedAt(); !ok {
		v := billingprofile.UpdateDefaultUpdatedAt()
		bpuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bpuo *BillingProfileUpdateOne) check() error {
	if v, ok := bpuo.mutation.ProviderConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "provider_config", err: fmt.Errorf(`db: validator failed for field "BillingProfile.provider_config": %w`, err)}
		}
	}
	if v, ok := bpuo.mutation.WorkflowConfigID(); ok {
		if err := billingprofile.WorkflowConfigIDValidator(v); err != nil {
			return &ValidationError{Name: "workflow_config_id", err: fmt.Errorf(`db: validator failed for field "BillingProfile.workflow_config_id": %w`, err)}
		}
	}
	if bpuo.mutation.BillingWorkflowConfigCleared() && len(bpuo.mutation.BillingWorkflowConfigIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingProfile.billing_workflow_config"`)
	}
	return nil
}

func (bpuo *BillingProfileUpdateOne) sqlSave(ctx context.Context) (_node *BillingProfile, err error) {
	if err := bpuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billingprofile.Table, billingprofile.Columns, sqlgraph.NewFieldSpec(billingprofile.FieldID, field.TypeString))
	id, ok := bpuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingProfile.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := bpuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billingprofile.FieldID)
		for _, f := range fields {
			if !billingprofile.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billingprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := bpuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bpuo.mutation.UpdatedAt(); ok {
		_spec.SetField(billingprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bpuo.mutation.DeletedAt(); ok {
		_spec.SetField(billingprofile.FieldDeletedAt, field.TypeTime, value)
	}
	if bpuo.mutation.DeletedAtCleared() {
		_spec.ClearField(billingprofile.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := bpuo.mutation.ProviderConfig(); ok {
		vv, err := billingprofile.ValueScanner.ProviderConfig.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(billingprofile.FieldProviderConfig, field.TypeString, vv)
	}
	if value, ok := bpuo.mutation.Timezone(); ok {
		_spec.SetField(billingprofile.FieldTimezone, field.TypeString, value)
	}
	if value, ok := bpuo.mutation.Default(); ok {
		_spec.SetField(billingprofile.FieldDefault, field.TypeBool, value)
	}
	if bpuo.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpuo.mutation.RemovedBillingInvoicesIDs(); len(nodes) > 0 && !bpuo.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpuo.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if bpuo.mutation.BillingWorkflowConfigCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bpuo.mutation.BillingWorkflowConfigIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &BillingProfile{config: bpuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, bpuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billingprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	bpuo.mutation.done = true
	return _node, nil
}
