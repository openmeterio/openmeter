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
	"github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// PlanRateCardUpdate is the builder for updating PlanRateCard entities.
type PlanRateCardUpdate struct {
	config
	hooks    []Hook
	mutation *PlanRateCardMutation
}

// Where appends a list predicates to the PlanRateCardUpdate builder.
func (prcu *PlanRateCardUpdate) Where(ps ...predicate.PlanRateCard) *PlanRateCardUpdate {
	prcu.mutation.Where(ps...)
	return prcu
}

// SetMetadata sets the "metadata" field.
func (prcu *PlanRateCardUpdate) SetMetadata(m map[string]string) *PlanRateCardUpdate {
	prcu.mutation.SetMetadata(m)
	return prcu
}

// ClearMetadata clears the value of the "metadata" field.
func (prcu *PlanRateCardUpdate) ClearMetadata() *PlanRateCardUpdate {
	prcu.mutation.ClearMetadata()
	return prcu
}

// SetUpdatedAt sets the "updated_at" field.
func (prcu *PlanRateCardUpdate) SetUpdatedAt(t time.Time) *PlanRateCardUpdate {
	prcu.mutation.SetUpdatedAt(t)
	return prcu
}

// SetDeletedAt sets the "deleted_at" field.
func (prcu *PlanRateCardUpdate) SetDeletedAt(t time.Time) *PlanRateCardUpdate {
	prcu.mutation.SetDeletedAt(t)
	return prcu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableDeletedAt(t *time.Time) *PlanRateCardUpdate {
	if t != nil {
		prcu.SetDeletedAt(*t)
	}
	return prcu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (prcu *PlanRateCardUpdate) ClearDeletedAt() *PlanRateCardUpdate {
	prcu.mutation.ClearDeletedAt()
	return prcu
}

// SetName sets the "name" field.
func (prcu *PlanRateCardUpdate) SetName(s string) *PlanRateCardUpdate {
	prcu.mutation.SetName(s)
	return prcu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableName(s *string) *PlanRateCardUpdate {
	if s != nil {
		prcu.SetName(*s)
	}
	return prcu
}

// SetDescription sets the "description" field.
func (prcu *PlanRateCardUpdate) SetDescription(s string) *PlanRateCardUpdate {
	prcu.mutation.SetDescription(s)
	return prcu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableDescription(s *string) *PlanRateCardUpdate {
	if s != nil {
		prcu.SetDescription(*s)
	}
	return prcu
}

// ClearDescription clears the value of the "description" field.
func (prcu *PlanRateCardUpdate) ClearDescription() *PlanRateCardUpdate {
	prcu.mutation.ClearDescription()
	return prcu
}

// SetFeatureKey sets the "feature_key" field.
func (prcu *PlanRateCardUpdate) SetFeatureKey(s string) *PlanRateCardUpdate {
	prcu.mutation.SetFeatureKey(s)
	return prcu
}

// SetNillableFeatureKey sets the "feature_key" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableFeatureKey(s *string) *PlanRateCardUpdate {
	if s != nil {
		prcu.SetFeatureKey(*s)
	}
	return prcu
}

// ClearFeatureKey clears the value of the "feature_key" field.
func (prcu *PlanRateCardUpdate) ClearFeatureKey() *PlanRateCardUpdate {
	prcu.mutation.ClearFeatureKey()
	return prcu
}

// SetEntitlementTemplate sets the "entitlement_template" field.
func (prcu *PlanRateCardUpdate) SetEntitlementTemplate(pt *productcatalog.EntitlementTemplate) *PlanRateCardUpdate {
	prcu.mutation.SetEntitlementTemplate(pt)
	return prcu
}

// ClearEntitlementTemplate clears the value of the "entitlement_template" field.
func (prcu *PlanRateCardUpdate) ClearEntitlementTemplate() *PlanRateCardUpdate {
	prcu.mutation.ClearEntitlementTemplate()
	return prcu
}

// SetTaxConfig sets the "tax_config" field.
func (prcu *PlanRateCardUpdate) SetTaxConfig(pc *productcatalog.TaxConfig) *PlanRateCardUpdate {
	prcu.mutation.SetTaxConfig(pc)
	return prcu
}

// ClearTaxConfig clears the value of the "tax_config" field.
func (prcu *PlanRateCardUpdate) ClearTaxConfig() *PlanRateCardUpdate {
	prcu.mutation.ClearTaxConfig()
	return prcu
}

// SetBillingCadence sets the "billing_cadence" field.
func (prcu *PlanRateCardUpdate) SetBillingCadence(i isodate.String) *PlanRateCardUpdate {
	prcu.mutation.SetBillingCadence(i)
	return prcu
}

// SetNillableBillingCadence sets the "billing_cadence" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableBillingCadence(i *isodate.String) *PlanRateCardUpdate {
	if i != nil {
		prcu.SetBillingCadence(*i)
	}
	return prcu
}

// ClearBillingCadence clears the value of the "billing_cadence" field.
func (prcu *PlanRateCardUpdate) ClearBillingCadence() *PlanRateCardUpdate {
	prcu.mutation.ClearBillingCadence()
	return prcu
}

// SetPrice sets the "price" field.
func (prcu *PlanRateCardUpdate) SetPrice(pr *productcatalog.Price) *PlanRateCardUpdate {
	prcu.mutation.SetPrice(pr)
	return prcu
}

// ClearPrice clears the value of the "price" field.
func (prcu *PlanRateCardUpdate) ClearPrice() *PlanRateCardUpdate {
	prcu.mutation.ClearPrice()
	return prcu
}

// SetPhaseID sets the "phase_id" field.
func (prcu *PlanRateCardUpdate) SetPhaseID(s string) *PlanRateCardUpdate {
	prcu.mutation.SetPhaseID(s)
	return prcu
}

// SetNillablePhaseID sets the "phase_id" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillablePhaseID(s *string) *PlanRateCardUpdate {
	if s != nil {
		prcu.SetPhaseID(*s)
	}
	return prcu
}

// SetFeatureID sets the "feature_id" field.
func (prcu *PlanRateCardUpdate) SetFeatureID(s string) *PlanRateCardUpdate {
	prcu.mutation.SetFeatureID(s)
	return prcu
}

// SetNillableFeatureID sets the "feature_id" field if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableFeatureID(s *string) *PlanRateCardUpdate {
	if s != nil {
		prcu.SetFeatureID(*s)
	}
	return prcu
}

