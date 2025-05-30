// Code generated by ent, DO NOT EDIT.

package planaddon

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldDeletedAt, v))
}

// PlanID applies equality check predicate on the "plan_id" field. It's identical to PlanIDEQ.
func PlanID(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldPlanID, v))
}

// AddonID applies equality check predicate on the "addon_id" field. It's identical to AddonIDEQ.
func AddonID(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldAddonID, v))
}

// FromPlanPhase applies equality check predicate on the "from_plan_phase" field. It's identical to FromPlanPhaseEQ.
func FromPlanPhase(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldFromPlanPhase, v))
}

// MaxQuantity applies equality check predicate on the "max_quantity" field. It's identical to MaxQuantityEQ.
func MaxQuantity(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldMaxQuantity, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContainsFold(FieldNamespace, v))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotNull(FieldMetadata))
}

// AnnotationsIsNil applies the IsNil predicate on the "annotations" field.
func AnnotationsIsNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIsNull(FieldAnnotations))
}

// AnnotationsNotNil applies the NotNil predicate on the "annotations" field.
func AnnotationsNotNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotNull(FieldAnnotations))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotNull(FieldDeletedAt))
}

// PlanIDEQ applies the EQ predicate on the "plan_id" field.
func PlanIDEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldPlanID, v))
}

// PlanIDNEQ applies the NEQ predicate on the "plan_id" field.
func PlanIDNEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldPlanID, v))
}

// PlanIDIn applies the In predicate on the "plan_id" field.
func PlanIDIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldPlanID, vs...))
}

// PlanIDNotIn applies the NotIn predicate on the "plan_id" field.
func PlanIDNotIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldPlanID, vs...))
}

// PlanIDGT applies the GT predicate on the "plan_id" field.
func PlanIDGT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldPlanID, v))
}

// PlanIDGTE applies the GTE predicate on the "plan_id" field.
func PlanIDGTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldPlanID, v))
}

// PlanIDLT applies the LT predicate on the "plan_id" field.
func PlanIDLT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldPlanID, v))
}

// PlanIDLTE applies the LTE predicate on the "plan_id" field.
func PlanIDLTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldPlanID, v))
}

// PlanIDContains applies the Contains predicate on the "plan_id" field.
func PlanIDContains(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContains(FieldPlanID, v))
}

// PlanIDHasPrefix applies the HasPrefix predicate on the "plan_id" field.
func PlanIDHasPrefix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasPrefix(FieldPlanID, v))
}

// PlanIDHasSuffix applies the HasSuffix predicate on the "plan_id" field.
func PlanIDHasSuffix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasSuffix(FieldPlanID, v))
}

// PlanIDEqualFold applies the EqualFold predicate on the "plan_id" field.
func PlanIDEqualFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEqualFold(FieldPlanID, v))
}

// PlanIDContainsFold applies the ContainsFold predicate on the "plan_id" field.
func PlanIDContainsFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContainsFold(FieldPlanID, v))
}

// AddonIDEQ applies the EQ predicate on the "addon_id" field.
func AddonIDEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldAddonID, v))
}

// AddonIDNEQ applies the NEQ predicate on the "addon_id" field.
func AddonIDNEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldAddonID, v))
}

// AddonIDIn applies the In predicate on the "addon_id" field.
func AddonIDIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldAddonID, vs...))
}

// AddonIDNotIn applies the NotIn predicate on the "addon_id" field.
func AddonIDNotIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldAddonID, vs...))
}

// AddonIDGT applies the GT predicate on the "addon_id" field.
func AddonIDGT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldAddonID, v))
}

// AddonIDGTE applies the GTE predicate on the "addon_id" field.
func AddonIDGTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldAddonID, v))
}

// AddonIDLT applies the LT predicate on the "addon_id" field.
func AddonIDLT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldAddonID, v))
}

// AddonIDLTE applies the LTE predicate on the "addon_id" field.
func AddonIDLTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldAddonID, v))
}

// AddonIDContains applies the Contains predicate on the "addon_id" field.
func AddonIDContains(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContains(FieldAddonID, v))
}

// AddonIDHasPrefix applies the HasPrefix predicate on the "addon_id" field.
func AddonIDHasPrefix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasPrefix(FieldAddonID, v))
}

// AddonIDHasSuffix applies the HasSuffix predicate on the "addon_id" field.
func AddonIDHasSuffix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasSuffix(FieldAddonID, v))
}

// AddonIDEqualFold applies the EqualFold predicate on the "addon_id" field.
func AddonIDEqualFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEqualFold(FieldAddonID, v))
}

// AddonIDContainsFold applies the ContainsFold predicate on the "addon_id" field.
func AddonIDContainsFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContainsFold(FieldAddonID, v))
}

// FromPlanPhaseEQ applies the EQ predicate on the "from_plan_phase" field.
func FromPlanPhaseEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldFromPlanPhase, v))
}

