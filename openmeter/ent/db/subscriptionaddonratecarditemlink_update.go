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
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonratecarditemlink"
)

// SubscriptionAddonRateCardItemLinkUpdate is the builder for updating SubscriptionAddonRateCardItemLink entities.
type SubscriptionAddonRateCardItemLinkUpdate struct {
	config
	hooks    []Hook
	mutation *SubscriptionAddonRateCardItemLinkMutation
}

// Where appends a list predicates to the SubscriptionAddonRateCardItemLinkUpdate builder.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) Where(ps ...predicate.SubscriptionAddonRateCardItemLink) *SubscriptionAddonRateCardItemLinkUpdate {
	sarcilu.mutation.Where(ps...)
	return sarcilu
}

// SetUpdatedAt sets the "updated_at" field.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) SetUpdatedAt(t time.Time) *SubscriptionAddonRateCardItemLinkUpdate {
	sarcilu.mutation.SetUpdatedAt(t)
	return sarcilu
}

// SetDeletedAt sets the "deleted_at" field.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) SetDeletedAt(t time.Time) *SubscriptionAddonRateCardItemLinkUpdate {
	sarcilu.mutation.SetDeletedAt(t)
	return sarcilu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) SetNillableDeletedAt(t *time.Time) *SubscriptionAddonRateCardItemLinkUpdate {
	if t != nil {
		sarcilu.SetDeletedAt(*t)
	}
	return sarcilu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) ClearDeletedAt() *SubscriptionAddonRateCardItemLinkUpdate {
	sarcilu.mutation.ClearDeletedAt()
	return sarcilu
}

