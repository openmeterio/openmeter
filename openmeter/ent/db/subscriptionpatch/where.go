// Code generated by ent, DO NOT EDIT.

package subscriptionpatch

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldDeletedAt, v))
}

// SubscriptionID applies equality check predicate on the "subscription_id" field. It's identical to SubscriptionIDEQ.
func SubscriptionID(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldSubscriptionID, v))
}

// AppliedAt applies equality check predicate on the "applied_at" field. It's identical to AppliedAtEQ.
func AppliedAt(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldAppliedAt, v))
}

// BatchIndex applies equality check predicate on the "batch_index" field. It's identical to BatchIndexEQ.
func BatchIndex(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldBatchIndex, v))
}

// Operation applies equality check predicate on the "operation" field. It's identical to OperationEQ.
func Operation(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldOperation, v))
}

// Path applies equality check predicate on the "path" field. It's identical to PathEQ.
func Path(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldPath, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotNull(FieldDeletedAt))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotNull(FieldMetadata))
}

// SubscriptionIDEQ applies the EQ predicate on the "subscription_id" field.
func SubscriptionIDEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldSubscriptionID, v))
}

// SubscriptionIDNEQ applies the NEQ predicate on the "subscription_id" field.
func SubscriptionIDNEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldSubscriptionID, v))
}

// SubscriptionIDIn applies the In predicate on the "subscription_id" field.
func SubscriptionIDIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldSubscriptionID, vs...))
}

// SubscriptionIDNotIn applies the NotIn predicate on the "subscription_id" field.
func SubscriptionIDNotIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldSubscriptionID, vs...))
}

// SubscriptionIDGT applies the GT predicate on the "subscription_id" field.
func SubscriptionIDGT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldSubscriptionID, v))
}

// SubscriptionIDGTE applies the GTE predicate on the "subscription_id" field.
func SubscriptionIDGTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldSubscriptionID, v))
}

// SubscriptionIDLT applies the LT predicate on the "subscription_id" field.
func SubscriptionIDLT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldSubscriptionID, v))
}

// SubscriptionIDLTE applies the LTE predicate on the "subscription_id" field.
func SubscriptionIDLTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldSubscriptionID, v))
}

// SubscriptionIDContains applies the Contains predicate on the "subscription_id" field.
func SubscriptionIDContains(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContains(FieldSubscriptionID, v))
}

// SubscriptionIDHasPrefix applies the HasPrefix predicate on the "subscription_id" field.
func SubscriptionIDHasPrefix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasPrefix(FieldSubscriptionID, v))
}

// SubscriptionIDHasSuffix applies the HasSuffix predicate on the "subscription_id" field.
func SubscriptionIDHasSuffix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasSuffix(FieldSubscriptionID, v))
}

// SubscriptionIDEqualFold applies the EqualFold predicate on the "subscription_id" field.
func SubscriptionIDEqualFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEqualFold(FieldSubscriptionID, v))
}

// SubscriptionIDContainsFold applies the ContainsFold predicate on the "subscription_id" field.
func SubscriptionIDContainsFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContainsFold(FieldSubscriptionID, v))
}

// AppliedAtEQ applies the EQ predicate on the "applied_at" field.
func AppliedAtEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldAppliedAt, v))
}

// AppliedAtNEQ applies the NEQ predicate on the "applied_at" field.
func AppliedAtNEQ(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldAppliedAt, v))
}

// AppliedAtIn applies the In predicate on the "applied_at" field.
func AppliedAtIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldAppliedAt, vs...))
}

// AppliedAtNotIn applies the NotIn predicate on the "applied_at" field.
func AppliedAtNotIn(vs ...time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldAppliedAt, vs...))
}

// AppliedAtGT applies the GT predicate on the "applied_at" field.
func AppliedAtGT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldAppliedAt, v))
}

// AppliedAtGTE applies the GTE predicate on the "applied_at" field.
func AppliedAtGTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldAppliedAt, v))
}

// AppliedAtLT applies the LT predicate on the "applied_at" field.
func AppliedAtLT(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldAppliedAt, v))
}

// AppliedAtLTE applies the LTE predicate on the "applied_at" field.
func AppliedAtLTE(v time.Time) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldAppliedAt, v))
}

// BatchIndexEQ applies the EQ predicate on the "batch_index" field.
func BatchIndexEQ(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldBatchIndex, v))
}

// BatchIndexNEQ applies the NEQ predicate on the "batch_index" field.
func BatchIndexNEQ(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldBatchIndex, v))
}

// BatchIndexIn applies the In predicate on the "batch_index" field.
func BatchIndexIn(vs ...int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldBatchIndex, vs...))
}

// BatchIndexNotIn applies the NotIn predicate on the "batch_index" field.
func BatchIndexNotIn(vs ...int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldBatchIndex, vs...))
}

// BatchIndexGT applies the GT predicate on the "batch_index" field.
func BatchIndexGT(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldBatchIndex, v))
}

// BatchIndexGTE applies the GTE predicate on the "batch_index" field.
func BatchIndexGTE(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldBatchIndex, v))
}

// BatchIndexLT applies the LT predicate on the "batch_index" field.
func BatchIndexLT(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldBatchIndex, v))
}

// BatchIndexLTE applies the LTE predicate on the "batch_index" field.
func BatchIndexLTE(v int) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldBatchIndex, v))
}

