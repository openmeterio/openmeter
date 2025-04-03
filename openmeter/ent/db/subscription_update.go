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
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
)

// SubscriptionUpdate is the builder for updating Subscription entities.
type SubscriptionUpdate struct {
	config
	hooks    []Hook
	mutation *SubscriptionMutation
}

// Where appends a list predicates to the SubscriptionUpdate builder.
func (su *SubscriptionUpdate) Where(ps ...predicate.Subscription) *SubscriptionUpdate {
	su.mutation.Where(ps...)
	return su
}

// SetUpdatedAt sets the "updated_at" field.
func (su *SubscriptionUpdate) SetUpdatedAt(t time.Time) *SubscriptionUpdate {
	su.mutation.SetUpdatedAt(t)
	return su
}

// SetDeletedAt sets the "deleted_at" field.
func (su *SubscriptionUpdate) SetDeletedAt(t time.Time) *SubscriptionUpdate {
	su.mutation.SetDeletedAt(t)
	return su
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillableDeletedAt(t *time.Time) *SubscriptionUpdate {
	if t != nil {
		su.SetDeletedAt(*t)
	}
	return su
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (su *SubscriptionUpdate) ClearDeletedAt() *SubscriptionUpdate {
	su.mutation.ClearDeletedAt()
	return su
}

// SetMetadata sets the "metadata" field.
func (su *SubscriptionUpdate) SetMetadata(m map[string]string) *SubscriptionUpdate {
	su.mutation.SetMetadata(m)
	return su
}

// ClearMetadata clears the value of the "metadata" field.
func (su *SubscriptionUpdate) ClearMetadata() *SubscriptionUpdate {
	su.mutation.ClearMetadata()
	return su
}

// SetActiveTo sets the "active_to" field.
func (su *SubscriptionUpdate) SetActiveTo(t time.Time) *SubscriptionUpdate {
	su.mutation.SetActiveTo(t)
	return su
}

// SetNillableActiveTo sets the "active_to" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillableActiveTo(t *time.Time) *SubscriptionUpdate {
	if t != nil {
		su.SetActiveTo(*t)
	}
	return su
}

// ClearActiveTo clears the value of the "active_to" field.
func (su *SubscriptionUpdate) ClearActiveTo() *SubscriptionUpdate {
	su.mutation.ClearActiveTo()
	return su
}

// SetBillablesMustAlign sets the "billables_must_align" field.
func (su *SubscriptionUpdate) SetBillablesMustAlign(b bool) *SubscriptionUpdate {
	su.mutation.SetBillablesMustAlign(b)
	return su
}

// SetNillableBillablesMustAlign sets the "billables_must_align" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillableBillablesMustAlign(b *bool) *SubscriptionUpdate {
	if b != nil {
		su.SetBillablesMustAlign(*b)
	}
	return su
}

// SetName sets the "name" field.
func (su *SubscriptionUpdate) SetName(s string) *SubscriptionUpdate {
	su.mutation.SetName(s)
	return su
}

// SetNillableName sets the "name" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillableName(s *string) *SubscriptionUpdate {
	if s != nil {
		su.SetName(*s)
	}
	return su
}

