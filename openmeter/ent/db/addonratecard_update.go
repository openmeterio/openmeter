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
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonratecard"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// AddonRateCardUpdate is the builder for updating AddonRateCard entities.
type AddonRateCardUpdate struct {
	config
	hooks    []Hook
	mutation *AddonRateCardMutation
}

// Where appends a list predicates to the AddonRateCardUpdate builder.
func (arcu *AddonRateCardUpdate) Where(ps ...predicate.AddonRateCard) *AddonRateCardUpdate {
	arcu.mutation.Where(ps...)
	return arcu
}

// SetMetadata sets the "metadata" field.
func (arcu *AddonRateCardUpdate) SetMetadata(m map[string]string) *AddonRateCardUpdate {
	arcu.mutation.SetMetadata(m)
	return arcu
}

// ClearMetadata clears the value of the "metadata" field.
func (arcu *AddonRateCardUpdate) ClearMetadata() *AddonRateCardUpdate {
	arcu.mutation.ClearMetadata()
	return arcu
}

// SetUpdatedAt sets the "updated_at" field.
func (arcu *AddonRateCardUpdate) SetUpdatedAt(t time.Time) *AddonRateCardUpdate {
	arcu.mutation.SetUpdatedAt(t)
	return arcu
}

// SetDeletedAt sets the "deleted_at" field.
func (arcu *AddonRateCardUpdate) SetDeletedAt(t time.Time) *AddonRateCardUpdate {
	arcu.mutation.SetDeletedAt(t)
	return arcu
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableDeletedAt(t *time.Time) *AddonRateCardUpdate {
	if t != nil {
		arcu.SetDeletedAt(*t)
	}
	return arcu
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (arcu *AddonRateCardUpdate) ClearDeletedAt() *AddonRateCardUpdate {
	arcu.mutation.ClearDeletedAt()
	return arcu
}

// SetName sets the "name" field.
func (arcu *AddonRateCardUpdate) SetName(s string) *AddonRateCardUpdate {
	arcu.mutation.SetName(s)
	return arcu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableName(s *string) *AddonRateCardUpdate {
	if s != nil {
		arcu.SetName(*s)
	}
	return arcu
}

// SetDescription sets the "description" field.
func (arcu *AddonRateCardUpdate) SetDescription(s string) *AddonRateCardUpdate {
	arcu.mutation.SetDescription(s)
	return arcu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableDescription(s *string) *AddonRateCardUpdate {
	if s != nil {
		arcu.SetDescription(*s)
	}
	return arcu
}

// ClearDescription clears the value of the "description" field.
func (arcu *AddonRateCardUpdate) ClearDescription() *AddonRateCardUpdate {
	arcu.mutation.ClearDescription()
	return arcu
}

// SetFeatureKey sets the "feature_key" field.
func (arcu *AddonRateCardUpdate) SetFeatureKey(s string) *AddonRateCardUpdate {
	arcu.mutation.SetFeatureKey(s)
	return arcu
}

// SetNillableFeatureKey sets the "feature_key" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableFeatureKey(s *string) *AddonRateCardUpdate {
	if s != nil {
		arcu.SetFeatureKey(*s)
	}
	return arcu
}

// ClearFeatureKey clears the value of the "feature_key" field.
func (arcu *AddonRateCardUpdate) ClearFeatureKey() *AddonRateCardUpdate {
	arcu.mutation.ClearFeatureKey()
	return arcu
}

// SetEntitlementTemplate sets the "entitlement_template" field.
func (arcu *AddonRateCardUpdate) SetEntitlementTemplate(pt *productcatalog.EntitlementTemplate) *AddonRateCardUpdate {
	arcu.mutation.SetEntitlementTemplate(pt)
	return arcu
}

// ClearEntitlementTemplate clears the value of the "entitlement_template" field.
func (arcu *AddonRateCardUpdate) ClearEntitlementTemplate() *AddonRateCardUpdate {
	arcu.mutation.ClearEntitlementTemplate()
	return arcu
}

// SetTaxConfig sets the "tax_config" field.
func (arcu *AddonRateCardUpdate) SetTaxConfig(pc *productcatalog.TaxConfig) *AddonRateCardUpdate {
	arcu.mutation.SetTaxConfig(pc)
	return arcu
}

// ClearTaxConfig clears the value of the "tax_config" field.
func (arcu *AddonRateCardUpdate) ClearTaxConfig() *AddonRateCardUpdate {
	arcu.mutation.ClearTaxConfig()
	return arcu
}

// SetBillingCadence sets the "billing_cadence" field.
func (arcu *AddonRateCardUpdate) SetBillingCadence(i isodate.String) *AddonRateCardUpdate {
	arcu.mutation.SetBillingCadence(i)
	return arcu
}

// SetNillableBillingCadence sets the "billing_cadence" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableBillingCadence(i *isodate.String) *AddonRateCardUpdate {
	if i != nil {
		arcu.SetBillingCadence(*i)
	}
	return arcu
}

// ClearBillingCadence clears the value of the "billing_cadence" field.
func (arcu *AddonRateCardUpdate) ClearBillingCadence() *AddonRateCardUpdate {
	arcu.mutation.ClearBillingCadence()
	return arcu
}

// SetPrice sets the "price" field.
func (arcu *AddonRateCardUpdate) SetPrice(pr *productcatalog.Price) *AddonRateCardUpdate {
	arcu.mutation.SetPrice(pr)
	return arcu
}

// ClearPrice clears the value of the "price" field.
func (arcu *AddonRateCardUpdate) ClearPrice() *AddonRateCardUpdate {
	arcu.mutation.ClearPrice()
	return arcu
}

// SetDiscounts sets the "discounts" field.
func (arcu *AddonRateCardUpdate) SetDiscounts(pr *productcatalog.Discounts) *AddonRateCardUpdate {
	arcu.mutation.SetDiscounts(pr)
	return arcu
}

// ClearDiscounts clears the value of the "discounts" field.
func (arcu *AddonRateCardUpdate) ClearDiscounts() *AddonRateCardUpdate {
	arcu.mutation.ClearDiscounts()
	return arcu
}

// SetAddonID sets the "addon_id" field.
func (arcu *AddonRateCardUpdate) SetAddonID(s string) *AddonRateCardUpdate {
	arcu.mutation.SetAddonID(s)
	return arcu
}

// SetNillableAddonID sets the "addon_id" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableAddonID(s *string) *AddonRateCardUpdate {
	if s != nil {
		arcu.SetAddonID(*s)
	}
	return arcu
}

// SetFeatureID sets the "feature_id" field.
func (arcu *AddonRateCardUpdate) SetFeatureID(s string) *AddonRateCardUpdate {
	arcu.mutation.SetFeatureID(s)
	return arcu
}

// SetNillableFeatureID sets the "feature_id" field if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableFeatureID(s *string) *AddonRateCardUpdate {
	if s != nil {
		arcu.SetFeatureID(*s)
	}
	return arcu
}

