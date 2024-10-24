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
	"github.com/alpacahq/alpacadecimal"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicemanuallineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceLineUpdate is the builder for updating BillingInvoiceLine entities.
type BillingInvoiceLineUpdate struct {
	config
	hooks    []Hook
	mutation *BillingInvoiceLineMutation
}

// Where appends a list predicates to the BillingInvoiceLineUpdate builder.
func (bilu *BillingInvoiceLineUpdate) Where(ps ...predicate.BillingInvoiceLine) *BillingInvoiceLineUpdate {
	bilu.mutation.Where(ps...)
	return bilu
}

// SetMetadata sets the "metadata" field.
func (bilu *BillingInvoiceLineUpdate) SetMetadata(m map[string]string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetMetadata(m)
	return bilu
}

// ClearMetadata clears the value of the "metadata" field.
func (bilu *BillingInvoiceLineUpdate) ClearMetadata() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearMetadata()
	return bilu
}

// SetUpdatedAt sets the "updated_at" field.
func (bilu *BillingInvoiceLineUpdate) SetUpdatedAt(t time.Time) *BillingInvoiceLineUpdate {
	bilu.mutation.SetUpdatedAt(t)
	return bilu
}

// SetDeletedAt sets the "deleted_at" field.
func (bilu *BillingInvoiceLineUpdate) SetDeletedAt(t time.Time) *BillingInvoiceLineUpdate {
	bilu.mutation.SetDeletedAt(t)
	return bilu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableDeletedAt(t *time.Time) *BillingInvoiceLineUpdate {
	if t != nil {
		bilu.SetDeletedAt(*t)
	}
	return bilu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (bilu *BillingInvoiceLineUpdate) ClearDeletedAt() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearDeletedAt()
	return bilu
}

// SetName sets the "name" field.
func (bilu *BillingInvoiceLineUpdate) SetName(s string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetName(s)
	return bilu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableName(s *string) *BillingInvoiceLineUpdate {
	if s != nil {
		bilu.SetName(*s)
	}
	return bilu
}

// SetDescription sets the "description" field.
func (bilu *BillingInvoiceLineUpdate) SetDescription(s string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetDescription(s)
	return bilu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableDescription(s *string) *BillingInvoiceLineUpdate {
	if s != nil {
		bilu.SetDescription(*s)
	}
	return bilu
}

// ClearDescription clears the value of the "description" field.
func (bilu *BillingInvoiceLineUpdate) ClearDescription() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearDescription()
	return bilu
}

// SetInvoiceID sets the "invoice_id" field.
func (bilu *BillingInvoiceLineUpdate) SetInvoiceID(s string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetInvoiceID(s)
	return bilu
}

// SetNillableInvoiceID sets the "invoice_id" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableInvoiceID(s *string) *BillingInvoiceLineUpdate {
	if s != nil {
		bilu.SetInvoiceID(*s)
	}
	return bilu
}

// SetPeriodStart sets the "period_start" field.
func (bilu *BillingInvoiceLineUpdate) SetPeriodStart(t time.Time) *BillingInvoiceLineUpdate {
	bilu.mutation.SetPeriodStart(t)
	return bilu
}

// SetNillablePeriodStart sets the "period_start" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillablePeriodStart(t *time.Time) *BillingInvoiceLineUpdate {
	if t != nil {
		bilu.SetPeriodStart(*t)
	}
	return bilu
}

// SetPeriodEnd sets the "period_end" field.
func (bilu *BillingInvoiceLineUpdate) SetPeriodEnd(t time.Time) *BillingInvoiceLineUpdate {
	bilu.mutation.SetPeriodEnd(t)
	return bilu
}

// SetNillablePeriodEnd sets the "period_end" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillablePeriodEnd(t *time.Time) *BillingInvoiceLineUpdate {
	if t != nil {
		bilu.SetPeriodEnd(*t)
	}
	return bilu
}

