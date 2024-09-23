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
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingWorkflowConfigUpdate is the builder for updating BillingWorkflowConfig entities.
type BillingWorkflowConfigUpdate struct {
	config
	hooks    []Hook
	mutation *BillingWorkflowConfigMutation
}

// Where appends a list predicates to the BillingWorkflowConfigUpdate builder.
func (bwcu *BillingWorkflowConfigUpdate) Where(ps ...predicate.BillingWorkflowConfig) *BillingWorkflowConfigUpdate {
	bwcu.mutation.Where(ps...)
	return bwcu
}

// SetUpdatedAt sets the "updated_at" field.
func (bwcu *BillingWorkflowConfigUpdate) SetUpdatedAt(t time.Time) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetUpdatedAt(t)
	return bwcu
}

// SetDeletedAt sets the "deleted_at" field.
func (bwcu *BillingWorkflowConfigUpdate) SetDeletedAt(t time.Time) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetDeletedAt(t)
	return bwcu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableDeletedAt(t *time.Time) *BillingWorkflowConfigUpdate {
	if t != nil {
		bwcu.SetDeletedAt(*t)
	}
	return bwcu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bwcu *BillingWorkflowConfigUpdate) ClearDeletedAt() *BillingWorkflowConfigUpdate {
	bwcu.mutation.ClearDeletedAt()
	return bwcu
}

// SetAlignment sets the "alignment" field.
func (bwcu *BillingWorkflowConfigUpdate) SetAlignment(bk billing.AlignmentKind) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetAlignment(bk)
	return bwcu
}

// SetNillableAlignment sets the "alignment" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableAlignment(bk *billing.AlignmentKind) *BillingWorkflowConfigUpdate {
	if bk != nil {
		bwcu.SetAlignment(*bk)
	}
	return bwcu
}

// SetItemCollectionPeriodSeconds sets the "item_collection_period_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) SetItemCollectionPeriodSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.ResetItemCollectionPeriodSeconds()
	bwcu.mutation.SetItemCollectionPeriodSeconds(i)
	return bwcu
}

// SetNillableItemCollectionPeriodSeconds sets the "item_collection_period_seconds" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableItemCollectionPeriodSeconds(i *int64) *BillingWorkflowConfigUpdate {
	if i != nil {
		bwcu.SetItemCollectionPeriodSeconds(*i)
	}
	return bwcu
}

// AddItemCollectionPeriodSeconds adds i to the "item_collection_period_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) AddItemCollectionPeriodSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.AddItemCollectionPeriodSeconds(i)
	return bwcu
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceAutoAdvance(b bool) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetInvoiceAutoAdvance(b)
	return bwcu
}

// SetNillableInvoiceAutoAdvance sets the "invoice_auto_advance" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceAutoAdvance(b *bool) *BillingWorkflowConfigUpdate {
	if b != nil {
		bwcu.SetInvoiceAutoAdvance(*b)
	}
	return bwcu
}

// SetInvoiceDraftPeriodSeconds sets the "invoice_draft_period_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceDraftPeriodSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.ResetInvoiceDraftPeriodSeconds()
	bwcu.mutation.SetInvoiceDraftPeriodSeconds(i)
	return bwcu
}

// SetNillableInvoiceDraftPeriodSeconds sets the "invoice_draft_period_seconds" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceDraftPeriodSeconds(i *int64) *BillingWorkflowConfigUpdate {
	if i != nil {
		bwcu.SetInvoiceDraftPeriodSeconds(*i)
	}
	return bwcu
}

// AddInvoiceDraftPeriodSeconds adds i to the "invoice_draft_period_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) AddInvoiceDraftPeriodSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.AddInvoiceDraftPeriodSeconds(i)
	return bwcu
}

// SetInvoiceDueAfterSeconds sets the "invoice_due_after_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceDueAfterSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.ResetInvoiceDueAfterSeconds()
	bwcu.mutation.SetInvoiceDueAfterSeconds(i)
	return bwcu
}