// ClearFeatureID clears the value of the "feature_id" field.
func (arcu *AddonRateCardUpdate) ClearFeatureID() *AddonRateCardUpdate {
	arcu.mutation.ClearFeatureID()
	return arcu
}

// SetAddon sets the "addon" edge to the Addon entity.
func (arcu *AddonRateCardUpdate) SetAddon(a *Addon) *AddonRateCardUpdate {
	return arcu.SetAddonID(a.ID)
}

// SetFeaturesID sets the "features" edge to the Feature entity by ID.
func (arcu *AddonRateCardUpdate) SetFeaturesID(id string) *AddonRateCardUpdate {
	arcu.mutation.SetFeaturesID(id)
	return arcu
}

// SetNillableFeaturesID sets the "features" edge to the Feature entity by ID if the given value is not nil.
func (arcu *AddonRateCardUpdate) SetNillableFeaturesID(id *string) *AddonRateCardUpdate {
	if id != nil {
		arcu = arcu.SetFeaturesID(*id)
	}
	return arcu
}

// SetFeatures sets the "features" edge to the Feature entity.
func (arcu *AddonRateCardUpdate) SetFeatures(f *Feature) *AddonRateCardUpdate {
	return arcu.SetFeaturesID(f.ID)
}

// AddSubscriptionAddonRateCardIDs adds the "subscription_addon_rate_cards" edge to the SubscriptionAddonRateCard entity by IDs.
func (arcu *AddonRateCardUpdate) AddSubscriptionAddonRateCardIDs(ids ...string) *AddonRateCardUpdate {
	arcu.mutation.AddSubscriptionAddonRateCardIDs(ids...)
	return arcu
}

