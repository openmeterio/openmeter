// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

// BillingInvoiceUsageBasedLineConfigUpdate is the builder for updating BillingInvoiceUsageBasedLineConfig entities.
type BillingInvoiceUsageBasedLineConfigUpdate struct {
	config
	hooks    []Hook
	mutation *BillingInvoiceUsageBasedLineConfigMutation
}

// Where appends a list predicates to the BillingInvoiceUsageBasedLineConfigUpdate builder.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) Where(ps ...predicate.BillingInvoiceUsageBasedLineConfig) *BillingInvoiceUsageBasedLineConfigUpdate {
	biublcu.mutation.Where(ps...)
	return biublcu
}

// SetPriceType sets the "price_type" field.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) SetPriceType(pt plan.PriceType) *BillingInvoiceUsageBasedLineConfigUpdate {
	biublcu.mutation.SetPriceType(pt)
	return biublcu
}

// SetNillablePriceType sets the "price_type" field if the given value is not nil.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) SetNillablePriceType(pt *plan.PriceType) *BillingInvoiceUsageBasedLineConfigUpdate {
	if pt != nil {
		biublcu.SetPriceType(*pt)
	}
	return biublcu
}

// SetPrice sets the "price" field.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) SetPrice(pl *plan.Price) *BillingInvoiceUsageBasedLineConfigUpdate {
	biublcu.mutation.SetPrice(pl)
	return biublcu
}

// Mutation returns the BillingInvoiceUsageBasedLineConfigMutation object of the builder.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) Mutation() *BillingInvoiceUsageBasedLineConfigMutation {
	return biublcu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, biublcu.sqlSave, biublcu.mutation, biublcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) SaveX(ctx context.Context) int {
	affected, err := biublcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) Exec(ctx context.Context) error {
	_, err := biublcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) ExecX(ctx context.Context) {
	if err := biublcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) check() error {
	if v, ok := biublcu.mutation.PriceType(); ok {
		if err := billinginvoiceusagebasedlineconfig.PriceTypeValidator(v); err != nil {
			return &ValidationError{Name: "price_type", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceUsageBasedLineConfig.price_type": %w`, err)}
		}
	}
	if v, ok := biublcu.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceUsageBasedLineConfig.price": %w`, err)}
		}
	}
	return nil
}

func (biublcu *BillingInvoiceUsageBasedLineConfigUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := biublcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceusagebasedlineconfig.Table, billinginvoiceusagebasedlineconfig.Columns, sqlgraph.NewFieldSpec(billinginvoiceusagebasedlineconfig.FieldID, field.TypeString))
	if ps := biublcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := biublcu.mutation.PriceType(); ok {
		_spec.SetField(billinginvoiceusagebasedlineconfig.FieldPriceType, field.TypeEnum, value)
	}
	if value, ok := biublcu.mutation.Price(); ok {
		vv, err := billinginvoiceusagebasedlineconfig.ValueScanner.Price.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(billinginvoiceusagebasedlineconfig.FieldPrice, field.TypeString, vv)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, biublcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceusagebasedlineconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	biublcu.mutation.done = true
	return n, nil
}

// BillingInvoiceUsageBasedLineConfigUpdateOne is the builder for updating a single BillingInvoiceUsageBasedLineConfig entity.
type BillingInvoiceUsageBasedLineConfigUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *BillingInvoiceUsageBasedLineConfigMutation
}

// SetPriceType sets the "price_type" field.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) SetPriceType(pt plan.PriceType) *BillingInvoiceUsageBasedLineConfigUpdateOne {
	biublcuo.mutation.SetPriceType(pt)
	return biublcuo
}

// SetNillablePriceType sets the "price_type" field if the given value is not nil.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) SetNillablePriceType(pt *plan.PriceType) *BillingInvoiceUsageBasedLineConfigUpdateOne {
	if pt != nil {
		biublcuo.SetPriceType(*pt)
	}
	return biublcuo
}

// SetPrice sets the "price" field.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) SetPrice(pl *plan.Price) *BillingInvoiceUsageBasedLineConfigUpdateOne {
	biublcuo.mutation.SetPrice(pl)
	return biublcuo
}

// Mutation returns the BillingInvoiceUsageBasedLineConfigMutation object of the builder.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) Mutation() *BillingInvoiceUsageBasedLineConfigMutation {
	return biublcuo.mutation
}

// Where appends a list predicates to the BillingInvoiceUsageBasedLineConfigUpdate builder.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) Where(ps ...predicate.BillingInvoiceUsageBasedLineConfig) *BillingInvoiceUsageBasedLineConfigUpdateOne {
	biublcuo.mutation.Where(ps...)
	return biublcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) Select(field string, fields ...string) *BillingInvoiceUsageBasedLineConfigUpdateOne {
	biublcuo.fields = append([]string{field}, fields...)
	return biublcuo
}

// Save executes the query and returns the updated BillingInvoiceUsageBasedLineConfig entity.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) Save(ctx context.Context) (*BillingInvoiceUsageBasedLineConfig, error) {
	return withHooks(ctx, biublcuo.sqlSave, biublcuo.mutation, biublcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) SaveX(ctx context.Context) *BillingInvoiceUsageBasedLineConfig {
	node, err := biublcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) Exec(ctx context.Context) error {
	_, err := biublcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) ExecX(ctx context.Context) {
	if err := biublcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) check() error {
	if v, ok := biublcuo.mutation.PriceType(); ok {
		if err := billinginvoiceusagebasedlineconfig.PriceTypeValidator(v); err != nil {
			return &ValidationError{Name: "price_type", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceUsageBasedLineConfig.price_type": %w`, err)}
		}
	}
	if v, ok := biublcuo.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "BillingInvoiceUsageBasedLineConfig.price": %w`, err)}
		}
	}
	return nil
}

func (biublcuo *BillingInvoiceUsageBasedLineConfigUpdateOne) sqlSave(ctx context.Context) (_node *BillingInvoiceUsageBasedLineConfig, err error) {
	if err := biublcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(billinginvoiceusagebasedlineconfig.Table, billinginvoiceusagebasedlineconfig.Columns, sqlgraph.NewFieldSpec(billinginvoiceusagebasedlineconfig.FieldID, field.TypeString))
	id, ok := biublcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "BillingInvoiceUsageBasedLineConfig.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := biublcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoiceusagebasedlineconfig.FieldID)
		for _, f := range fields {
			if !billinginvoiceusagebasedlineconfig.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != billinginvoiceusagebasedlineconfig.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := biublcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := biublcuo.mutation.PriceType(); ok {
		_spec.SetField(billinginvoiceusagebasedlineconfig.FieldPriceType, field.TypeEnum, value)
	}
	if value, ok := biublcuo.mutation.Price(); ok {
		vv, err := billinginvoiceusagebasedlineconfig.ValueScanner.Price.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(billinginvoiceusagebasedlineconfig.FieldPrice, field.TypeString, vv)
	}
	_node = &BillingInvoiceUsageBasedLineConfig{config: biublcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, biublcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{billinginvoiceusagebasedlineconfig.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	biublcuo.mutation.done = true
	return _node, nil
}