// SetInvoiceAt sets the "invoice_at" field.
func (bilu *BillingInvoiceLineUpdate) SetInvoiceAt(t time.Time) *BillingInvoiceLineUpdate {
	bilu.mutation.SetInvoiceAt(t)
	return bilu
}

// SetNillableInvoiceAt sets the "invoice_at" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableInvoiceAt(t *time.Time) *BillingInvoiceLineUpdate {
	if t != nil {
		bilu.SetInvoiceAt(*t)
	}
	return bilu
}

// SetType sets the "type" field.
func (bilu *BillingInvoiceLineUpdate) SetType(blt billingentity.InvoiceLineType) *BillingInvoiceLineUpdate {
	bilu.mutation.SetType(blt)
	return bilu
}

// SetNillableType sets the "type" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableType(blt *billingentity.InvoiceLineType) *BillingInvoiceLineUpdate {
	if blt != nil {
		bilu.SetType(*blt)
	}
	return bilu
}

// SetStatus sets the "status" field.
func (bilu *BillingInvoiceLineUpdate) SetStatus(bls billingentity.InvoiceLineStatus) *BillingInvoiceLineUpdate {
	bilu.mutation.SetStatus(bls)
	return bilu
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableStatus(bls *billingentity.InvoiceLineStatus) *BillingInvoiceLineUpdate {
	if bls != nil {
		bilu.SetStatus(*bls)
	}
	return bilu
}

// SetQuantity sets the "quantity" field.
func (bilu *BillingInvoiceLineUpdate) SetQuantity(a alpacadecimal.Decimal) *BillingInvoiceLineUpdate {
	bilu.mutation.SetQuantity(a)
	return bilu
}

// SetNillableQuantity sets the "quantity" field if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableQuantity(a *alpacadecimal.Decimal) *BillingInvoiceLineUpdate {
	if a != nil {
		bilu.SetQuantity(*a)
	}
	return bilu
}

// ClearQuantity clears the value of the "quantity" field.
func (bilu *BillingInvoiceLineUpdate) ClearQuantity() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearQuantity()
	return bilu
}

// SetTaxOverrides sets the "tax_overrides" field.
func (bilu *BillingInvoiceLineUpdate) SetTaxOverrides(bo *billingentity.TaxOverrides) *BillingInvoiceLineUpdate {
	bilu.mutation.SetTaxOverrides(bo)
	return bilu
}

// ClearTaxOverrides clears the value of the "tax_overrides" field.
func (bilu *BillingInvoiceLineUpdate) ClearTaxOverrides() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearTaxOverrides()
	return bilu
}

// SetBillingInvoiceID sets the "billing_invoice" edge to the BillingInvoice entity by ID.
func (bilu *BillingInvoiceLineUpdate) SetBillingInvoiceID(id string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetBillingInvoiceID(id)
	return bilu
}

// SetBillingInvoice sets the "billing_invoice" edge to the BillingInvoice entity.
func (bilu *BillingInvoiceLineUpdate) SetBillingInvoice(b *BillingInvoice) *BillingInvoiceLineUpdate {
	return bilu.SetBillingInvoiceID(b.ID)
}

// SetBillingInvoiceManualLinesID sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity by ID.
func (bilu *BillingInvoiceLineUpdate) SetBillingInvoiceManualLinesID(id string) *BillingInvoiceLineUpdate {
	bilu.mutation.SetBillingInvoiceManualLinesID(id)
	return bilu
}

// SetNillableBillingInvoiceManualLinesID sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity by ID if the given value is not nil.
func (bilu *BillingInvoiceLineUpdate) SetNillableBillingInvoiceManualLinesID(id *string) *BillingInvoiceLineUpdate {
	if id != nil {
		bilu = bilu.SetBillingInvoiceManualLinesID(*id)
	}
	return bilu
}

// SetBillingInvoiceManualLines sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity.
func (bilu *BillingInvoiceLineUpdate) SetBillingInvoiceManualLines(b *BillingInvoiceManualLineConfig) *BillingInvoiceLineUpdate {
	return bilu.SetBillingInvoiceManualLinesID(b.ID)
}

