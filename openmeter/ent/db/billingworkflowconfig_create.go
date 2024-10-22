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
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/pkg/datex"
)

// BillingWorkflowConfigCreate is the builder for creating a BillingWorkflowConfig entity.
type BillingWorkflowConfigCreate struct {
	config
	mutation *BillingWorkflowConfigMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetNamespace sets the "namespace" field.
func (bwcc *BillingWorkflowConfigCreate) SetNamespace(s string) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetNamespace(s)
	return bwcc
}

// SetCreatedAt sets the "created_at" field.
func (bwcc *BillingWorkflowConfigCreate) SetCreatedAt(t time.Time) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetCreatedAt(t)
	return bwcc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableCreatedAt(t *time.Time) *BillingWorkflowConfigCreate {
	if t != nil {
		bwcc.SetCreatedAt(*t)
	}
	return bwcc
}

// SetUpdatedAt sets the "updated_at" field.
func (bwcc *BillingWorkflowConfigCreate) SetUpdatedAt(t time.Time) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetUpdatedAt(t)
	return bwcc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableUpdatedAt(t *time.Time) *BillingWorkflowConfigCreate {
	if t != nil {
		bwcc.SetUpdatedAt(*t)
	}
	return bwcc
}

// SetDeletedAt sets the "deleted_at" field.
func (bwcc *BillingWorkflowConfigCreate) SetDeletedAt(t time.Time) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetDeletedAt(t)
	return bwcc
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableDeletedAt(t *time.Time) *BillingWorkflowConfigCreate {
	if t != nil {
		bwcc.SetDeletedAt(*t)
	}
	return bwcc
}

// SetCollectionAlignment sets the "collection_alignment" field.
func (bwcc *BillingWorkflowConfigCreate) SetCollectionAlignment(bk billingentity.AlignmentKind) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetCollectionAlignment(bk)
	return bwcc
}

// SetLineCollectionPeriod sets the "line_collection_period" field.
func (bwcc *BillingWorkflowConfigCreate) SetLineCollectionPeriod(ds datex.ISOString) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetLineCollectionPeriod(ds)
	return bwcc
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (bwcc *BillingWorkflowConfigCreate) SetInvoiceAutoAdvance(b bool) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetInvoiceAutoAdvance(b)
	return bwcc
}

// SetInvoiceDraftPeriod sets the "invoice_draft_period" field.
func (bwcc *BillingWorkflowConfigCreate) SetInvoiceDraftPeriod(ds datex.ISOString) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetInvoiceDraftPeriod(ds)
	return bwcc
}

// SetInvoiceDueAfter sets the "invoice_due_after" field.
func (bwcc *BillingWorkflowConfigCreate) SetInvoiceDueAfter(ds datex.ISOString) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetInvoiceDueAfter(ds)
	return bwcc
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (bwcc *BillingWorkflowConfigCreate) SetInvoiceCollectionMethod(bm billingentity.CollectionMethod) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetInvoiceCollectionMethod(bm)
	return bwcc
}

// SetID sets the "id" field.
func (bwcc *BillingWorkflowConfigCreate) SetID(s string) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetID(s)
	return bwcc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableID(s *string) *BillingWorkflowConfigCreate {
	if s != nil {
		bwcc.SetID(*s)
	}
	return bwcc
}

// SetBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID.
func (bwcc *BillingWorkflowConfigCreate) SetBillingInvoicesID(id string) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetBillingInvoicesID(id)
	return bwcc
}

// SetNillableBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableBillingInvoicesID(id *string) *BillingWorkflowConfigCreate {
	if id != nil {
		bwcc = bwcc.SetBillingInvoicesID(*id)
	}
	return bwcc
}

// SetBillingInvoices sets the "billing_invoices" edge to the BillingInvoice entity.
func (bwcc *BillingWorkflowConfigCreate) SetBillingInvoices(b *BillingInvoice) *BillingWorkflowConfigCreate {
	return bwcc.SetBillingInvoicesID(b.ID)
}

// SetBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID.
func (bwcc *BillingWorkflowConfigCreate) SetBillingProfileID(id string) *BillingWorkflowConfigCreate {
	bwcc.mutation.SetBillingProfileID(id)
	return bwcc
}

// SetNillableBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID if the given value is not nil.
func (bwcc *BillingWorkflowConfigCreate) SetNillableBillingProfileID(id *string) *BillingWorkflowConfigCreate {
	if id != nil {
		bwcc = bwcc.SetBillingProfileID(*id)
	}
	return bwcc
}