// FromPlanPhaseNEQ applies the NEQ predicate on the "from_plan_phase" field.
func FromPlanPhaseNEQ(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldFromPlanPhase, v))
}

// FromPlanPhaseIn applies the In predicate on the "from_plan_phase" field.
func FromPlanPhaseIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldFromPlanPhase, vs...))
}

// FromPlanPhaseNotIn applies the NotIn predicate on the "from_plan_phase" field.
func FromPlanPhaseNotIn(vs ...string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldFromPlanPhase, vs...))
}

// FromPlanPhaseGT applies the GT predicate on the "from_plan_phase" field.
func FromPlanPhaseGT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldFromPlanPhase, v))
}

// FromPlanPhaseGTE applies the GTE predicate on the "from_plan_phase" field.
func FromPlanPhaseGTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldFromPlanPhase, v))
}

// FromPlanPhaseLT applies the LT predicate on the "from_plan_phase" field.
func FromPlanPhaseLT(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldFromPlanPhase, v))
}

// FromPlanPhaseLTE applies the LTE predicate on the "from_plan_phase" field.
func FromPlanPhaseLTE(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldFromPlanPhase, v))
}

// FromPlanPhaseContains applies the Contains predicate on the "from_plan_phase" field.
func FromPlanPhaseContains(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContains(FieldFromPlanPhase, v))
}

// FromPlanPhaseHasPrefix applies the HasPrefix predicate on the "from_plan_phase" field.
func FromPlanPhaseHasPrefix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasPrefix(FieldFromPlanPhase, v))
}

// FromPlanPhaseHasSuffix applies the HasSuffix predicate on the "from_plan_phase" field.
func FromPlanPhaseHasSuffix(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldHasSuffix(FieldFromPlanPhase, v))
}

// FromPlanPhaseEqualFold applies the EqualFold predicate on the "from_plan_phase" field.
func FromPlanPhaseEqualFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEqualFold(FieldFromPlanPhase, v))
}

// FromPlanPhaseContainsFold applies the ContainsFold predicate on the "from_plan_phase" field.
func FromPlanPhaseContainsFold(v string) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldContainsFold(FieldFromPlanPhase, v))
}

// MaxQuantityEQ applies the EQ predicate on the "max_quantity" field.
func MaxQuantityEQ(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldEQ(FieldMaxQuantity, v))
}

// MaxQuantityNEQ applies the NEQ predicate on the "max_quantity" field.
func MaxQuantityNEQ(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNEQ(FieldMaxQuantity, v))
}

// MaxQuantityIn applies the In predicate on the "max_quantity" field.
func MaxQuantityIn(vs ...int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIn(FieldMaxQuantity, vs...))
}

// MaxQuantityNotIn applies the NotIn predicate on the "max_quantity" field.
func MaxQuantityNotIn(vs ...int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotIn(FieldMaxQuantity, vs...))
}

// MaxQuantityGT applies the GT predicate on the "max_quantity" field.
func MaxQuantityGT(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGT(FieldMaxQuantity, v))
}

// MaxQuantityGTE applies the GTE predicate on the "max_quantity" field.
func MaxQuantityGTE(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldGTE(FieldMaxQuantity, v))
}

// MaxQuantityLT applies the LT predicate on the "max_quantity" field.
func MaxQuantityLT(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLT(FieldMaxQuantity, v))
}

// MaxQuantityLTE applies the LTE predicate on the "max_quantity" field.
func MaxQuantityLTE(v int) predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldLTE(FieldMaxQuantity, v))
}

// MaxQuantityIsNil applies the IsNil predicate on the "max_quantity" field.
func MaxQuantityIsNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldIsNull(FieldMaxQuantity))
}

// MaxQuantityNotNil applies the NotNil predicate on the "max_quantity" field.
func MaxQuantityNotNil() predicate.PlanAddon {
	return predicate.PlanAddon(sql.FieldNotNull(FieldMaxQuantity))
}

// HasPlan applies the HasEdge predicate on the "plan" edge.
func HasPlan() predicate.PlanAddon {
	return predicate.PlanAddon(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanWith applies the HasEdge predicate on the "plan" edge with a given conditions (other predicates).
func HasPlanWith(preds ...predicate.Plan) predicate.PlanAddon {
	return predicate.PlanAddon(func(s *sql.Selector) {
		step := newPlanStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAddon applies the HasEdge predicate on the "addon" edge.
func HasAddon() predicate.PlanAddon {
	return predicate.PlanAddon(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, AddonTable, AddonColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAddonWith applies the HasEdge predicate on the "addon" edge with a given conditions (other predicates).
func HasAddonWith(preds ...predicate.Addon) predicate.PlanAddon {
	return predicate.PlanAddon(func(s *sql.Selector) {
		step := newAddonStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.PlanAddon) predicate.PlanAddon {
	return predicate.PlanAddon(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.PlanAddon) predicate.PlanAddon {
	return predicate.PlanAddon(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.PlanAddon) predicate.PlanAddon {
	return predicate.PlanAddon(sql.NotPredicates(p))
}
