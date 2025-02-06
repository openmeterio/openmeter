// Code generated by ent, DO NOT EDIT.

package subscription

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldDeletedAt, v))
}

// ActiveFrom applies equality check predicate on the "active_from" field. It's identical to ActiveFromEQ.
func ActiveFrom(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldActiveFrom, v))
}

// ActiveTo applies equality check predicate on the "active_to" field. It's identical to ActiveToEQ.
func ActiveTo(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldActiveTo, v))
}

// BillablesMustAlign applies equality check predicate on the "billables_must_align" field. It's identical to BillablesMustAlignEQ.
func BillablesMustAlign(v bool) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldBillablesMustAlign, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldName, v))
}

// Description applies equality check predicate on the "description" field. It's identical to DescriptionEQ.
func Description(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldDescription, v))
}

// PlanID applies equality check predicate on the "plan_id" field. It's identical to PlanIDEQ.
func PlanID(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldPlanID, v))
}

// CustomerID applies equality check predicate on the "customer_id" field. It's identical to CustomerIDEQ.
func CustomerID(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldCustomerID, v))
}

// Currency applies equality check predicate on the "currency" field. It's identical to CurrencyEQ.
func Currency(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldEQ(FieldCurrency, vc))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldNotNull(FieldDeletedAt))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldNotNull(FieldMetadata))
}

// ActiveFromEQ applies the EQ predicate on the "active_from" field.
func ActiveFromEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldActiveFrom, v))
}

// ActiveFromNEQ applies the NEQ predicate on the "active_from" field.
func ActiveFromNEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldActiveFrom, v))
}

// ActiveFromIn applies the In predicate on the "active_from" field.
func ActiveFromIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldActiveFrom, vs...))
}

// ActiveFromNotIn applies the NotIn predicate on the "active_from" field.
func ActiveFromNotIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldActiveFrom, vs...))
}

// ActiveFromGT applies the GT predicate on the "active_from" field.
func ActiveFromGT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldActiveFrom, v))
}

// ActiveFromGTE applies the GTE predicate on the "active_from" field.
func ActiveFromGTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldActiveFrom, v))
}

// ActiveFromLT applies the LT predicate on the "active_from" field.
func ActiveFromLT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldActiveFrom, v))
}

// ActiveFromLTE applies the LTE predicate on the "active_from" field.
func ActiveFromLTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldActiveFrom, v))
}

// ActiveToEQ applies the EQ predicate on the "active_to" field.
func ActiveToEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldActiveTo, v))
}

// ActiveToNEQ applies the NEQ predicate on the "active_to" field.
func ActiveToNEQ(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldActiveTo, v))
}

// ActiveToIn applies the In predicate on the "active_to" field.
func ActiveToIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldActiveTo, vs...))
}

// ActiveToNotIn applies the NotIn predicate on the "active_to" field.
func ActiveToNotIn(vs ...time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldActiveTo, vs...))
}

// ActiveToGT applies the GT predicate on the "active_to" field.
func ActiveToGT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldActiveTo, v))
}

// ActiveToGTE applies the GTE predicate on the "active_to" field.
func ActiveToGTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldActiveTo, v))
}

// ActiveToLT applies the LT predicate on the "active_to" field.
func ActiveToLT(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldActiveTo, v))
}

// ActiveToLTE applies the LTE predicate on the "active_to" field.
func ActiveToLTE(v time.Time) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldActiveTo, v))
}

// ActiveToIsNil applies the IsNil predicate on the "active_to" field.
func ActiveToIsNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldIsNull(FieldActiveTo))
}

// ActiveToNotNil applies the NotNil predicate on the "active_to" field.
func ActiveToNotNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldNotNull(FieldActiveTo))
}

// BillablesMustAlignEQ applies the EQ predicate on the "billables_must_align" field.
func BillablesMustAlignEQ(v bool) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldBillablesMustAlign, v))
}

// BillablesMustAlignNEQ applies the NEQ predicate on the "billables_must_align" field.
func BillablesMustAlignNEQ(v bool) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldBillablesMustAlign, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldName, v))
}

// DescriptionEQ applies the EQ predicate on the "description" field.
func DescriptionEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldDescription, v))
}

// DescriptionNEQ applies the NEQ predicate on the "description" field.
func DescriptionNEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldDescription, v))
}

// DescriptionIn applies the In predicate on the "description" field.
func DescriptionIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldDescription, vs...))
}

// DescriptionNotIn applies the NotIn predicate on the "description" field.
func DescriptionNotIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldDescription, vs...))
}

// DescriptionGT applies the GT predicate on the "description" field.
func DescriptionGT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldDescription, v))
}