// SetBillingProfile sets the "billing_profile" edge to the BillingProfile entity.
func (bwcc *BillingWorkflowConfigCreate) SetBillingProfile(b *BillingProfile) *BillingWorkflowConfigCreate {
	return bwcc.SetBillingProfileID(b.ID)
}

// Mutation returns the BillingWorkflowConfigMutation object of the builder.
func (bwcc *BillingWorkflowConfigCreate) Mutation() *BillingWorkflowConfigMutation {
	return bwcc.mutation
}

// Save creates the BillingWorkflowConfig in the database.
func (bwcc *BillingWorkflowConfigCreate) Save(ctx context.Context) (*BillingWorkflowConfig, error) {
	bwcc.defaults()
	return withHooks(ctx, bwcc.sqlSave, bwcc.mutation, bwcc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (bwcc *BillingWorkflowConfigCreate) SaveX(ctx context.Context) *BillingWorkflowConfig {
	v, err := bwcc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bwcc *BillingWorkflowConfigCreate) Exec(ctx context.Context) error {
	_, err := bwcc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bwcc *BillingWorkflowConfigCreate) ExecX(ctx context.Context) {
	if err := bwcc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bwcc *BillingWorkflowConfigCreate) defaults() {
	if _, ok := bwcc.mutation.CreatedAt(); !ok {
		v := billingworkflowconfig.DefaultCreatedAt()
		bwcc.mutation.SetCreatedAt(v)
	}
	if _, ok := bwcc.mutation.UpdatedAt(); !ok {
		v := billingworkflowconfig.DefaultUpdatedAt()
		bwcc.mutation.SetUpdatedAt(v)
	}
	if _, ok := bwcc.mutation.ID(); !ok {
		v := billingworkflowconfig.DefaultID()
		bwcc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bwcc *BillingWorkflowConfigCreate) check() error {
	if _, ok := bwcc.mutation.Namespace(); !ok {
		return &ValidationError{Name: "namespace", err: errors.New(`db: missing required field "BillingWorkflowConfig.namespace"`)}
	}
	if v, ok := bwcc.mutation.Namespace(); ok {
		if err := billingworkflowconfig.NamespaceValidator(v); err != nil {
			return &ValidationError{Name: "namespace", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.namespace": %w`, err)}
		}
	}
	if _, ok := bwcc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`db: missing required field "BillingWorkflowConfig.created_at"`)}
	}
	if _, ok := bwcc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`db: missing required field "BillingWorkflowConfig.updated_at"`)}
	}
	if _, ok := bwcc.mutation.CollectionAlignment(); !ok {
		return &ValidationError{Name: "collection_alignment", err: errors.New(`db: missing required field "BillingWorkflowConfig.collection_alignment"`)}
	}
	if v, ok := bwcc.mutation.CollectionAlignment(); ok {
		if err := billingworkflowconfig.CollectionAlignmentValidator(v); err != nil {
			return &ValidationError{Name: "collection_alignment", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.collection_alignment": %w`, err)}
		}
	}
	if _, ok := bwcc.mutation.LineCollectionPeriod(); !ok {
		return &ValidationError{Name: "line_collection_period", err: errors.New(`db: missing required field "BillingWorkflowConfig.line_collection_period"`)}
	}
	if _, ok := bwcc.mutation.InvoiceAutoAdvance(); !ok {
		return &ValidationError{Name: "invoice_auto_advance", err: errors.New(`db: missing required field "BillingWorkflowConfig.invoice_auto_advance"`)}
	}
	if _, ok := bwcc.mutation.InvoiceDraftPeriod(); !ok {
		return &ValidationError{Name: "invoice_draft_period", err: errors.New(`db: missing required field "BillingWorkflowConfig.invoice_draft_period"`)}
	}
	if _, ok := bwcc.mutation.InvoiceDueAfter(); !ok {
		return &ValidationError{Name: "invoice_due_after", err: errors.New(`db: missing required field "BillingWorkflowConfig.invoice_due_after"`)}
	}
	if _, ok := bwcc.mutation.InvoiceCollectionMethod(); !ok {
		return &ValidationError{Name: "invoice_collection_method", err: errors.New(`db: missing required field "BillingWorkflowConfig.invoice_collection_method"`)}
	}
	if v, ok := bwcc.mutation.InvoiceCollectionMethod(); ok {
		if err := billingworkflowconfig.InvoiceCollectionMethodValidator(v); err != nil {
			return &ValidationError{Name: "invoice_collection_method", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.invoice_collection_method": %w`, err)}
		}
	}
	return nil
}

func (bwcc *BillingWorkflowConfigCreate) sqlSave(ctx context.Context) (*BillingWorkflowConfig, error) {
	if err := bwcc.check(); err != nil {
		return nil, err
	}
	_node, _spec := bwcc.createSpec()
	if err := sqlgraph.CreateNode(ctx, bwcc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected BillingWorkflowConfig.ID type: %T", _spec.ID.Value)
		}
	}
	bwcc.mutation.id = &_node.ID
	bwcc.mutation.done = true
	return _node, nil
}

