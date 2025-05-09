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
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceLineDiscountUpdate is the builder for updating BillingInvoiceLineDiscount entities.
type BillingInvoiceLineDiscountUpdate struct {
	config
	hooks    []Hook
	mutation *BillingInvoiceLineDiscountMutation
}

// Where appends a list predicates to the BillingInvoiceLineDiscountUpdate builder.
func (_u *BillingInvoiceLineDiscountUpdate) Where(ps ...predicate.BillingInvoiceLineDiscount) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.Where(ps...)
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetUpdatedAt(v time.Time) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetDeletedAt(v time.Time) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableDeletedAt(v *time.Time) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearDeletedAt() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetLineID sets the "line_id" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetLineID(v string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetLineID(v)
	return _u
}

// SetNillableLineID sets the "line_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableLineID(v *string) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetLineID(*v)
	}
	return _u
}

// SetChildUniqueReferenceID sets the "child_unique_reference_id" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetChildUniqueReferenceID(v string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetChildUniqueReferenceID(v)
	return _u
}

// SetNillableChildUniqueReferenceID sets the "child_unique_reference_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableChildUniqueReferenceID(v *string) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetChildUniqueReferenceID(*v)
	}
	return _u
}

// ClearChildUniqueReferenceID clears the value of the "child_unique_reference_id" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearChildUniqueReferenceID() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearChildUniqueReferenceID()
	return _u
}

// SetDescription sets the "description" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetDescription(v string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableDescription(v *string) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearDescription() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearDescription()
	return _u
}

// SetReason sets the "reason" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetReason(v billing.DiscountReasonType) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetReason(v)
	return _u
}

// SetNillableReason sets the "reason" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableReason(v *billing.DiscountReasonType) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetReason(*v)
	}
	return _u
}

// SetInvoicingAppExternalID sets the "invoicing_app_external_id" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetInvoicingAppExternalID(v string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetInvoicingAppExternalID(v)
	return _u
}

// SetNillableInvoicingAppExternalID sets the "invoicing_app_external_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableInvoicingAppExternalID(v *string) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetInvoicingAppExternalID(*v)
	}
	return _u
}

// ClearInvoicingAppExternalID clears the value of the "invoicing_app_external_id" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearInvoicingAppExternalID() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearInvoicingAppExternalID()
	return _u
}

// SetAmount sets the "amount" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetAmount(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetAmount(v)
	return _u
}

// SetNillableAmount sets the "amount" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableAmount(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetAmount(*v)
	}
	return _u
}

// SetRoundingAmount sets the "rounding_amount" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetRoundingAmount(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetRoundingAmount(v)
	return _u
}

// SetNillableRoundingAmount sets the "rounding_amount" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableRoundingAmount(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetRoundingAmount(*v)
	}
	return _u
}

// ClearRoundingAmount clears the value of the "rounding_amount" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearRoundingAmount() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearRoundingAmount()
	return _u
}

// SetSourceDiscount sets the "source_discount" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetSourceDiscount(v *billing.DiscountReason) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetSourceDiscount(v)
	return _u
}

// ClearSourceDiscount clears the value of the "source_discount" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearSourceDiscount() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearSourceDiscount()
	return _u
}

// SetType sets the "type" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetType(v string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetType(v)
	return _u
}

// SetNillableType sets the "type" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableType(v *string) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetType(*v)
	}
	return _u
}

// ClearType clears the value of the "type" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearType() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearType()
	return _u
}

// SetQuantity sets the "quantity" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetQuantity(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetQuantity(v)
	return _u
}

// SetNillableQuantity sets the "quantity" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillableQuantity(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetQuantity(*v)
	}
	return _u
}

// ClearQuantity clears the value of the "quantity" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearQuantity() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearQuantity()
	return _u
}

// SetPreLinePeriodQuantity sets the "pre_line_period_quantity" field.
func (_u *BillingInvoiceLineDiscountUpdate) SetPreLinePeriodQuantity(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetPreLinePeriodQuantity(v)
	return _u
}

