// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// BillingInvoiceFlatFeeLineConfigUpdate is the builder for updating BillingInvoiceFlatFeeLineConfig entities.
type BillingInvoiceFlatFeeLineConfigUpdate struct {
	config
	hooks    []Hook
	mutation *BillingInvoiceFlatFeeLineConfigMutation
}

// Where appends a list predicates to the BillingInvoiceFlatFeeLineConfigUpdate builder.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) Where(ps ...predicate.BillingInvoiceFlatFeeLineConfig) *BillingInvoiceFlatFeeLineConfigUpdate {
	bifflcu.mutation.Where(ps...)
	return bifflcu
}

// SetPerUnitAmount sets the "per_unit_amount" field.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetPerUnitAmount(a alpacadecimal.Decimal) *BillingInvoiceFlatFeeLineConfigUpdate {
	bifflcu.mutation.SetPerUnitAmount(a)
	return bifflcu
}

// SetNillablePerUnitAmount sets the "per_unit_amount" field if the given value is not nil.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetNillablePerUnitAmount(a *alpacadecimal.Decimal) *BillingInvoiceFlatFeeLineConfigUpdate {
	if a != nil {
		bifflcu.SetPerUnitAmount(*a)
	}
	return bifflcu
}

// SetCategory sets the "category" field.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetCategory(bfc billing.FlatFeeCategory) *BillingInvoiceFlatFeeLineConfigUpdate {
	bifflcu.mutation.SetCategory(bfc)
	return bifflcu
}

// SetNillableCategory sets the "category" field if the given value is not nil.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetNillableCategory(bfc *billing.FlatFeeCategory) *BillingInvoiceFlatFeeLineConfigUpdate {
	if bfc != nil {
		bifflcu.SetCategory(*bfc)
	}
	return bifflcu
}

// SetPaymentTerm sets the "payment_term" field.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetPaymentTerm(ptt productcatalog.PaymentTermType) *BillingInvoiceFlatFeeLineConfigUpdate {
	bifflcu.mutation.SetPaymentTerm(ptt)
	return bifflcu
}

// SetNillablePaymentTerm sets the "payment_term" field if the given value is not nil.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SetNillablePaymentTerm(ptt *productcatalog.PaymentTermType) *BillingInvoiceFlatFeeLineConfigUpdate {
	if ptt != nil {
		bifflcu.SetPaymentTerm(*ptt)
	}
	return bifflcu
}

// Mutation returns the BillingInvoiceFlatFeeLineConfigMutation object of the builder.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) Mutation() *BillingInvoiceFlatFeeLineConfigMutation {
	return bifflcu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, bifflcu.sqlSave, bifflcu.mutation, bifflcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) SaveX(ctx context.Context) int {
	affected, err := bifflcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) Exec(ctx context.Context) error {
	_, err := bifflcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) ExecX(ctx context.Context) {
	if err := bifflcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) check() error {
	if v, ok := bifflcu.mutation.Category(); ok {
		if err := billinginvoiceflatfeelineconfig.CategoryValidator(v); err != nil {
			return &ValidationError{Name: "category", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceFlatFeeLineConfig.category": %w`, err)}
		}
	}
	if v, ok := bifflcu.mutation.PaymentTerm(); ok {
		if err := billinginvoiceflatfeelineconfig.PaymentTermValidator(v); err != nil {
			return &ValidationError{Name: "payment_term", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceFlatFeeLineConfig.payment_term": %w`, err)}
		}
	}
	return nil
}

func (bifflcu *BillingInvoiceFlatFeeLineConfigUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := bifflcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceflatfeelineconfig.Table, billinginvoiceflatfeelineconfig.Columns, sqlgraph.NewFieldSpec(billinginvoiceflatfeelineconfig.FieldID, field.TypeString))
	if ps := bifflcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bifflcu.mutation.PerUnitAmount(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldPerUnitAmount, field.TypeOther, value)
	}
	if value, ok := bifflcu.mutation.Category(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldCategory, field.TypeEnum, value)
	}
	if value, ok := bifflcu.mutation.PaymentTerm(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldPaymentTerm, field.TypeEnum, value)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, bifflcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceflatfeelineconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	bifflcu.mutation.done = true
	return n, nil
}

// BillingInvoiceFlatFeeLineConfigUpdateOne is the builder for updating a single BillingInvoiceFlatFeeLineConfig entity.
type BillingInvoiceFlatFeeLineConfigUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingInvoiceFlatFeeLineConfigMutation
}