// Mutation returns the BillingInvoiceLineMutation object of the builder.
func (bilu *BillingInvoiceLineUpdate) Mutation() *BillingInvoiceLineMutation {
	return bilu.mutation
}

// ClearBillingInvoice clears the "billing_invoice" edge to the BillingInvoice entity.
func (bilu *BillingInvoiceLineUpdate) ClearBillingInvoice() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearBillingInvoice()
	return bilu
}

// ClearBillingInvoiceManualLines clears the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity.
func (bilu *BillingInvoiceLineUpdate) ClearBillingInvoiceManualLines() *BillingInvoiceLineUpdate {
	bilu.mutation.ClearBillingInvoiceManualLines()
	return bilu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (bilu *BillingInvoiceLineUpdate) Save(ctx context.Context) (int, error) {
	bilu.defaults()
	return withHooks(ctx, bilu.sqlSave, bilu.mutation, bilu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bilu *BillingInvoiceLineUpdate) SaveX(ctx context.Context) int {
	affected, err := bilu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (bilu *BillingInvoiceLineUpdate) Exec(ctx context.Context) error {
	_, err := bilu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bilu *BillingInvoiceLineUpdate) ExecX(ctx context.Context) {
	if err := bilu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bilu *BillingInvoiceLineUpdate) defaults() {
	if _, ok := bilu.mutation.UpdatedAt(); !ok {
		v := billinginvoiceline.UpdateDefaultUpdatedAt()
		bilu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bilu *BillingInvoiceLineUpdate) check() error {
	if v, ok := bilu.mutation.GetType(); ok {
		if err := billinginvoiceline.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLine.type": %w`, err)}
		}
	}
	if v, ok := bilu.mutation.Status(); ok {
		if err := billinginvoiceline.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLine.status": %w`, err)}
		}
	}
	if bilu.mutation.BillingInvoiceCleared() && len(bilu.mutation.BillingInvoiceIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingInvoiceLine.billing_invoice"`)
	}
	return nil
}

func (bilu *BillingInvoiceLineUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := bilu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceline.Table, billinginvoiceline.Columns, sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString))
	if ps := bilu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bilu.mutation.Metadata(); ok {
		_spec.SetField(billinginvoiceline.FieldMetadata, field.TypeJSON, value)
	}
	if bilu.mutation.MetadataCleared() {
		_spec.ClearField(billinginvoiceline.FieldMetadata, field.TypeJSON)
	}
	if value, ok := bilu.mutation.UpdatedAt(); ok {
		_spec.SetField(billinginvoiceline.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := bilu.mutation.DeletedAt(); ok {
		_spec.SetField(billinginvoiceline.FieldDeletedAt, field.TypeTime, value)
	}
	if bilu.mutation.DeletedAtCleared() {
		_spec.ClearField(billinginvoiceline.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := bilu.mutation.Name(); ok {
		_spec.SetField(billinginvoiceline.FieldName, field.TypeString, value)
	}
	if value, ok := bilu.mutation.Description(); ok {
		_spec.SetField(billinginvoiceline.FieldDescription, field.TypeString, value)
	}
	if bilu.mutation.DescriptionCleared() {
		_spec.ClearField(billinginvoiceline.FieldDescription, field.TypeString)
	}
	if value, ok := bilu.mutation.PeriodStart(); ok {
		_spec.SetField(billinginvoiceline.FieldPeriodStart, field.TypeTime, value)
	}
	if value, ok := bilu.mutation.PeriodEnd(); ok {
		_spec.SetField(billinginvoiceline.FieldPeriodEnd, field.TypeTime, value)
	}
	if value, ok := bilu.mutation.InvoiceAt(); ok {
		_spec.SetField(billinginvoiceline.FieldInvoiceAt, field.TypeTime, value)
	}
	if value, ok := bilu.mutation.GetType(); ok {
		_spec.SetField(billinginvoiceline.FieldType, field.TypeEnum, value)
	}
	if value, ok := bilu.mutation.Status(); ok {
		_spec.SetField(billinginvoiceline.FieldStatus, field.TypeEnum, value)
	}
	if value, ok := bilu.mutation.Quantity(); ok {
		_spec.SetField(billinginvoiceline.FieldQuantity, field.TypeOther, value)
	}
	if bilu.mutation.QuantityCleared() {
		_spec.ClearField(billinginvoiceline.FieldQuantity, field.TypeOther)
	}
	if value, ok := bilu.mutation.TaxOverrides(); ok {
		_spec.SetField(billinginvoiceline.FieldTaxOverrides, field.TypeJSON, value)
	}
	if bilu.mutation.TaxOverridesCleared() {
		_spec.ClearField(billinginvoiceline.FieldTaxOverrides, field.TypeJSON)
	}
	if bilu.mutation.BillingInvoiceCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoiceline.BillingInvoiceTable,
			Columns: []string{billinginvoiceline.BillingInvoiceColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoice.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bilu.mutation.BillingInvoiceIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoiceline.BillingInvoiceTable,
			Columns: []string{billinginvoiceline.BillingInvoiceColumn},
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
	if bilu.mutation.BillingInvoiceManualLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   billinginvoiceline.BillingInvoiceManualLinesTable,
			Columns: []string{billinginvoiceline.BillingInvoiceManualLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoicemanuallineconfig.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := bilu.mutation.BillingInvoiceManualLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   billinginvoiceline.BillingInvoiceManualLinesTable,
			Columns: []string{billinginvoiceline.BillingInvoiceManualLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoicemanuallineconfig.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, bilu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceline.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	bilu.mutation.done = true
	return n, nil
}

// BillingInvoiceLineUpdateOne is the builder for updating a single BillingInvoiceLine entity.
type BillingInvoiceLineUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingInvoiceLineMutation
}

// SetMetadata sets the "metadata" field.
func (biluo *BillingInvoiceLineUpdateOne) SetMetadata(m map[string]string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetMetadata(m)
	return biluo
}

// ClearMetadata clears the value of the "metadata" field.
func (biluo *BillingInvoiceLineUpdateOne) ClearMetadata() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearMetadata()
	return biluo
}

// SetUpdatedAt sets the "updated_at" field.
func (biluo *BillingInvoiceLineUpdateOne) SetUpdatedAt(t time.Time) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetUpdatedAt(t)
	return biluo
}

// SetDeletedAt sets the "deleted_at" field.
func (biluo *BillingInvoiceLineUpdateOne) SetDeletedAt(t time.Time) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetDeletedAt(t)
	return biluo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableDeletedAt(t *time.Time) *BillingInvoiceLineUpdateOne {
	if t != nil {
		biluo.SetDeletedAt(*t)
	}
	return biluo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (biluo *BillingInvoiceLineUpdateOne) ClearDeletedAt() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearDeletedAt()
	return biluo
}

// SetName sets the "name" field.
func (biluo *BillingInvoiceLineUpdateOne) SetName(s string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetName(s)
	return biluo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableName(s *string) *BillingInvoiceLineUpdateOne {
	if s != nil {
		biluo.SetName(*s)
	}
	return biluo
}

// SetDescription sets the "description" field.
func (biluo *BillingInvoiceLineUpdateOne) SetDescription(s string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetDescription(s)
	return biluo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableDescription(s *string) *BillingInvoiceLineUpdateOne {
	if s != nil {
		biluo.SetDescription(*s)
	}
	return biluo
}

// ClearDescription clears the value of the "description" field.
func (biluo *BillingInvoiceLineUpdateOne) ClearDescription() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearDescription()
	return biluo
}

// SetInvoiceID sets the "invoice_id" field.
func (biluo *BillingInvoiceLineUpdateOne) SetInvoiceID(s string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetInvoiceID(s)
	return biluo
}

// SetNillableInvoiceID sets the "invoice_id" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableInvoiceID(s *string) *BillingInvoiceLineUpdateOne {
	if s != nil {
		biluo.SetInvoiceID(*s)
	}
	return biluo
}

// SetPeriodStart sets the "period_start" field.
func (biluo *BillingInvoiceLineUpdateOne) SetPeriodStart(t time.Time) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetPeriodStart(t)
	return biluo
}

// SetNillablePeriodStart sets the "period_start" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillablePeriodStart(t *time.Time) *BillingInvoiceLineUpdateOne {
	if t != nil {
		biluo.SetPeriodStart(*t)
	}
	return biluo
}

// SetPeriodEnd sets the "period_end" field.
func (biluo *BillingInvoiceLineUpdateOne) SetPeriodEnd(t time.Time) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetPeriodEnd(t)
	return biluo
}

// SetNillablePeriodEnd sets the "period_end" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillablePeriodEnd(t *time.Time) *BillingInvoiceLineUpdateOne {
	if t != nil {
		biluo.SetPeriodEnd(*t)
	}
	return biluo
}

