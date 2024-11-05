// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueextendphase"
)

// SubscriptionPatchValueExtendPhaseUpdate is the builder for updating SubscriptionPatchValueExtendPhase entities.
type SubscriptionPatchValueExtendPhaseUpdate struct {
	config
	hooks    []Hook
	mutation *SubscriptionPatchValueExtendPhaseMutation
}

// Where appends a list predicates to the SubscriptionPatchValueExtendPhaseUpdate builder.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) Where(ps ...predicate.SubscriptionPatchValueExtendPhase) *SubscriptionPatchValueExtendPhaseUpdate {
	spvepu.mutation.Where(ps...)
	return spvepu
}

// Mutation returns the SubscriptionPatchValueExtendPhaseMutation object of the builder.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) Mutation() *SubscriptionPatchValueExtendPhaseMutation {
	return spvepu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, spvepu.sqlSave, spvepu.mutation, spvepu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) SaveX(ctx context.Context) int {
	affected, err := spvepu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) Exec(ctx context.Context) error {
	_, err := spvepu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) ExecX(ctx context.Context) {
	if err := spvepu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) check() error {
	if spvepu.mutation.SubscriptionPatchCleared() && len(spvepu.mutation.SubscriptionPatchIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionPatchValueExtendPhase.subscription_patch"`)
	}
	return nil
}

func (spvepu *SubscriptionPatchValueExtendPhaseUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := spvepu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionpatchvalueextendphase.Table, subscriptionpatchvalueextendphase.Columns, sqlgraph.NewFieldSpec(subscriptionpatchvalueextendphase.FieldID, field.TypeString))
	if ps := spvepu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if n, err = sqlgraph.UpdateNodes(ctx, spvepu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionpatchvalueextendphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	spvepu.mutation.done = true
	return n, nil
}

// SubscriptionPatchValueExtendPhaseUpdateOne is the builder for updating a single SubscriptionPatchValueExtendPhase entity.
type SubscriptionPatchValueExtendPhaseUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *SubscriptionPatchValueExtendPhaseMutation
}

// Mutation returns the SubscriptionPatchValueExtendPhaseMutation object of the builder.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) Mutation() *SubscriptionPatchValueExtendPhaseMutation {
	return spvepuo.mutation
}

// Where appends a list predicates to the SubscriptionPatchValueExtendPhaseUpdate builder.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) Where(ps ...predicate.SubscriptionPatchValueExtendPhase) *SubscriptionPatchValueExtendPhaseUpdateOne {
	spvepuo.mutation.Where(ps...)
	return spvepuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) Select(field string, fields ...string) *SubscriptionPatchValueExtendPhaseUpdateOne {
	spvepuo.fields = append([]string{field}, fields...)
	return spvepuo
}

// Save executes the query and returns the updated SubscriptionPatchValueExtendPhase entity.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) Save(ctx context.Context) (*SubscriptionPatchValueExtendPhase, error) {
	return withHooks(ctx, spvepuo.sqlSave, spvepuo.mutation, spvepuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) SaveX(ctx context.Context) *SubscriptionPatchValueExtendPhase {
	node, err := spvepuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) Exec(ctx context.Context) error {
	_, err := spvepuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) ExecX(ctx context.Context) {
	if err := spvepuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) check() error {
	if spvepuo.mutation.SubscriptionPatchCleared() && len(spvepuo.mutation.SubscriptionPatchIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionPatchValueExtendPhase.subscription_patch"`)
	}
	return nil
}

func (spvepuo *SubscriptionPatchValueExtendPhaseUpdateOne) sqlSave(ctx context.Context) (_node *SubscriptionPatchValueExtendPhase, err error) {
	if err := spvepuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionpatchvalueextendphase.Table, subscriptionpatchvalueextendphase.Columns, sqlgraph.NewFieldSpec(subscriptionpatchvalueextendphase.FieldID, field.TypeString))
	id, ok := spvepuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "SubscriptionPatchValueExtendPhase.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := spvepuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionpatchvalueextendphase.FieldID)
		for _, f := range fields {
			if !subscriptionpatchvalueextendphase.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != subscriptionpatchvalueextendphase.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := spvepuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	_node = &SubscriptionPatchValueExtendPhase{config: spvepuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, spvepuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionpatchvalueextendphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	spvepuo.mutation.done = true
	return _node, nil
}