// SetPerUnitAmount sets the "per_unit_amount" field.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetPerUnitAmount(a alpacadecimal.Decimal) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	bifflcuo.mutation.SetPerUnitAmount(a)
	return bifflcuo
}

// SetNillablePerUnitAmount sets the "per_unit_amount" field if the given value is not nil.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetNillablePerUnitAmount(a *alpacadecimal.Decimal) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	if a != nil {
		bifflcuo.SetPerUnitAmount(*a)
	}
	return bifflcuo
}

// SetCategory sets the "category" field.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetCategory(bfc billing.FlatFeeCategory) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	bifflcuo.mutation.SetCategory(bfc)
	return bifflcuo
}

// SetNillableCategory sets the "category" field if the given value is not nil.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetNillableCategory(bfc *billing.FlatFeeCategory) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	if bfc != nil {
		bifflcuo.SetCategory(*bfc)
	}
	return bifflcuo
}

// SetPaymentTerm sets the "payment_term" field.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetPaymentTerm(ptt productcatalog.PaymentTermType) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	bifflcuo.mutation.SetPaymentTerm(ptt)
	return bifflcuo
}

// SetNillablePaymentTerm sets the "payment_term" field if the given value is not nil.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SetNillablePaymentTerm(ptt *productcatalog.PaymentTermType) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	if ptt != nil {
		bifflcuo.SetPaymentTerm(*ptt)
	}
	return bifflcuo
}

// Mutation returns the BillingInvoiceFlatFeeLineConfigMutation object of the builder.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) Mutation() *BillingInvoiceFlatFeeLineConfigMutation {
	return bifflcuo.mutation
}

// Where appends a list predicates to the BillingInvoiceFlatFeeLineConfigUpdate builder.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) Where(ps ...predicate.BillingInvoiceFlatFeeLineConfig) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	bifflcuo.mutation.Where(ps...)
	return bifflcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) Select(field string, fields ...string) *BillingInvoiceFlatFeeLineConfigUpdateOne {
	bifflcuo.fields = append([]string{field}, fields...)
	return bifflcuo
}

// Save executes the query and returns the updated BillingInvoiceFlatFeeLineConfig entity.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) Save(ctx context.Context) (*BillingInvoiceFlatFeeLineConfig, error) {
	return withHooks(ctx, bifflcuo.sqlSave, bifflcuo.mutation, bifflcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) SaveX(ctx context.Context) *BillingInvoiceFlatFeeLineConfig {
	node, err := bifflcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) Exec(ctx context.Context) error {
	_, err := bifflcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) ExecX(ctx context.Context) {
	if err := bifflcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) check() error {
	if v, ok := bifflcuo.mutation.Category(); ok {
		if err := billinginvoiceflatfeelineconfig.CategoryValidator(v); err != nil {
			return &ValidationError{Name: "category", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceFlatFeeLineConfig.category": %w`, err)}
		}
	}
	if v, ok := bifflcuo.mutation.PaymentTerm(); ok {
		if err := billinginvoiceflatfeelineconfig.PaymentTermValidator(v); err != nil {
			return &ValidationError{Name: "payment_term", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceFlatFeeLineConfig.payment_term": %w`, err)}
		}
	}
	return nil
}

func (bifflcuo *BillingInvoiceFlatFeeLineConfigUpdateOne) sqlSave(ctx context.Context) (_node *BillingInvoiceFlatFeeLineConfig, err error) {
	if err := bifflcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceflatfeelineconfig.Table, billinginvoiceflatfeelineconfig.Columns, sqlgraph.NewFieldSpec(billinginvoiceflatfeelineconfig.FieldID, field.TypeString))
	id, ok := bifflcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingInvoiceFlatFeeLineConfig.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := bifflcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoiceflatfeelineconfig.FieldID)
		for _, f := range fields {
			if !billinginvoiceflatfeelineconfig.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billinginvoiceflatfeelineconfig.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := bifflcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := bifflcuo.mutation.PerUnitAmount(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldPerUnitAmount, field.TypeOther, value)
	}
	if value, ok := bifflcuo.mutation.Category(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldCategory, field.TypeEnum, value)
	}
	if value, ok := bifflcuo.mutation.PaymentTerm(); ok {
		_spec.SetField(billinginvoiceflatfeelineconfig.FieldPaymentTerm, field.TypeEnum, value)
	}
	_node = &BillingInvoiceFlatFeeLineConfig{config: bifflcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, bifflcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceflatfeelineconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	bifflcuo.mutation.done = true
	return _node, nil
}