// SetNillablePreLinePeriodQuantity sets the "pre_line_period_quantity" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdate) SetNillablePreLinePeriodQuantity(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdate {
	if v != nil {
		_u.SetPreLinePeriodQuantity(*v)
	}
	return _u
}

// ClearPreLinePeriodQuantity clears the value of the "pre_line_period_quantity" field.
func (_u *BillingInvoiceLineDiscountUpdate) ClearPreLinePeriodQuantity() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearPreLinePeriodQuantity()
	return _u
}

// SetBillingInvoiceLineID sets the "billing_invoice_line" edge to the BillingInvoiceLine entity by ID.
func (_u *BillingInvoiceLineDiscountUpdate) SetBillingInvoiceLineID(id string) *BillingInvoiceLineDiscountUpdate {
	_u.mutation.SetBillingInvoiceLineID(id)
	return _u
}

// SetBillingInvoiceLine sets the "billing_invoice_line" edge to the BillingInvoiceLine entity.
func (_u *BillingInvoiceLineDiscountUpdate) SetBillingInvoiceLine(v *BillingInvoiceLine) *BillingInvoiceLineDiscountUpdate {
	return _u.SetBillingInvoiceLineID(v.ID)
}

// Mutation returns the BillingInvoiceLineDiscountMutation object of the builder.
func (_u *BillingInvoiceLineDiscountUpdate) Mutation() *BillingInvoiceLineDiscountMutation {
	return _u.mutation
}

// ClearBillingInvoiceLine clears the "billing_invoice_line" edge to the BillingInvoiceLine entity.
func (_u *BillingInvoiceLineDiscountUpdate) ClearBillingInvoiceLine() *BillingInvoiceLineDiscountUpdate {
	_u.mutation.ClearBillingInvoiceLine()
	return _u
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (_u *BillingInvoiceLineDiscountUpdate) Save(ctx context.Context) (int, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *BillingInvoiceLineDiscountUpdate) SaveX(ctx context.Context) int {
	affected, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (_u *BillingInvoiceLineDiscountUpdate) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *BillingInvoiceLineDiscountUpdate) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *BillingInvoiceLineDiscountUpdate) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := billinginvoicelinediscount.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *BillingInvoiceLineDiscountUpdate) check() error {
	if v, ok := _u.mutation.Reason(); ok {
		if err := billinginvoicelinediscount.ReasonValidator(v); err != nil {
			return &ValidationError{Name: "reason", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLineDiscount.reason": %w`, err)}
		}
	}
	if v, ok := _u.mutation.SourceDiscount(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "source_discount", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLineDiscount.source_discount": %w`, err)}
		}
	}
	if _u.mutation.BillingInvoiceLineCleared() && len(_u.mutation.BillingInvoiceLineIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingInvoiceLineDiscount.billing_invoice_line"`)
	}
	return nil
}