// SetNillableInvoiceDueAfterSeconds sets the "invoice_due_after_seconds" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceDueAfterSeconds(i *int64) *BillingWorkflowConfigUpdate {
	if i != nil {
		bwcu.SetInvoiceDueAfterSeconds(*i)
	}
	return bwcu
}

// AddInvoiceDueAfterSeconds adds i to the "invoice_due_after_seconds" field.
func (bwcu *BillingWorkflowConfigUpdate) AddInvoiceDueAfterSeconds(i int64) *BillingWorkflowConfigUpdate {
	bwcu.mutation.AddInvoiceDueAfterSeconds(i)
	return bwcu
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceCollectionMethod(bm billing.CollectionMethod) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetInvoiceCollectionMethod(bm)
	return bwcu
}

// SetNillableInvoiceCollectionMethod sets the "invoice_collection_method" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceCollectionMethod(bm *billing.CollectionMethod) *BillingWorkflowConfigUpdate {
	if bm != nil {
		bwcu.SetInvoiceCollectionMethod(*bm)
	}
	return bwcu
}

// SetInvoiceLineItemResolution sets the "invoice_line_item_resolution" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceLineItemResolution(br billing.GranualityResolution) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetInvoiceLineItemResolution(br)
	return bwcu
}

// SetNillableInvoiceLineItemResolution sets the "invoice_line_item_resolution" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceLineItemResolution(br *billing.GranualityResolution) *BillingWorkflowConfigUpdate {
	if br != nil {
		bwcu.SetInvoiceLineItemResolution(*br)
	}
	return bwcu
}

// SetInvoiceLineItemPerSubject sets the "invoice_line_item_per_subject" field.
func (bwcu *BillingWorkflowConfigUpdate) SetInvoiceLineItemPerSubject(b bool) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetInvoiceLineItemPerSubject(b)
	return bwcu
}

// SetNillableInvoiceLineItemPerSubject sets the "invoice_line_item_per_subject" field if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableInvoiceLineItemPerSubject(b *bool) *BillingWorkflowConfigUpdate {
	if b != nil {
		bwcu.SetInvoiceLineItemPerSubject(*b)
	}
	return bwcu
}

// SetBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID.
func (bwcu *BillingWorkflowConfigUpdate) SetBillingInvoicesID(id string) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetBillingInvoicesID(id)
	return bwcu
}

// SetNillableBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableBillingInvoicesID(id *string) *BillingWorkflowConfigUpdate {
	if id != nil {
		bwcu = bwcu.SetBillingInvoicesID(*id)
	}
	return bwcu
}

// SetBillingInvoices sets the "billing_invoices" edge to the BillingInvoice entity.
func (bwcu *BillingWorkflowConfigUpdate) SetBillingInvoices(b *BillingInvoice) *BillingWorkflowConfigUpdate {
	return bwcu.SetBillingInvoicesID(b.ID)
}

// SetBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID.
func (bwcu *BillingWorkflowConfigUpdate) SetBillingProfileID(id string) *BillingWorkflowConfigUpdate {
	bwcu.mutation.SetBillingProfileID(id)
	return bwcu
}

// SetNillableBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID if the given value is not nil.
func (bwcu *BillingWorkflowConfigUpdate) SetNillableBillingProfileID(id *string) *BillingWorkflowConfigUpdate {
	if id != nil {
		bwcu = bwcu.SetBillingProfileID(*id)
	}
	return bwcu
}

// SetBillingProfile sets the "billing_profile" edge to the BillingProfile entity.
func (bwcu *BillingWorkflowConfigUpdate) SetBillingProfile(b *BillingProfile) *BillingWorkflowConfigUpdate {
	return bwcu.SetBillingProfileID(b.ID)
}

// Mutation returns the BillingWorkflowConfigMutation object of the builder.
func (bwcu *BillingWorkflowConfigUpdate) Mutation() *BillingWorkflowConfigMutation {
	return bwcu.mutation
}

// ClearBillingInvoices clears the "billing_invoices" edge to the BillingInvoice entity.
func (bwcu *BillingWorkflowConfigUpdate) ClearBillingInvoices() *BillingWorkflowConfigUpdate {
	bwcu.mutation.ClearBillingInvoices()
	return bwcu
}