func (bwcc *BillingWorkflowConfigCreate) createSpec() (*BillingWorkflowConfig, *sqlgraph.CreateSpec) {
	var (
		_node = &BillingWorkflowConfig{config: bwcc.config}
		_spec = sqlgraph.NewCreateSpec(billingworkflowconfig.Table, sqlgraph.NewFieldSpec(billingworkflowconfig.FieldID, field.TypeString))
	)
	_spec.OnConflict = bwcc.conflict
	if id, ok := bwcc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := bwcc.mutation.Namespace(); ok {
		_spec.SetField(billingworkflowconfig.FieldNamespace, field.TypeString, value)
		_node.Namespace = value
	}
	if value, ok := bwcc.mutation.CreatedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := bwcc.mutation.UpdatedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := bwcc.mutation.DeletedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldDeletedAt, field.TypeTime, value)
		_node.DeletedAt = &value
	}
	if value, ok := bwcc.mutation.CollectionAlignment(); ok {
		_spec.SetField(billingworkflowconfig.FieldCollectionAlignment, field.TypeEnum, value)
		_node.CollectionAlignment = value
	}
	if value, ok := bwcc.mutation.LineCollectionPeriod(); ok {
		_spec.SetField(billingworkflowconfig.FieldLineCollectionPeriod, field.TypeString, value)
		_node.LineCollectionPeriod = value
	}
	if value, ok := bwcc.mutation.InvoiceAutoAdvance(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceAutoAdvance, field.TypeBool, value)
		_node.InvoiceAutoAdvance = value
	}
	if value, ok := bwcc.mutation.InvoiceDraftPeriod(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDraftPeriod, field.TypeString, value)
		_node.InvoiceDraftPeriod = value
	}
	if value, ok := bwcc.mutation.InvoiceDueAfter(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDueAfter, field.TypeString, value)
		_node.InvoiceDueAfter = value
	}
	if value, ok := bwcc.mutation.InvoiceCollectionMethod(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceCollectionMethod, field.TypeEnum, value)
		_node.InvoiceCollectionMethod = value
	}
	if nodes := bwcc.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   billingworkflowconfig.BillingInvoicesTable,
			Columns: []string{billingworkflowconfig.BillingInvoicesColumn},
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
	if nodes := bwcc.mutation.BillingProfileIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   billingworkflowconfig.BillingProfileTable,
			Columns: []string{billingworkflowconfig.BillingProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billingprofile.FieldID, field.TypeString),
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
//	client.BillingWorkflowConfig.Create().
//		SetNamespace(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.BillingWorkflowConfigUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (bwcc *BillingWorkflowConfigCreate) OnConflict(opts ...sql.ConflictOption) *BillingWorkflowConfigUpsertOne {
	bwcc.conflict = opts
	return &BillingWorkflowConfigUpsertOne{
		create: bwcc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (bwcc *BillingWorkflowConfigCreate) OnConflictColumns(columns ...string) *BillingWorkflowConfigUpsertOne {
	bwcc.conflict = append(bwcc.conflict, sql.ConflictColumns(columns...))
	return &BillingWorkflowConfigUpsertOne{
		create: bwcc,
	}
}

type (
	// BillingWorkflowConfigUpsertOne is the builder for "upsert"-ing
	//  one BillingWorkflowConfig node.
	BillingWorkflowConfigUpsertOne struct {
		create *BillingWorkflowConfigCreate
	}

	// BillingWorkflowConfigUpsert is the "OnConflict" setter.
	BillingWorkflowConfigUpsert struct {
		*sql.UpdateSet
	}
)

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingWorkflowConfigUpsert) SetUpdatedAt(v time.Time) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldUpdatedAt, v)
	return u
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateUpdatedAt() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldUpdatedAt)
	return u
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingWorkflowConfigUpsert) SetDeletedAt(v time.Time) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldDeletedAt, v)
	return u
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateDeletedAt() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldDeletedAt)
	return u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingWorkflowConfigUpsert) ClearDeletedAt() *BillingWorkflowConfigUpsert {
	u.SetNull(billingworkflowconfig.FieldDeletedAt)
	return u
}

