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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
)

// SubscriptionPhaseUpdate is the builder for updating SubscriptionPhase entities.
type SubscriptionPhaseUpdate struct {
	config
	hooks    []Hook
	mutation *SubscriptionPhaseMutation
}

// Where appends a list predicates to the SubscriptionPhaseUpdate builder.
func (spu *SubscriptionPhaseUpdate) Where(ps ...predicate.SubscriptionPhase) *SubscriptionPhaseUpdate {
	spu.mutation.Where(ps...)
	return spu
}

// SetUpdatedAt sets the "updated_at" field.
func (spu *SubscriptionPhaseUpdate) SetUpdatedAt(t time.Time) *SubscriptionPhaseUpdate {
	spu.mutation.SetUpdatedAt(t)
	return spu
}

// SetDeletedAt sets the "deleted_at" field.
func (spu *SubscriptionPhaseUpdate) SetDeletedAt(t time.Time) *SubscriptionPhaseUpdate {
	spu.mutation.SetDeletedAt(t)
	return spu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (spu *SubscriptionPhaseUpdate) SetNillableDeletedAt(t *time.Time) *SubscriptionPhaseUpdate {
	if t != nil {
		spu.SetDeletedAt(*t)
	}
	return spu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (spu *SubscriptionPhaseUpdate) ClearDeletedAt() *SubscriptionPhaseUpdate {
	spu.mutation.ClearDeletedAt()
	return spu
}

// SetMetadata sets the "metadata" field.
func (spu *SubscriptionPhaseUpdate) SetMetadata(m map[string]string) *SubscriptionPhaseUpdate {
	spu.mutation.SetMetadata(m)
	return spu
}

// ClearMetadata clears the value of the "metadata" field.
func (spu *SubscriptionPhaseUpdate) ClearMetadata() *SubscriptionPhaseUpdate {
	spu.mutation.ClearMetadata()
	return spu
}

// SetName sets the "name" field.
func (spu *SubscriptionPhaseUpdate) SetName(s string) *SubscriptionPhaseUpdate {
	spu.mutation.SetName(s)
	return spu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (spu *SubscriptionPhaseUpdate) SetNillableName(s *string) *SubscriptionPhaseUpdate {
	if s != nil {
		spu.SetName(*s)
	}
	return spu
}

// SetDescription sets the "description" field.
func (spu *SubscriptionPhaseUpdate) SetDescription(s string) *SubscriptionPhaseUpdate {
	spu.mutation.SetDescription(s)
	return spu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (spu *SubscriptionPhaseUpdate) SetNillableDescription(s *string) *SubscriptionPhaseUpdate {
	if s != nil {
		spu.SetDescription(*s)
	}
	return spu
}

// ClearDescription clears the value of the "description" field.
func (spu *SubscriptionPhaseUpdate) ClearDescription() *SubscriptionPhaseUpdate {
	spu.mutation.ClearDescription()
	return spu
}

// AddItemIDs adds the "items" edge to the SubscriptionItem entity by IDs.
func (spu *SubscriptionPhaseUpdate) AddItemIDs(ids ...string) *SubscriptionPhaseUpdate {
	spu.mutation.AddItemIDs(ids...)
	return spu
}

// AddItems adds the "items" edges to the SubscriptionItem entity.
func (spu *SubscriptionPhaseUpdate) AddItems(s ...*SubscriptionItem) *SubscriptionPhaseUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return spu.AddItemIDs(ids...)
}

// AddBillingLineIDs adds the "billing_lines" edge to the BillingInvoiceLine entity by IDs.
func (spu *SubscriptionPhaseUpdate) AddBillingLineIDs(ids ...string) *SubscriptionPhaseUpdate {
	spu.mutation.AddBillingLineIDs(ids...)
	return spu
}

// AddBillingLines adds the "billing_lines" edges to the BillingInvoiceLine entity.
func (spu *SubscriptionPhaseUpdate) AddBillingLines(b ...*BillingInvoiceLine) *SubscriptionPhaseUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return spu.AddBillingLineIDs(ids...)
}

// Mutation returns the SubscriptionPhaseMutation object of the builder.
func (spu *SubscriptionPhaseUpdate) Mutation() *SubscriptionPhaseMutation {
	return spu.mutation
}

// ClearItems clears all "items" edges to the SubscriptionItem entity.
func (spu *SubscriptionPhaseUpdate) ClearItems() *SubscriptionPhaseUpdate {
	spu.mutation.ClearItems()
	return spu
}

// RemoveItemIDs removes the "items" edge to SubscriptionItem entities by IDs.
func (spu *SubscriptionPhaseUpdate) RemoveItemIDs(ids ...string) *SubscriptionPhaseUpdate {
	spu.mutation.RemoveItemIDs(ids...)
	return spu
}

// RemoveItems removes "items" edges to SubscriptionItem entities.
func (spu *SubscriptionPhaseUpdate) RemoveItems(s ...*SubscriptionItem) *SubscriptionPhaseUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return spu.RemoveItemIDs(ids...)
}

// ClearBillingLines clears all "billing_lines" edges to the BillingInvoiceLine entity.
func (spu *SubscriptionPhaseUpdate) ClearBillingLines() *SubscriptionPhaseUpdate {
	spu.mutation.ClearBillingLines()
	return spu
}

// RemoveBillingLineIDs removes the "billing_lines" edge to BillingInvoiceLine entities by IDs.
func (spu *SubscriptionPhaseUpdate) RemoveBillingLineIDs(ids ...string) *SubscriptionPhaseUpdate {
	spu.mutation.RemoveBillingLineIDs(ids...)
	return spu
}

// RemoveBillingLines removes "billing_lines" edges to BillingInvoiceLine entities.
func (spu *SubscriptionPhaseUpdate) RemoveBillingLines(b ...*BillingInvoiceLine) *SubscriptionPhaseUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return spu.RemoveBillingLineIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (spu *SubscriptionPhaseUpdate) Save(ctx context.Context) (int, error) {
	spu.defaults()
	return withHooks(ctx, spu.sqlSave, spu.mutation, spu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (spu *SubscriptionPhaseUpdate) SaveX(ctx context.Context) int {
	affected, err := spu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (spu *SubscriptionPhaseUpdate) Exec(ctx context.Context) error {
	_, err := spu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (spu *SubscriptionPhaseUpdate) ExecX(ctx context.Context) {
	if err := spu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (spu *SubscriptionPhaseUpdate) defaults() {
	if _, ok := spu.mutation.UpdatedAt(); !ok {
		v := subscriptionphase.UpdateDefaultUpdatedAt()
		spu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (spu *SubscriptionPhaseUpdate) check() error {
	if v, ok := spu.mutation.Name(); ok {
		if err := subscriptionphase.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "SubscriptionPhase.name": %w`, err)}
		}
	}
	if spu.mutation.SubscriptionCleared() && len(spu.mutation.SubscriptionIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionPhase.subscription"`)
	}
	return nil
}

func (spu *SubscriptionPhaseUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := spu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionphase.Table, subscriptionphase.Columns, sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString))
	if ps := spu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := spu.mutation.UpdatedAt(); ok {
		_spec.SetField(subscriptionphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := spu.mutation.DeletedAt(); ok {
		_spec.SetField(subscriptionphase.FieldDeletedAt, field.TypeTime, value)
	}
	if spu.mutation.DeletedAtCleared() {
		_spec.ClearField(subscriptionphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := spu.mutation.Metadata(); ok {
		_spec.SetField(subscriptionphase.FieldMetadata, field.TypeJSON, value)
	}
	if spu.mutation.MetadataCleared() {
		_spec.ClearField(subscriptionphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := spu.mutation.Name(); ok {
		_spec.SetField(subscriptionphase.FieldName, field.TypeString, value)
	}
	if value, ok := spu.mutation.Description(); ok {
		_spec.SetField(subscriptionphase.FieldDescription, field.TypeString, value)
	}
	if spu.mutation.DescriptionCleared() {
		_spec.ClearField(subscriptionphase.FieldDescription, field.TypeString)
	}
	if spu.mutation.ItemsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spu.mutation.RemovedItemsIDs(); len(nodes) > 0 && !spu.mutation.ItemsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spu.mutation.ItemsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if spu.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spu.mutation.RemovedBillingLinesIDs(); len(nodes) > 0 && !spu.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spu.mutation.BillingLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, spu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	spu.mutation.done = true
	return n, nil
}

// SubscriptionPhaseUpdateOne is the builder for updating a single SubscriptionPhase entity.
type SubscriptionPhaseUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *SubscriptionPhaseMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (spuo *SubscriptionPhaseUpdateOne) SetUpdatedAt(t time.Time) *SubscriptionPhaseUpdateOne {
	spuo.mutation.SetUpdatedAt(t)
	return spuo
}

// SetDeletedAt sets the "deleted_at" field.
func (spuo *SubscriptionPhaseUpdateOne) SetDeletedAt(t time.Time) *SubscriptionPhaseUpdateOne {
	spuo.mutation.SetDeletedAt(t)
	return spuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (spuo *SubscriptionPhaseUpdateOne) SetNillableDeletedAt(t *time.Time) *SubscriptionPhaseUpdateOne {
	if t != nil {
		spuo.SetDeletedAt(*t)
	}
	return spuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (spuo *SubscriptionPhaseUpdateOne) ClearDeletedAt() *SubscriptionPhaseUpdateOne {
	spuo.mutation.ClearDeletedAt()
	return spuo
}

// SetMetadata sets the "metadata" field.
func (spuo *SubscriptionPhaseUpdateOne) SetMetadata(m map[string]string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.SetMetadata(m)
	return spuo
}

// ClearMetadata clears the value of the "metadata" field.
func (spuo *SubscriptionPhaseUpdateOne) ClearMetadata() *SubscriptionPhaseUpdateOne {
	spuo.mutation.ClearMetadata()
	return spuo
}

// SetName sets the "name" field.
func (spuo *SubscriptionPhaseUpdateOne) SetName(s string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.SetName(s)
	return spuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (spuo *SubscriptionPhaseUpdateOne) SetNillableName(s *string) *SubscriptionPhaseUpdateOne {
	if s != nil {
		spuo.SetName(*s)
	}
	return spuo
}

// SetDescription sets the "description" field.
func (spuo *SubscriptionPhaseUpdateOne) SetDescription(s string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.SetDescription(s)
	return spuo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (spuo *SubscriptionPhaseUpdateOne) SetNillableDescription(s *string) *SubscriptionPhaseUpdateOne {
	if s != nil {
		spuo.SetDescription(*s)
	}
	return spuo
}

// ClearDescription clears the value of the "description" field.
func (spuo *SubscriptionPhaseUpdateOne) ClearDescription() *SubscriptionPhaseUpdateOne {
	spuo.mutation.ClearDescription()
	return spuo
}

// AddItemIDs adds the "items" edge to the SubscriptionItem entity by IDs.
func (spuo *SubscriptionPhaseUpdateOne) AddItemIDs(ids ...string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.AddItemIDs(ids...)
	return spuo
}

// AddItems adds the "items" edges to the SubscriptionItem entity.
func (spuo *SubscriptionPhaseUpdateOne) AddItems(s ...*SubscriptionItem) *SubscriptionPhaseUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return spuo.AddItemIDs(ids...)
}

// AddBillingLineIDs adds the "billing_lines" edge to the BillingInvoiceLine entity by IDs.
func (spuo *SubscriptionPhaseUpdateOne) AddBillingLineIDs(ids ...string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.AddBillingLineIDs(ids...)
	return spuo
}

// AddBillingLines adds the "billing_lines" edges to the BillingInvoiceLine entity.
func (spuo *SubscriptionPhaseUpdateOne) AddBillingLines(b ...*BillingInvoiceLine) *SubscriptionPhaseUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return spuo.AddBillingLineIDs(ids...)
}

// Mutation returns the SubscriptionPhaseMutation object of the builder.
func (spuo *SubscriptionPhaseUpdateOne) Mutation() *SubscriptionPhaseMutation {
	return spuo.mutation
}

// ClearItems clears all "items" edges to the SubscriptionItem entity.
func (spuo *SubscriptionPhaseUpdateOne) ClearItems() *SubscriptionPhaseUpdateOne {
	spuo.mutation.ClearItems()
	return spuo
}

// RemoveItemIDs removes the "items" edge to SubscriptionItem entities by IDs.
func (spuo *SubscriptionPhaseUpdateOne) RemoveItemIDs(ids ...string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.RemoveItemIDs(ids...)
	return spuo
}

// RemoveItems removes "items" edges to SubscriptionItem entities.
func (spuo *SubscriptionPhaseUpdateOne) RemoveItems(s ...*SubscriptionItem) *SubscriptionPhaseUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return spuo.RemoveItemIDs(ids...)
}

// ClearBillingLines clears all "billing_lines" edges to the BillingInvoiceLine entity.
func (spuo *SubscriptionPhaseUpdateOne) ClearBillingLines() *SubscriptionPhaseUpdateOne {
	spuo.mutation.ClearBillingLines()
	return spuo
}

// RemoveBillingLineIDs removes the "billing_lines" edge to BillingInvoiceLine entities by IDs.
func (spuo *SubscriptionPhaseUpdateOne) RemoveBillingLineIDs(ids ...string) *SubscriptionPhaseUpdateOne {
	spuo.mutation.RemoveBillingLineIDs(ids...)
	return spuo
}

// RemoveBillingLines removes "billing_lines" edges to BillingInvoiceLine entities.
func (spuo *SubscriptionPhaseUpdateOne) RemoveBillingLines(b ...*BillingInvoiceLine) *SubscriptionPhaseUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return spuo.RemoveBillingLineIDs(ids...)
}

// Where appends a list predicates to the SubscriptionPhaseUpdate builder.
func (spuo *SubscriptionPhaseUpdateOne) Where(ps ...predicate.SubscriptionPhase) *SubscriptionPhaseUpdateOne {
	spuo.mutation.Where(ps...)
	return spuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (spuo *SubscriptionPhaseUpdateOne) Select(field string, fields ...string) *SubscriptionPhaseUpdateOne {
	spuo.fields = append([]string{field}, fields...)
	return spuo
}

// Save executes the query and returns the updated SubscriptionPhase entity.
func (spuo *SubscriptionPhaseUpdateOne) Save(ctx context.Context) (*SubscriptionPhase, error) {
	spuo.defaults()
	return withHooks(ctx, spuo.sqlSave, spuo.mutation, spuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (spuo *SubscriptionPhaseUpdateOne) SaveX(ctx context.Context) *SubscriptionPhase {
	node, err := spuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (spuo *SubscriptionPhaseUpdateOne) Exec(ctx context.Context) error {
	_, err := spuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (spuo *SubscriptionPhaseUpdateOne) ExecX(ctx context.Context) {
	if err := spuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (spuo *SubscriptionPhaseUpdateOne) defaults() {
	if _, ok := spuo.mutation.UpdatedAt(); !ok {
		v := subscriptionphase.UpdateDefaultUpdatedAt()
		spuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (spuo *SubscriptionPhaseUpdateOne) check() error {
	if v, ok := spuo.mutation.Name(); ok {
		if err := subscriptionphase.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "SubscriptionPhase.name": %w`, err)}
		}
	}
	if spuo.mutation.SubscriptionCleared() && len(spuo.mutation.SubscriptionIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "SubscriptionPhase.subscription"`)
	}
	return nil
}

func (spuo *SubscriptionPhaseUpdateOne) sqlSave(ctx context.Context) (_node *SubscriptionPhase, err error) {
	if err := spuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscriptionphase.Table, subscriptionphase.Columns, sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString))
	id, ok := spuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "SubscriptionPhase.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := spuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionphase.FieldID)
		for _, f := range fields {
			if !subscriptionphase.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != subscriptionphase.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := spuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := spuo.mutation.UpdatedAt(); ok {
		_spec.SetField(subscriptionphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := spuo.mutation.DeletedAt(); ok {
		_spec.SetField(subscriptionphase.FieldDeletedAt, field.TypeTime, value)
	}
	if spuo.mutation.DeletedAtCleared() {
		_spec.ClearField(subscriptionphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := spuo.mutation.Metadata(); ok {
		_spec.SetField(subscriptionphase.FieldMetadata, field.TypeJSON, value)
	}
	if spuo.mutation.MetadataCleared() {
		_spec.ClearField(subscriptionphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := spuo.mutation.Name(); ok {
		_spec.SetField(subscriptionphase.FieldName, field.TypeString, value)
	}
	if value, ok := spuo.mutation.Description(); ok {
		_spec.SetField(subscriptionphase.FieldDescription, field.TypeString, value)
	}
	if spuo.mutation.DescriptionCleared() {
		_spec.ClearField(subscriptionphase.FieldDescription, field.TypeString)
	}
	if spuo.mutation.ItemsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spuo.mutation.RemovedItemsIDs(); len(nodes) > 0 && !spuo.mutation.ItemsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spuo.mutation.ItemsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.ItemsTable,
			Columns: []string{subscriptionphase.ItemsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionitem.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if spuo.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spuo.mutation.RemovedBillingLinesIDs(); len(nodes) > 0 && !spuo.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := spuo.mutation.BillingLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscriptionphase.BillingLinesTable,
			Columns: []string{subscriptionphase.BillingLinesColumn},
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
	_node = &SubscriptionPhase{config: spuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, spuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscriptionphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	spuo.mutation.done = true
	return _node, nil
}
