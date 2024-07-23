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
	"github.com/openmeterio/openmeter/internal/ent/db/feature"
	"github.com/openmeterio/openmeter/internal/ent/db/predicate"
)

// FeatureUpdate is the builder for updating Feature entities.
type FeatureUpdate struct {
	config
	hooks    []Hook
	mutation *FeatureMutation
}

// Where appends a list predicates to the FeatureUpdate builder.
func (fu *FeatureUpdate) Where(ps ...predicate.Feature) *FeatureUpdate {
	fu.mutation.Where(ps...)
	return fu
}

// SetUpdatedAt sets the "updated_at" field.
func (fu *FeatureUpdate) SetUpdatedAt(t time.Time) *FeatureUpdate {
	fu.mutation.SetUpdatedAt(t)
	return fu
}

// SetDeletedAt sets the "deleted_at" field.
func (fu *FeatureUpdate) SetDeletedAt(t time.Time) *FeatureUpdate {
	fu.mutation.SetDeletedAt(t)
	return fu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (fu *FeatureUpdate) SetNillableDeletedAt(t *time.Time) *FeatureUpdate {
	if t != nil {
		fu.SetDeletedAt(*t)
	}
	return fu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (fu *FeatureUpdate) ClearDeletedAt() *FeatureUpdate {
	fu.mutation.ClearDeletedAt()
	return fu
}

// SetMetadata sets the "metadata" field.
func (fu *FeatureUpdate) SetMetadata(m map[string]string) *FeatureUpdate {
	fu.mutation.SetMetadata(m)
	return fu
}

// ClearMetadata clears the value of the "metadata" field.
func (fu *FeatureUpdate) ClearMetadata() *FeatureUpdate {
	fu.mutation.ClearMetadata()
	return fu
}

// SetName sets the "name" field.
func (fu *FeatureUpdate) SetName(s string) *FeatureUpdate {
	fu.mutation.SetName(s)
	return fu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (fu *FeatureUpdate) SetNillableName(s *string) *FeatureUpdate {
	if s != nil {
		fu.SetName(*s)
	}
	return fu
}

// SetMeterGroupByFilters sets the "meter_group_by_filters" field.
func (fu *FeatureUpdate) SetMeterGroupByFilters(m map[string]string) *FeatureUpdate {
	fu.mutation.SetMeterGroupByFilters(m)
	return fu
}

// ClearMeterGroupByFilters clears the value of the "meter_group_by_filters" field.
func (fu *FeatureUpdate) ClearMeterGroupByFilters() *FeatureUpdate {
	fu.mutation.ClearMeterGroupByFilters()
	return fu
}

// SetArchivedAt sets the "archived_at" field.
func (fu *FeatureUpdate) SetArchivedAt(t time.Time) *FeatureUpdate {
	fu.mutation.SetArchivedAt(t)
	return fu
}

// SetNillableArchivedAt sets the "archived_at" field if the given value is not nil.
func (fu *FeatureUpdate) SetNillableArchivedAt(t *time.Time) *FeatureUpdate {
	if t != nil {
		fu.SetArchivedAt(*t)
	}
	return fu
}

// ClearArchivedAt clears the value of the "archived_at" field.
func (fu *FeatureUpdate) ClearArchivedAt() *FeatureUpdate {
	fu.mutation.ClearArchivedAt()
	return fu
}

// Mutation returns the FeatureMutation object of the builder.
func (fu *FeatureUpdate) Mutation() *FeatureMutation {
	return fu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (fu *FeatureUpdate) Save(ctx context.Context) (int, error) {
	fu.defaults()
	return withHooks(ctx, fu.sqlSave, fu.mutation, fu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (fu *FeatureUpdate) SaveX(ctx context.Context) int {
	affected, err := fu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (fu *FeatureUpdate) Exec(ctx context.Context) error {
	_, err := fu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fu *FeatureUpdate) ExecX(ctx context.Context) {
	if err := fu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (fu *FeatureUpdate) defaults() {
	if _, ok := fu.mutation.UpdatedAt(); !ok {
		v := feature.UpdateDefaultUpdatedAt()
		fu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fu *FeatureUpdate) check() error {
	if v, ok := fu.mutation.Name(); ok {
		if err := feature.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "Feature.name": %w`, err)}
		}
	}
	return nil
}

func (fu *FeatureUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := fu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(feature.Table, feature.Columns, sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString))
	if ps := fu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := fu.mutation.UpdatedAt(); ok {
		_spec.SetField(feature.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := fu.mutation.DeletedAt(); ok {
		_spec.SetField(feature.FieldDeletedAt, field.TypeTime, value)
	}
	if fu.mutation.DeletedAtCleared() {
		_spec.ClearField(feature.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := fu.mutation.Metadata(); ok {
		_spec.SetField(feature.FieldMetadata, field.TypeJSON, value)
	}
	if fu.mutation.MetadataCleared() {
		_spec.ClearField(feature.FieldMetadata, field.TypeJSON)
	}
	if value, ok := fu.mutation.Name(); ok {
		_spec.SetField(feature.FieldName, field.TypeString, value)
	}
	if fu.mutation.MeterSlugCleared() {
		_spec.ClearField(feature.FieldMeterSlug, field.TypeString)
	}
	if value, ok := fu.mutation.MeterGroupByFilters(); ok {
		_spec.SetField(feature.FieldMeterGroupByFilters, field.TypeJSON, value)
	}
	if fu.mutation.MeterGroupByFiltersCleared() {
		_spec.ClearField(feature.FieldMeterGroupByFilters, field.TypeJSON)
	}
	if value, ok := fu.mutation.ArchivedAt(); ok {
		_spec.SetField(feature.FieldArchivedAt, field.TypeTime, value)
	}
	if fu.mutation.ArchivedAtCleared() {
		_spec.ClearField(feature.FieldArchivedAt, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, fu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{feature.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	fu.mutation.done = true
	return n, nil
}

// FeatureUpdateOne is the builder for updating a single Feature entity.
type FeatureUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *FeatureMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (fuo *FeatureUpdateOne) SetUpdatedAt(t time.Time) *FeatureUpdateOne {
	fuo.mutation.SetUpdatedAt(t)
	return fuo
}

// SetDeletedAt sets the "deleted_at" field.
func (fuo *FeatureUpdateOne) SetDeletedAt(t time.Time) *FeatureUpdateOne {
	fuo.mutation.SetDeletedAt(t)
	return fuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (fuo *FeatureUpdateOne) SetNillableDeletedAt(t *time.Time) *FeatureUpdateOne {
	if t != nil {
		fuo.SetDeletedAt(*t)
	}
	return fuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (fuo *FeatureUpdateOne) ClearDeletedAt() *FeatureUpdateOne {
	fuo.mutation.ClearDeletedAt()
	return fuo
}

// SetMetadata sets the "metadata" field.
func (fuo *FeatureUpdateOne) SetMetadata(m map[string]string) *FeatureUpdateOne {
	fuo.mutation.SetMetadata(m)
	return fuo
}

// ClearMetadata clears the value of the "metadata" field.
func (fuo *FeatureUpdateOne) ClearMetadata() *FeatureUpdateOne {
	fuo.mutation.ClearMetadata()
	return fuo
}

// SetName sets the "name" field.
func (fuo *FeatureUpdateOne) SetName(s string) *FeatureUpdateOne {
	fuo.mutation.SetName(s)
	return fuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (fuo *FeatureUpdateOne) SetNillableName(s *string) *FeatureUpdateOne {
	if s != nil {
		fuo.SetName(*s)
	}
	return fuo
}

// SetMeterGroupByFilters sets the "meter_group_by_filters" field.
func (fuo *FeatureUpdateOne) SetMeterGroupByFilters(m map[string]string) *FeatureUpdateOne {
	fuo.mutation.SetMeterGroupByFilters(m)
	return fuo
}

// ClearMeterGroupByFilters clears the value of the "meter_group_by_filters" field.
func (fuo *FeatureUpdateOne) ClearMeterGroupByFilters() *FeatureUpdateOne {
	fuo.mutation.ClearMeterGroupByFilters()
	return fuo
}

// SetArchivedAt sets the "archived_at" field.
func (fuo *FeatureUpdateOne) SetArchivedAt(t time.Time) *FeatureUpdateOne {
	fuo.mutation.SetArchivedAt(t)
	return fuo
}

// SetNillableArchivedAt sets the "archived_at" field if the given value is not nil.
func (fuo *FeatureUpdateOne) SetNillableArchivedAt(t *time.Time) *FeatureUpdateOne {
	if t != nil {
		fuo.SetArchivedAt(*t)
	}
	return fuo
}

// ClearArchivedAt clears the value of the "archived_at" field.
func (fuo *FeatureUpdateOne) ClearArchivedAt() *FeatureUpdateOne {
	fuo.mutation.ClearArchivedAt()
	return fuo
}

// Mutation returns the FeatureMutation object of the builder.
func (fuo *FeatureUpdateOne) Mutation() *FeatureMutation {
	return fuo.mutation
}

// Where appends a list predicates to the FeatureUpdate builder.
func (fuo *FeatureUpdateOne) Where(ps ...predicate.Feature) *FeatureUpdateOne {
	fuo.mutation.Where(ps...)
	return fuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (fuo *FeatureUpdateOne) Select(field string, fields ...string) *FeatureUpdateOne {
	fuo.fields = append([]string{field}, fields...)
	return fuo
}

// Save executes the query and returns the updated Feature entity.
func (fuo *FeatureUpdateOne) Save(ctx context.Context) (*Feature, error) {
	fuo.defaults()
	return withHooks(ctx, fuo.sqlSave, fuo.mutation, fuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (fuo *FeatureUpdateOne) SaveX(ctx context.Context) *Feature {
	node, err := fuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (fuo *FeatureUpdateOne) Exec(ctx context.Context) error {
	_, err := fuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fuo *FeatureUpdateOne) ExecX(ctx context.Context) {
	if err := fuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (fuo *FeatureUpdateOne) defaults() {
	if _, ok := fuo.mutation.UpdatedAt(); !ok {
		v := feature.UpdateDefaultUpdatedAt()
		fuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fuo *FeatureUpdateOne) check() error {
	if v, ok := fuo.mutation.Name(); ok {
		if err := feature.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`db: validator failed for field "Feature.name": %w`, err)}
		}
	}
	return nil
}

func (fuo *FeatureUpdateOne) sqlSave(ctx context.Context) (_node *Feature, err error) {
	if err := fuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(feature.Table, feature.Columns, sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString))
	id, ok := fuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "Feature.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := fuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, feature.FieldID)
		for _, f := range fields {
			if !feature.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != feature.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := fuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := fuo.mutation.UpdatedAt(); ok {
		_spec.SetField(feature.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := fuo.mutation.DeletedAt(); ok {
		_spec.SetField(feature.FieldDeletedAt, field.TypeTime, value)
	}
	if fuo.mutation.DeletedAtCleared() {
		_spec.ClearField(feature.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := fuo.mutation.Metadata(); ok {
		_spec.SetField(feature.FieldMetadata, field.TypeJSON, value)
	}
	if fuo.mutation.MetadataCleared() {
		_spec.ClearField(feature.FieldMetadata, field.TypeJSON)
	}
	if value, ok := fuo.mutation.Name(); ok {
		_spec.SetField(feature.FieldName, field.TypeString, value)
	}
	if fuo.mutation.MeterSlugCleared() {
		_spec.ClearField(feature.FieldMeterSlug, field.TypeString)
	}
	if value, ok := fuo.mutation.MeterGroupByFilters(); ok {
		_spec.SetField(feature.FieldMeterGroupByFilters, field.TypeJSON, value)
	}
	if fuo.mutation.MeterGroupByFiltersCleared() {
		_spec.ClearField(feature.FieldMeterGroupByFilters, field.TypeJSON)
	}
	if value, ok := fuo.mutation.ArchivedAt(); ok {
		_spec.SetField(feature.FieldArchivedAt, field.TypeTime, value)
	}
	if fuo.mutation.ArchivedAtCleared() {
		_spec.ClearField(feature.FieldArchivedAt, field.TypeTime)
	}
	_node = &Feature{config: fuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, fuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{feature.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	fuo.mutation.done = true
	return _node, nil
}