// SetInvoiceAt sets the "invoice_at" field.
func (biluo *BillingInvoiceLineUpdateOne) SetInvoiceAt(t time.Time) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetInvoiceAt(t)
	return biluo
}

// SetNillableInvoiceAt sets the "invoice_at" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableInvoiceAt(t *time.Time) *BillingInvoiceLineUpdateOne {
	if t != nil {
		biluo.SetInvoiceAt(*t)
	}
	return biluo
}

// SetType sets the "type" field.
func (biluo *BillingInvoiceLineUpdateOne) SetType(blt billingentity.InvoiceLineType) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetType(blt)
	return biluo
}

// SetNillableType sets the "type" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableType(blt *billingentity.InvoiceLineType) *BillingInvoiceLineUpdateOne {
	if blt != nil {
		biluo.SetType(*blt)
	}
	return biluo
}

// SetStatus sets the "status" field.
func (biluo *BillingInvoiceLineUpdateOne) SetStatus(bls billingentity.InvoiceLineStatus) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetStatus(bls)
	return biluo
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableStatus(bls *billingentity.InvoiceLineStatus) *BillingInvoiceLineUpdateOne {
	if bls != nil {
		biluo.SetStatus(*bls)
	}
	return biluo
}

// SetQuantity sets the "quantity" field.
func (biluo *BillingInvoiceLineUpdateOne) SetQuantity(a alpacadecimal.Decimal) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetQuantity(a)
	return biluo
}