// AddSubscriptionAddonRateCards adds the "subscription_addon_rate_cards" edges to the SubscriptionAddonRateCard entity.
func (arcu *AddonRateCardUpdate) AddSubscriptionAddonRateCards(s ...*SubscriptionAddonRateCard) *AddonRateCardUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return arcu.AddSubscriptionAddonRateCardIDs(ids...)
}

// Mutation returns the AddonRateCardMutation object of the builder.
func (arcu *AddonRateCardUpdate) Mutation() *AddonRateCardMutation {
	return arcu.mutation
}

// ClearAddon clears the "addon" edge to the Addon entity.
func (arcu *AddonRateCardUpdate) ClearAddon() *AddonRateCardUpdate {
	arcu.mutation.ClearAddon()
	return arcu
}

// ClearFeatures clears the "features" edge to the Feature entity.
func (arcu *AddonRateCardUpdate) ClearFeatures() *AddonRateCardUpdate {
	arcu.mutation.ClearFeatures()
	return arcu
}

// ClearSubscriptionAddonRateCards clears all "subscription_addon_rate_cards" edges to the SubscriptionAddonRateCard entity.
func (arcu *AddonRateCardUpdate) ClearSubscriptionAddonRateCards() *AddonRateCardUpdate {
	arcu.mutation.ClearSubscriptionAddonRateCards()
	return arcu
}

// RemoveSubscriptionAddonRateCardIDs removes the "subscription_addon_rate_cards" edge to SubscriptionAddonRateCard entities by IDs.
func (arcu *AddonRateCardUpdate) RemoveSubscriptionAddonRateCardIDs(ids ...string) *AddonRateCardUpdate {
	arcu.mutation.RemoveSubscriptionAddonRateCardIDs(ids...)
	return arcu
}