func (_u *BillingInvoiceLineDiscountUpdate) sqlSave(ctx context.Context) (_node int, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoicelinediscount.Table, billinginvoicelinediscount.Columns, sqlgraph.NewFieldSpec(billinginvoicelinediscount.FieldID, field.TypeString))
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.ChildUniqueReferenceID(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldChildUniqueReferenceID, field.TypeString, value)
	}
	if _u.mutation.ChildUniqueReferenceIDCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldChildUniqueReferenceID, field.TypeString)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.Reason(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldReason, field.TypeEnum, value)
	}
	if value, ok := _u.mutation.InvoicingAppExternalID(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldInvoicingAppExternalID, field.TypeString, value)
	}
	if _u.mutation.InvoicingAppExternalIDCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldInvoicingAppExternalID, field.TypeString)
	}
	if value, ok := _u.mutation.Amount(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldAmount, field.TypeOther, value)
	}
	if value, ok := _u.mutation.RoundingAmount(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldRoundingAmount, field.TypeOther, value)
	}
	if _u.mutation.RoundingAmountCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldRoundingAmount, field.TypeOther)
	}
	if value, ok := _u.mutation.SourceDiscount(); ok {
		vv, err := billinginvoicelinediscount.ValueScanner.SourceDiscount.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(billinginvoicelinediscount.FieldSourceDiscount, field.TypeString, vv)
	}
	if _u.mutation.SourceDiscountCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldSourceDiscount, field.TypeString)
	}
	if value, ok := _u.mutation.GetType(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldType, field.TypeString, value)
	}
	if _u.mutation.TypeCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldType, field.TypeString)
	}
	if value, ok := _u.mutation.Quantity(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldQuantity, field.TypeOther, value)
	}
	if _u.mutation.QuantityCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldQuantity, field.TypeOther)
	}
	if value, ok := _u.mutation.PreLinePeriodQuantity(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldPreLinePeriodQuantity, field.TypeOther, value)
	}
	if _u.mutation.PreLinePeriodQuantityCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldPreLinePeriodQuantity, field.TypeOther)
	}
	if _u.mutation.BillingInvoiceLineCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoicelinediscount.BillingInvoiceLineTable,
			Columns: []string{billinginvoicelinediscount.BillingInvoiceLineColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.BillingInvoiceLineIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoicelinediscount.BillingInvoiceLineTable,
			Columns: []string{billinginvoicelinediscount.BillingInvoiceLineColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if _node, err = sqlgraph.UpdateNodes(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoicelinediscount.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	_u.mutation.done = true
	return _node, nil
}

// BillingInvoiceLineDiscountUpdateOne is the builder for updating a single BillingInvoiceLineDiscount entity.
type BillingInvoiceLineDiscountUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingInvoiceLineDiscountMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetUpdatedAt(v time.Time) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetDeletedAt(v time.Time) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableDeletedAt(v *time.Time) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearDeletedAt() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetLineID sets the "line_id" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetLineID(v string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetLineID(v)
	return _u
}

// SetNillableLineID sets the "line_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableLineID(v *string) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetLineID(*v)
	}
	return _u
}

// SetChildUniqueReferenceID sets the "child_unique_reference_id" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetChildUniqueReferenceID(v string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetChildUniqueReferenceID(v)
	return _u
}

// SetNillableChildUniqueReferenceID sets the "child_unique_reference_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableChildUniqueReferenceID(v *string) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetChildUniqueReferenceID(*v)
	}
	return _u
}

// ClearChildUniqueReferenceID clears the value of the "child_unique_reference_id" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearChildUniqueReferenceID() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearChildUniqueReferenceID()
	return _u
}

// SetDescription sets the "description" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetDescription(v string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableDescription(v *string) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearDescription() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearDescription()
	return _u
}

// SetReason sets the "reason" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetReason(v billing.DiscountReasonType) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetReason(v)
	return _u
}

// SetNillableReason sets the "reason" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableReason(v *billing.DiscountReasonType) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetReason(*v)
	}
	return _u
}

// SetInvoicingAppExternalID sets the "invoicing_app_external_id" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetInvoicingAppExternalID(v string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetInvoicingAppExternalID(v)
	return _u
}

// SetNillableInvoicingAppExternalID sets the "invoicing_app_external_id" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableInvoicingAppExternalID(v *string) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetInvoicingAppExternalID(*v)
	}
	return _u
}

// ClearInvoicingAppExternalID clears the value of the "invoicing_app_external_id" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearInvoicingAppExternalID() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearInvoicingAppExternalID()
	return _u
}

// SetAmount sets the "amount" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetAmount(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetAmount(v)
	return _u
}

// SetNillableAmount sets the "amount" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableAmount(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetAmount(*v)
	}
	return _u
}

// SetRoundingAmount sets the "rounding_amount" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetRoundingAmount(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetRoundingAmount(v)
	return _u
}