// SetNillableQuantity sets the "quantity" field if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableQuantity(a *alpacadecimal.Decimal) *BillingInvoiceLineUpdateOne {
	if a != nil {
		biluo.SetQuantity(*a)
	}
	return biluo
}

// ClearQuantity clears the value of the "quantity" field.
func (biluo *BillingInvoiceLineUpdateOne) ClearQuantity() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearQuantity()
	return biluo
}

// SetTaxOverrides sets the "tax_overrides" field.
func (biluo *BillingInvoiceLineUpdateOne) SetTaxOverrides(bo *billingentity.TaxOverrides) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetTaxOverrides(bo)
	return biluo
}

// ClearTaxOverrides clears the value of the "tax_overrides" field.
func (biluo *BillingInvoiceLineUpdateOne) ClearTaxOverrides() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearTaxOverrides()
	return biluo
}

// SetBillingInvoiceID sets the "billing_invoice" edge to the BillingInvoice entity by ID.
func (biluo *BillingInvoiceLineUpdateOne) SetBillingInvoiceID(id string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetBillingInvoiceID(id)
	return biluo
}

// SetBillingInvoice sets the "billing_invoice" edge to the BillingInvoice entity.
func (biluo *BillingInvoiceLineUpdateOne) SetBillingInvoice(b *BillingInvoice) *BillingInvoiceLineUpdateOne {
	return biluo.SetBillingInvoiceID(b.ID)
}

