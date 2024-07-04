// Code generated by ent, DO NOT EDIT.

package entitlement

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldDeletedAt, v))
}

// FeatureID applies equality check predicate on the "feature_id" field. It's identical to FeatureIDEQ.
func FeatureID(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldFeatureID, v))
}

// FeatureKey applies equality check predicate on the "feature_key" field. It's identical to FeatureKeyEQ.
func FeatureKey(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldFeatureKey, v))
}

// SubjectKey applies equality check predicate on the "subject_key" field. It's identical to SubjectKeyEQ.
func SubjectKey(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldSubjectKey, v))
}

// MeasureUsageFrom applies equality check predicate on the "measure_usage_from" field. It's identical to MeasureUsageFromEQ.
func MeasureUsageFrom(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldMeasureUsageFrom, v))
}

// IssueAfterReset applies equality check predicate on the "issue_after_reset" field. It's identical to IssueAfterResetEQ.
func IssueAfterReset(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIssueAfterReset, v))
}

// IssueAfterResetPriority applies equality check predicate on the "issue_after_reset_priority" field. It's identical to IssueAfterResetPriorityEQ.
func IssueAfterResetPriority(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIssueAfterResetPriority, v))
}

// IsSoftLimit applies equality check predicate on the "is_soft_limit" field. It's identical to IsSoftLimitEQ.
func IsSoftLimit(v bool) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIsSoftLimit, v))
}

// UsagePeriodAnchor applies equality check predicate on the "usage_period_anchor" field. It's identical to UsagePeriodAnchorEQ.
func UsagePeriodAnchor(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldUsagePeriodAnchor, v))
}

// CurrentUsagePeriodStart applies equality check predicate on the "current_usage_period_start" field. It's identical to CurrentUsagePeriodStartEQ.
func CurrentUsagePeriodStart(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodEnd applies equality check predicate on the "current_usage_period_end" field. It's identical to CurrentUsagePeriodEndEQ.
func CurrentUsagePeriodEnd(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCurrentUsagePeriodEnd, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContainsFold(FieldNamespace, v))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldMetadata))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldDeletedAt))
}

// EntitlementTypeEQ applies the EQ predicate on the "entitlement_type" field.
func EntitlementTypeEQ(v EntitlementType) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldEntitlementType, v))
}

// EntitlementTypeNEQ applies the NEQ predicate on the "entitlement_type" field.
func EntitlementTypeNEQ(v EntitlementType) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldEntitlementType, v))
}

// EntitlementTypeIn applies the In predicate on the "entitlement_type" field.
func EntitlementTypeIn(vs ...EntitlementType) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldEntitlementType, vs...))
}

// EntitlementTypeNotIn applies the NotIn predicate on the "entitlement_type" field.
func EntitlementTypeNotIn(vs ...EntitlementType) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldEntitlementType, vs...))
}

// FeatureIDEQ applies the EQ predicate on the "feature_id" field.
func FeatureIDEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldFeatureID, v))
}

// FeatureIDNEQ applies the NEQ predicate on the "feature_id" field.
func FeatureIDNEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldFeatureID, v))
}

// FeatureIDIn applies the In predicate on the "feature_id" field.
func FeatureIDIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldFeatureID, vs...))
}

// FeatureIDNotIn applies the NotIn predicate on the "feature_id" field.
func FeatureIDNotIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldFeatureID, vs...))
}

// FeatureIDGT applies the GT predicate on the "feature_id" field.
func FeatureIDGT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldFeatureID, v))
}

// FeatureIDGTE applies the GTE predicate on the "feature_id" field.
func FeatureIDGTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldFeatureID, v))
}

// FeatureIDLT applies the LT predicate on the "feature_id" field.
func FeatureIDLT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldFeatureID, v))
}

// FeatureIDLTE applies the LTE predicate on the "feature_id" field.
func FeatureIDLTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldFeatureID, v))
}

// FeatureIDContains applies the Contains predicate on the "feature_id" field.
func FeatureIDContains(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContains(FieldFeatureID, v))
}

// FeatureIDHasPrefix applies the HasPrefix predicate on the "feature_id" field.
func FeatureIDHasPrefix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasPrefix(FieldFeatureID, v))
}

// FeatureIDHasSuffix applies the HasSuffix predicate on the "feature_id" field.
func FeatureIDHasSuffix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasSuffix(FieldFeatureID, v))
}

// FeatureIDEqualFold applies the EqualFold predicate on the "feature_id" field.
func FeatureIDEqualFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEqualFold(FieldFeatureID, v))
}