// ClearBillingProfile clears the "billing_profile" edge to the BillingProfile entity.
func (bwcu *BillingWorkflowConfigUpdate) ClearBillingProfile() *BillingWorkflowConfigUpdate {
	bwcu.mutation.ClearBillingProfile()
	return bwcu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (bwcu *BillingWorkflowConfigUpdate) Save(ctx context.Context) (int, error) {
	bwcu.defaults()
	return withHooks(ctx, bwcu.sqlSave, bwcu.mutation, bwcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bwcu *BillingWorkflowConfigUpdate) SaveX(ctx context.Context) int {
	affected, err := bwcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (bwcu *BillingWorkflowConfigUpdate) Exec(ctx context.Context) error {
	_, err := bwcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bwcu *BillingWorkflowConfigUpdate) ExecX(ctx context.Context) {
	if err := bwcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bwcu *BillingWorkflowConfigUpdate) defaults() {
	if _, ok := bwcu.mutation.UpdatedAt(); !ok {
		v := billingworkflowconfig.UpdateDefaultUpdatedAt()
		bwcu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bwcu *BillingWorkflowConfigUpdate) check() error {
	if v, ok := bwcu.mutation.Alignment(); ok {
		if err := billingworkflowconfig.AlignmentValidator(v); err != nil {
			return &ValidationError{Name: "alignment", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.alignment": %w`, err)}
		}
	}
	if v, ok := bwcu.mutation.InvoiceCollectionMethod(); ok {
		if err := billingworkflowconfig.InvoiceCollectionMethodValidator(v); err != nil {
			return &ValidationError{Name: "invoice_collection_method", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.invoice_collection_method": %w`, err)}
		}
	}
	if v, ok := bwcu.mutation.InvoiceLineItemResolution(); ok {
		if err := billingworkflowconfig.InvoiceLineItemResolutionValidator(v); err != nil {
			return &ValidationError{Name: "invoice_line_item_resolution", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.invoice_line_item_resolution": %w`, err)}
		}
	}
	return nil
}

func (bwcu *BillingWorkflowConfigUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := bwcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(billingworkflowconfig.Table, billingworkflowconfig.Columns, sqlgraph.NewFieldSpec(billingworkflowconfig.FieldID, field.TypeString))
	if ps := bwcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bwcu.mutation.UpdatedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bwcu.mutation.DeletedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldDeletedAt, field.TypeTime, value)
	}
	if bwcu.mutation.DeletedAtCleared() {
		_spec.ClearField(billingworkflowconfig.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := bwcu.mutation.Alignment(); ok {
		_spec.SetField(billingworkflowconfig.FieldAlignment, field.TypeEnum, value)
	}
	if value, ok := bwcu.mutation.ItemCollectionPeriodSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldItemCollectionPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.AddedItemCollectionPeriodSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldItemCollectionPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.InvoiceAutoAdvance(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceAutoAdvance, field.TypeBool, value)
	}
	if value, ok := bwcu.mutation.InvoiceDraftPeriodSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDraftPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.AddedInvoiceDraftPeriodSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldInvoiceDraftPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.InvoiceDueAfterSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDueAfterSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.AddedInvoiceDueAfterSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldInvoiceDueAfterSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcu.mutation.InvoiceCollectionMethod(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceCollectionMethod, field.TypeEnum, value)
	}
	if value, ok := bwcu.mutation.InvoiceLineItemResolution(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceLineItemResolution, field.TypeEnum, value)
	}
	if value, ok := bwcu.mutation.InvoiceLineItemPerSubject(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceLineItemPerSubject, field.TypeBool, value)
	}
	if bwcu.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bwcu.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if bwcu.mutation.BillingProfileCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bwcu.mutation.BillingProfileIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, bwcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billingworkflowconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	bwcu.mutation.done = true
	return n, nil
}

// BillingWorkflowConfigUpdateOne is the builder for updating a single BillingWorkflowConfig entity.
type BillingWorkflowConfigUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingWorkflowConfigMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetUpdatedAt(t time.Time) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetUpdatedAt(t)
	return bwcuo
}

// SetDeletedAt sets the "deleted_at" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetDeletedAt(t time.Time) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetDeletedAt(t)
	return bwcuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableDeletedAt(t *time.Time) *BillingWorkflowConfigUpdateOne {
	if t != nil {
		bwcuo.SetDeletedAt(*t)
	}
	return bwcuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) ClearDeletedAt() *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ClearDeletedAt()
	return bwcuo
}

// SetAlignment sets the "alignment" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetAlignment(bk billing.AlignmentKind) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetAlignment(bk)
	return bwcuo
}