// SetNillableRoundingAmount sets the "rounding_amount" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableRoundingAmount(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetRoundingAmount(*v)
	}
	return _u
}

// ClearRoundingAmount clears the value of the "rounding_amount" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearRoundingAmount() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearRoundingAmount()
	return _u
}

// SetSourceDiscount sets the "source_discount" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetSourceDiscount(v *billing.DiscountReason) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetSourceDiscount(v)
	return _u
}

// ClearSourceDiscount clears the value of the "source_discount" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearSourceDiscount() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearSourceDiscount()
	return _u
}

// SetType sets the "type" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetType(v string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetType(v)
	return _u
}

// SetNillableType sets the "type" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableType(v *string) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetType(*v)
	}
	return _u
}

// ClearType clears the value of the "type" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearType() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearType()
	return _u
}

// SetQuantity sets the "quantity" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetQuantity(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetQuantity(v)
	return _u
}

// SetNillableQuantity sets the "quantity" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillableQuantity(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetQuantity(*v)
	}
	return _u
}

// ClearQuantity clears the value of the "quantity" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearQuantity() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearQuantity()
	return _u
}

// SetPreLinePeriodQuantity sets the "pre_line_period_quantity" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetPreLinePeriodQuantity(v alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetPreLinePeriodQuantity(v)
	return _u
}

// SetNillablePreLinePeriodQuantity sets the "pre_line_period_quantity" field if the given value is not nil.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetNillablePreLinePeriodQuantity(v *alpacadecimal.Decimal) *BillingInvoiceLineDiscountUpdateOne {
	if v != nil {
		_u.SetPreLinePeriodQuantity(*v)
	}
	return _u
}

// ClearPreLinePeriodQuantity clears the value of the "pre_line_period_quantity" field.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearPreLinePeriodQuantity() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearPreLinePeriodQuantity()
	return _u
}

// SetBillingInvoiceLineID sets the "billing_invoice_line" edge to the BillingInvoiceLine entity by ID.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetBillingInvoiceLineID(id string) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.SetBillingInvoiceLineID(id)
	return _u
}

// SetBillingInvoiceLine sets the "billing_invoice_line" edge to the BillingInvoiceLine entity.
func (_u *BillingInvoiceLineDiscountUpdateOne) SetBillingInvoiceLine(v *BillingInvoiceLine) *BillingInvoiceLineDiscountUpdateOne {
	return _u.SetBillingInvoiceLineID(v.ID)
}

// Mutation returns the BillingInvoiceLineDiscountMutation object of the builder.
func (_u *BillingInvoiceLineDiscountUpdateOne) Mutation() *BillingInvoiceLineDiscountMutation {
	return _u.mutation
}

// ClearBillingInvoiceLine clears the "billing_invoice_line" edge to the BillingInvoiceLine entity.
func (_u *BillingInvoiceLineDiscountUpdateOne) ClearBillingInvoiceLine() *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.ClearBillingInvoiceLine()
	return _u
}