// FeatureIDContainsFold applies the ContainsFold predicate on the "feature_id" field.
func FeatureIDContainsFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContainsFold(FieldFeatureID, v))
}

// FeatureKeyEQ applies the EQ predicate on the "feature_key" field.
func FeatureKeyEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldFeatureKey, v))
}

// FeatureKeyNEQ applies the NEQ predicate on the "feature_key" field.
func FeatureKeyNEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldFeatureKey, v))
}

// FeatureKeyIn applies the In predicate on the "feature_key" field.
func FeatureKeyIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldFeatureKey, vs...))
}

// FeatureKeyNotIn applies the NotIn predicate on the "feature_key" field.
func FeatureKeyNotIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldFeatureKey, vs...))
}

// FeatureKeyGT applies the GT predicate on the "feature_key" field.
func FeatureKeyGT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldFeatureKey, v))
}

// FeatureKeyGTE applies the GTE predicate on the "feature_key" field.
func FeatureKeyGTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldFeatureKey, v))
}

// FeatureKeyLT applies the LT predicate on the "feature_key" field.
func FeatureKeyLT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldFeatureKey, v))
}

// FeatureKeyLTE applies the LTE predicate on the "feature_key" field.
func FeatureKeyLTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldFeatureKey, v))
}

// FeatureKeyContains applies the Contains predicate on the "feature_key" field.
func FeatureKeyContains(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContains(FieldFeatureKey, v))
}

// FeatureKeyHasPrefix applies the HasPrefix predicate on the "feature_key" field.
func FeatureKeyHasPrefix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasPrefix(FieldFeatureKey, v))
}

// FeatureKeyHasSuffix applies the HasSuffix predicate on the "feature_key" field.
func FeatureKeyHasSuffix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasSuffix(FieldFeatureKey, v))
}

// FeatureKeyEqualFold applies the EqualFold predicate on the "feature_key" field.
func FeatureKeyEqualFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEqualFold(FieldFeatureKey, v))
}

// FeatureKeyContainsFold applies the ContainsFold predicate on the "feature_key" field.
func FeatureKeyContainsFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContainsFold(FieldFeatureKey, v))
}

// SubjectKeyEQ applies the EQ predicate on the "subject_key" field.
func SubjectKeyEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldSubjectKey, v))
}

// SubjectKeyNEQ applies the NEQ predicate on the "subject_key" field.
func SubjectKeyNEQ(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldSubjectKey, v))
}

// SubjectKeyIn applies the In predicate on the "subject_key" field.
func SubjectKeyIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldSubjectKey, vs...))
}

// SubjectKeyNotIn applies the NotIn predicate on the "subject_key" field.
func SubjectKeyNotIn(vs ...string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldSubjectKey, vs...))
}

// SubjectKeyGT applies the GT predicate on the "subject_key" field.
func SubjectKeyGT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldSubjectKey, v))
}

// SubjectKeyGTE applies the GTE predicate on the "subject_key" field.
func SubjectKeyGTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldSubjectKey, v))
}

// SubjectKeyLT applies the LT predicate on the "subject_key" field.
func SubjectKeyLT(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldSubjectKey, v))
}

// SubjectKeyLTE applies the LTE predicate on the "subject_key" field.
func SubjectKeyLTE(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldSubjectKey, v))
}

// SubjectKeyContains applies the Contains predicate on the "subject_key" field.
func SubjectKeyContains(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContains(FieldSubjectKey, v))
}

// SubjectKeyHasPrefix applies the HasPrefix predicate on the "subject_key" field.
func SubjectKeyHasPrefix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasPrefix(FieldSubjectKey, v))
}

// SubjectKeyHasSuffix applies the HasSuffix predicate on the "subject_key" field.
func SubjectKeyHasSuffix(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldHasSuffix(FieldSubjectKey, v))
}

// SubjectKeyEqualFold applies the EqualFold predicate on the "subject_key" field.
func SubjectKeyEqualFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEqualFold(FieldSubjectKey, v))
}

// SubjectKeyContainsFold applies the ContainsFold predicate on the "subject_key" field.
func SubjectKeyContainsFold(v string) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldContainsFold(FieldSubjectKey, v))
}

// MeasureUsageFromEQ applies the EQ predicate on the "measure_usage_from" field.
func MeasureUsageFromEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromNEQ applies the NEQ predicate on the "measure_usage_from" field.
func MeasureUsageFromNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromIn applies the In predicate on the "measure_usage_from" field.
func MeasureUsageFromIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldMeasureUsageFrom, vs...))
}

// MeasureUsageFromNotIn applies the NotIn predicate on the "measure_usage_from" field.
func MeasureUsageFromNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldMeasureUsageFrom, vs...))
}