// SetNillableAlignment sets the "alignment" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableAlignment(bk *billing.AlignmentKind) *BillingWorkflowConfigUpdateOne {
	if bk != nil {
		bwcuo.SetAlignment(*bk)
	}
	return bwcuo
}

// SetItemCollectionPeriodSeconds sets the "item_collection_period_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetItemCollectionPeriodSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ResetItemCollectionPeriodSeconds()
	bwcuo.mutation.SetItemCollectionPeriodSeconds(i)
	return bwcuo
}

// SetNillableItemCollectionPeriodSeconds sets the "item_collection_period_seconds" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableItemCollectionPeriodSeconds(i *int64) *BillingWorkflowConfigUpdateOne {
	if i != nil {
		bwcuo.SetItemCollectionPeriodSeconds(*i)
	}
	return bwcuo
}

// AddItemCollectionPeriodSeconds adds i to the "item_collection_period_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) AddItemCollectionPeriodSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.AddItemCollectionPeriodSeconds(i)
	return bwcuo
}

// SetInvoiceAutoAdvance sets the "invoice_auto_advance" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceAutoAdvance(b bool) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetInvoiceAutoAdvance(b)
	return bwcuo
}

// SetNillableInvoiceAutoAdvance sets the "invoice_auto_advance" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceAutoAdvance(b *bool) *BillingWorkflowConfigUpdateOne {
	if b != nil {
		bwcuo.SetInvoiceAutoAdvance(*b)
	}
	return bwcuo
}

// SetInvoiceDraftPeriodSeconds sets the "invoice_draft_period_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceDraftPeriodSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ResetInvoiceDraftPeriodSeconds()
	bwcuo.mutation.SetInvoiceDraftPeriodSeconds(i)
	return bwcuo
}

// SetNillableInvoiceDraftPeriodSeconds sets the "invoice_draft_period_seconds" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceDraftPeriodSeconds(i *int64) *BillingWorkflowConfigUpdateOne {
	if i != nil {
		bwcuo.SetInvoiceDraftPeriodSeconds(*i)
	}
	return bwcuo
}

// AddInvoiceDraftPeriodSeconds adds i to the "invoice_draft_period_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) AddInvoiceDraftPeriodSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.AddInvoiceDraftPeriodSeconds(i)
	return bwcuo
}

// SetInvoiceDueAfterSeconds sets the "invoice_due_after_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceDueAfterSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ResetInvoiceDueAfterSeconds()
	bwcuo.mutation.SetInvoiceDueAfterSeconds(i)
	return bwcuo
}

// SetNillableInvoiceDueAfterSeconds sets the "invoice_due_after_seconds" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceDueAfterSeconds(i *int64) *BillingWorkflowConfigUpdateOne {
	if i != nil {
		bwcuo.SetInvoiceDueAfterSeconds(*i)
	}
	return bwcuo
}

// AddInvoiceDueAfterSeconds adds i to the "invoice_due_after_seconds" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) AddInvoiceDueAfterSeconds(i int64) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.AddInvoiceDueAfterSeconds(i)
	return bwcuo
}

// SetInvoiceCollectionMethod sets the "invoice_collection_method" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceCollectionMethod(bm billing.CollectionMethod) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetInvoiceCollectionMethod(bm)
	return bwcuo
}

// SetNillableInvoiceCollectionMethod sets the "invoice_collection_method" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceCollectionMethod(bm *billing.CollectionMethod) *BillingWorkflowConfigUpdateOne {
	if bm != nil {
		bwcuo.SetInvoiceCollectionMethod(*bm)
	}
	return bwcuo
}

