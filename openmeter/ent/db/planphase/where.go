// Code generated by ent, DO NOT EDIT.

package planphase

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldDeletedAt, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldName, v))
}

// Description applies equality check predicate on the "description" field. It's identical to DescriptionEQ.
func Description(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldDescription, v))
}

// Key applies equality check predicate on the "key" field. It's identical to KeyEQ.
func Key(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldKey, v))
}

// PlanID applies equality check predicate on the "plan_id" field. It's identical to PlanIDEQ.
func PlanID(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldPlanID, v))
}

// Index applies equality check predicate on the "index" field. It's identical to IndexEQ.
func Index(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldIndex, v))
}

// Duration applies equality check predicate on the "duration" field. It's identical to DurationEQ.
func Duration(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldEQ(FieldDuration, vc))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldNamespace, v))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotNull(FieldMetadata))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotNull(FieldDeletedAt))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldName, v))
}

// DescriptionEQ applies the EQ predicate on the "description" field.
func DescriptionEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldDescription, v))
}

// DescriptionNEQ applies the NEQ predicate on the "description" field.
func DescriptionNEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldDescription, v))
}

// DescriptionIn applies the In predicate on the "description" field.
func DescriptionIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldDescription, vs...))
}

// DescriptionNotIn applies the NotIn predicate on the "description" field.
func DescriptionNotIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldDescription, vs...))
}

// DescriptionGT applies the GT predicate on the "description" field.
func DescriptionGT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldDescription, v))
}

// DescriptionGTE applies the GTE predicate on the "description" field.
func DescriptionGTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldDescription, v))
}

// DescriptionLT applies the LT predicate on the "description" field.
func DescriptionLT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldDescription, v))
}

// DescriptionLTE applies the LTE predicate on the "description" field.
func DescriptionLTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldDescription, v))
}

// DescriptionContains applies the Contains predicate on the "description" field.
func DescriptionContains(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContains(FieldDescription, v))
}

// DescriptionHasPrefix applies the HasPrefix predicate on the "description" field.
func DescriptionHasPrefix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldDescription, v))
}

// DescriptionHasSuffix applies the HasSuffix predicate on the "description" field.
func DescriptionHasSuffix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldDescription, v))
}

// DescriptionIsNil applies the IsNil predicate on the "description" field.
func DescriptionIsNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIsNull(FieldDescription))
}

// DescriptionNotNil applies the NotNil predicate on the "description" field.
func DescriptionNotNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotNull(FieldDescription))
}

// DescriptionEqualFold applies the EqualFold predicate on the "description" field.
func DescriptionEqualFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldDescription, v))
}

// DescriptionContainsFold applies the ContainsFold predicate on the "description" field.
func DescriptionContainsFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldDescription, v))
}

// KeyEQ applies the EQ predicate on the "key" field.
func KeyEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldKey, v))
}

// KeyNEQ applies the NEQ predicate on the "key" field.
func KeyNEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldKey, v))
}

// KeyIn applies the In predicate on the "key" field.
func KeyIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldKey, vs...))
}

// KeyNotIn applies the NotIn predicate on the "key" field.
func KeyNotIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldKey, vs...))
}

// KeyGT applies the GT predicate on the "key" field.
func KeyGT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldKey, v))
}

// KeyGTE applies the GTE predicate on the "key" field.
func KeyGTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldKey, v))
}

// KeyLT applies the LT predicate on the "key" field.
func KeyLT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldKey, v))
}

// KeyLTE applies the LTE predicate on the "key" field.
func KeyLTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldKey, v))
}

// KeyContains applies the Contains predicate on the "key" field.
func KeyContains(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContains(FieldKey, v))
}

// KeyHasPrefix applies the HasPrefix predicate on the "key" field.
func KeyHasPrefix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldKey, v))
}

// KeyHasSuffix applies the HasSuffix predicate on the "key" field.
func KeyHasSuffix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldKey, v))
}

// KeyEqualFold applies the EqualFold predicate on the "key" field.
func KeyEqualFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldKey, v))
}

// KeyContainsFold applies the ContainsFold predicate on the "key" field.
func KeyContainsFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldKey, v))
}

// PlanIDEQ applies the EQ predicate on the "plan_id" field.
func PlanIDEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldPlanID, v))
}

// PlanIDNEQ applies the NEQ predicate on the "plan_id" field.
func PlanIDNEQ(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldPlanID, v))
}

// PlanIDIn applies the In predicate on the "plan_id" field.
func PlanIDIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldPlanID, vs...))
}

// PlanIDNotIn applies the NotIn predicate on the "plan_id" field.
func PlanIDNotIn(vs ...string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldPlanID, vs...))
}

// PlanIDGT applies the GT predicate on the "plan_id" field.
func PlanIDGT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldPlanID, v))
}

// PlanIDGTE applies the GTE predicate on the "plan_id" field.
func PlanIDGTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldPlanID, v))
}

// PlanIDLT applies the LT predicate on the "plan_id" field.
func PlanIDLT(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldPlanID, v))
}

// PlanIDLTE applies the LTE predicate on the "plan_id" field.
func PlanIDLTE(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldPlanID, v))
}

// PlanIDContains applies the Contains predicate on the "plan_id" field.
func PlanIDContains(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContains(FieldPlanID, v))
}

// PlanIDHasPrefix applies the HasPrefix predicate on the "plan_id" field.
func PlanIDHasPrefix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldPlanID, v))
}