// OperationEQ applies the EQ predicate on the "operation" field.
func OperationEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldOperation, v))
}

// OperationNEQ applies the NEQ predicate on the "operation" field.
func OperationNEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldOperation, v))
}

// OperationIn applies the In predicate on the "operation" field.
func OperationIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldOperation, vs...))
}

// OperationNotIn applies the NotIn predicate on the "operation" field.
func OperationNotIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldOperation, vs...))
}

// OperationGT applies the GT predicate on the "operation" field.
func OperationGT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldOperation, v))
}

// OperationGTE applies the GTE predicate on the "operation" field.
func OperationGTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldOperation, v))
}

// OperationLT applies the LT predicate on the "operation" field.
func OperationLT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldOperation, v))
}

// OperationLTE applies the LTE predicate on the "operation" field.
func OperationLTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldOperation, v))
}

// OperationContains applies the Contains predicate on the "operation" field.
func OperationContains(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContains(FieldOperation, v))
}

// OperationHasPrefix applies the HasPrefix predicate on the "operation" field.
func OperationHasPrefix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasPrefix(FieldOperation, v))
}

// OperationHasSuffix applies the HasSuffix predicate on the "operation" field.
func OperationHasSuffix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasSuffix(FieldOperation, v))
}

// OperationEqualFold applies the EqualFold predicate on the "operation" field.
func OperationEqualFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEqualFold(FieldOperation, v))
}

// OperationContainsFold applies the ContainsFold predicate on the "operation" field.
func OperationContainsFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContainsFold(FieldOperation, v))
}

// PathEQ applies the EQ predicate on the "path" field.
func PathEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEQ(FieldPath, v))
}

// PathNEQ applies the NEQ predicate on the "path" field.
func PathNEQ(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNEQ(FieldPath, v))
}

// PathIn applies the In predicate on the "path" field.
func PathIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldIn(FieldPath, vs...))
}

// PathNotIn applies the NotIn predicate on the "path" field.
func PathNotIn(vs ...string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldNotIn(FieldPath, vs...))
}

// PathGT applies the GT predicate on the "path" field.
func PathGT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGT(FieldPath, v))
}

// PathGTE applies the GTE predicate on the "path" field.
func PathGTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldGTE(FieldPath, v))
}

// PathLT applies the LT predicate on the "path" field.
func PathLT(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLT(FieldPath, v))
}

// PathLTE applies the LTE predicate on the "path" field.
func PathLTE(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldLTE(FieldPath, v))
}

// PathContains applies the Contains predicate on the "path" field.
func PathContains(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContains(FieldPath, v))
}

// PathHasPrefix applies the HasPrefix predicate on the "path" field.
func PathHasPrefix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasPrefix(FieldPath, v))
}

// PathHasSuffix applies the HasSuffix predicate on the "path" field.
func PathHasSuffix(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldHasSuffix(FieldPath, v))
}

// PathEqualFold applies the EqualFold predicate on the "path" field.
func PathEqualFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldEqualFold(FieldPath, v))
}

// PathContainsFold applies the ContainsFold predicate on the "path" field.
func PathContainsFold(v string) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.FieldContainsFold(FieldPath, v))
}

// HasSubscription applies the HasEdge predicate on the "subscription" edge.
func HasSubscription() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionTable, SubscriptionColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSubscriptionWith applies the HasEdge predicate on the "subscription" edge with a given conditions (other predicates).
func HasSubscriptionWith(preds ...predicate.Subscription) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := newSubscriptionStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasValueAddItem applies the HasEdge predicate on the "value_add_item" edge.
func HasValueAddItem() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ValueAddItemTable, ValueAddItemColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasValueAddItemWith applies the HasEdge predicate on the "value_add_item" edge with a given conditions (other predicates).
func HasValueAddItemWith(preds ...predicate.SubscriptionPatchValueAddItem) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := newValueAddItemStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasValueAddPhase applies the HasEdge predicate on the "value_add_phase" edge.
func HasValueAddPhase() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ValueAddPhaseTable, ValueAddPhaseColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasValueAddPhaseWith applies the HasEdge predicate on the "value_add_phase" edge with a given conditions (other predicates).
func HasValueAddPhaseWith(preds ...predicate.SubscriptionPatchValueAddPhase) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := newValueAddPhaseStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasValueRemovePhase applies the HasEdge predicate on the "value_remove_phase" edge.
func HasValueRemovePhase() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ValueRemovePhaseTable, ValueRemovePhaseColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasValueRemovePhaseWith applies the HasEdge predicate on the "value_remove_phase" edge with a given conditions (other predicates).
func HasValueRemovePhaseWith(preds ...predicate.SubscriptionPatchValueRemovePhase) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := newValueRemovePhaseStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasValueExtendPhase applies the HasEdge predicate on the "value_extend_phase" edge.
func HasValueExtendPhase() predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ValueExtendPhaseTable, ValueExtendPhaseColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasValueExtendPhaseWith applies the HasEdge predicate on the "value_extend_phase" edge with a given conditions (other predicates).
func HasValueExtendPhaseWith(preds ...predicate.SubscriptionPatchValueExtendPhase) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(func(s *sql.Selector) {
		step := newValueExtendPhaseStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.SubscriptionPatch) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.SubscriptionPatch) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.SubscriptionPatch) predicate.SubscriptionPatch {
	return predicate.SubscriptionPatch(sql.NotPredicates(p))
}