// SetCollectionAlignment sets the "collection_alignment" field.
func (u *BillingWorkflowConfigUpsert) SetCollectionAlignment(v billingentity.AlignmentKind) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldCollectionAlignment, v)
	return u
}

// UpdateCollectionAlignment sets the "collection_alignment" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateCollectionAlignment() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldCollectionAlignment)
	return u
}

// SetLineCollectionPeriod sets the "line_collection_period" field.
func (u *BillingWorkflowConfigUpsert) SetLineCollectionPeriod(v datex.ISOString) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldLineCollectionPeriod, v)
	return u
}

// UpdateLineCollectionPeriod sets the "line_collection_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateLineCollectionPeriod() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldLineCollectionPeriod)
	return u
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (u *BillingWorkflowConfigUpsert) SetInvoiceAutoAdvance(v bool) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldInvoiceAutoAdvance, v)
	return u
}

// UpdateInvoiceAutoAdvance sets the "invoice_auto_advance" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateInvoiceAutoAdvance() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldInvoiceAutoAdvance)
	return u
}

// SetInvoiceDraftPeriod sets the "invoice_draft_period" field.
func (u *BillingWorkflowConfigUpsert) SetInvoiceDraftPeriod(v datex.ISOString) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldInvoiceDraftPeriod, v)
	return u
}

// UpdateInvoiceDraftPeriod sets the "invoice_draft_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateInvoiceDraftPeriod() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldInvoiceDraftPeriod)
	return u
}

// SetInvoiceDueAfter sets the "invoice_due_after" field.
func (u *BillingWorkflowConfigUpsert) SetInvoiceDueAfter(v datex.ISOString) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldInvoiceDueAfter, v)
	return u
}

// UpdateInvoiceDueAfter sets the "invoice_due_after" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateInvoiceDueAfter() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldInvoiceDueAfter)
	return u
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (u *BillingWorkflowConfigUpsert) SetInvoiceCollectionMethod(v billingentity.CollectionMethod) *BillingWorkflowConfigUpsert {
	u.Set(billingworkflowconfig.FieldInvoiceCollectionMethod, v)
	return u
}

// UpdateInvoiceCollectionMethod sets the "invoice_collection_method" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsert) UpdateInvoiceCollectionMethod() *BillingWorkflowConfigUpsert {
	u.SetExcluded(billingworkflowconfig.FieldInvoiceCollectionMethod)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(billingworkflowconfig.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *BillingWorkflowConfigUpsertOne) UpdateNewValues() *BillingWorkflowConfigUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(billingworkflowconfig.FieldID)
		}
		if _, exists := u.create.mutation.Namespace(); exists {
			s.SetIgnore(billingworkflowconfig.FieldNamespace)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(billingworkflowconfig.FieldCreatedAt)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *BillingWorkflowConfigUpsertOne) Ignore() *BillingWorkflowConfigUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *BillingWorkflowConfigUpsertOne) DoNothing() *BillingWorkflowConfigUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the BillingWorkflowConfigCreate.OnConflict
// documentation for more info.
func (u *BillingWorkflowConfigUpsertOne) Update(set func(*BillingWorkflowConfigUpsert)) *BillingWorkflowConfigUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&BillingWorkflowConfigUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingWorkflowConfigUpsertOne) SetUpdatedAt(v time.Time) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateUpdatedAt() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingWorkflowConfigUpsertOne) SetDeletedAt(v time.Time) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateDeletedAt() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingWorkflowConfigUpsertOne) ClearDeletedAt() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.ClearDeletedAt()
	})
}

// SetCollectionAlignment sets the "collection_alignment" field.
func (u *BillingWorkflowConfigUpsertOne) SetCollectionAlignment(v billingentity.AlignmentKind) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetCollectionAlignment(v)
	})
}

// UpdateCollectionAlignment sets the "collection_alignment" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateCollectionAlignment() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateCollectionAlignment()
	})
}