// SetBillingInvoiceManualLinesID sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity by ID.
func (biluo *BillingInvoiceLineUpdateOne) SetBillingInvoiceManualLinesID(id string) *BillingInvoiceLineUpdateOne {
	biluo.mutation.SetBillingInvoiceManualLinesID(id)
	return biluo
}

// SetNillableBillingInvoiceManualLinesID sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity by ID if the given value is not nil.
func (biluo *BillingInvoiceLineUpdateOne) SetNillableBillingInvoiceManualLinesID(id *string) *BillingInvoiceLineUpdateOne {
	if id != nil {
		biluo = biluo.SetBillingInvoiceManualLinesID(*id)
	}
	return biluo
}

// SetBillingInvoiceManualLines sets the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity.
func (biluo *BillingInvoiceLineUpdateOne) SetBillingInvoiceManualLines(b *BillingInvoiceManualLineConfig) *BillingInvoiceLineUpdateOne {
	return biluo.SetBillingInvoiceManualLinesID(b.ID)
}

// Mutation returns the BillingInvoiceLineMutation object of the builder.
func (biluo *BillingInvoiceLineUpdateOne) Mutation() *BillingInvoiceLineMutation {
	return biluo.mutation
}

// ClearBillingInvoice clears the "billing_invoice" edge to the BillingInvoice entity.
func (biluo *BillingInvoiceLineUpdateOne) ClearBillingInvoice() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearBillingInvoice()
	return biluo
}

// ClearBillingInvoiceManualLines clears the "billing_invoice_manual_lines" edge to the BillingInvoiceManualLineConfig entity.
func (biluo *BillingInvoiceLineUpdateOne) ClearBillingInvoiceManualLines() *BillingInvoiceLineUpdateOne {
	biluo.mutation.ClearBillingInvoiceManualLines()
	return biluo
}