// ClearFeatureID clears the value of the "feature_id" field.
func (prcu *PlanRateCardUpdate) ClearFeatureID() *PlanRateCardUpdate {
	prcu.mutation.ClearFeatureID()
	return prcu
}

// SetPhase sets the "phase" edge to the PlanPhase entity.
func (prcu *PlanRateCardUpdate) SetPhase(p *PlanPhase) *PlanRateCardUpdate {
	return prcu.SetPhaseID(p.ID)
}

// SetFeaturesID sets the "features" edge to the Feature entity by ID.
func (prcu *PlanRateCardUpdate) SetFeaturesID(id string) *PlanRateCardUpdate {
	prcu.mutation.SetFeaturesID(id)
	return prcu
}

// SetNillableFeaturesID sets the "features" edge to the Feature entity by ID if the given value is not nil.
func (prcu *PlanRateCardUpdate) SetNillableFeaturesID(id *string) *PlanRateCardUpdate {
	if id != nil {
		prcu = prcu.SetFeaturesID(*id)
	}
	return prcu
}

// SetFeatures sets the "features" edge to the Feature entity.
func (prcu *PlanRateCardUpdate) SetFeatures(f *Feature) *PlanRateCardUpdate {
	return prcu.SetFeaturesID(f.ID)
}

// Mutation returns the PlanRateCardMutation object of the builder.
func (prcu *PlanRateCardUpdate) Mutation() *PlanRateCardMutation {
	return prcu.mutation
}

// ClearPhase clears the "phase" edge to the PlanPhase entity.
func (prcu *PlanRateCardUpdate) ClearPhase() *PlanRateCardUpdate {
	prcu.mutation.ClearPhase()
	return prcu
}