// SetInvoiceLineItemResolution sets the "invoice_line_item_resolution" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceLineItemResolution(br billing.GranualityResolution) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetInvoiceLineItemResolution(br)
	return bwcuo
}

// SetNillableInvoiceLineItemResolution sets the "invoice_line_item_resolution" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceLineItemResolution(br *billing.GranualityResolution) *BillingWorkflowConfigUpdateOne {
	if br != nil {
		bwcuo.SetInvoiceLineItemResolution(*br)
	}
	return bwcuo
}

// SetInvoiceLineItemPerSubject sets the "invoice_line_item_per_subject" field.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetInvoiceLineItemPerSubject(b bool) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetInvoiceLineItemPerSubject(b)
	return bwcuo
}

// SetNillableInvoiceLineItemPerSubject sets the "invoice_line_item_per_subject" field if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableInvoiceLineItemPerSubject(b *bool) *BillingWorkflowConfigUpdateOne {
	if b != nil {
		bwcuo.SetInvoiceLineItemPerSubject(*b)
	}
	return bwcuo
}

// SetBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetBillingInvoicesID(id string) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetBillingInvoicesID(id)
	return bwcuo
}

// SetNillableBillingInvoicesID sets the "billing_invoices" edge to the BillingInvoice entity by ID if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableBillingInvoicesID(id *string) *BillingWorkflowConfigUpdateOne {
	if id != nil {
		bwcuo = bwcuo.SetBillingInvoicesID(*id)
	}
	return bwcuo
}

// SetBillingInvoices sets the "billing_invoices" edge to the BillingInvoice entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetBillingInvoices(b *BillingInvoice) *BillingWorkflowConfigUpdateOne {
	return bwcuo.SetBillingInvoicesID(b.ID)
}

// SetBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetBillingProfileID(id string) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.SetBillingProfileID(id)
	return bwcuo
}

// SetNillableBillingProfileID sets the "billing_profile" edge to the BillingProfile entity by ID if the given value is not nil.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetNillableBillingProfileID(id *string) *BillingWorkflowConfigUpdateOne {
	if id != nil {
		bwcuo = bwcuo.SetBillingProfileID(*id)
	}
	return bwcuo
}

// SetBillingProfile sets the "billing_profile" edge to the BillingProfile entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) SetBillingProfile(b *BillingProfile) *BillingWorkflowConfigUpdateOne {
	return bwcuo.SetBillingProfileID(b.ID)
}

// Mutation returns the BillingWorkflowConfigMutation object of the builder.
func (bwcuo *BillingWorkflowConfigUpdateOne) Mutation() *BillingWorkflowConfigMutation {
	return bwcuo.mutation
}

// ClearBillingInvoices clears the "billing_invoices" edge to the BillingInvoice entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) ClearBillingInvoices() *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ClearBillingInvoices()
	return bwcuo
}

// ClearBillingProfile clears the "billing_profile" edge to the BillingProfile entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) ClearBillingProfile() *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.ClearBillingProfile()
	return bwcuo
}