// MeasureUsageFromGT applies the GT predicate on the "measure_usage_from" field.
func MeasureUsageFromGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromGTE applies the GTE predicate on the "measure_usage_from" field.
func MeasureUsageFromGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromLT applies the LT predicate on the "measure_usage_from" field.
func MeasureUsageFromLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromLTE applies the LTE predicate on the "measure_usage_from" field.
func MeasureUsageFromLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldMeasureUsageFrom, v))
}

// MeasureUsageFromIsNil applies the IsNil predicate on the "measure_usage_from" field.
func MeasureUsageFromIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldMeasureUsageFrom))
}

// MeasureUsageFromNotNil applies the NotNil predicate on the "measure_usage_from" field.
func MeasureUsageFromNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldMeasureUsageFrom))
}

// IssueAfterResetEQ applies the EQ predicate on the "issue_after_reset" field.
func IssueAfterResetEQ(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIssueAfterReset, v))
}

// IssueAfterResetNEQ applies the NEQ predicate on the "issue_after_reset" field.
func IssueAfterResetNEQ(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldIssueAfterReset, v))
}

// IssueAfterResetIn applies the In predicate on the "issue_after_reset" field.
func IssueAfterResetIn(vs ...float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldIssueAfterReset, vs...))
}

// IssueAfterResetNotIn applies the NotIn predicate on the "issue_after_reset" field.
func IssueAfterResetNotIn(vs ...float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldIssueAfterReset, vs...))
}

// IssueAfterResetGT applies the GT predicate on the "issue_after_reset" field.
func IssueAfterResetGT(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldIssueAfterReset, v))
}

// IssueAfterResetGTE applies the GTE predicate on the "issue_after_reset" field.
func IssueAfterResetGTE(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldIssueAfterReset, v))
}

// IssueAfterResetLT applies the LT predicate on the "issue_after_reset" field.
func IssueAfterResetLT(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldIssueAfterReset, v))
}

// IssueAfterResetLTE applies the LTE predicate on the "issue_after_reset" field.
func IssueAfterResetLTE(v float64) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldIssueAfterReset, v))
}

// IssueAfterResetIsNil applies the IsNil predicate on the "issue_after_reset" field.
func IssueAfterResetIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldIssueAfterReset))
}

// IssueAfterResetNotNil applies the NotNil predicate on the "issue_after_reset" field.
func IssueAfterResetNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldIssueAfterReset))
}

// IssueAfterResetPriorityEQ applies the EQ predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityEQ(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityNEQ applies the NEQ predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityNEQ(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityIn applies the In predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityIn(vs ...uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldIssueAfterResetPriority, vs...))
}

// IssueAfterResetPriorityNotIn applies the NotIn predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityNotIn(vs ...uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldIssueAfterResetPriority, vs...))
}

// IssueAfterResetPriorityGT applies the GT predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityGT(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityGTE applies the GTE predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityGTE(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityLT applies the LT predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityLT(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityLTE applies the LTE predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityLTE(v uint8) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldIssueAfterResetPriority, v))
}

// IssueAfterResetPriorityIsNil applies the IsNil predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldIssueAfterResetPriority))
}

// IssueAfterResetPriorityNotNil applies the NotNil predicate on the "issue_after_reset_priority" field.
func IssueAfterResetPriorityNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldIssueAfterResetPriority))
}

// IsSoftLimitEQ applies the EQ predicate on the "is_soft_limit" field.
func IsSoftLimitEQ(v bool) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldIsSoftLimit, v))
}

// IsSoftLimitNEQ applies the NEQ predicate on the "is_soft_limit" field.
func IsSoftLimitNEQ(v bool) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldIsSoftLimit, v))
}

// IsSoftLimitIsNil applies the IsNil predicate on the "is_soft_limit" field.
func IsSoftLimitIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldIsSoftLimit))
}

// IsSoftLimitNotNil applies the NotNil predicate on the "is_soft_limit" field.
func IsSoftLimitNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldIsSoftLimit))
}

// ConfigIsNil applies the IsNil predicate on the "config" field.
func ConfigIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldConfig))
}

// ConfigNotNil applies the NotNil predicate on the "config" field.
func ConfigNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldConfig))
}

// UsagePeriodIntervalEQ applies the EQ predicate on the "usage_period_interval" field.
func UsagePeriodIntervalEQ(v UsagePeriodInterval) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldUsagePeriodInterval, v))
}