// PlanIDHasSuffix applies the HasSuffix predicate on the "plan_id" field.
func PlanIDHasSuffix(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldPlanID, v))
}

// PlanIDEqualFold applies the EqualFold predicate on the "plan_id" field.
func PlanIDEqualFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEqualFold(FieldPlanID, v))
}

// PlanIDContainsFold applies the ContainsFold predicate on the "plan_id" field.
func PlanIDContainsFold(v string) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldContainsFold(FieldPlanID, v))
}

// IndexEQ applies the EQ predicate on the "index" field.
func IndexEQ(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldEQ(FieldIndex, v))
}

// IndexNEQ applies the NEQ predicate on the "index" field.
func IndexNEQ(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNEQ(FieldIndex, v))
}

// IndexIn applies the In predicate on the "index" field.
func IndexIn(vs ...uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIn(FieldIndex, vs...))
}

// IndexNotIn applies the NotIn predicate on the "index" field.
func IndexNotIn(vs ...uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotIn(FieldIndex, vs...))
}

// IndexGT applies the GT predicate on the "index" field.
func IndexGT(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGT(FieldIndex, v))
}

// IndexGTE applies the GTE predicate on the "index" field.
func IndexGTE(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldGTE(FieldIndex, v))
}

// IndexLT applies the LT predicate on the "index" field.
func IndexLT(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLT(FieldIndex, v))
}

// IndexLTE applies the LTE predicate on the "index" field.
func IndexLTE(v uint8) predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldLTE(FieldIndex, v))
}

// DurationEQ applies the EQ predicate on the "duration" field.
func DurationEQ(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldEQ(FieldDuration, vc))
}

// DurationNEQ applies the NEQ predicate on the "duration" field.
func DurationNEQ(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldNEQ(FieldDuration, vc))
}

// DurationIn applies the In predicate on the "duration" field.
func DurationIn(vs ...isodate.String) predicate.PlanPhase {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.PlanPhase(sql.FieldIn(FieldDuration, v...))
}

// DurationNotIn applies the NotIn predicate on the "duration" field.
func DurationNotIn(vs ...isodate.String) predicate.PlanPhase {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.PlanPhase(sql.FieldNotIn(FieldDuration, v...))
}

// DurationGT applies the GT predicate on the "duration" field.
func DurationGT(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldGT(FieldDuration, vc))
}

// DurationGTE applies the GTE predicate on the "duration" field.
func DurationGTE(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldGTE(FieldDuration, vc))
}

// DurationLT applies the LT predicate on the "duration" field.
func DurationLT(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldLT(FieldDuration, vc))
}

// DurationLTE applies the LTE predicate on the "duration" field.
func DurationLTE(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldLTE(FieldDuration, vc))
}

// DurationContains applies the Contains predicate on the "duration" field.
func DurationContains(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldContains(FieldDuration, vc))
}

// DurationHasPrefix applies the HasPrefix predicate on the "duration" field.
func DurationHasPrefix(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldHasPrefix(FieldDuration, vc))
}

// DurationHasSuffix applies the HasSuffix predicate on the "duration" field.
func DurationHasSuffix(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldHasSuffix(FieldDuration, vc))
}

// DurationIsNil applies the IsNil predicate on the "duration" field.
func DurationIsNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIsNull(FieldDuration))
}

// DurationNotNil applies the NotNil predicate on the "duration" field.
func DurationNotNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotNull(FieldDuration))
}

// DurationEqualFold applies the EqualFold predicate on the "duration" field.
func DurationEqualFold(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldEqualFold(FieldDuration, vc))
}

// DurationContainsFold applies the ContainsFold predicate on the "duration" field.
func DurationContainsFold(v isodate.String) predicate.PlanPhase {
	vc := string(v)
	return predicate.PlanPhase(sql.FieldContainsFold(FieldDuration, vc))
}

// DiscountsIsNil applies the IsNil predicate on the "discounts" field.
func DiscountsIsNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldIsNull(FieldDiscounts))
}

// DiscountsNotNil applies the NotNil predicate on the "discounts" field.
func DiscountsNotNil() predicate.PlanPhase {
	return predicate.PlanPhase(sql.FieldNotNull(FieldDiscounts))
}

// HasPlan applies the HasEdge predicate on the "plan" edge.
func HasPlan() predicate.PlanPhase {
	return predicate.PlanPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanWith applies the HasEdge predicate on the "plan" edge with a given conditions (other predicates).
func HasPlanWith(preds ...predicate.Plan) predicate.PlanPhase {
	return predicate.PlanPhase(func(s *sql.Selector) {
		step := newPlanStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasRatecards applies the HasEdge predicate on the "ratecards" edge.
func HasRatecards() predicate.PlanPhase {
	return predicate.PlanPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, RatecardsTable, RatecardsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasRatecardsWith applies the HasEdge predicate on the "ratecards" edge with a given conditions (other predicates).
func HasRatecardsWith(preds ...predicate.PlanRateCard) predicate.PlanPhase {
	return predicate.PlanPhase(func(s *sql.Selector) {
		step := newRatecardsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.PlanPhase) predicate.PlanPhase {
	return predicate.PlanPhase(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.PlanPhase) predicate.PlanPhase {
	return predicate.PlanPhase(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.PlanPhase) predicate.PlanPhase {
	return predicate.PlanPhase(sql.NotPredicates(p))
}