// Where appends a list predicates to the BillingInvoiceLineDiscountUpdate builder.
func (_u *BillingInvoiceLineDiscountUpdateOne) Where(ps ...predicate.BillingInvoiceLineDiscount) *BillingInvoiceLineDiscountUpdateOne {
	_u.mutation.Where(ps...)
	return _u
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (_u *BillingInvoiceLineDiscountUpdateOne) Select(field string, fields ...string) *BillingInvoiceLineDiscountUpdateOne {
	_u.fields = append([]string{field}, fields...)
	return _u
}

// Save executes the query and returns the updated BillingInvoiceLineDiscount entity.
func (_u *BillingInvoiceLineDiscountUpdateOne) Save(ctx context.Context) (*BillingInvoiceLineDiscount, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *BillingInvoiceLineDiscountUpdateOne) SaveX(ctx context.Context) *BillingInvoiceLineDiscount {
	node, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (_u *BillingInvoiceLineDiscountUpdateOne) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *BillingInvoiceLineDiscountUpdateOne) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *BillingInvoiceLineDiscountUpdateOne) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := billinginvoicelinediscount.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *BillingInvoiceLineDiscountUpdateOne) check() error {
	if v, ok := _u.mutation.Reason(); ok {
		if err := billinginvoicelinediscount.ReasonValidator(v); err != nil {
			return &ValidationError{Name: "reason", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLineDiscount.reason": %w`, err)}
		}
	}
	if v, ok := _u.mutation.SourceDiscount(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "source_discount", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceLineDiscount.source_discount": %w`, err)}
		}
	}
	if _u.mutation.BillingInvoiceLineCleared() && len(_u.mutation.BillingInvoiceLineIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "BillingInvoiceLineDiscount.billing_invoice_line"`)
	}
	return nil
}

func (_u *BillingInvoiceLineDiscountUpdateOne) sqlSave(ctx context.Context) (_node *BillingInvoiceLineDiscount, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoicelinediscount.Table, billinginvoicelinediscount.Columns, sqlgraph.NewFieldSpec(billinginvoicelinediscount.FieldID, field.TypeString))
	id, ok := _u.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingInvoiceLineDiscount.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := _u.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoicelinediscount.FieldID)
		for _, f := range fields {
			if !billinginvoicelinediscount.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billinginvoicelinediscount.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.ChildUniqueReferenceID(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldChildUniqueReferenceID, field.TypeString, value)
	}
	if _u.mutation.ChildUniqueReferenceIDCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldChildUniqueReferenceID, field.TypeString)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.Reason(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldReason, field.TypeEnum, value)
	}
	if value, ok := _u.mutation.InvoicingAppExternalID(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldInvoicingAppExternalID, field.TypeString, value)
	}
	if _u.mutation.InvoicingAppExternalIDCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldInvoicingAppExternalID, field.TypeString)
	}
	if value, ok := _u.mutation.Amount(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldAmount, field.TypeOther, value)
	}
	if value, ok := _u.mutation.RoundingAmount(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldRoundingAmount, field.TypeOther, value)
	}
	if _u.mutation.RoundingAmountCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldRoundingAmount, field.TypeOther)
	}
	if value, ok := _u.mutation.SourceDiscount(); ok {
		vv, err := billinginvoicelinediscount.ValueScanner.SourceDiscount.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(billinginvoicelinediscount.FieldSourceDiscount, field.TypeString, vv)
	}
	if _u.mutation.SourceDiscountCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldSourceDiscount, field.TypeString)
	}
	if value, ok := _u.mutation.GetType(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldType, field.TypeString, value)
	}
	if _u.mutation.TypeCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldType, field.TypeString)
	}
	if value, ok := _u.mutation.Quantity(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldQuantity, field.TypeOther, value)
	}
	if _u.mutation.QuantityCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldQuantity, field.TypeOther)
	}
	if value, ok := _u.mutation.PreLinePeriodQuantity(); ok {
		_spec.SetField(billinginvoicelinediscount.FieldPreLinePeriodQuantity, field.TypeOther, value)
	}
	if _u.mutation.PreLinePeriodQuantityCleared() {
		_spec.ClearField(billinginvoicelinediscount.FieldPreLinePeriodQuantity, field.TypeOther)
	}
	if _u.mutation.BillingInvoiceLineCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoicelinediscount.BillingInvoiceLineTable,
			Columns: []string{billinginvoicelinediscount.BillingInvoiceLineColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := _u.mutation.BillingInvoiceLineIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   billinginvoicelinediscount.BillingInvoiceLineTable,
			Columns: []string{billinginvoicelinediscount.BillingInvoiceLineColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &BillingInvoiceLineDiscount{config: _u.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoicelinediscount.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	_u.mutation.done = true
	return _node, nil
}