// RemoveSubscriptionAddonRateCards removes "subscription_addon_rate_cards" edges to SubscriptionAddonRateCard entities.
func (arcu *AddonRateCardUpdate) RemoveSubscriptionAddonRateCards(s ...*SubscriptionAddonRateCard) *AddonRateCardUpdate {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return arcu.RemoveSubscriptionAddonRateCardIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (arcu *AddonRateCardUpdate) Save(ctx context.Context) (int, error) {
	arcu.defaults()
	return withHooks(ctx, arcu.sqlSave, arcu.mutation, arcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (arcu *AddonRateCardUpdate) SaveX(ctx context.Context) int {
	affected, err := arcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (arcu *AddonRateCardUpdate) Exec(ctx context.Context) error {
	_, err := arcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (arcu *AddonRateCardUpdate) ExecX(ctx context.Context) {
	if err := arcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (arcu *AddonRateCardUpdate) defaults() {
	if _, ok := arcu.mutation.UpdatedAt(); !ok {
		v := addonratecard.UpdateDefaultUpdatedAt()
		arcu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (arcu *AddonRateCardUpdate) check() error {
	if v, ok := arcu.mutation.EntitlementTemplate(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "entitlement_template", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.entitlement_template": %w`, err)}
		}
	}
	if v, ok := arcu.mutation.TaxConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "tax_config", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.tax_config": %w`, err)}
		}
	}
	if v, ok := arcu.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.price": %w`, err)}
		}
	}
	if v, ok := arcu.mutation.AddonID(); ok {
		if err := addonratecard.AddonIDValidator(v); err != nil {
			return &ValidationError{Name: "addon_id", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.addon_id": %w`, err)}
		}
	}
	if arcu.mutation.AddonCleared() && len(arcu.mutation.AddonIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AddonRateCard.addon"`)
	}
	return nil
}

func (arcu *AddonRateCardUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := arcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(addonratecard.Table, addonratecard.Columns, sqlgraph.NewFieldSpec(addonratecard.FieldID, field.TypeString))
	if ps := arcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := arcu.mutation.Metadata(); ok {
		_spec.SetField(addonratecard.FieldMetadata, field.TypeJSON, value)
	}
	if arcu.mutation.MetadataCleared() {
		_spec.ClearField(addonratecard.FieldMetadata, field.TypeJSON)
	}
	if value, ok := arcu.mutation.UpdatedAt(); ok {
		_spec.SetField(addonratecard.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := arcu.mutation.DeletedAt(); ok {
		_spec.SetField(addonratecard.FieldDeletedAt, field.TypeTime, value)
	}
	if arcu.mutation.DeletedAtCleared() {
		_spec.ClearField(addonratecard.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := arcu.mutation.Name(); ok {
		_spec.SetField(addonratecard.FieldName, field.TypeString, value)
	}
	if value, ok := arcu.mutation.Description(); ok {
		_spec.SetField(addonratecard.FieldDescription, field.TypeString, value)
	}
	if arcu.mutation.DescriptionCleared() {
		_spec.ClearField(addonratecard.FieldDescription, field.TypeString)
	}
	if value, ok := arcu.mutation.FeatureKey(); ok {
		_spec.SetField(addonratecard.FieldFeatureKey, field.TypeString, value)
	}
	if arcu.mutation.FeatureKeyCleared() {
		_spec.ClearField(addonratecard.FieldFeatureKey, field.TypeString)
	}
	if value, ok := arcu.mutation.EntitlementTemplate(); ok {
		vv, err := addonratecard.ValueScanner.EntitlementTemplate.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(addonratecard.FieldEntitlementTemplate, field.TypeString, vv)
	}
	if arcu.mutation.EntitlementTemplateCleared() {
		_spec.ClearField(addonratecard.FieldEntitlementTemplate, field.TypeString)
	}
	if value, ok := arcu.mutation.TaxConfig(); ok {
		vv, err := addonratecard.ValueScanner.TaxConfig.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(addonratecard.FieldTaxConfig, field.TypeString, vv)
	}
	if arcu.mutation.TaxConfigCleared() {
		_spec.ClearField(addonratecard.FieldTaxConfig, field.TypeString)
	}
	if value, ok := arcu.mutation.BillingCadence(); ok {
		_spec.SetField(addonratecard.FieldBillingCadence, field.TypeString, value)
	}
	if arcu.mutation.BillingCadenceCleared() {
		_spec.ClearField(addonratecard.FieldBillingCadence, field.TypeString)
	}
	if value, ok := arcu.mutation.Price(); ok {
		vv, err := addonratecard.ValueScanner.Price.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(addonratecard.FieldPrice, field.TypeString, vv)
	}
	if arcu.mutation.PriceCleared() {
		_spec.ClearField(addonratecard.FieldPrice, field.TypeString)
	}
	if value, ok := arcu.mutation.Discounts(); ok {
		vv, err := addonratecard.ValueScanner.Discounts.Value(value)
		if err != nil {
			return 0, err
		}
		_spec.SetField(addonratecard.FieldDiscounts, field.TypeString, vv)
	}
	if arcu.mutation.DiscountsCleared() {
		_spec.ClearField(addonratecard.FieldDiscounts, field.TypeString)
	}
	if arcu.mutation.AddonCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.AddonTable,
			Columns: []string{addonratecard.AddonColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcu.mutation.AddonIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.AddonTable,
			Columns: []string{addonratecard.AddonColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if arcu.mutation.FeaturesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.FeaturesTable,
			Columns: []string{addonratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcu.mutation.FeaturesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.FeaturesTable,
			Columns: []string{addonratecard.FeaturesColumn},
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
	if arcu.mutation.SubscriptionAddonRateCardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcu.mutation.RemovedSubscriptionAddonRateCardsIDs(); len(nodes) > 0 && !arcu.mutation.SubscriptionAddonRateCardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcu.mutation.SubscriptionAddonRateCardsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, arcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{addonratecard.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	arcu.mutation.done = true
	return n, nil
}

// AddonRateCardUpdateOne is the builder for updating a single AddonRateCard entity.
type AddonRateCardUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *AddonRateCardMutation
}

// SetMetadata sets the "metadata" field.
func (arcuo *AddonRateCardUpdateOne) SetMetadata(m map[string]string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetMetadata(m)
	return arcuo
}

// ClearMetadata clears the value of the "metadata" field.
func (arcuo *AddonRateCardUpdateOne) ClearMetadata() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearMetadata()
	return arcuo
}

// SetUpdatedAt sets the "updated_at" field.
func (arcuo *AddonRateCardUpdateOne) SetUpdatedAt(t time.Time) *AddonRateCardUpdateOne {
	arcuo.mutation.SetUpdatedAt(t)
	return arcuo
}

// SetDeletedAt sets the "deleted_at" field.
func (arcuo *AddonRateCardUpdateOne) SetDeletedAt(t time.Time) *AddonRateCardUpdateOne {
	arcuo.mutation.SetDeletedAt(t)
	return arcuo
}

// SetNillableDeletedAt sets the "deleted_at" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableDeletedAt(t *time.Time) *AddonRateCardUpdateOne {
	if t != nil {
		arcuo.SetDeletedAt(*t)
	}
	return arcuo
}

// ClearDeletedAt clears the value of the "deleted_at" field.
func (arcuo *AddonRateCardUpdateOne) ClearDeletedAt() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearDeletedAt()
	return arcuo
}

// SetName sets the "name" field.
func (arcuo *AddonRateCardUpdateOne) SetName(s string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetName(s)
	return arcuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableName(s *string) *AddonRateCardUpdateOne {
	if s != nil {
		arcuo.SetName(*s)
	}
	return arcuo
}

// SetDescription sets the "description" field.
func (arcuo *AddonRateCardUpdateOne) SetDescription(s string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetDescription(s)
	return arcuo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableDescription(s *string) *AddonRateCardUpdateOne {
	if s != nil {
		arcuo.SetDescription(*s)
	}
	return arcuo
}

// ClearDescription clears the value of the "description" field.
func (arcuo *AddonRateCardUpdateOne) ClearDescription() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearDescription()
	return arcuo
}

// SetFeatureKey sets the "feature_key" field.
func (arcuo *AddonRateCardUpdateOne) SetFeatureKey(s string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetFeatureKey(s)
	return arcuo
}

// SetNillableFeatureKey sets the "feature_key" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableFeatureKey(s *string) *AddonRateCardUpdateOne {
	if s != nil {
		arcuo.SetFeatureKey(*s)
	}
	return arcuo
}

// ClearFeatureKey clears the value of the "feature_key" field.
func (arcuo *AddonRateCardUpdateOne) ClearFeatureKey() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearFeatureKey()
	return arcuo
}

// SetEntitlementTemplate sets the "entitlement_template" field.
func (arcuo *AddonRateCardUpdateOne) SetEntitlementTemplate(pt *productcatalog.EntitlementTemplate) *AddonRateCardUpdateOne {
	arcuo.mutation.SetEntitlementTemplate(pt)
	return arcuo
}

// ClearEntitlementTemplate clears the value of the "entitlement_template" field.
func (arcuo *AddonRateCardUpdateOne) ClearEntitlementTemplate() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearEntitlementTemplate()
	return arcuo
}

// SetTaxConfig sets the "tax_config" field.
func (arcuo *AddonRateCardUpdateOne) SetTaxConfig(pc *productcatalog.TaxConfig) *AddonRateCardUpdateOne {
	arcuo.mutation.SetTaxConfig(pc)
	return arcuo
}

// ClearTaxConfig clears the value of the "tax_config" field.
func (arcuo *AddonRateCardUpdateOne) ClearTaxConfig() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearTaxConfig()
	return arcuo
}

// SetBillingCadence sets the "billing_cadence" field.
func (arcuo *AddonRateCardUpdateOne) SetBillingCadence(i isodate.String) *AddonRateCardUpdateOne {
	arcuo.mutation.SetBillingCadence(i)
	return arcuo
}

// SetNillableBillingCadence sets the "billing_cadence" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableBillingCadence(i *isodate.String) *AddonRateCardUpdateOne {
	if i != nil {
		arcuo.SetBillingCadence(*i)
	}
	return arcuo
}

// ClearBillingCadence clears the value of the "billing_cadence" field.
func (arcuo *AddonRateCardUpdateOne) ClearBillingCadence() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearBillingCadence()
	return arcuo
}

// SetPrice sets the "price" field.
func (arcuo *AddonRateCardUpdateOne) SetPrice(pr *productcatalog.Price) *AddonRateCardUpdateOne {
	arcuo.mutation.SetPrice(pr)
	return arcuo
}

// ClearPrice clears the value of the "price" field.
func (arcuo *AddonRateCardUpdateOne) ClearPrice() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearPrice()
	return arcuo
}

// SetDiscounts sets the "discounts" field.
func (arcuo *AddonRateCardUpdateOne) SetDiscounts(pr *productcatalog.Discounts) *AddonRateCardUpdateOne {
	arcuo.mutation.SetDiscounts(pr)
	return arcuo
}

// ClearDiscounts clears the value of the "discounts" field.
func (arcuo *AddonRateCardUpdateOne) ClearDiscounts() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearDiscounts()
	return arcuo
}

// SetAddonID sets the "addon_id" field.
func (arcuo *AddonRateCardUpdateOne) SetAddonID(s string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetAddonID(s)
	return arcuo
}

// SetNillableAddonID sets the "addon_id" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableAddonID(s *string) *AddonRateCardUpdateOne {
	if s != nil {
		arcuo.SetAddonID(*s)
	}
	return arcuo
}