// SetLineCollectionPeriod sets the "line_collection_period" field.
func (u *BillingWorkflowConfigUpsertOne) SetLineCollectionPeriod(v datex.ISOString) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetLineCollectionPeriod(v)
	})
}

// UpdateLineCollectionPeriod sets the "line_collection_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateLineCollectionPeriod() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateLineCollectionPeriod()
	})
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (u *BillingWorkflowConfigUpsertOne) SetInvoiceAutoAdvance(v bool) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceAutoAdvance(v)
	})
}

// UpdateInvoiceAutoAdvance sets the "invoice_auto_advance" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateInvoiceAutoAdvance() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceAutoAdvance()
	})
}

// SetInvoiceDraftPeriod sets the "invoice_draft_period" field.
func (u *BillingWorkflowConfigUpsertOne) SetInvoiceDraftPeriod(v datex.ISOString) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceDraftPeriod(v)
	})
}

// UpdateInvoiceDraftPeriod sets the "invoice_draft_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateInvoiceDraftPeriod() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceDraftPeriod()
	})
}

// SetInvoiceDueAfter sets the "invoice_due_after" field.
func (u *BillingWorkflowConfigUpsertOne) SetInvoiceDueAfter(v datex.ISOString) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceDueAfter(v)
	})
}

// UpdateInvoiceDueAfter sets the "invoice_due_after" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateInvoiceDueAfter() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceDueAfter()
	})
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (u *BillingWorkflowConfigUpsertOne) SetInvoiceCollectionMethod(v billingentity.CollectionMethod) *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceCollectionMethod(v)
	})
}

// UpdateInvoiceCollectionMethod sets the "invoice_collection_method" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertOne) UpdateInvoiceCollectionMethod() *BillingWorkflowConfigUpsertOne {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceCollectionMethod()
	})
}

// Exec executes the query.
func (u *BillingWorkflowConfigUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for BillingWorkflowConfigCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *BillingWorkflowConfigUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *BillingWorkflowConfigUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("db: BillingWorkflowConfigUpsertOne.ID is not supported by MySQL driver. Use BillingWorkflowConfigUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *BillingWorkflowConfigUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// BillingWorkflowConfigCreateBulk is the builder for creating many BillingWorkflowConfig entities in bulk.
type BillingWorkflowConfigCreateBulk struct {
	config
	err      error
	builders []*BillingWorkflowConfigCreate
	conflict []sql.ConflictOption
}

// Save creates the BillingWorkflowConfig entities in the database.
func (bwccb *BillingWorkflowConfigCreateBulk) Save(ctx context.Context) ([]*BillingWorkflowConfig, error) {
	if bwccb.err != nil {
		return nil, bwccb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(bwccb.builders))
	nodes := make([]*BillingWorkflowConfig, len(bwccb.builders))
	mutators := make([]Mutator, len(bwccb.builders))
	for i := range bwccb.builders {
		func(i int, root context.Context) {
			builder := bwccb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*BillingWorkflowConfigMutation)
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
					_, err = mutators[i+1].Mutate(root, bwccb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = bwccb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, bwccb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, bwccb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (bwccb *BillingWorkflowConfigCreateBulk) SaveX(ctx context.Context) []*BillingWorkflowConfig {
	v, err := bwccb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bwccb *BillingWorkflowConfigCreateBulk) Exec(ctx context.Context) error {
	_, err := bwccb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bwccb *BillingWorkflowConfigCreateBulk) ExecX(ctx context.Context) {
	if err := bwccb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.BillingWorkflowConfig.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.BillingWorkflowConfigUpsert) {
//			SetNamespace(v+v).
//		}).
//		Exec(ctx)
func (bwccb *BillingWorkflowConfigCreateBulk) OnConflict(opts ...sql.ConflictOption) *BillingWorkflowConfigUpsertBulk {
	bwccb.conflict = opts
	return &BillingWorkflowConfigUpsertBulk{
		create: bwccb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (bwccb *BillingWorkflowConfigCreateBulk) OnConflictColumns(columns ...string) *BillingWorkflowConfigUpsertBulk {
	bwccb.conflict = append(bwccb.conflict, sql.ConflictColumns(columns...))
	return &BillingWorkflowConfigUpsertBulk{
		create: bwccb,
	}
}

// BillingWorkflowConfigUpsertBulk is the builder for "upsert"-ing
// a bulk of BillingWorkflowConfig nodes.
type BillingWorkflowConfigUpsertBulk struct {
	create *BillingWorkflowConfigCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(billingworkflowconfig.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *BillingWorkflowConfigUpsertBulk) UpdateNewValues() *BillingWorkflowConfigUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(billingworkflowconfig.FieldID)
			}
			if _, exists := b.mutation.Namespace(); exists {
				s.SetIgnore(billingworkflowconfig.FieldNamespace)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(billingworkflowconfig.FieldCreatedAt)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.BillingWorkflowConfig.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *BillingWorkflowConfigUpsertBulk) Ignore() *BillingWorkflowConfigUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *BillingWorkflowConfigUpsertBulk) DoNothing() *BillingWorkflowConfigUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the BillingWorkflowConfigCreateBulk.OnConflict
// documentation for more info.
func (u *BillingWorkflowConfigUpsertBulk) Update(set func(*BillingWorkflowConfigUpsert)) *BillingWorkflowConfigUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&BillingWorkflowConfigUpsert{UpdateSet: update})
	}))
	return u
}

// SetUpdatedAt sets the "updated_at" field.
func (u *BillingWorkflowConfigUpsertBulk) SetUpdatedAt(v time.Time) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetUpdatedAt(v)
	})
}

