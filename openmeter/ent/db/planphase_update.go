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
	"github.com/openmeterio/openmeter/pkg/datetime"
)

// PlanPhaseUpdate is the builder for updating PlanPhase entities.
type PlanPhaseUpdate struct {
	config
	hooks    []Hook
	mutation *PlanPhaseMutation
}

// Where appends a list predicates to the PlanPhaseUpdate builder.
func (_u *PlanPhaseUpdate) Where(ps ...predicate.PlanPhase) *PlanPhaseUpdate {
	_u.mutation.Where(ps...)
	return _u
}

// SetMetadata sets the "metadata" field.
func (_u *PlanPhaseUpdate) SetMetadata(v map[string]string) *PlanPhaseUpdate {
	_u.mutation.SetMetadata(v)
	return _u
}

// ClearMetadata clears the value of the "metadata" field.
func (_u *PlanPhaseUpdate) ClearMetadata() *PlanPhaseUpdate {
	_u.mutation.ClearMetadata()
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *PlanPhaseUpdate) SetUpdatedAt(v time.Time) *PlanPhaseUpdate {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *PlanPhaseUpdate) SetDeletedAt(v time.Time) *PlanPhaseUpdate {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillableDeletedAt(v *time.Time) *PlanPhaseUpdate {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *PlanPhaseUpdate) ClearDeletedAt() *PlanPhaseUpdate {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetName sets the "name" field.
func (_u *PlanPhaseUpdate) SetName(v string) *PlanPhaseUpdate {
	_u.mutation.SetName(v)
	return _u
}

// SetNillableName sets the "name" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillableName(v *string) *PlanPhaseUpdate {
	if v != nil {
		_u.SetName(*v)
	}
	return _u
}

// SetDescription sets the "description" field.
func (_u *PlanPhaseUpdate) SetDescription(v string) *PlanPhaseUpdate {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillableDescription(v *string) *PlanPhaseUpdate {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *PlanPhaseUpdate) ClearDescription() *PlanPhaseUpdate {
	_u.mutation.ClearDescription()
	return _u
}

// SetPlanID sets the "plan_id" field.
func (_u *PlanPhaseUpdate) SetPlanID(v string) *PlanPhaseUpdate {
	_u.mutation.SetPlanID(v)
	return _u
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillablePlanID(v *string) *PlanPhaseUpdate {
	if v != nil {
		_u.SetPlanID(*v)
	}
	return _u
}

// SetIndex sets the "index" field.
func (_u *PlanPhaseUpdate) SetIndex(v uint8) *PlanPhaseUpdate {
	_u.mutation.ResetIndex()
	_u.mutation.SetIndex(v)
	return _u
}

// SetNillableIndex sets the "index" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillableIndex(v *uint8) *PlanPhaseUpdate {
	if v != nil {
		_u.SetIndex(*v)
	}
	return _u
}

// AddIndex adds value to the "index" field.
func (_u *PlanPhaseUpdate) AddIndex(v int8) *PlanPhaseUpdate {
	_u.mutation.AddIndex(v)
	return _u
}

// SetDuration sets the "duration" field.
func (_u *PlanPhaseUpdate) SetDuration(v datetime.ISODurationString) *PlanPhaseUpdate {
	_u.mutation.SetDuration(v)
	return _u
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (_u *PlanPhaseUpdate) SetNillableDuration(v *datetime.ISODurationString) *PlanPhaseUpdate {
	if v != nil {
		_u.SetDuration(*v)
	}
	return _u
}

// ClearDuration clears the value of the "duration" field.
func (_u *PlanPhaseUpdate) ClearDuration() *PlanPhaseUpdate {
	_u.mutation.ClearDuration()
	return _u
}

// SetPlan sets the "plan" edge to the Plan entity.
func (_u *PlanPhaseUpdate) SetPlan(v *Plan) *PlanPhaseUpdate {
	return _u.SetPlanID(v.ID)
}

// AddRatecardIDs adds the "ratecards" edge to the PlanRateCard entity by IDs.
func (_u *PlanPhaseUpdate) AddRatecardIDs(ids ...string) *PlanPhaseUpdate {
	_u.mutation.AddRatecardIDs(ids...)
	return _u
}

// AddRatecards adds the "ratecards" edges to the PlanRateCard entity.
func (_u *PlanPhaseUpdate) AddRatecards(v ...*PlanRateCard) *PlanPhaseUpdate {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.AddRatecardIDs(ids...)
}

// Mutation returns the PlanPhaseMutation object of the builder.
func (_u *PlanPhaseUpdate) Mutation() *PlanPhaseMutation {
	return _u.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (_u *PlanPhaseUpdate) ClearPlan() *PlanPhaseUpdate {
	_u.mutation.ClearPlan()
	return _u
}

// ClearRatecards clears all "ratecards" edges to the PlanRateCard entity.
func (_u *PlanPhaseUpdate) ClearRatecards() *PlanPhaseUpdate {
	_u.mutation.ClearRatecards()
	return _u
}

// RemoveRatecardIDs removes the "ratecards" edge to PlanRateCard entities by IDs.
func (_u *PlanPhaseUpdate) RemoveRatecardIDs(ids ...string) *PlanPhaseUpdate {
	_u.mutation.RemoveRatecardIDs(ids...)
	return _u
}

// RemoveRatecards removes "ratecards" edges to PlanRateCard entities.
func (_u *PlanPhaseUpdate) RemoveRatecards(v ...*PlanRateCard) *PlanPhaseUpdate {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.RemoveRatecardIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (_u *PlanPhaseUpdate) Save(ctx context.Context) (int, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *PlanPhaseUpdate) SaveX(ctx context.Context) int {
	affected, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (_u *PlanPhaseUpdate) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *PlanPhaseUpdate) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *PlanPhaseUpdate) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := planphase.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *PlanPhaseUpdate) check() error {
	if v, ok := _u.mutation.PlanID(); ok {
		if err := planphase.PlanIDValidator(v); err != nil {
			return &ValidationError{Name: "plan_id", err: fmt.Errorf(`db: validator failed for field "PlanPhase.plan_id": %w`, err)}
		}
	}
	if _u.mutation.PlanCleared() && len(_u.mutation.PlanIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanPhase.plan"`)
	}
	return nil
}

func (_u *PlanPhaseUpdate) sqlSave(ctx context.Context) (_node int, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(planphase.Table, planphase.Columns, sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString))
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.Metadata(); ok {
		_spec.SetField(planphase.FieldMetadata, field.TypeJSON, value)
	}
	if _u.mutation.MetadataCleared() {
		_spec.ClearField(planphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(planphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(planphase.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(planphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.Name(); ok {
		_spec.SetField(planphase.FieldName, field.TypeString, value)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(planphase.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(planphase.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.Index(); ok {
		_spec.SetField(planphase.FieldIndex, field.TypeUint8, value)
	}
	if value, ok := _u.mutation.AddedIndex(); ok {
		_spec.AddField(planphase.FieldIndex, field.TypeUint8, value)
	}
	if value, ok := _u.mutation.Duration(); ok {
		_spec.SetField(planphase.FieldDuration, field.TypeString, value)
	}
	if _u.mutation.DurationCleared() {
		_spec.ClearField(planphase.FieldDuration, field.TypeString)
	}
	if _u.mutation.PlanCleared() {
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
	if nodes := _u.mutation.PlanIDs(); len(nodes) > 0 {
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
	if _u.mutation.RatecardsCleared() {
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
	if nodes := _u.mutation.RemovedRatecardsIDs(); len(nodes) > 0 && !_u.mutation.RatecardsCleared() {
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
	if nodes := _u.mutation.RatecardsIDs(); len(nodes) > 0 {
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
	if _node, err = sqlgraph.UpdateNodes(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	_u.mutation.done = true
	return _node, nil
}

// PlanPhaseUpdateOne is the builder for updating a single PlanPhase entity.
type PlanPhaseUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *PlanPhaseMutation
}

// SetMetadata sets the "metadata" field.
func (_u *PlanPhaseUpdateOne) SetMetadata(v map[string]string) *PlanPhaseUpdateOne {
	_u.mutation.SetMetadata(v)
	return _u
}

// ClearMetadata clears the value of the "metadata" field.
func (_u *PlanPhaseUpdateOne) ClearMetadata() *PlanPhaseUpdateOne {
	_u.mutation.ClearMetadata()
	return _u
}

// SetUpdatedAt sets the "updated_at" field.
func (_u *PlanPhaseUpdateOne) SetUpdatedAt(v time.Time) *PlanPhaseUpdateOne {
	_u.mutation.SetUpdatedAt(v)
	return _u
}

// SetDeletedAt sets the "deleted_at" field.
func (_u *PlanPhaseUpdateOne) SetDeletedAt(v time.Time) *PlanPhaseUpdateOne {
	_u.mutation.SetDeletedAt(v)
	return _u
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillableDeletedAt(v *time.Time) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetDeletedAt(*v)
	}
	return _u
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (_u *PlanPhaseUpdateOne) ClearDeletedAt() *PlanPhaseUpdateOne {
	_u.mutation.ClearDeletedAt()
	return _u
}

// SetName sets the "name" field.
func (_u *PlanPhaseUpdateOne) SetName(v string) *PlanPhaseUpdateOne {
	_u.mutation.SetName(v)
	return _u
}

// SetNillableName sets the "name" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillableName(v *string) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetName(*v)
	}
	return _u
}

// SetDescription sets the "description" field.
func (_u *PlanPhaseUpdateOne) SetDescription(v string) *PlanPhaseUpdateOne {
	_u.mutation.SetDescription(v)
	return _u
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillableDescription(v *string) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetDescription(*v)
	}
	return _u
}

// ClearDescription clears the value of the "description" field.
func (_u *PlanPhaseUpdateOne) ClearDescription() *PlanPhaseUpdateOne {
	_u.mutation.ClearDescription()
	return _u
}

// SetPlanID sets the "plan_id" field.
func (_u *PlanPhaseUpdateOne) SetPlanID(v string) *PlanPhaseUpdateOne {
	_u.mutation.SetPlanID(v)
	return _u
}

// SetNillablePlanID sets the "plan_id" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillablePlanID(v *string) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetPlanID(*v)
	}
	return _u
}

// SetIndex sets the "index" field.
func (_u *PlanPhaseUpdateOne) SetIndex(v uint8) *PlanPhaseUpdateOne {
	_u.mutation.ResetIndex()
	_u.mutation.SetIndex(v)
	return _u
}

// SetNillableIndex sets the "index" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillableIndex(v *uint8) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetIndex(*v)
	}
	return _u
}

// AddIndex adds value to the "index" field.
func (_u *PlanPhaseUpdateOne) AddIndex(v int8) *PlanPhaseUpdateOne {
	_u.mutation.AddIndex(v)
	return _u
}

// SetDuration sets the "duration" field.
func (_u *PlanPhaseUpdateOne) SetDuration(v datetime.ISODurationString) *PlanPhaseUpdateOne {
	_u.mutation.SetDuration(v)
	return _u
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (_u *PlanPhaseUpdateOne) SetNillableDuration(v *datetime.ISODurationString) *PlanPhaseUpdateOne {
	if v != nil {
		_u.SetDuration(*v)
	}
	return _u
}

// ClearDuration clears the value of the "duration" field.
func (_u *PlanPhaseUpdateOne) ClearDuration() *PlanPhaseUpdateOne {
	_u.mutation.ClearDuration()
	return _u
}

// SetPlan sets the "plan" edge to the Plan entity.
func (_u *PlanPhaseUpdateOne) SetPlan(v *Plan) *PlanPhaseUpdateOne {
	return _u.SetPlanID(v.ID)
}

// AddRatecardIDs adds the "ratecards" edge to the PlanRateCard entity by IDs.
func (_u *PlanPhaseUpdateOne) AddRatecardIDs(ids ...string) *PlanPhaseUpdateOne {
	_u.mutation.AddRatecardIDs(ids...)
	return _u
}

// AddRatecards adds the "ratecards" edges to the PlanRateCard entity.
func (_u *PlanPhaseUpdateOne) AddRatecards(v ...*PlanRateCard) *PlanPhaseUpdateOne {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.AddRatecardIDs(ids...)
}

// Mutation returns the PlanPhaseMutation object of the builder.
func (_u *PlanPhaseUpdateOne) Mutation() *PlanPhaseMutation {
	return _u.mutation
}

// ClearPlan clears the "plan" edge to the Plan entity.
func (_u *PlanPhaseUpdateOne) ClearPlan() *PlanPhaseUpdateOne {
	_u.mutation.ClearPlan()
	return _u
}

// ClearRatecards clears all "ratecards" edges to the PlanRateCard entity.
func (_u *PlanPhaseUpdateOne) ClearRatecards() *PlanPhaseUpdateOne {
	_u.mutation.ClearRatecards()
	return _u
}

// RemoveRatecardIDs removes the "ratecards" edge to PlanRateCard entities by IDs.
func (_u *PlanPhaseUpdateOne) RemoveRatecardIDs(ids ...string) *PlanPhaseUpdateOne {
	_u.mutation.RemoveRatecardIDs(ids...)
	return _u
}

// RemoveRatecards removes "ratecards" edges to PlanRateCard entities.
func (_u *PlanPhaseUpdateOne) RemoveRatecards(v ...*PlanRateCard) *PlanPhaseUpdateOne {
	ids := make([]string, len(v))
	for i := range v {
		ids[i] = v[i].ID
	}
	return _u.RemoveRatecardIDs(ids...)
}

// Where appends a list predicates to the PlanPhaseUpdate builder.
func (_u *PlanPhaseUpdateOne) Where(ps ...predicate.PlanPhase) *PlanPhaseUpdateOne {
	_u.mutation.Where(ps...)
	return _u
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (_u *PlanPhaseUpdateOne) Select(field string, fields ...string) *PlanPhaseUpdateOne {
	_u.fields = append([]string{field}, fields...)
	return _u
}

// Save executes the query and returns the updated PlanPhase entity.
func (_u *PlanPhaseUpdateOne) Save(ctx context.Context) (*PlanPhase, error) {
	_u.defaults()
	return withHooks(ctx, _u.sqlSave, _u.mutation, _u.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (_u *PlanPhaseUpdateOne) SaveX(ctx context.Context) *PlanPhase {
	node, err := _u.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (_u *PlanPhaseUpdateOne) Exec(ctx context.Context) error {
	_, err := _u.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (_u *PlanPhaseUpdateOne) ExecX(ctx context.Context) {
	if err := _u.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (_u *PlanPhaseUpdateOne) defaults() {
	if _, ok := _u.mutation.UpdatedAt(); !ok {
		v := planphase.UpdateDefaultUpdatedAt()
		_u.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (_u *PlanPhaseUpdateOne) check() error {
	if v, ok := _u.mutation.PlanID(); ok {
		if err := planphase.PlanIDValidator(v); err != nil {
			return &ValidationError{Name: "plan_id", err: fmt.Errorf(`db: validator failed for field "PlanPhase.plan_id": %w`, err)}
		}
	}
	if _u.mutation.PlanCleared() && len(_u.mutation.PlanIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanPhase.plan"`)
	}
	return nil
}

func (_u *PlanPhaseUpdateOne) sqlSave(ctx context.Context) (_node *PlanPhase, err error) {
	if err := _u.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(planphase.Table, planphase.Columns, sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString))
	id, ok := _u.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "PlanPhase.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := _u.fields; len(fields) > 0 {
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
	if ps := _u.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := _u.mutation.Metadata(); ok {
		_spec.SetField(planphase.FieldMetadata, field.TypeJSON, value)
	}
	if _u.mutation.MetadataCleared() {
		_spec.ClearField(planphase.FieldMetadata, field.TypeJSON)
	}
	if value, ok := _u.mutation.UpdatedAt(); ok {
		_spec.SetField(planphase.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := _u.mutation.DeletedAt(); ok {
		_spec.SetField(planphase.FieldDeletedAt, field.TypeTime, value)
	}
	if _u.mutation.DeletedAtCleared() {
		_spec.ClearField(planphase.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := _u.mutation.Name(); ok {
		_spec.SetField(planphase.FieldName, field.TypeString, value)
	}
	if value, ok := _u.mutation.Description(); ok {
		_spec.SetField(planphase.FieldDescription, field.TypeString, value)
	}
	if _u.mutation.DescriptionCleared() {
		_spec.ClearField(planphase.FieldDescription, field.TypeString)
	}
	if value, ok := _u.mutation.Index(); ok {
		_spec.SetField(planphase.FieldIndex, field.TypeUint8, value)
	}
	if value, ok := _u.mutation.AddedIndex(); ok {
		_spec.AddField(planphase.FieldIndex, field.TypeUint8, value)
	}
	if value, ok := _u.mutation.Duration(); ok {
		_spec.SetField(planphase.FieldDuration, field.TypeString, value)
	}
	if _u.mutation.DurationCleared() {
		_spec.ClearField(planphase.FieldDuration, field.TypeString)
	}
	if _u.mutation.PlanCleared() {
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
	if nodes := _u.mutation.PlanIDs(); len(nodes) > 0 {
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
	if _u.mutation.RatecardsCleared() {
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
	if nodes := _u.mutation.RemovedRatecardsIDs(); len(nodes) > 0 && !_u.mutation.RatecardsCleared() {
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
	if nodes := _u.mutation.RatecardsIDs(); len(nodes) > 0 {
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
	_node = &PlanPhase{config: _u.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, _u.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planphase.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	_u.mutation.done = true
	return _node, nil
}