// SetFeatureID sets the "feature_id" field.
func (arcuo *AddonRateCardUpdateOne) SetFeatureID(s string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetFeatureID(s)
	return arcuo
}

// SetNillableFeatureID sets the "feature_id" field if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableFeatureID(s *string) *AddonRateCardUpdateOne {
	if s != nil {
		arcuo.SetFeatureID(*s)
	}
	return arcuo
}

// ClearFeatureID clears the value of the "feature_id" field.
func (arcuo *AddonRateCardUpdateOne) ClearFeatureID() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearFeatureID()
	return arcuo
}

// SetAddon sets the "addon" edge to the Addon entity.
func (arcuo *AddonRateCardUpdateOne) SetAddon(a *Addon) *AddonRateCardUpdateOne {
	return arcuo.SetAddonID(a.ID)
}

// SetFeaturesID sets the "features" edge to the Feature entity by ID.
func (arcuo *AddonRateCardUpdateOne) SetFeaturesID(id string) *AddonRateCardUpdateOne {
	arcuo.mutation.SetFeaturesID(id)
	return arcuo
}

// SetNillableFeaturesID sets the "features" edge to the Feature entity by ID if the given value is not nil.
func (arcuo *AddonRateCardUpdateOne) SetNillableFeaturesID(id *string) *AddonRateCardUpdateOne {
	if id != nil {
		arcuo = arcuo.SetFeaturesID(*id)
	}
	return arcuo
}