// ClearFeatures clears the "features" edge to the Feature entity.
func (prcu *PlanRateCardUpdate) ClearFeatures() *PlanRateCardUpdate {
	prcu.mutation.ClearFeatures()
	return prcu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (prcu *PlanRateCardUpdate) Save(ctx context.Context) (int, error) {
	prcu.defaults()
	return withHooks(ctx, prcu.sqlSave, prcu.mutation, prcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (prcu *PlanRateCardUpdate) SaveX(ctx context.Context) int {
	affected, err := prcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (prcu *PlanRateCardUpdate) Exec(ctx context.Context) error {
	_, err := prcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (prcu *PlanRateCardUpdate) ExecX(ctx context.Context) {
	if err := prcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (prcu *PlanRateCardUpdate) defaults() {
	if _, ok := prcu.mutation.UpdatedAt(); !ok {
		v := planratecard.UpdateDefaultUpdatedAt()
		prcu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (prcu *PlanRateCardUpdate) check() error {
	if v, ok := prcu.mutation.EntitlementTemplate(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "entitlement_template", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.entitlement_template": %w`, err)}
		}
	}
	if v, ok := prcu.mutation.TaxConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "tax_config", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.tax_config": %w`, err)}
		}
	}
	if v, ok := prcu.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.price": %w`, err)}
		}
	}
	if v, ok := prcu.mutation.PhaseID(); ok {
		if err := planratecard.PhaseIDValidator(v); err != nil {
			return &ValidationError{Name: "phase_id", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.phase_id": %w`, err)}
		}
	}
	if prcu.mutation.PhaseCleared() && len(prcu.mutation.PhaseIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanRateCard.phase"`)
	}
	return nil
}

func (prcu *PlanRateCardUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := prcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(planratecard.Table, planratecard.Columns, sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString))
	if ps := prcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := prcu.mutation.Metadata(); ok {
		_spec.SetField(planratecard.FieldMetadata, field.TypeJSON, value)
	}
	if prcu.mutation.MetadataCleared() {
		_spec.ClearField(planratecard.FieldMetadata, field.TypeJSON)
	}
	if value, ok := prcu.mutation.UpdatedAt(); ok {
		_spec.SetField(planratecard.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := prcu.mutation.DeletedAt(); ok {
		_spec.SetField(planratecard.FieldDeletedAt, field.TypeTime, value)
	}
	if prcu.mutation.DeletedAtCleared() {
		_spec.ClearField(planratecard.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := prcu.mutation.Name(); ok {
		_spec.SetField(planratecard.FieldName, field.TypeString, value)
	}
	if value, ok := prcu.mutation.Description(); ok {
		_spec.SetField(planratecard.FieldDescription, field.TypeString, value)
	}
	if prcu.mutation.DescriptionCleared() {
		_spec.ClearField(planratecard.FieldDescription, field.TypeString)
	}
	if value, ok := prcu.mutation.FeatureKey(); ok {
		_spec.SetField(planratecard.FieldFeatureKey, field.TypeString, value)
	}
	if prcu.mutation.FeatureKeyCleared() {
		_spec.ClearField(planratecard.FieldFeatureKey, field.TypeString)
	}
	if value, ok := prcu.mutation.EntitlementTemplate(); ok {
		vv, err := planratecard.ValueScanner.EntitlementTemplate.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(planratecard.FieldEntitlementTemplate, field.TypeString, vv)
	}
	if prcu.mutation.EntitlementTemplateCleared() {
		_spec.ClearField(planratecard.FieldEntitlementTemplate, field.TypeString)
	}
	if value, ok := prcu.mutation.TaxConfig(); ok {
		vv, err := planratecard.ValueScanner.TaxConfig.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(planratecard.FieldTaxConfig, field.TypeString, vv)
	}
	if prcu.mutation.TaxConfigCleared() {
		_spec.ClearField(planratecard.FieldTaxConfig, field.TypeString)
	}
	if value, ok := prcu.mutation.BillingCadence(); ok {
		_spec.SetField(planratecard.FieldBillingCadence, field.TypeString, value)
	}
	if prcu.mutation.BillingCadenceCleared() {
		_spec.ClearField(planratecard.FieldBillingCadence, field.TypeString)
	}
	if value, ok := prcu.mutation.Price(); ok {
		vv, err := planratecard.ValueScanner.Price.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(planratecard.FieldPrice, field.TypeString, vv)
	}
	if prcu.mutation.PriceCleared() {
		_spec.ClearField(planratecard.FieldPrice, field.TypeString)
	}
	if prcu.mutation.PhaseCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.PhaseTable,
			Columns: []string{planratecard.PhaseColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := prcu.mutation.PhaseIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.PhaseTable,
			Columns: []string{planratecard.PhaseColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if prcu.mutation.FeaturesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.FeaturesTable,
			Columns: []string{planratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := prcu.mutation.FeaturesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.FeaturesTable,
			Columns: []string{planratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, prcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planratecard.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	prcu.mutation.done = true
	return n, nil
}

// PlanRateCardUpdateOne is the builder for updating a single PlanRateCard entity.
type PlanRateCardUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *PlanRateCardMutation
}

// SetMetadata sets the "metadata" field.
func (prcuo *PlanRateCardUpdateOne) SetMetadata(m map[string]string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetMetadata(m)
	return prcuo
}

// ClearMetadata clears the value of the "metadata" field.
func (prcuo *PlanRateCardUpdateOne) ClearMetadata() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearMetadata()
	return prcuo
}

// SetUpdatedAt sets the "updated_at" field.
func (prcuo *PlanRateCardUpdateOne) SetUpdatedAt(t time.Time) *PlanRateCardUpdateOne {
	prcuo.mutation.SetUpdatedAt(t)
	return prcuo
}

// SetDeletedAt sets the "deleted_at" field.
func (prcuo *PlanRateCardUpdateOne) SetDeletedAt(t time.Time) *PlanRateCardUpdateOne {
	prcuo.mutation.SetDeletedAt(t)
	return prcuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableDeletedAt(t *time.Time) *PlanRateCardUpdateOne {
	if t != nil {
		prcuo.SetDeletedAt(*t)
	}
	return prcuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (prcuo *PlanRateCardUpdateOne) ClearDeletedAt() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearDeletedAt()
	return prcuo
}

// SetName sets the "name" field.
func (prcuo *PlanRateCardUpdateOne) SetName(s string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetName(s)
	return prcuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableName(s *string) *PlanRateCardUpdateOne {
	if s != nil {
		prcuo.SetName(*s)
	}
	return prcuo
}

// SetDescription sets the "description" field.
func (prcuo *PlanRateCardUpdateOne) SetDescription(s string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetDescription(s)
	return prcuo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableDescription(s *string) *PlanRateCardUpdateOne {
	if s != nil {
		prcuo.SetDescription(*s)
	}
	return prcuo
}

// ClearDescription clears the value of the "description" field.
func (prcuo *PlanRateCardUpdateOne) ClearDescription() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearDescription()
	return prcuo
}

// SetFeatureKey sets the "feature_key" field.
func (prcuo *PlanRateCardUpdateOne) SetFeatureKey(s string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetFeatureKey(s)
	return prcuo
}

// SetNillableFeatureKey sets the "feature_key" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableFeatureKey(s *string) *PlanRateCardUpdateOne {
	if s != nil {
		prcuo.SetFeatureKey(*s)
	}
	return prcuo
}

// ClearFeatureKey clears the value of the "feature_key" field.
func (prcuo *PlanRateCardUpdateOne) ClearFeatureKey() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearFeatureKey()
	return prcuo
}

// SetEntitlementTemplate sets the "entitlement_template" field.
func (prcuo *PlanRateCardUpdateOne) SetEntitlementTemplate(pt *productcatalog.EntitlementTemplate) *PlanRateCardUpdateOne {
	prcuo.mutation.SetEntitlementTemplate(pt)
	return prcuo
}

// ClearEntitlementTemplate clears the value of the "entitlement_template" field.
func (prcuo *PlanRateCardUpdateOne) ClearEntitlementTemplate() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearEntitlementTemplate()
	return prcuo
}

// SetTaxConfig sets the "tax_config" field.
func (prcuo *PlanRateCardUpdateOne) SetTaxConfig(pc *productcatalog.TaxConfig) *PlanRateCardUpdateOne {
	prcuo.mutation.SetTaxConfig(pc)
	return prcuo
}

// ClearTaxConfig clears the value of the "tax_config" field.
func (prcuo *PlanRateCardUpdateOne) ClearTaxConfig() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearTaxConfig()
	return prcuo
}

// SetBillingCadence sets the "billing_cadence" field.
func (prcuo *PlanRateCardUpdateOne) SetBillingCadence(i isodate.String) *PlanRateCardUpdateOne {
	prcuo.mutation.SetBillingCadence(i)
	return prcuo
}

// SetNillableBillingCadence sets the "billing_cadence" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableBillingCadence(i *isodate.String) *PlanRateCardUpdateOne {
	if i != nil {
		prcuo.SetBillingCadence(*i)
	}
	return prcuo
}

// ClearBillingCadence clears the value of the "billing_cadence" field.
func (prcuo *PlanRateCardUpdateOne) ClearBillingCadence() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearBillingCadence()
	return prcuo
}

// SetPrice sets the "price" field.
func (prcuo *PlanRateCardUpdateOne) SetPrice(pr *productcatalog.Price) *PlanRateCardUpdateOne {
	prcuo.mutation.SetPrice(pr)
	return prcuo
}

// ClearPrice clears the value of the "price" field.
func (prcuo *PlanRateCardUpdateOne) ClearPrice() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearPrice()
	return prcuo
}

// SetPhaseID sets the "phase_id" field.
func (prcuo *PlanRateCardUpdateOne) SetPhaseID(s string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetPhaseID(s)
	return prcuo
}

// SetNillablePhaseID sets the "phase_id" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillablePhaseID(s *string) *PlanRateCardUpdateOne {
	if s != nil {
		prcuo.SetPhaseID(*s)
	}
	return prcuo
}

// SetFeatureID sets the "feature_id" field.
func (prcuo *PlanRateCardUpdateOne) SetFeatureID(s string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetFeatureID(s)
	return prcuo
}

// SetNillableFeatureID sets the "feature_id" field if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableFeatureID(s *string) *PlanRateCardUpdateOne {
	if s != nil {
		prcuo.SetFeatureID(*s)
	}
	return prcuo
}

// ClearFeatureID clears the value of the "feature_id" field.
func (prcuo *PlanRateCardUpdateOne) ClearFeatureID() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearFeatureID()
	return prcuo
}

// SetPhase sets the "phase" edge to the PlanPhase entity.
func (prcuo *PlanRateCardUpdateOne) SetPhase(p *PlanPhase) *PlanRateCardUpdateOne {
	return prcuo.SetPhaseID(p.ID)
}

// SetFeaturesID sets the "features" edge to the Feature entity by ID.
func (prcuo *PlanRateCardUpdateOne) SetFeaturesID(id string) *PlanRateCardUpdateOne {
	prcuo.mutation.SetFeaturesID(id)
	return prcuo
}

// SetNillableFeaturesID sets the "features" edge to the Feature entity by ID if the given value is not nil.
func (prcuo *PlanRateCardUpdateOne) SetNillableFeaturesID(id *string) *PlanRateCardUpdateOne {
	if id != nil {
		prcuo = prcuo.SetFeaturesID(*id)
	}
	return prcuo
}

// SetFeatures sets the "features" edge to the Feature entity.
func (prcuo *PlanRateCardUpdateOne) SetFeatures(f *Feature) *PlanRateCardUpdateOne {
	return prcuo.SetFeaturesID(f.ID)
}

// Mutation returns the PlanRateCardMutation object of the builder.
func (prcuo *PlanRateCardUpdateOne) Mutation() *PlanRateCardMutation {
	return prcuo.mutation
}

// ClearPhase clears the "phase" edge to the PlanPhase entity.
func (prcuo *PlanRateCardUpdateOne) ClearPhase() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearPhase()
	return prcuo
}