// Where appends a list predicates to the BillingInvoiceLineUpdate builder.
func (biluo *BillingInvoiceLineUpdateOne) Where(ps ...predicate.BillingInvoiceLine) *BillingInvoiceLineUpdateOne {
	biluo.mutation.Where(ps...)
	return biluo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (biluo *BillingInvoiceLineUpdateOne) Select(field string, fields ...string) *BillingInvoiceLineUpdateOne {
	biluo.fields = append([]string{field}, fields...)
	return biluo
}

// Save executes the query and returns the updated BillingInvoiceLine entity.
func (biluo *BillingInvoiceLineUpdateOne) Save(ctx context.Context) (*BillingInvoiceLine, error) {
	biluo.defaults()
	return withHooks(ctx, biluo.sqlSave, biluo.mutation, biluo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (biluo *BillingInvoiceLineUpdateOne) SaveX(ctx context.Context) *BillingInvoiceLine {
	node, err := biluo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (biluo *BillingInvoiceLineUpdateOne) Exec(ctx context.Context) error {
	_, err := biluo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (biluo *BillingInvoiceLineUpdateOne) ExecX(ctx context.Context) {
	if err := biluo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (biluo *BillingInvoiceLineUpdateOne) defaults() {
	if _, ok := biluo.mutation.UpdatedAt(); !ok {
		v := billinginvoiceline.UpdateDefaultUpdatedAt()
		biluo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (biluo *BillingInvoiceLineUpdateOne) check() error {
	if v, ok := biluo.mutation.GetType(); ok {
		if err := billinginvoiceline.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLine.type": %w`, err)}
		}
	}
	if v, ok := biluo.mutation.Status(); ok {
		if err := billinginvoiceline.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLine.status": %w`, err)}
		}
	}
	if biluo.mutation.BillingInvoiceCleared() && len(biluo.mutation.BillingInvoiceIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingInvoiceLine.billing_invoice"`)
	}
	return nil
}

func (biluo *BillingInvoiceLineUpdateOne) sqlSave(ctx context.Context) (_node *BillingInvoiceLine, err error) {
	if err := biluo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceline.Table, billinginvoiceline.Columns, sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString))
	id, ok := biluo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingInvoiceLine.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := biluo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoiceline.FieldID)
		for _, f := range fields {
			if !billinginvoiceline.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billinginvoiceline.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := biluo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := biluo.mutation.Metadata(); ok {
		_spec.SetField(billinginvoiceline.FieldMetadata, field.TypeJSON, value)
	}
	if biluo.mutation.MetadataCleared() {
		_spec.ClearField(billinginvoiceline.FieldMetadata, field.TypeJSON)
	}
	if value, ok := biluo.mutation.UpdatedAt(); ok {
		_spec.SetField(billinginvoiceline.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := biluo.mutation.DeletedAt(); ok {
		_spec.SetField(billinginvoiceline.FieldDeletedAt, field.TypeTime, value)
	}
	if biluo.mutation.DeletedAtCleared() {
		_spec.ClearField(billinginvoiceline.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := biluo.mutation.Name(); ok {
		_spec.SetField(billinginvoiceline.FieldName, field.TypeString, value)
	}
	if value, ok := biluo.mutation.Description(); ok {
		_spec.SetField(billinginvoiceline.FieldDescription, field.TypeString, value)
	}
	if biluo.mutation.DescriptionCleared() {
		_spec.ClearField(billinginvoiceline.FieldDescription, field.TypeString)
	}
	if value, ok := biluo.mutation.PeriodStart(); ok {
		_spec.SetField(billinginvoiceline.FieldPeriodStart, field.TypeTime, value)
	}
	if value, ok := biluo.mutation.PeriodEnd(); ok {
		_spec.SetField(billinginvoiceline.FieldPeriodEnd, field.TypeTime, value)
	}
	if value, ok := biluo.mutation.InvoiceAt(); ok {
		_spec.SetField(billinginvoiceline.FieldInvoiceAt, field.TypeTime, value)
	}
	if value, ok := biluo.mutation.GetType(); ok {
		_spec.SetField(billinginvoiceline.FieldType, field.TypeEnum, value)
	}
	if value, ok := biluo.mutation.Status(); ok {
		_spec.SetField(billinginvoiceline.FieldStatus, field.TypeEnum, value)
	}
	if value, ok := biluo.mutation.Quantity(); ok {
		_spec.SetField(billinginvoiceline.FieldQuantity, field.TypeOther, value)
	}
	if biluo.mutation.QuantityCleared() {
		_spec.ClearField(billinginvoiceline.FieldQuantity, field.TypeOther)
	}
	if value, ok := biluo.mutation.TaxOverrides(); ok {
		_spec.SetField(billinginvoiceline.FieldTaxOverrides, field.TypeJSON, value)
	}
	if biluo.mutation.TaxOverridesCleared() {
		_spec.ClearField(billinginvoiceline.FieldTaxOverrides, field.TypeJSON)
	}
	if biluo.mutation.BillingInvoiceCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoiceline.BillingInvoiceTable,
			Columns: []string{billinginvoiceline.BillingInvoiceColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoice.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := biluo.mutation.BillingInvoiceIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoiceline.BillingInvoiceTable,
			Columns: []string{billinginvoiceline.BillingInvoiceColumn},
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
	if biluo.mutation.BillingInvoiceManualLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   billinginvoiceline.BillingInvoiceManualLinesTable,
			Columns: []string{billinginvoiceline.BillingInvoiceManualLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoicemanuallineconfig.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := biluo.mutation.BillingInvoiceManualLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   billinginvoiceline.BillingInvoiceManualLinesTable,
			Columns: []string{billinginvoiceline.BillingInvoiceManualLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoicemanuallineconfig.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &BillingInvoiceLine{config: biluo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, biluo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceline.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	biluo.mutation.done = true
	return _node, nil
}