// DescriptionGTE applies the GTE predicate on the "description" field.
func DescriptionGTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldDescription, v))
}

// DescriptionLT applies the LT predicate on the "description" field.
func DescriptionLT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldDescription, v))
}

// DescriptionLTE applies the LTE predicate on the "description" field.
func DescriptionLTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldDescription, v))
}

// DescriptionContains applies the Contains predicate on the "description" field.
func DescriptionContains(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContains(FieldDescription, v))
}

// DescriptionHasPrefix applies the HasPrefix predicate on the "description" field.
func DescriptionHasPrefix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasPrefix(FieldDescription, v))
}

// DescriptionHasSuffix applies the HasSuffix predicate on the "description" field.
func DescriptionHasSuffix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasSuffix(FieldDescription, v))
}

// DescriptionIsNil applies the IsNil predicate on the "description" field.
func DescriptionIsNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldIsNull(FieldDescription))
}

// DescriptionNotNil applies the NotNil predicate on the "description" field.
func DescriptionNotNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldNotNull(FieldDescription))
}

// DescriptionEqualFold applies the EqualFold predicate on the "description" field.
func DescriptionEqualFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldDescription, v))
}

// DescriptionContainsFold applies the ContainsFold predicate on the "description" field.
func DescriptionContainsFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldDescription, v))
}

// PlanIDEQ applies the EQ predicate on the "plan_id" field.
func PlanIDEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldPlanID, v))
}

// PlanIDNEQ applies the NEQ predicate on the "plan_id" field.
func PlanIDNEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldPlanID, v))
}

// PlanIDIn applies the In predicate on the "plan_id" field.
func PlanIDIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldPlanID, vs...))
}

// PlanIDNotIn applies the NotIn predicate on the "plan_id" field.
func PlanIDNotIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldPlanID, vs...))
}

// PlanIDGT applies the GT predicate on the "plan_id" field.
func PlanIDGT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldPlanID, v))
}

// PlanIDGTE applies the GTE predicate on the "plan_id" field.
func PlanIDGTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldPlanID, v))
}

// PlanIDLT applies the LT predicate on the "plan_id" field.
func PlanIDLT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldPlanID, v))
}

// PlanIDLTE applies the LTE predicate on the "plan_id" field.
func PlanIDLTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldPlanID, v))
}

// PlanIDContains applies the Contains predicate on the "plan_id" field.
func PlanIDContains(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContains(FieldPlanID, v))
}

// PlanIDHasPrefix applies the HasPrefix predicate on the "plan_id" field.
func PlanIDHasPrefix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasPrefix(FieldPlanID, v))
}

// PlanIDHasSuffix applies the HasSuffix predicate on the "plan_id" field.
func PlanIDHasSuffix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasSuffix(FieldPlanID, v))
}

// PlanIDIsNil applies the IsNil predicate on the "plan_id" field.
func PlanIDIsNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldIsNull(FieldPlanID))
}

// PlanIDNotNil applies the NotNil predicate on the "plan_id" field.
func PlanIDNotNil() predicate.Subscription {
	return predicate.Subscription(sql.FieldNotNull(FieldPlanID))
}

// PlanIDEqualFold applies the EqualFold predicate on the "plan_id" field.
func PlanIDEqualFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldPlanID, v))
}

// PlanIDContainsFold applies the ContainsFold predicate on the "plan_id" field.
func PlanIDContainsFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldPlanID, v))
}

// CustomerIDEQ applies the EQ predicate on the "customer_id" field.
func CustomerIDEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEQ(FieldCustomerID, v))
}

// CustomerIDNEQ applies the NEQ predicate on the "customer_id" field.
func CustomerIDNEQ(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNEQ(FieldCustomerID, v))
}

// CustomerIDIn applies the In predicate on the "customer_id" field.
func CustomerIDIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldIn(FieldCustomerID, vs...))
}

// CustomerIDNotIn applies the NotIn predicate on the "customer_id" field.
func CustomerIDNotIn(vs ...string) predicate.Subscription {
	return predicate.Subscription(sql.FieldNotIn(FieldCustomerID, vs...))
}

// CustomerIDGT applies the GT predicate on the "customer_id" field.
func CustomerIDGT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGT(FieldCustomerID, v))
}

// CustomerIDGTE applies the GTE predicate on the "customer_id" field.
func CustomerIDGTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldGTE(FieldCustomerID, v))
}

// CustomerIDLT applies the LT predicate on the "customer_id" field.
func CustomerIDLT(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLT(FieldCustomerID, v))
}

// CustomerIDLTE applies the LTE predicate on the "customer_id" field.
func CustomerIDLTE(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldLTE(FieldCustomerID, v))
}