// Where appends a list predicates to the BillingWorkflowConfigUpdate builder.
func (bwcuo *BillingWorkflowConfigUpdateOne) Where(ps ...predicate.BillingWorkflowConfig) *BillingWorkflowConfigUpdateOne {
	bwcuo.mutation.Where(ps...)
	return bwcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (bwcuo *BillingWorkflowConfigUpdateOne) Select(field string, fields ...string) *BillingWorkflowConfigUpdateOne {
	bwcuo.fields = append([]string{field}, fields...)
	return bwcuo
}

// Save executes the query and returns the updated BillingWorkflowConfig entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) Save(ctx context.Context) (*BillingWorkflowConfig, error) {
	bwcuo.defaults()
	return withHooks(ctx, bwcuo.sqlSave, bwcuo.mutation, bwcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bwcuo *BillingWorkflowConfigUpdateOne) SaveX(ctx context.Context) *BillingWorkflowConfig {
	node, err := bwcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (bwcuo *BillingWorkflowConfigUpdateOne) Exec(ctx context.Context) error {
	_, err := bwcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bwcuo *BillingWorkflowConfigUpdateOne) ExecX(ctx context.Context) {
	if err := bwcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bwcuo *BillingWorkflowConfigUpdateOne) defaults() {
	if _, ok := bwcuo.mutation.UpdatedAt(); !ok {
		v := billingworkflowconfig.UpdateDefaultUpdatedAt()
		bwcuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bwcuo *BillingWorkflowConfigUpdateOne) check() error {
	if v, ok := bwcuo.mutation.Alignment(); ok {
		if err := billingworkflowconfig.AlignmentValidator(v); err != nil {
			return &ValidationError{Name: "alignment", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.alignment": %w`, err)}
		}
	}
	if v, ok := bwcuo.mutation.InvoiceCollectionMethod(); ok {
		if err := billingworkflowconfig.InvoiceCollectionMethodValidator(v); err != nil {
			return &ValidationError{Name: "invoice_collection_method", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.invoice_collection_method": %w`, err)}
		}
	}
	if v, ok := bwcuo.mutation.InvoiceLineItemResolution(); ok {
		if err := billingworkflowconfig.InvoiceLineItemResolutionValidator(v); err != nil {
			return &ValidationError{Name: "invoice_line_item_resolution", err: fmt.Errorf(`db: validator failed for field "BillingWorkflowConfig.invoice_line_item_resolution": %w`, err)}
		}
	}
	return nil
}

func (bwcuo *BillingWorkflowConfigUpdateOne) sqlSave(ctx context.Context) (_node *BillingWorkflowConfig, err error) {
	if err := bwcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billingworkflowconfig.Table, billingworkflowconfig.Columns, sqlgraph.NewFieldSpec(billingworkflowconfig.FieldID, field.TypeString))
	id, ok := bwcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingWorkflowConfig.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := bwcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billingworkflowconfig.FieldID)
		for _, f := range fields {
			if !billingworkflowconfig.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billingworkflowconfig.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := bwcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bwcuo.mutation.UpdatedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bwcuo.mutation.DeletedAt(); ok {
		_spec.SetField(billingworkflowconfig.FieldDeletedAt, field.TypeTime, value)
	}
	if bwcuo.mutation.DeletedAtCleared() {
		_spec.ClearField(billingworkflowconfig.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := bwcuo.mutation.Alignment(); ok {
		_spec.SetField(billingworkflowconfig.FieldAlignment, field.TypeEnum, value)
	}
	if value, ok := bwcuo.mutation.ItemCollectionPeriodSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldItemCollectionPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.AddedItemCollectionPeriodSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldItemCollectionPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.InvoiceAutoAdvance(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceAutoAdvance, field.TypeBool, value)
	}
	if value, ok := bwcuo.mutation.InvoiceDraftPeriodSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDraftPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.AddedInvoiceDraftPeriodSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldInvoiceDraftPeriodSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.InvoiceDueAfterSeconds(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceDueAfterSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.AddedInvoiceDueAfterSeconds(); ok {
		_spec.AddField(billingworkflowconfig.FieldInvoiceDueAfterSeconds, field.TypeInt64, value)
	}
	if value, ok := bwcuo.mutation.InvoiceCollectionMethod(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceCollectionMethod, field.TypeEnum, value)
	}
	if value, ok := bwcuo.mutation.InvoiceLineItemResolution(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceLineItemResolution, field.TypeEnum, value)
	}
	if value, ok := bwcuo.mutation.InvoiceLineItemPerSubject(); ok {
		_spec.SetField(billingworkflowconfig.FieldInvoiceLineItemPerSubject, field.TypeBool, value)
	}
	if bwcuo.mutation.BillingInvoicesCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bwcuo.mutation.BillingInvoicesIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if bwcuo.mutation.BillingProfileCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bwcuo.mutation.BillingProfileIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &BillingWorkflowConfig{config: bwcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, bwcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billingworkflowconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	bwcuo.mutation.done = true
	return _node, nil
}