// SetDescription sets the "description" field.
func (su *SubscriptionUpdate) SetDescription(s string) *SubscriptionUpdate {
	su.mutation.SetDescription(s)
	return su
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillableDescription(s *string) *SubscriptionUpdate {
	if s != nil {
		su.SetDescription(*s)
	}
	return su
}

// ClearDescription clears the value of the "description" field.
func (su *SubscriptionUpdate) ClearDescription() *SubscriptionUpdate {
	su.mutation.ClearDescription()
	return su
}

// SetPlanID sets the "plan_id" field.
func (su *SubscriptionUpdate) SetPlanID(s string) *SubscriptionUpdate {
	su.mutation.SetPlanID(s)
	return su
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (su *SubscriptionUpdate) SetNillablePlanID(s *string) *SubscriptionUpdate {
	if s != nil {
		su.SetPlanID(*s)
	}
	return su
}

// ClearPlanID clears the value of the "plan_id" field.
func (su *SubscriptionUpdate) ClearPlanID() *SubscriptionUpdate {
	su.mutation.ClearPlanID()
	return su
}

// SetPlan sets the "plan" edge to the Plan entity.
func (su *SubscriptionUpdate) SetPlan(p *Plan) *SubscriptionUpdate {
	return su.SetPlanID(p.ID)
}

// AddPhaseIDs adds the "phases" edge to the SubscriptionPhase entity by IDs.
func (su *SubscriptionUpdate) AddPhaseIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.AddPhaseIDs(ids...)
	return su
}

// AddPhases adds the "phases" edges to the SubscriptionPhase entity.
func (su *SubscriptionUpdate) AddPhases(s ...*SubscriptionPhase) *SubscriptionUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return su.AddPhaseIDs(ids...)
}

// AddBillingLineIDs adds the "billing_lines" edge to the BillingInvoiceLine entity by IDs.
func (su *SubscriptionUpdate) AddBillingLineIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.AddBillingLineIDs(ids...)
	return su
}

// AddBillingLines adds the "billing_lines" edges to the BillingInvoiceLine entity.
func (su *SubscriptionUpdate) AddBillingLines(b ...*BillingInvoiceLine) *SubscriptionUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return su.AddBillingLineIDs(ids...)
}

// AddAddonIDs adds the "addons" edge to the SubscriptionAddon entity by IDs.
func (su *SubscriptionUpdate) AddAddonIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.AddAddonIDs(ids...)
	return su
}

// AddAddons adds the "addons" edges to the SubscriptionAddon entity.
func (su *SubscriptionUpdate) AddAddons(s ...*SubscriptionAddon) *SubscriptionUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return su.AddAddonIDs(ids...)
}

// Mutation returns the SubscriptionMutation object of the builder.
func (su *SubscriptionUpdate) Mutation() *SubscriptionMutation {
	return su.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (su *SubscriptionUpdate) ClearPlan() *SubscriptionUpdate {
	su.mutation.ClearPlan()
	return su
}

// ClearPhases clears all "phases" edges to the SubscriptionPhase entity.
func (su *SubscriptionUpdate) ClearPhases() *SubscriptionUpdate {
	su.mutation.ClearPhases()
	return su
}

// RemovePhaseIDs removes the "phases" edge to SubscriptionPhase entities by IDs.
func (su *SubscriptionUpdate) RemovePhaseIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.RemovePhaseIDs(ids...)
	return su
}

// RemovePhases removes "phases" edges to SubscriptionPhase entities.
func (su *SubscriptionUpdate) RemovePhases(s ...*SubscriptionPhase) *SubscriptionUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return su.RemovePhaseIDs(ids...)
}

// ClearBillingLines clears all "billing_lines" edges to the BillingInvoiceLine entity.
func (su *SubscriptionUpdate) ClearBillingLines() *SubscriptionUpdate {
	su.mutation.ClearBillingLines()
	return su
}

// RemoveBillingLineIDs removes the "billing_lines" edge to BillingInvoiceLine entities by IDs.
func (su *SubscriptionUpdate) RemoveBillingLineIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.RemoveBillingLineIDs(ids...)
	return su
}

// RemoveBillingLines removes "billing_lines" edges to BillingInvoiceLine entities.
func (su *SubscriptionUpdate) RemoveBillingLines(b ...*BillingInvoiceLine) *SubscriptionUpdate {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return su.RemoveBillingLineIDs(ids...)
}

// ClearAddons clears all "addons" edges to the SubscriptionAddon entity.
func (su *SubscriptionUpdate) ClearAddons() *SubscriptionUpdate {
	su.mutation.ClearAddons()
	return su
}

// RemoveAddonIDs removes the "addons" edge to SubscriptionAddon entities by IDs.
func (su *SubscriptionUpdate) RemoveAddonIDs(ids ...string) *SubscriptionUpdate {
	su.mutation.RemoveAddonIDs(ids...)
	return su
}