// ClearFeatures clears the "features" edge to the Feature entity.
func (prcuo *PlanRateCardUpdateOne) ClearFeatures() *PlanRateCardUpdateOne {
	prcuo.mutation.ClearFeatures()
	return prcuo
}

// Where appends a list predicates to the PlanRateCardUpdate builder.
func (prcuo *PlanRateCardUpdateOne) Where(ps ...predicate.PlanRateCard) *PlanRateCardUpdateOne {
	prcuo.mutation.Where(ps...)
	return prcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (prcuo *PlanRateCardUpdateOne) Select(field string, fields ...string) *PlanRateCardUpdateOne {
	prcuo.fields = append([]string{field}, fields...)
	return prcuo
}

// Save executes the query and returns the updated PlanRateCard entity.
func (prcuo *PlanRateCardUpdateOne) Save(ctx context.Context) (*PlanRateCard, error) {
	prcuo.defaults()
	return withHooks(ctx, prcuo.sqlSave, prcuo.mutation, prcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (prcuo *PlanRateCardUpdateOne) SaveX(ctx context.Context) *PlanRateCard {
	node, err := prcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (prcuo *PlanRateCardUpdateOne) Exec(ctx context.Context) error {
	_, err := prcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (prcuo *PlanRateCardUpdateOne) ExecX(ctx context.Context) {
	if err := prcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (prcuo *PlanRateCardUpdateOne) defaults() {
	if _, ok := prcuo.mutation.UpdatedAt(); !ok {
		v := planratecard.UpdateDefaultUpdatedAt()
		prcuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (prcuo *PlanRateCardUpdateOne) check() error {
	if v, ok := prcuo.mutation.EntitlementTemplate(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "entitlement_template", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.entitlement_template": %w`, err)}
		}
	}
	if v, ok := prcuo.mutation.TaxConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "tax_config", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.tax_config": %w`, err)}
		}
	}
	if v, ok := prcuo.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.price": %w`, err)}
		}
	}
	if v, ok := prcuo.mutation.PhaseID(); ok {
		if err := planratecard.PhaseIDValidator(v); err != nil {
			return &ValidationError{Name: "phase_id", err: fmt.Errorf(`db: validator failed for field "PlanRateCard.phase_id": %w`, err)}
		}
	}
	if prcuo.mutation.PhaseCleared() && len(prcuo.mutation.PhaseIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "PlanRateCard.phase"`)
	}
	return nil
}

func (prcuo *PlanRateCardUpdateOne) sqlSave(ctx context.Context) (_node *PlanRateCard, err error) {
	if err := prcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(planratecard.Table, planratecard.Columns, sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString))
	id, ok := prcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "PlanRateCard.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := prcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, planratecard.FieldID)
		for _, f := range fields {
			if !planratecard.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != planratecard.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := prcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := prcuo.mutation.Metadata(); ok {
		_spec.SetField(planratecard.FieldMetadata, field.TypeJSON, value)
	}
	if prcuo.mutation.MetadataCleared() {
		_spec.ClearField(planratecard.FieldMetadata, field.TypeJSON)
	}
	if value, ok := prcuo.mutation.UpdatedAt(); ok {
		_spec.SetField(planratecard.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := prcuo.mutation.DeletedAt(); ok {
		_spec.SetField(planratecard.FieldDeletedAt, field.TypeTime, value)
	}
	if prcuo.mutation.DeletedAtCleared() {
		_spec.ClearField(planratecard.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := prcuo.mutation.Name(); ok {
		_spec.SetField(planratecard.FieldName, field.TypeString, value)
	}
	if value, ok := prcuo.mutation.Description(); ok {
		_spec.SetField(planratecard.FieldDescription, field.TypeString, value)
	}
	if prcuo.mutation.DescriptionCleared() {
		_spec.ClearField(planratecard.FieldDescription, field.TypeString)
	}
	if value, ok := prcuo.mutation.FeatureKey(); ok {
		_spec.SetField(planratecard.FieldFeatureKey, field.TypeString, value)
	}
	if prcuo.mutation.FeatureKeyCleared() {
		_spec.ClearField(planratecard.FieldFeatureKey, field.TypeString)
	}
	if value, ok := prcuo.mutation.EntitlementTemplate(); ok {
		vv, err := planratecard.ValueScanner.EntitlementTemplate.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(planratecard.FieldEntitlementTemplate, field.TypeString, vv)
	}
	if prcuo.mutation.EntitlementTemplateCleared() {
		_spec.ClearField(planratecard.FieldEntitlementTemplate, field.TypeString)
	}
	if value, ok := prcuo.mutation.TaxConfig(); ok {
		vv, err := planratecard.ValueScanner.TaxConfig.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(planratecard.FieldTaxConfig, field.TypeString, vv)
	}
	if prcuo.mutation.TaxConfigCleared() {
		_spec.ClearField(planratecard.FieldTaxConfig, field.TypeString)
	}
	if value, ok := prcuo.mutation.BillingCadence(); ok {
		_spec.SetField(planratecard.FieldBillingCadence, field.TypeString, value)
	}
	if prcuo.mutation.BillingCadenceCleared() {
		_spec.ClearField(planratecard.FieldBillingCadence, field.TypeString)
	}
	if value, ok := prcuo.mutation.Price(); ok {
		vv, err := planratecard.ValueScanner.Price.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(planratecard.FieldPrice, field.TypeString, vv)
	}
	if prcuo.mutation.PriceCleared() {
		_spec.ClearField(planratecard.FieldPrice, field.TypeString)
	}
	if prcuo.mutation.PhaseCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.PhaseTable,
			Columns: []string{planratecard.PhaseColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := prcuo.mutation.PhaseIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.PhaseTable,
			Columns: []string{planratecard.PhaseColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if prcuo.mutation.FeaturesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.FeaturesTable,
			Columns: []string{planratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := prcuo.mutation.FeaturesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   planratecard.FeaturesTable,
			Columns: []string{planratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &PlanRateCard{config: prcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, prcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{planratecard.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	prcuo.mutation.done = true
	return _node, nil
}