// Mutation returns the SubscriptionAddonRateCardItemLinkMutation object of the builder.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) Mutation() *SubscriptionAddonRateCardItemLinkMutation {
	return sarcilu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) Save(ctx context.Context) (int, error) {
	sarcilu.defaults()
	return withHooks(ctx, sarcilu.sqlSave, sarcilu.mutation, sarcilu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) SaveX(ctx context.Context) int {
	affected, err := sarcilu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) Exec(ctx context.Context) error {
	_, err := sarcilu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) ExecX(ctx context.Context) {
	if err := sarcilu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) defaults() {
	if _, ok := sarcilu.mutation.UpdatedAt(); !ok {
		v := subscriptionaddonratecarditemlink.UpdateDefaultUpdatedAt()
		sarcilu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) check() error {
	if sarcilu.mutation.SubscriptionAddonRateCardCleared() && len(sarcilu.mutation.SubscriptionAddonRateCardIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionAddonRateCardItemLink.subscription_addon_rate_card"`)
	}
	if sarcilu.mutation.SubscriptionItemCleared() && len(sarcilu.mutation.SubscriptionItemIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionAddonRateCardItemLink.subscription_item"`)
	}
	return nil
}

func (sarcilu *SubscriptionAddonRateCardItemLinkUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := sarcilu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionaddonratecarditemlink.Table, subscriptionaddonratecarditemlink.Columns, sqlgraph.NewFieldSpec(subscriptionaddonratecarditemlink.FieldID, field.TypeString))
	if ps := sarcilu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := sarcilu.mutation.UpdatedAt(); ok {
		_spec.SetField(subscriptionaddonratecarditemlink.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := sarcilu.mutation.DeletedAt(); ok {
		_spec.SetField(subscriptionaddonratecarditemlink.FieldDeletedAt, field.TypeTime, value)
	}
	if sarcilu.mutation.DeletedAtCleared() {
		_spec.ClearField(subscriptionaddonratecarditemlink.FieldDeletedAt, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, sarcilu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionaddonratecarditemlink.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	sarcilu.mutation.done = true
	return n, nil
}

// SubscriptionAddonRateCardItemLinkUpdateOne is the builder for updating a single SubscriptionAddonRateCardItemLink entity.
type SubscriptionAddonRateCardItemLinkUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *SubscriptionAddonRateCardItemLinkMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) SetUpdatedAt(t time.Time) *SubscriptionAddonRateCardItemLinkUpdateOne {
	sarciluo.mutation.SetUpdatedAt(t)
	return sarciluo
}

// SetDeletedAt sets the "deleted_at" field.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) SetDeletedAt(t time.Time) *SubscriptionAddonRateCardItemLinkUpdateOne {
	sarciluo.mutation.SetDeletedAt(t)
	return sarciluo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) SetNillableDeletedAt(t *time.Time) *SubscriptionAddonRateCardItemLinkUpdateOne {
	if t != nil {
		sarciluo.SetDeletedAt(*t)
	}
	return sarciluo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) ClearDeletedAt() *SubscriptionAddonRateCardItemLinkUpdateOne {
	sarciluo.mutation.ClearDeletedAt()
	return sarciluo
}

// Mutation returns the SubscriptionAddonRateCardItemLinkMutation object of the builder.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) Mutation() *SubscriptionAddonRateCardItemLinkMutation {
	return sarciluo.mutation
}

// Where appends a list predicates to the SubscriptionAddonRateCardItemLinkUpdate builder.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) Where(ps ...predicate.SubscriptionAddonRateCardItemLink) *SubscriptionAddonRateCardItemLinkUpdateOne {
	sarciluo.mutation.Where(ps...)
	return sarciluo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) Select(field string, fields ...string) *SubscriptionAddonRateCardItemLinkUpdateOne {
	sarciluo.fields = append([]string{field}, fields...)
	return sarciluo
}

// Save executes the query and returns the updated SubscriptionAddonRateCardItemLink entity.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) Save(ctx context.Context) (*SubscriptionAddonRateCardItemLink, error) {
	sarciluo.defaults()
	return withHooks(ctx, sarciluo.sqlSave, sarciluo.mutation, sarciluo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) SaveX(ctx context.Context) *SubscriptionAddonRateCardItemLink {
	node, err := sarciluo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) Exec(ctx context.Context) error {
	_, err := sarciluo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) ExecX(ctx context.Context) {
	if err := sarciluo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) defaults() {
	if _, ok := sarciluo.mutation.UpdatedAt(); !ok {
		v := subscriptionaddonratecarditemlink.UpdateDefaultUpdatedAt()
		sarciluo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) check() error {
	if sarciluo.mutation.SubscriptionAddonRateCardCleared() && len(sarciluo.mutation.SubscriptionAddonRateCardIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionAddonRateCardItemLink.subscription_addon_rate_card"`)
	}
	if sarciluo.mutation.SubscriptionItemCleared() && len(sarciluo.mutation.SubscriptionItemIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionAddonRateCardItemLink.subscription_item"`)
	}
	return nil
}

func (sarciluo *SubscriptionAddonRateCardItemLinkUpdateOne) sqlSave(ctx context.Context) (_node *SubscriptionAddonRateCardItemLink, err error) {
	if err := sarciluo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionaddonratecarditemlink.Table, subscriptionaddonratecarditemlink.Columns, sqlgraph.NewFieldSpec(subscriptionaddonratecarditemlink.FieldID, field.TypeString))
	id, ok := sarciluo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "SubscriptionAddonRateCardItemLink.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := sarciluo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionaddonratecarditemlink.FieldID)
		for _, f := range fields {
			if !subscriptionaddonratecarditemlink.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != subscriptionaddonratecarditemlink.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := sarciluo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := sarciluo.mutation.UpdatedAt(); ok {
		_spec.SetField(subscriptionaddonratecarditemlink.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := sarciluo.mutation.DeletedAt(); ok {
		_spec.SetField(subscriptionaddonratecarditemlink.FieldDeletedAt, field.TypeTime, value)
	}
	if sarciluo.mutation.DeletedAtCleared() {
		_spec.ClearField(subscriptionaddonratecarditemlink.FieldDeletedAt, field.TypeTime)
	}
	_node = &SubscriptionAddonRateCardItemLink{config: sarciluo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, sarciluo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionaddonratecarditemlink.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	sarciluo.mutation.done = true
	return _node, nil
}