// RemoveAddons removes "addons" edges to SubscriptionAddon entities.
func (su *SubscriptionUpdate) RemoveAddons(s ...*SubscriptionAddon) *SubscriptionUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return su.RemoveAddonIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (su *SubscriptionUpdate) Save(ctx context.Context) (int, error) {
	su.defaults()
	return withHooks(ctx, su.sqlSave, su.mutation, su.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (su *SubscriptionUpdate) SaveX(ctx context.Context) int {
	affected, err := su.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (su *SubscriptionUpdate) Exec(ctx context.Context) error {
	_, err := su.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (su *SubscriptionUpdate) ExecX(ctx context.Context) {
	if err := su.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (su *SubscriptionUpdate) defaults() {
	if _, ok := su.mutation.UpdatedAt(); !ok {
		v := subscription.UpdateDefaultUpdatedAt()
		su.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (su *SubscriptionUpdate) check() error {
	if v, ok := su.mutation.Name(); ok {
		if err := subscription.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "Subscription.name": %w`, err)}
		}
	}
	if su.mutation.CustomerCleared() && len(su.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "Subscription.customer"`)
	}
	return nil
}

func (su *SubscriptionUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := su.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscription.Table, subscription.Columns, sqlgraph.NewFieldSpec(subscription.FieldID, field.TypeString))
	if ps := su.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := su.mutation.UpdatedAt(); ok {
		_spec.SetField(subscription.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := su.mutation.DeletedAt(); ok {
		_spec.SetField(subscription.FieldDeletedAt, field.TypeTime, value)
	}
	if su.mutation.DeletedAtCleared() {
		_spec.ClearField(subscription.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := su.mutation.Metadata(); ok {
		_spec.SetField(subscription.FieldMetadata, field.TypeJSON, value)
	}
	if su.mutation.MetadataCleared() {
		_spec.ClearField(subscription.FieldMetadata, field.TypeJSON)
	}
	if value, ok := su.mutation.ActiveTo(); ok {
		_spec.SetField(subscription.FieldActiveTo, field.TypeTime, value)
	}
	if su.mutation.ActiveToCleared() {
		_spec.ClearField(subscription.FieldActiveTo, field.TypeTime)
	}
	if value, ok := su.mutation.BillablesMustAlign(); ok {
		_spec.SetField(subscription.FieldBillablesMustAlign, field.TypeBool, value)
	}
	if value, ok := su.mutation.Name(); ok {
		_spec.SetField(subscription.FieldName, field.TypeString, value)
	}
	if value, ok := su.mutation.Description(); ok {
		_spec.SetField(subscription.FieldDescription, field.TypeString, value)
	}
	if su.mutation.DescriptionCleared() {
		_spec.ClearField(subscription.FieldDescription, field.TypeString)
	}
	if su.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscription.PlanTable,
			Columns: []string{subscription.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscription.PlanTable,
			Columns: []string{subscription.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.PhasesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.RemovedPhasesIDs(); len(nodes) > 0 && !su.mutation.PhasesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.PhasesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.RemovedBillingLinesIDs(); len(nodes) > 0 && !su.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
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
	if nodes := su.mutation.BillingLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
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
	if su.mutation.AddonsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.RemovedAddonsIDs(); len(nodes) > 0 && !su.mutation.AddonsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.AddonsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, su.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscription.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	su.mutation.done = true
	return n, nil
}

// SubscriptionUpdateOne is the builder for updating a single Subscription entity.
type SubscriptionUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *SubscriptionMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (suo *SubscriptionUpdateOne) SetUpdatedAt(t time.Time) *SubscriptionUpdateOne {
	suo.mutation.SetUpdatedAt(t)
	return suo
}

// SetDeletedAt sets the "deleted_at" field.
func (suo *SubscriptionUpdateOne) SetDeletedAt(t time.Time) *SubscriptionUpdateOne {
	suo.mutation.SetDeletedAt(t)
	return suo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillableDeletedAt(t *time.Time) *SubscriptionUpdateOne {
	if t != nil {
		suo.SetDeletedAt(*t)
	}
	return suo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (suo *SubscriptionUpdateOne) ClearDeletedAt() *SubscriptionUpdateOne {
	suo.mutation.ClearDeletedAt()
	return suo
}

// SetMetadata sets the "metadata" field.
func (suo *SubscriptionUpdateOne) SetMetadata(m map[string]string) *SubscriptionUpdateOne {
	suo.mutation.SetMetadata(m)
	return suo
}

// ClearMetadata clears the value of the "metadata" field.
func (suo *SubscriptionUpdateOne) ClearMetadata() *SubscriptionUpdateOne {
	suo.mutation.ClearMetadata()
	return suo
}

// SetActiveTo sets the "active_to" field.
func (suo *SubscriptionUpdateOne) SetActiveTo(t time.Time) *SubscriptionUpdateOne {
	suo.mutation.SetActiveTo(t)
	return suo
}

// SetNillableActiveTo sets the "active_to" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillableActiveTo(t *time.Time) *SubscriptionUpdateOne {
	if t != nil {
		suo.SetActiveTo(*t)
	}
	return suo
}

// ClearActiveTo clears the value of the "active_to" field.
func (suo *SubscriptionUpdateOne) ClearActiveTo() *SubscriptionUpdateOne {
	suo.mutation.ClearActiveTo()
	return suo
}

// SetBillablesMustAlign sets the "billables_must_align" field.
func (suo *SubscriptionUpdateOne) SetBillablesMustAlign(b bool) *SubscriptionUpdateOne {
	suo.mutation.SetBillablesMustAlign(b)
	return suo
}

// SetNillableBillablesMustAlign sets the "billables_must_align" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillableBillablesMustAlign(b *bool) *SubscriptionUpdateOne {
	if b != nil {
		suo.SetBillablesMustAlign(*b)
	}
	return suo
}

// SetName sets the "name" field.
func (suo *SubscriptionUpdateOne) SetName(s string) *SubscriptionUpdateOne {
	suo.mutation.SetName(s)
	return suo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillableName(s *string) *SubscriptionUpdateOne {
	if s != nil {
		suo.SetName(*s)
	}
	return suo
}

// SetDescription sets the "description" field.
func (suo *SubscriptionUpdateOne) SetDescription(s string) *SubscriptionUpdateOne {
	suo.mutation.SetDescription(s)
	return suo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillableDescription(s *string) *SubscriptionUpdateOne {
	if s != nil {
		suo.SetDescription(*s)
	}
	return suo
}

// ClearDescription clears the value of the "description" field.
func (suo *SubscriptionUpdateOne) ClearDescription() *SubscriptionUpdateOne {
	suo.mutation.ClearDescription()
	return suo
}

// SetPlanID sets the "plan_id" field.
func (suo *SubscriptionUpdateOne) SetPlanID(s string) *SubscriptionUpdateOne {
	suo.mutation.SetPlanID(s)
	return suo
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (suo *SubscriptionUpdateOne) SetNillablePlanID(s *string) *SubscriptionUpdateOne {
	if s != nil {
		suo.SetPlanID(*s)
	}
	return suo
}

// ClearPlanID clears the value of the "plan_id" field.
func (suo *SubscriptionUpdateOne) ClearPlanID() *SubscriptionUpdateOne {
	suo.mutation.ClearPlanID()
	return suo
}

// SetPlan sets the "plan" edge to the Plan entity.
func (suo *SubscriptionUpdateOne) SetPlan(p *Plan) *SubscriptionUpdateOne {
	return suo.SetPlanID(p.ID)
}

// AddPhaseIDs adds the "phases" edge to the SubscriptionPhase entity by IDs.
func (suo *SubscriptionUpdateOne) AddPhaseIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.AddPhaseIDs(ids...)
	return suo
}

// AddPhases adds the "phases" edges to the SubscriptionPhase entity.
func (suo *SubscriptionUpdateOne) AddPhases(s ...*SubscriptionPhase) *SubscriptionUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return suo.AddPhaseIDs(ids...)
}

// AddBillingLineIDs adds the "billing_lines" edge to the BillingInvoiceLine entity by IDs.
func (suo *SubscriptionUpdateOne) AddBillingLineIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.AddBillingLineIDs(ids...)
	return suo
}

// AddBillingLines adds the "billing_lines" edges to the BillingInvoiceLine entity.
func (suo *SubscriptionUpdateOne) AddBillingLines(b ...*BillingInvoiceLine) *SubscriptionUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return suo.AddBillingLineIDs(ids...)
}

// AddAddonIDs adds the "addons" edge to the SubscriptionAddon entity by IDs.
func (suo *SubscriptionUpdateOne) AddAddonIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.AddAddonIDs(ids...)
	return suo
}

// AddAddons adds the "addons" edges to the SubscriptionAddon entity.
func (suo *SubscriptionUpdateOne) AddAddons(s ...*SubscriptionAddon) *SubscriptionUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return suo.AddAddonIDs(ids...)
}

// Mutation returns the SubscriptionMutation object of the builder.
func (suo *SubscriptionUpdateOne) Mutation() *SubscriptionMutation {
	return suo.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (suo *SubscriptionUpdateOne) ClearPlan() *SubscriptionUpdateOne {
	suo.mutation.ClearPlan()
	return suo
}

// ClearPhases clears all "phases" edges to the SubscriptionPhase entity.
func (suo *SubscriptionUpdateOne) ClearPhases() *SubscriptionUpdateOne {
	suo.mutation.ClearPhases()
	return suo
}

// RemovePhaseIDs removes the "phases" edge to SubscriptionPhase entities by IDs.
func (suo *SubscriptionUpdateOne) RemovePhaseIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.RemovePhaseIDs(ids...)
	return suo
}

// RemovePhases removes "phases" edges to SubscriptionPhase entities.
func (suo *SubscriptionUpdateOne) RemovePhases(s ...*SubscriptionPhase) *SubscriptionUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return suo.RemovePhaseIDs(ids...)
}

// ClearBillingLines clears all "billing_lines" edges to the BillingInvoiceLine entity.
func (suo *SubscriptionUpdateOne) ClearBillingLines() *SubscriptionUpdateOne {
	suo.mutation.ClearBillingLines()
	return suo
}

// RemoveBillingLineIDs removes the "billing_lines" edge to BillingInvoiceLine entities by IDs.
func (suo *SubscriptionUpdateOne) RemoveBillingLineIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.RemoveBillingLineIDs(ids...)
	return suo
}

// RemoveBillingLines removes "billing_lines" edges to BillingInvoiceLine entities.
func (suo *SubscriptionUpdateOne) RemoveBillingLines(b ...*BillingInvoiceLine) *SubscriptionUpdateOne {
	ids := make([]string, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return suo.RemoveBillingLineIDs(ids...)
}

// ClearAddons clears all "addons" edges to the SubscriptionAddon entity.
func (suo *SubscriptionUpdateOne) ClearAddons() *SubscriptionUpdateOne {
	suo.mutation.ClearAddons()
	return suo
}

// RemoveAddonIDs removes the "addons" edge to SubscriptionAddon entities by IDs.
func (suo *SubscriptionUpdateOne) RemoveAddonIDs(ids ...string) *SubscriptionUpdateOne {
	suo.mutation.RemoveAddonIDs(ids...)
	return suo
}

// RemoveAddons removes "addons" edges to SubscriptionAddon entities.
func (suo *SubscriptionUpdateOne) RemoveAddons(s ...*SubscriptionAddon) *SubscriptionUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return suo.RemoveAddonIDs(ids...)
}

// Where appends a list predicates to the SubscriptionUpdate builder.
func (suo *SubscriptionUpdateOne) Where(ps ...predicate.Subscription) *SubscriptionUpdateOne {
	suo.mutation.Where(ps...)
	return suo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (suo *SubscriptionUpdateOne) Select(field string, fields ...string) *SubscriptionUpdateOne {
	suo.fields = append([]string{field}, fields...)
	return suo
}

// Save executes the query and returns the updated Subscription entity.
func (suo *SubscriptionUpdateOne) Save(ctx context.Context) (*Subscription, error) {
	suo.defaults()
	return withHooks(ctx, suo.sqlSave, suo.mutation, suo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (suo *SubscriptionUpdateOne) SaveX(ctx context.Context) *Subscription {
	node, err := suo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (suo *SubscriptionUpdateOne) Exec(ctx context.Context) error {
	_, err := suo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (suo *SubscriptionUpdateOne) ExecX(ctx context.Context) {
	if err := suo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (suo *SubscriptionUpdateOne) defaults() {
	if _, ok := suo.mutation.UpdatedAt(); !ok {
		v := subscription.UpdateDefaultUpdatedAt()
		suo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (suo *SubscriptionUpdateOne) check() error {
	if v, ok := suo.mutation.Name(); ok {
		if err := subscription.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "Subscription.name": %w`, err)}
		}
	}
	if suo.mutation.CustomerCleared() && len(suo.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "Subscription.customer"`)
	}
	return nil
}

func (suo *SubscriptionUpdateOne) sqlSave(ctx context.Context) (_node *Subscription, err error) {
	if err := suo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(subscription.Table, subscription.Columns, sqlgraph.NewFieldSpec(subscription.FieldID, field.TypeString))
	id, ok := suo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "Subscription.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := suo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscription.FieldID)
		for _, f := range fields {
			if !subscription.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != subscription.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := suo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := suo.mutation.UpdatedAt(); ok {
		_spec.SetField(subscription.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := suo.mutation.DeletedAt(); ok {
		_spec.SetField(subscription.FieldDeletedAt, field.TypeTime, value)
	}
	if suo.mutation.DeletedAtCleared() {
		_spec.ClearField(subscription.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := suo.mutation.Metadata(); ok {
		_spec.SetField(subscription.FieldMetadata, field.TypeJSON, value)
	}
	if suo.mutation.MetadataCleared() {
		_spec.ClearField(subscription.FieldMetadata, field.TypeJSON)
	}
	if value, ok := suo.mutation.ActiveTo(); ok {
		_spec.SetField(subscription.FieldActiveTo, field.TypeTime, value)
	}
	if suo.mutation.ActiveToCleared() {
		_spec.ClearField(subscription.FieldActiveTo, field.TypeTime)
	}
	if value, ok := suo.mutation.BillablesMustAlign(); ok {
		_spec.SetField(subscription.FieldBillablesMustAlign, field.TypeBool, value)
	}
	if value, ok := suo.mutation.Name(); ok {
		_spec.SetField(subscription.FieldName, field.TypeString, value)
	}
	if value, ok := suo.mutation.Description(); ok {
		_spec.SetField(subscription.FieldDescription, field.TypeString, value)
	}
	if suo.mutation.DescriptionCleared() {
		_spec.ClearField(subscription.FieldDescription, field.TypeString)
	}
	if suo.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscription.PlanTable,
			Columns: []string{subscription.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   subscription.PlanTable,
			Columns: []string{subscription.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.PhasesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.RemovedPhasesIDs(); len(nodes) > 0 && !suo.mutation.PhasesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.PhasesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.PhasesTable,
			Columns: []string{subscription.PhasesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.RemovedBillingLinesIDs(); len(nodes) > 0 && !suo.mutation.BillingLinesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
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
	if nodes := suo.mutation.BillingLinesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.BillingLinesTable,
			Columns: []string{subscription.BillingLinesColumn},
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
	if suo.mutation.AddonsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.RemovedAddonsIDs(); len(nodes) > 0 && !suo.mutation.AddonsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.AddonsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   subscription.AddonsTable,
			Columns: []string{subscription.AddonsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Subscription{config: suo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, suo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{subscription.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	suo.mutation.done = true
	return _node, nil
}