// CustomerIDContains applies the Contains predicate on the "customer_id" field.
func CustomerIDContains(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContains(FieldCustomerID, v))
}

// CustomerIDHasPrefix applies the HasPrefix predicate on the "customer_id" field.
func CustomerIDHasPrefix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasPrefix(FieldCustomerID, v))
}

// CustomerIDHasSuffix applies the HasSuffix predicate on the "customer_id" field.
func CustomerIDHasSuffix(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldHasSuffix(FieldCustomerID, v))
}

// CustomerIDEqualFold applies the EqualFold predicate on the "customer_id" field.
func CustomerIDEqualFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldEqualFold(FieldCustomerID, v))
}

// CustomerIDContainsFold applies the ContainsFold predicate on the "customer_id" field.
func CustomerIDContainsFold(v string) predicate.Subscription {
	return predicate.Subscription(sql.FieldContainsFold(FieldCustomerID, v))
}

// CurrencyEQ applies the EQ predicate on the "currency" field.
func CurrencyEQ(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldEQ(FieldCurrency, vc))
}

// CurrencyNEQ applies the NEQ predicate on the "currency" field.
func CurrencyNEQ(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldNEQ(FieldCurrency, vc))
}

// CurrencyIn applies the In predicate on the "currency" field.
func CurrencyIn(vs ...currencyx.Code) predicate.Subscription {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.Subscription(sql.FieldIn(FieldCurrency, v...))
}

// CurrencyNotIn applies the NotIn predicate on the "currency" field.
func CurrencyNotIn(vs ...currencyx.Code) predicate.Subscription {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.Subscription(sql.FieldNotIn(FieldCurrency, v...))
}

// CurrencyGT applies the GT predicate on the "currency" field.
func CurrencyGT(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldGT(FieldCurrency, vc))
}

// CurrencyGTE applies the GTE predicate on the "currency" field.
func CurrencyGTE(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldGTE(FieldCurrency, vc))
}

// CurrencyLT applies the LT predicate on the "currency" field.
func CurrencyLT(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldLT(FieldCurrency, vc))
}

// CurrencyLTE applies the LTE predicate on the "currency" field.
func CurrencyLTE(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldLTE(FieldCurrency, vc))
}

// CurrencyContains applies the Contains predicate on the "currency" field.
func CurrencyContains(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldContains(FieldCurrency, vc))
}

// CurrencyHasPrefix applies the HasPrefix predicate on the "currency" field.
func CurrencyHasPrefix(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldHasPrefix(FieldCurrency, vc))
}

// CurrencyHasSuffix applies the HasSuffix predicate on the "currency" field.
func CurrencyHasSuffix(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldHasSuffix(FieldCurrency, vc))
}

// CurrencyEqualFold applies the EqualFold predicate on the "currency" field.
func CurrencyEqualFold(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldEqualFold(FieldCurrency, vc))
}

// CurrencyContainsFold applies the ContainsFold predicate on the "currency" field.
func CurrencyContainsFold(v currencyx.Code) predicate.Subscription {
	vc := string(v)
	return predicate.Subscription(sql.FieldContainsFold(FieldCurrency, vc))
}

// HasPlan applies the HasEdge predicate on the "plan" edge.
func HasPlan() predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanWith applies the HasEdge predicate on the "plan" edge with a given conditions (other predicates).
func HasPlanWith(preds ...predicate.Plan) predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := newPlanStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCustomer applies the HasEdge predicate on the "customer" edge.
func HasCustomer() predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, CustomerTable, CustomerColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCustomerWith applies the HasEdge predicate on the "customer" edge with a given conditions (other predicates).
func HasCustomerWith(preds ...predicate.Customer) predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := newCustomerStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPhases applies the HasEdge predicate on the "phases" edge.
func HasPhases() predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, PhasesTable, PhasesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPhasesWith applies the HasEdge predicate on the "phases" edge with a given conditions (other predicates).
func HasPhasesWith(preds ...predicate.SubscriptionPhase) predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := newPhasesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBillingLines applies the HasEdge predicate on the "billing_lines" edge.
func HasBillingLines() predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, BillingLinesTable, BillingLinesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBillingLinesWith applies the HasEdge predicate on the "billing_lines" edge with a given conditions (other predicates).
func HasBillingLinesWith(preds ...predicate.BillingInvoiceLine) predicate.Subscription {
	return predicate.Subscription(func(s *sql.Selector) {
		step := newBillingLinesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Subscription) predicate.Subscription {
	return predicate.Subscription(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Subscription) predicate.Subscription {
	return predicate.Subscription(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Subscription) predicate.Subscription {
	return predicate.Subscription(sql.NotPredicates(p))
}