// SetFeatures sets the "features" edge to the Feature entity.
func (arcuo *AddonRateCardUpdateOne) SetFeatures(f *Feature) *AddonRateCardUpdateOne {
	return arcuo.SetFeaturesID(f.ID)
}

// AddSubscriptionAddonRateCardIDs adds the "subscription_addon_rate_cards" edge to the SubscriptionAddonRateCard entity by IDs.
func (arcuo *AddonRateCardUpdateOne) AddSubscriptionAddonRateCardIDs(ids ...string) *AddonRateCardUpdateOne {
	arcuo.mutation.AddSubscriptionAddonRateCardIDs(ids...)
	return arcuo
}

// AddSubscriptionAddonRateCards adds the "subscription_addon_rate_cards" edges to the SubscriptionAddonRateCard entity.
func (arcuo *AddonRateCardUpdateOne) AddSubscriptionAddonRateCards(s ...*SubscriptionAddonRateCard) *AddonRateCardUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return arcuo.AddSubscriptionAddonRateCardIDs(ids...)
}

// Mutation returns the AddonRateCardMutation object of the builder.
func (arcuo *AddonRateCardUpdateOne) Mutation() *AddonRateCardMutation {
	return arcuo.mutation
}

// ClearAddon clears the "addon" edge to the Addon entity.
func (arcuo *AddonRateCardUpdateOne) ClearAddon() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearAddon()
	return arcuo
}

// ClearFeatures clears the "features" edge to the Feature entity.
func (arcuo *AddonRateCardUpdateOne) ClearFeatures() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearFeatures()
	return arcuo
}

