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
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datex"
)

// PlanPhaseUpdate is the builder for updating PlanPhase entities.
type PlanPhaseUpdate struct {
	config
	hooks    []Hook
	mutation *PlanPhaseMutation
}

// Where appends a list predicates to the PlanPhaseUpdate builder.
func (ppu *PlanPhaseUpdate) Where(ps ...predicate.PlanPhase) *PlanPhaseUpdate {
	ppu.mutation.Where(ps...)
	return ppu
}

// SetMetadata sets the "metadata" field.
func (ppu *PlanPhaseUpdate) SetMetadata(m map[string]string) *PlanPhaseUpdate {
	ppu.mutation.SetMetadata(m)
	return ppu
}

// ClearMetadata clears the value of the "metadata" field.
func (ppu *PlanPhaseUpdate) ClearMetadata() *PlanPhaseUpdate {
	ppu.mutation.ClearMetadata()
	return ppu
}

// SetUpdatedAt sets the "updated_at" field.
func (ppu *PlanPhaseUpdate) SetUpdatedAt(t time.Time) *PlanPhaseUpdate {
	ppu.mutation.SetUpdatedAt(t)
	return ppu
}

// SetDeletedAt sets the "deleted_at" field.
func (ppu *PlanPhaseUpdate) SetDeletedAt(t time.Time) *PlanPhaseUpdate {
	ppu.mutation.SetDeletedAt(t)
	return ppu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillableDeletedAt(t *time.Time) *PlanPhaseUpdate {
	if t != nil {
		ppu.SetDeletedAt(*t)
	}
	return ppu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (ppu *PlanPhaseUpdate) ClearDeletedAt() *PlanPhaseUpdate {
	ppu.mutation.ClearDeletedAt()
	return ppu
}

// SetName sets the "name" field.
func (ppu *PlanPhaseUpdate) SetName(s string) *PlanPhaseUpdate {
	ppu.mutation.SetName(s)
	return ppu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillableName(s *string) *PlanPhaseUpdate {
	if s != nil {
		ppu.SetName(*s)
	}
	return ppu
}

// SetDescription sets the "description" field.
func (ppu *PlanPhaseUpdate) SetDescription(s string) *PlanPhaseUpdate {
	ppu.mutation.SetDescription(s)
	return ppu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillableDescription(s *string) *PlanPhaseUpdate {
	if s != nil {
		ppu.SetDescription(*s)
	}
	return ppu
}

// ClearDescription clears the value of the "description" field.
func (ppu *PlanPhaseUpdate) ClearDescription() *PlanPhaseUpdate {
	ppu.mutation.ClearDescription()
	return ppu
}

// SetDuration sets the "duration" field.
func (ppu *PlanPhaseUpdate) SetDuration(ds datex.ISOString) *PlanPhaseUpdate {
	ppu.mutation.SetDuration(ds)
	return ppu
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillableDuration(ds *datex.ISOString) *PlanPhaseUpdate {
	if ds != nil {
		ppu.SetDuration(*ds)
	}
	return ppu
}

// ClearDuration clears the value of the "duration" field.
func (ppu *PlanPhaseUpdate) ClearDuration() *PlanPhaseUpdate {
	ppu.mutation.ClearDuration()
	return ppu
}

// SetDiscounts sets the "discounts" field.
func (ppu *PlanPhaseUpdate) SetDiscounts(pr []productcatalog.Discount) *PlanPhaseUpdate {
	ppu.mutation.SetDiscounts(pr)
	return ppu
}

// ClearDiscounts clears the value of the "discounts" field.
func (ppu *PlanPhaseUpdate) ClearDiscounts() *PlanPhaseUpdate {
	ppu.mutation.ClearDiscounts()
	return ppu
}

// SetPlanID sets the "plan_id" field.
func (ppu *PlanPhaseUpdate) SetPlanID(s string) *PlanPhaseUpdate {
	ppu.mutation.SetPlanID(s)
	return ppu
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillablePlanID(s *string) *PlanPhaseUpdate {
	if s != nil {
		ppu.SetPlanID(*s)
	}
	return ppu
}

// SetIndex sets the "index" field.
func (ppu *PlanPhaseUpdate) SetIndex(i int) *PlanPhaseUpdate {
	ppu.mutation.ResetIndex()
	ppu.mutation.SetIndex(i)
	return ppu
}

// SetNillableIndex sets the "index" field if the given value is not nil.
func (ppu *PlanPhaseUpdate) SetNillableIndex(i *int) *PlanPhaseUpdate {
	if i != nil {
		ppu.SetIndex(*i)
	}
	return ppu
}

// AddIndex adds i to the "index" field.
func (ppu *PlanPhaseUpdate) AddIndex(i int) *PlanPhaseUpdate {
	ppu.mutation.AddIndex(i)
	return ppu
}

// SetPlan sets the "plan" edge to the Plan entity.
func (ppu *PlanPhaseUpdate) SetPlan(p *Plan) *PlanPhaseUpdate {
	return ppu.SetPlanID(p.ID)
}

// AddRatecardIDs adds the "ratecards" edge to the PlanRateCard entity by IDs.
func (ppu *PlanPhaseUpdate) AddRatecardIDs(ids ...string) *PlanPhaseUpdate {
	ppu.mutation.AddRatecardIDs(ids...)
	return ppu
}

// AddRatecards adds the "ratecards" edges to the PlanRateCard entity.
func (ppu *PlanPhaseUpdate) AddRatecards(p ...*PlanRateCard) *PlanPhaseUpdate {
	ids := make([]string, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return ppu.AddRatecardIDs(ids...)
}

// Mutation returns the PlanPhaseMutation object of the builder.
func (ppu *PlanPhaseUpdate) Mutation() *PlanPhaseMutation {
	return ppu.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (ppu *PlanPhaseUpdate) ClearPlan() *PlanPhaseUpdate {
	ppu.mutation.ClearPlan()
	return ppu
}

// ClearRatecards clears all "ratecards" edges to the PlanRateCard entity.
func (ppu *PlanPhaseUpdate) ClearRatecards() *PlanPhaseUpdate {
	ppu.mutation.ClearRatecards()
	return ppu
}

// RemoveRatecardIDs removes the "ratecards" edge to PlanRateCard entities by IDs.
func (ppu *PlanPhaseUpdate) RemoveRatecardIDs(ids ...string) *PlanPhaseUpdate {
	ppu.mutation.RemoveRatecardIDs(ids...)
	return ppu
}

// RemoveRatecards removes "ratecards" edges to PlanRateCard entities.
func (ppu *PlanPhaseUpdate) RemoveRatecards(p ...*PlanRateCard) *PlanPhaseUpdate {
	ids := make([]string, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return ppu.RemoveRatecardIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (ppu *PlanPhaseUpdate) Save(ctx context.Context) (int, error) {
	ppu.defaults()
	return withHooks(ctx, ppu.sqlSave, ppu.mutation, ppu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (ppu *PlanPhaseUpdate) SaveX(ctx context.Context) int {
	affected, err := ppu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (ppu *PlanPhaseUpdate) Exec(ctx context.Context) error {
	_, err := ppu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ppu *PlanPhaseUpdate) ExecX(ctx context.Context) {
	if err := ppu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ppu *PlanPhaseUpdate) defaults() {
	if _, ok := ppu.mutation.UpdatedAt(); !ok {
		v := planphase.UpdateDefaultUpdatedAt()
		ppu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ppu *PlanPhaseUpdate) check() error {
	if v, ok := ppu.mutation.PlanID(); ok {
		if err := planphase.PlanIDValidator(v); err != nil {
			return &ValidationError{Name: "plan_id", err: fmt.Errorf(`db: validator failed for field "PlanPhase.plan_id": %w`, err)}
		}
	}
	if ppu.mutation.PlanCleared() && len(ppu.mutation.PlanIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanPhase.plan"`)
	}
	return nil
}

func (ppu *PlanPhaseUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := ppu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(planphase.Table, planphase.Columns, sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString))
	if ps := ppu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := ppu.mutation.Metadata(); ok {
		_spec.SetField(planphase.FieldMetadata, field.TypeJSON, value)
	}
	if ppu.mutation.MetadataCleared() {
		_spec.ClearField(planphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := ppu.mutation.UpdatedAt(); ok {
		_spec.SetField(planphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := ppu.mutation.DeletedAt(); ok {
		_spec.SetField(planphase.FieldDeletedAt, field.TypeTime, value)
	}
	if ppu.mutation.DeletedAtCleared() {
		_spec.ClearField(planphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := ppu.mutation.Name(); ok {
		_spec.SetField(planphase.FieldName, field.TypeString, value)
	}
	if value, ok := ppu.mutation.Description(); ok {
		_spec.SetField(planphase.FieldDescription, field.TypeString, value)
	}
	if ppu.mutation.DescriptionCleared() {
		_spec.ClearField(planphase.FieldDescription, field.TypeString)
	}
	if value, ok := ppu.mutation.Duration(); ok {
		_spec.SetField(planphase.FieldDuration, field.TypeString, value)
	}
	if ppu.mutation.DurationCleared() {
		_spec.ClearField(planphase.FieldDuration, field.TypeString)
	}
	if value, ok := ppu.mutation.Discounts(); ok {
		vv, err := planphase.ValueScanner.Discounts.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(planphase.FieldDiscounts, field.TypeString, vv)
	}
	if ppu.mutation.DiscountsCleared() {
		_spec.ClearField(planphase.FieldDiscounts, field.TypeString)
	}
	if value, ok := ppu.mutation.Index(); ok {
		_spec.SetField(planphase.FieldIndex, field.TypeInt, value)
	}
	if value, ok := ppu.mutation.AddedIndex(); ok {
		_spec.AddField(planphase.FieldIndex, field.TypeInt, value)
	}
	if ppu.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planphase.PlanTable,
			Columns: []string{planphase.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppu.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planphase.PlanTable,
			Columns: []string{planphase.PlanColumn},
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
	if ppu.mutation.RatecardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppu.mutation.RemovedRatecardsIDs(); len(nodes) > 0 && !ppu.mutation.RatecardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppu.mutation.RatecardsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, ppu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	ppu.mutation.done = true
	return n, nil
}

// PlanPhaseUpdateOne is the builder for updating a single PlanPhase entity.
type PlanPhaseUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *PlanPhaseMutation
}

// SetMetadata sets the "metadata" field.
func (ppuo *PlanPhaseUpdateOne) SetMetadata(m map[string]string) *PlanPhaseUpdateOne {
	ppuo.mutation.SetMetadata(m)
	return ppuo
}

// ClearMetadata clears the value of the "metadata" field.
func (ppuo *PlanPhaseUpdateOne) ClearMetadata() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearMetadata()
	return ppuo
}

// SetUpdatedAt sets the "updated_at" field.
func (ppuo *PlanPhaseUpdateOne) SetUpdatedAt(t time.Time) *PlanPhaseUpdateOne {
	ppuo.mutation.SetUpdatedAt(t)
	return ppuo
}

// SetDeletedAt sets the "deleted_at" field.
func (ppuo *PlanPhaseUpdateOne) SetDeletedAt(t time.Time) *PlanPhaseUpdateOne {
	ppuo.mutation.SetDeletedAt(t)
	return ppuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillableDeletedAt(t *time.Time) *PlanPhaseUpdateOne {
	if t != nil {
		ppuo.SetDeletedAt(*t)
	}
	return ppuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (ppuo *PlanPhaseUpdateOne) ClearDeletedAt() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearDeletedAt()
	return ppuo
}

// SetName sets the "name" field.
func (ppuo *PlanPhaseUpdateOne) SetName(s string) *PlanPhaseUpdateOne {
	ppuo.mutation.SetName(s)
	return ppuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillableName(s *string) *PlanPhaseUpdateOne {
	if s != nil {
		ppuo.SetName(*s)
	}
	return ppuo
}

// SetDescription sets the "description" field.
func (ppuo *PlanPhaseUpdateOne) SetDescription(s string) *PlanPhaseUpdateOne {
	ppuo.mutation.SetDescription(s)
	return ppuo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillableDescription(s *string) *PlanPhaseUpdateOne {
	if s != nil {
		ppuo.SetDescription(*s)
	}
	return ppuo
}

// ClearDescription clears the value of the "description" field.
func (ppuo *PlanPhaseUpdateOne) ClearDescription() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearDescription()
	return ppuo
}

// SetDuration sets the "duration" field.
func (ppuo *PlanPhaseUpdateOne) SetDuration(ds datex.ISOString) *PlanPhaseUpdateOne {
	ppuo.mutation.SetDuration(ds)
	return ppuo
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillableDuration(ds *datex.ISOString) *PlanPhaseUpdateOne {
	if ds != nil {
		ppuo.SetDuration(*ds)
	}
	return ppuo
}

// ClearDuration clears the value of the "duration" field.
func (ppuo *PlanPhaseUpdateOne) ClearDuration() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearDuration()
	return ppuo
}

// SetDiscounts sets the "discounts" field.
func (ppuo *PlanPhaseUpdateOne) SetDiscounts(pr []productcatalog.Discount) *PlanPhaseUpdateOne {
	ppuo.mutation.SetDiscounts(pr)
	return ppuo
}

// ClearDiscounts clears the value of the "discounts" field.
func (ppuo *PlanPhaseUpdateOne) ClearDiscounts() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearDiscounts()
	return ppuo
}

// SetPlanID sets the "plan_id" field.
func (ppuo *PlanPhaseUpdateOne) SetPlanID(s string) *PlanPhaseUpdateOne {
	ppuo.mutation.SetPlanID(s)
	return ppuo
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillablePlanID(s *string) *PlanPhaseUpdateOne {
	if s != nil {
		ppuo.SetPlanID(*s)
	}
	return ppuo
}

// SetIndex sets the "index" field.
func (ppuo *PlanPhaseUpdateOne) SetIndex(i int) *PlanPhaseUpdateOne {
	ppuo.mutation.ResetIndex()
	ppuo.mutation.SetIndex(i)
	return ppuo
}

// SetNillableIndex sets the "index" field if the given value is not nil.
func (ppuo *PlanPhaseUpdateOne) SetNillableIndex(i *int) *PlanPhaseUpdateOne {
	if i != nil {
		ppuo.SetIndex(*i)
	}
	return ppuo
}

// AddIndex adds i to the "index" field.
func (ppuo *PlanPhaseUpdateOne) AddIndex(i int) *PlanPhaseUpdateOne {
	ppuo.mutation.AddIndex(i)
	return ppuo
}

// SetPlan sets the "plan" edge to the Plan entity.
func (ppuo *PlanPhaseUpdateOne) SetPlan(p *Plan) *PlanPhaseUpdateOne {
	return ppuo.SetPlanID(p.ID)
}

// AddRatecardIDs adds the "ratecards" edge to the PlanRateCard entity by IDs.
func (ppuo *PlanPhaseUpdateOne) AddRatecardIDs(ids ...string) *PlanPhaseUpdateOne {
	ppuo.mutation.AddRatecardIDs(ids...)
	return ppuo
}

// AddRatecards adds the "ratecards" edges to the PlanRateCard entity.
func (ppuo *PlanPhaseUpdateOne) AddRatecards(p ...*PlanRateCard) *PlanPhaseUpdateOne {
	ids := make([]string, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return ppuo.AddRatecardIDs(ids...)
}

// Mutation returns the PlanPhaseMutation object of the builder.
func (ppuo *PlanPhaseUpdateOne) Mutation() *PlanPhaseMutation {
	return ppuo.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (ppuo *PlanPhaseUpdateOne) ClearPlan() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearPlan()
	return ppuo
}

// ClearRatecards clears all "ratecards" edges to the PlanRateCard entity.
func (ppuo *PlanPhaseUpdateOne) ClearRatecards() *PlanPhaseUpdateOne {
	ppuo.mutation.ClearRatecards()
	return ppuo
}

// RemoveRatecardIDs removes the "ratecards" edge to PlanRateCard entities by IDs.
func (ppuo *PlanPhaseUpdateOne) RemoveRatecardIDs(ids ...string) *PlanPhaseUpdateOne {
	ppuo.mutation.RemoveRatecardIDs(ids...)
	return ppuo
}

// RemoveRatecards removes "ratecards" edges to PlanRateCard entities.
func (ppuo *PlanPhaseUpdateOne) RemoveRatecards(p ...*PlanRateCard) *PlanPhaseUpdateOne {
	ids := make([]string, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return ppuo.RemoveRatecardIDs(ids...)
}

// Where appends a list predicates to the PlanPhaseUpdate builder.
func (ppuo *PlanPhaseUpdateOne) Where(ps ...predicate.PlanPhase) *PlanPhaseUpdateOne {
	ppuo.mutation.Where(ps...)
	return ppuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (ppuo *PlanPhaseUpdateOne) Select(field string, fields ...string) *PlanPhaseUpdateOne {
	ppuo.fields = append([]string{field}, fields...)
	return ppuo
}

// Save executes the query and returns the updated PlanPhase entity.
func (ppuo *PlanPhaseUpdateOne) Save(ctx context.Context) (*PlanPhase, error) {
	ppuo.defaults()
	return withHooks(ctx, ppuo.sqlSave, ppuo.mutation, ppuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (ppuo *PlanPhaseUpdateOne) SaveX(ctx context.Context) *PlanPhase {
	node, err := ppuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (ppuo *PlanPhaseUpdateOne) Exec(ctx context.Context) error {
	_, err := ppuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ppuo *PlanPhaseUpdateOne) ExecX(ctx context.Context) {
	if err := ppuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ppuo *PlanPhaseUpdateOne) defaults() {
	if _, ok := ppuo.mutation.UpdatedAt(); !ok {
		v := planphase.UpdateDefaultUpdatedAt()
		ppuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ppuo *PlanPhaseUpdateOne) check() error {
	if v, ok := ppuo.mutation.PlanID(); ok {
		if err := planphase.PlanIDValidator(v); err != nil {
			return &ValidationError{Name: "plan_id", err: fmt.Errorf(`db: validator failed for field "PlanPhase.plan_id": %w`, err)}
		}
	}
	if ppuo.mutation.PlanCleared() && len(ppuo.mutation.PlanIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanPhase.plan"`)
	}
	return nil
}

func (ppuo *PlanPhaseUpdateOne) sqlSave(ctx context.Context) (_node *PlanPhase, err error) {
	if err := ppuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(planphase.Table, planphase.Columns, sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString))
	id, ok := ppuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "PlanPhase.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := ppuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, planphase.FieldID)
		for _, f := range fields {
			if !planphase.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != planphase.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := ppuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := ppuo.mutation.Metadata(); ok {
		_spec.SetField(planphase.FieldMetadata, field.TypeJSON, value)
	}
	if ppuo.mutation.MetadataCleared() {
		_spec.ClearField(planphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := ppuo.mutation.UpdatedAt(); ok {
		_spec.SetField(planphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := ppuo.mutation.DeletedAt(); ok {
		_spec.SetField(planphase.FieldDeletedAt, field.TypeTime, value)
	}
	if ppuo.mutation.DeletedAtCleared() {
		_spec.ClearField(planphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := ppuo.mutation.Name(); ok {
		_spec.SetField(planphase.FieldName, field.TypeString, value)
	}
	if value, ok := ppuo.mutation.Description(); ok {
		_spec.SetField(planphase.FieldDescription, field.TypeString, value)
	}
	if ppuo.mutation.DescriptionCleared() {
		_spec.ClearField(planphase.FieldDescription, field.TypeString)
	}
	if value, ok := ppuo.mutation.Duration(); ok {
		_spec.SetField(planphase.FieldDuration, field.TypeString, value)
	}
	if ppuo.mutation.DurationCleared() {
		_spec.ClearField(planphase.FieldDuration, field.TypeString)
	}
	if value, ok := ppuo.mutation.Discounts(); ok {
		vv, err := planphase.ValueScanner.Discounts.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(planphase.FieldDiscounts, field.TypeString, vv)
	}
	if ppuo.mutation.DiscountsCleared() {
		_spec.ClearField(planphase.FieldDiscounts, field.TypeString)
	}
	if value, ok := ppuo.mutation.Index(); ok {
		_spec.SetField(planphase.FieldIndex, field.TypeInt, value)
	}
	if value, ok := ppuo.mutation.AddedIndex(); ok {
		_spec.AddField(planphase.FieldIndex, field.TypeInt, value)
	}
	if ppuo.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planphase.PlanTable,
			Columns: []string{planphase.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppuo.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planphase.PlanTable,
			Columns: []string{planphase.PlanColumn},
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
	if ppuo.mutation.RatecardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppuo.mutation.RemovedRatecardsIDs(); len(nodes) > 0 && !ppuo.mutation.RatecardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := ppuo.mutation.RatecardsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   planphase.RatecardsTable,
			Columns: []string{planphase.RatecardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &PlanPhase{config: ppuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, ppuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	ppuo.mutation.done = true
	return _node, nil
}
