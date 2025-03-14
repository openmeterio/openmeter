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
	"github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// CustomerSubjectsUpdate is the builder for updating CustomerSubjects entities.
type CustomerSubjectsUpdate struct {
	config
	hooks    []Hook
	mutation *CustomerSubjectsMutation
}

// Where appends a list predicates to the CustomerSubjectsUpdate builder.
func (csu *CustomerSubjectsUpdate) Where(ps ...predicate.CustomerSubjects) *CustomerSubjectsUpdate {
	csu.mutation.Where(ps...)
	return csu
}

// SetDeletedAt sets the "deleted_at" field.
func (csu *CustomerSubjectsUpdate) SetDeletedAt(t time.Time) *CustomerSubjectsUpdate {
	csu.mutation.SetDeletedAt(t)
	return csu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (csu *CustomerSubjectsUpdate) SetNillableDeletedAt(t *time.Time) *CustomerSubjectsUpdate {
	if t != nil {
		csu.SetDeletedAt(*t)
	}
	return csu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (csu *CustomerSubjectsUpdate) ClearDeletedAt() *CustomerSubjectsUpdate {
	csu.mutation.ClearDeletedAt()
	return csu
}

// Mutation returns the CustomerSubjectsMutation object of the builder.
func (csu *CustomerSubjectsUpdate) Mutation() *CustomerSubjectsMutation {
	return csu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (csu *CustomerSubjectsUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, csu.sqlSave, csu.mutation, csu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (csu *CustomerSubjectsUpdate) SaveX(ctx context.Context) int {
	affected, err := csu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (csu *CustomerSubjectsUpdate) Exec(ctx context.Context) error {
	_, err := csu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (csu *CustomerSubjectsUpdate) ExecX(ctx context.Context) {
	if err := csu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (csu *CustomerSubjectsUpdate) check() error {
	if csu.mutation.CustomerCleared() && len(csu.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "CustomerSubjects.customer"`)
	}
	return nil
}

func (csu *CustomerSubjectsUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := csu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(customersubjects.Table, customersubjects.Columns, sqlgraph.NewFieldSpec(customersubjects.FieldID, field.TypeInt))
	if ps := csu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := csu.mutation.DeletedAt(); ok {
		_spec.SetField(customersubjects.FieldDeletedAt, field.TypeTime, value)
	}
	if csu.mutation.DeletedAtCleared() {
		_spec.ClearField(customersubjects.FieldDeletedAt, field.TypeTime)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, csu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{customersubjects.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	csu.mutation.done = true
	return n, nil
}

// CustomerSubjectsUpdateOne is the builder for updating a single CustomerSubjects entity.
type CustomerSubjectsUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *CustomerSubjectsMutation
}

// SetDeletedAt sets the "deleted_at" field.
func (csuo *CustomerSubjectsUpdateOne) SetDeletedAt(t time.Time) *CustomerSubjectsUpdateOne {
	csuo.mutation.SetDeletedAt(t)
	return csuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (csuo *CustomerSubjectsUpdateOne) SetNillableDeletedAt(t *time.Time) *CustomerSubjectsUpdateOne {
	if t != nil {
		csuo.SetDeletedAt(*t)
	}
	return csuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (csuo *CustomerSubjectsUpdateOne) ClearDeletedAt() *CustomerSubjectsUpdateOne {
	csuo.mutation.ClearDeletedAt()
	return csuo
}

// Mutation returns the CustomerSubjectsMutation object of the builder.
func (csuo *CustomerSubjectsUpdateOne) Mutation() *CustomerSubjectsMutation {
	return csuo.mutation
}

// Where appends a list predicates to the CustomerSubjectsUpdate builder.
func (csuo *CustomerSubjectsUpdateOne) Where(ps ...predicate.CustomerSubjects) *CustomerSubjectsUpdateOne {
	csuo.mutation.Where(ps...)
	return csuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (csuo *CustomerSubjectsUpdateOne) Select(field string, fields ...string) *CustomerSubjectsUpdateOne {
	csuo.fields = append([]string{field}, fields...)
	return csuo
}

// Save executes the query and returns the updated CustomerSubjects entity.
func (csuo *CustomerSubjectsUpdateOne) Save(ctx context.Context) (*CustomerSubjects, error) {
	return withHooks(ctx, csuo.sqlSave, csuo.mutation, csuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (csuo *CustomerSubjectsUpdateOne) SaveX(ctx context.Context) *CustomerSubjects {
	node, err := csuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (csuo *CustomerSubjectsUpdateOne) Exec(ctx context.Context) error {
	_, err := csuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (csuo *CustomerSubjectsUpdateOne) ExecX(ctx context.Context) {
	if err := csuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (csuo *CustomerSubjectsUpdateOne) check() error {
	if csuo.mutation.CustomerCleared() && len(csuo.mutation.CustomerIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "CustomerSubjects.customer"`)
	}
	return nil
}

func (csuo *CustomerSubjectsUpdateOne) sqlSave(ctx context.Context) (_node *CustomerSubjects, err error) {
	if err := csuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(customersubjects.Table, customersubjects.Columns, sqlgraph.NewFieldSpec(customersubjects.FieldID, field.TypeInt))
	id, ok := csuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "CustomerSubjects.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := csuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, customersubjects.FieldID)
		for _, f := range fields {
			if !customersubjects.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != customersubjects.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := csuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := csuo.mutation.DeletedAt(); ok {
		_spec.SetField(customersubjects.FieldDeletedAt, field.TypeTime, value)
	}
	if csuo.mutation.DeletedAtCleared() {
		_spec.ClearField(customersubjects.FieldDeletedAt, field.TypeTime)
	}
	_node = &CustomerSubjects{config: csuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, csuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{customersubjects.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	csuo.mutation.done = true
	return _node, nil
}