// UsagePeriodIntervalNEQ applies the NEQ predicate on the "usage_period_interval" field.
func UsagePeriodIntervalNEQ(v UsagePeriodInterval) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldUsagePeriodInterval, v))
}

// UsagePeriodIntervalIn applies the In predicate on the "usage_period_interval" field.
func UsagePeriodIntervalIn(vs ...UsagePeriodInterval) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldUsagePeriodInterval, vs...))
}

// UsagePeriodIntervalNotIn applies the NotIn predicate on the "usage_period_interval" field.
func UsagePeriodIntervalNotIn(vs ...UsagePeriodInterval) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldUsagePeriodInterval, vs...))
}

// UsagePeriodIntervalIsNil applies the IsNil predicate on the "usage_period_interval" field.
func UsagePeriodIntervalIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldUsagePeriodInterval))
}

// UsagePeriodIntervalNotNil applies the NotNil predicate on the "usage_period_interval" field.
func UsagePeriodIntervalNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldUsagePeriodInterval))
}

// UsagePeriodAnchorEQ applies the EQ predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorNEQ applies the NEQ predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorIn applies the In predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldUsagePeriodAnchor, vs...))
}

// UsagePeriodAnchorNotIn applies the NotIn predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldUsagePeriodAnchor, vs...))
}

// UsagePeriodAnchorGT applies the GT predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorGTE applies the GTE predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorLT applies the LT predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorLTE applies the LTE predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldUsagePeriodAnchor, v))
}

// UsagePeriodAnchorIsNil applies the IsNil predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldUsagePeriodAnchor))
}

// UsagePeriodAnchorNotNil applies the NotNil predicate on the "usage_period_anchor" field.
func UsagePeriodAnchorNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldUsagePeriodAnchor))
}

// CurrentUsagePeriodStartEQ applies the EQ predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartNEQ applies the NEQ predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartIn applies the In predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldCurrentUsagePeriodStart, vs...))
}

// CurrentUsagePeriodStartNotIn applies the NotIn predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldCurrentUsagePeriodStart, vs...))
}

// CurrentUsagePeriodStartGT applies the GT predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartGTE applies the GTE predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartLT applies the LT predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartLTE applies the LTE predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldCurrentUsagePeriodStart, v))
}

// CurrentUsagePeriodStartIsNil applies the IsNil predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldCurrentUsagePeriodStart))
}

// CurrentUsagePeriodStartNotNil applies the NotNil predicate on the "current_usage_period_start" field.
func CurrentUsagePeriodStartNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldCurrentUsagePeriodStart))
}

// CurrentUsagePeriodEndEQ applies the EQ predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldEQ(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndNEQ applies the NEQ predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndNEQ(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNEQ(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndIn applies the In predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIn(FieldCurrentUsagePeriodEnd, vs...))
}

// CurrentUsagePeriodEndNotIn applies the NotIn predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndNotIn(vs ...time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotIn(FieldCurrentUsagePeriodEnd, vs...))
}

// CurrentUsagePeriodEndGT applies the GT predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndGT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGT(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndGTE applies the GTE predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndGTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldGTE(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndLT applies the LT predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndLT(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLT(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndLTE applies the LTE predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndLTE(v time.Time) predicate.Entitlement {
	return predicate.Entitlement(sql.FieldLTE(FieldCurrentUsagePeriodEnd, v))
}

// CurrentUsagePeriodEndIsNil applies the IsNil predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndIsNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldIsNull(FieldCurrentUsagePeriodEnd))
}

// CurrentUsagePeriodEndNotNil applies the NotNil predicate on the "current_usage_period_end" field.
func CurrentUsagePeriodEndNotNil() predicate.Entitlement {
	return predicate.Entitlement(sql.FieldNotNull(FieldCurrentUsagePeriodEnd))
}

// HasUsageReset applies the HasEdge predicate on the "usage_reset" edge.
func HasUsageReset() predicate.Entitlement {
	return predicate.Entitlement(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, UsageResetTable, UsageResetColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasUsageResetWith applies the HasEdge predicate on the "usage_reset" edge with a given conditions (other predicates).
func HasUsageResetWith(preds ...predicate.UsageReset) predicate.Entitlement {
	return predicate.Entitlement(func(s *sql.Selector) {
		step := newUsageResetStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Entitlement) predicate.Entitlement {
	return predicate.Entitlement(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Entitlement) predicate.Entitlement {
	return predicate.Entitlement(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Entitlement) predicate.Entitlement {
	return predicate.Entitlement(sql.NotPredicates(p))
}