// UpdateUpdatedAt sets the "updated_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateUpdatedAt() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateUpdatedAt()
	})
}

// SetDeletedAt sets the "deleted_at" field.
func (u *BillingWorkflowConfigUpsertBulk) SetDeletedAt(v time.Time) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetDeletedAt(v)
	})
}

// UpdateDeletedAt sets the "deleted_at" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateDeletedAt() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateDeletedAt()
	})
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (u *BillingWorkflowConfigUpsertBulk) ClearDeletedAt() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.ClearDeletedAt()
	})
}

// SetCollectionAlignment sets the "collection_alignment" field.
func (u *BillingWorkflowConfigUpsertBulk) SetCollectionAlignment(v billingentity.AlignmentKind) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetCollectionAlignment(v)
	})
}

// UpdateCollectionAlignment sets the "collection_alignment" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateCollectionAlignment() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateCollectionAlignment()
	})
}

// SetLineCollectionPeriod sets the "line_collection_period" field.
func (u *BillingWorkflowConfigUpsertBulk) SetLineCollectionPeriod(v datex.ISOString) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetLineCollectionPeriod(v)
	})
}

// UpdateLineCollectionPeriod sets the "line_collection_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateLineCollectionPeriod() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateLineCollectionPeriod()
	})
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (u *BillingWorkflowConfigUpsertBulk) SetInvoiceAutoAdvance(v bool) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceAutoAdvance(v)
	})
}

// UpdateInvoiceAutoAdvance sets the "invoice_auto_advance" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateInvoiceAutoAdvance() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceAutoAdvance()
	})
}

// SetInvoiceDraftPeriod sets the "invoice_draft_period" field.
func (u *BillingWorkflowConfigUpsertBulk) SetInvoiceDraftPeriod(v datex.ISOString) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceDraftPeriod(v)
	})
}

// UpdateInvoiceDraftPeriod sets the "invoice_draft_period" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateInvoiceDraftPeriod() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceDraftPeriod()
	})
}

// SetInvoiceDueAfter sets the "invoice_due_after" field.
func (u *BillingWorkflowConfigUpsertBulk) SetInvoiceDueAfter(v datex.ISOString) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceDueAfter(v)
	})
}

// UpdateInvoiceDueAfter sets the "invoice_due_after" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateInvoiceDueAfter() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceDueAfter()
	})
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (u *BillingWorkflowConfigUpsertBulk) SetInvoiceCollectionMethod(v billingentity.CollectionMethod) *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.SetInvoiceCollectionMethod(v)
	})
}

// UpdateInvoiceCollectionMethod sets the "invoice_collection_method" field to the value that was provided on create.
func (u *BillingWorkflowConfigUpsertBulk) UpdateInvoiceCollectionMethod() *BillingWorkflowConfigUpsertBulk {
	return u.Update(func(s *BillingWorkflowConfigUpsert) {
		s.UpdateInvoiceCollectionMethod()
	})
}

// Exec executes the query.
func (u *BillingWorkflowConfigUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("db: OnConflict was set for builder %d. Set it on the BillingWorkflowConfigCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("db: missing options for BillingWorkflowConfigCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *BillingWorkflowConfigUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