// ClearSubscriptionAddonRateCards clears all "subscription_addon_rate_cards" edges to the SubscriptionAddonRateCard entity.
func (arcuo *AddonRateCardUpdateOne) ClearSubscriptionAddonRateCards() *AddonRateCardUpdateOne {
	arcuo.mutation.ClearSubscriptionAddonRateCards()
	return arcuo
}

// RemoveSubscriptionAddonRateCardIDs removes the "subscription_addon_rate_cards" edge to SubscriptionAddonRateCard entities by IDs.
func (arcuo *AddonRateCardUpdateOne) RemoveSubscriptionAddonRateCardIDs(ids ...string) *AddonRateCardUpdateOne {
	arcuo.mutation.RemoveSubscriptionAddonRateCardIDs(ids...)
	return arcuo
}

// RemoveSubscriptionAddonRateCards removes "subscription_addon_rate_cards" edges to SubscriptionAddonRateCard entities.
func (arcuo *AddonRateCardUpdateOne) RemoveSubscriptionAddonRateCards(s ...*SubscriptionAddonRateCard) *AddonRateCardUpdateOne {
	ids := make([]string, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return arcuo.RemoveSubscriptionAddonRateCardIDs(ids...)
}

// Where appends a list predicates to the AddonRateCardUpdate builder.
func (arcuo *AddonRateCardUpdateOne) Where(ps ...predicate.AddonRateCard) *AddonRateCardUpdateOne {
	arcuo.mutation.Where(ps...)
	return arcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (arcuo *AddonRateCardUpdateOne) Select(field string, fields ...string) *AddonRateCardUpdateOne {
	arcuo.fields = append([]string{field}, fields...)
	return arcuo
}

// Save executes the query and returns the updated AddonRateCard entity.
func (arcuo *AddonRateCardUpdateOne) Save(ctx context.Context) (*AddonRateCard, error) {
	arcuo.defaults()
	return withHooks(ctx, arcuo.sqlSave, arcuo.mutation, arcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (arcuo *AddonRateCardUpdateOne) SaveX(ctx context.Context) *AddonRateCard {
	node, err := arcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (arcuo *AddonRateCardUpdateOne) Exec(ctx context.Context) error {
	_, err := arcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (arcuo *AddonRateCardUpdateOne) ExecX(ctx context.Context) {
	if err := arcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (arcuo *AddonRateCardUpdateOne) defaults() {
	if _, ok := arcuo.mutation.UpdatedAt(); !ok {
		v := addonratecard.UpdateDefaultUpdatedAt()
		arcuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (arcuo *AddonRateCardUpdateOne) check() error {
	if v, ok := arcuo.mutation.EntitlementTemplate(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "entitlement_template", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.entitlement_template": %w`, err)}
		}
	}
	if v, ok := arcuo.mutation.TaxConfig(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "tax_config", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.tax_config": %w`, err)}
		}
	}
	if v, ok := arcuo.mutation.Price(); ok {
		if err := v.Validate(); err != nil {
			return &ValidationError{Name: "price", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.price": %w`, err)}
		}
	}
	if v, ok := arcuo.mutation.AddonID(); ok {
		if err := addonratecard.AddonIDValidator(v); err != nil {
			return &ValidationError{Name: "addon_id", err: fmt.Errorf(`db: validator failed for field "AddonRateCard.addon_id": %w`, err)}
		}
	}
	if arcuo.mutation.AddonCleared() && len(arcuo.mutation.AddonIDs()) > 0 {
		return errors.New(`db: clearing a required unique edge "AddonRateCard.addon"`)
	}
	return nil
}

func (arcuo *AddonRateCardUpdateOne) sqlSave(ctx context.Context) (_node *AddonRateCard, err error) {
	if err := arcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(addonratecard.Table, addonratecard.Columns, sqlgraph.NewFieldSpec(addonratecard.FieldID, field.TypeString))
	id, ok := arcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "AddonRateCard.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := arcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, addonratecard.FieldID)
		for _, f := range fields {
			if !addonratecard.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != addonratecard.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := arcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := arcuo.mutation.Metadata(); ok {
		_spec.SetField(addonratecard.FieldMetadata, field.TypeJSON, value)
	}
	if arcuo.mutation.MetadataCleared() {
		_spec.ClearField(addonratecard.FieldMetadata, field.TypeJSON)
	}
	if value, ok := arcuo.mutation.UpdatedAt(); ok {
		_spec.SetField(addonratecard.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := arcuo.mutation.DeletedAt(); ok {
		_spec.SetField(addonratecard.FieldDeletedAt, field.TypeTime, value)
	}
	if arcuo.mutation.DeletedAtCleared() {
		_spec.ClearField(addonratecard.FieldDeletedAt, field.TypeTime)
	}
	if value, ok := arcuo.mutation.Name(); ok {
		_spec.SetField(addonratecard.FieldName, field.TypeString, value)
	}
	if value, ok := arcuo.mutation.Description(); ok {
		_spec.SetField(addonratecard.FieldDescription, field.TypeString, value)
	}
	if arcuo.mutation.DescriptionCleared() {
		_spec.ClearField(addonratecard.FieldDescription, field.TypeString)
	}
	if value, ok := arcuo.mutation.FeatureKey(); ok {
		_spec.SetField(addonratecard.FieldFeatureKey, field.TypeString, value)
	}
	if arcuo.mutation.FeatureKeyCleared() {
		_spec.ClearField(addonratecard.FieldFeatureKey, field.TypeString)
	}
	if value, ok := arcuo.mutation.EntitlementTemplate(); ok {
		vv, err := addonratecard.ValueScanner.EntitlementTemplate.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(addonratecard.FieldEntitlementTemplate, field.TypeString, vv)
	}
	if arcuo.mutation.EntitlementTemplateCleared() {
		_spec.ClearField(addonratecard.FieldEntitlementTemplate, field.TypeString)
	}
	if value, ok := arcuo.mutation.TaxConfig(); ok {
		vv, err := addonratecard.ValueScanner.TaxConfig.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(addonratecard.FieldTaxConfig, field.TypeString, vv)
	}
	if arcuo.mutation.TaxConfigCleared() {
		_spec.ClearField(addonratecard.FieldTaxConfig, field.TypeString)
	}
	if value, ok := arcuo.mutation.BillingCadence(); ok {
		_spec.SetField(addonratecard.FieldBillingCadence, field.TypeString, value)
	}
	if arcuo.mutation.BillingCadenceCleared() {
		_spec.ClearField(addonratecard.FieldBillingCadence, field.TypeString)
	}
	if value, ok := arcuo.mutation.Price(); ok {
		vv, err := addonratecard.ValueScanner.Price.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(addonratecard.FieldPrice, field.TypeString, vv)
	}
	if arcuo.mutation.PriceCleared() {
		_spec.ClearField(addonratecard.FieldPrice, field.TypeString)
	}
	if value, ok := arcuo.mutation.Discounts(); ok {
		vv, err := addonratecard.ValueScanner.Discounts.Value(value)
		if err != nil {
			return nil, err
		}
		_spec.SetField(addonratecard.FieldDiscounts, field.TypeString, vv)
	}
	if arcuo.mutation.DiscountsCleared() {
		_spec.ClearField(addonratecard.FieldDiscounts, field.TypeString)
	}
	if arcuo.mutation.AddonCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.AddonTable,
			Columns: []string{addonratecard.AddonColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcuo.mutation.AddonIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.AddonTable,
			Columns: []string{addonratecard.AddonColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if arcuo.mutation.FeaturesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.FeaturesTable,
			Columns: []string{addonratecard.FeaturesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(feature.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcuo.mutation.FeaturesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   addonratecard.FeaturesTable,
			Columns: []string{addonratecard.FeaturesColumn},
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
	if arcuo.mutation.SubscriptionAddonRateCardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcuo.mutation.RemovedSubscriptionAddonRateCardsIDs(); len(nodes) > 0 && !arcuo.mutation.SubscriptionAddonRateCardsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := arcuo.mutation.SubscriptionAddonRateCardsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   addonratecard.SubscriptionAddonRateCardsTable,
			Columns: []string{addonratecard.SubscriptionAddonRateCardsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(subscriptionaddonratecard.FieldID, field.TypeString),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &AddonRateCard{config: arcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, arcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{addonratecard.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	arcuo.mutation.done = true
	return _node, nil
}
