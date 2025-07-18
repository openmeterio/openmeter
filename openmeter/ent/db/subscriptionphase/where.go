// Code generated by ent, DO NOT EDIT.

package subscriptionphase

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldDeletedAt, v))
}

// SubscriptionID applies equality check predicate on the "subscription_id" field. It's identical to SubscriptionIDEQ.
func SubscriptionID(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldSubscriptionID, v))
}

// Key applies equality check predicate on the "key" field. It's identical to KeyEQ.
func Key(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldKey, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldName, v))
}

// Description applies equality check predicate on the "description" field. It's identical to DescriptionEQ.
func Description(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldDescription, v))
}

// ActiveFrom applies equality check predicate on the "active_from" field. It's identical to ActiveFromEQ.
func ActiveFrom(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldActiveFrom, v))
}

// SortHint applies equality check predicate on the "sort_hint" field. It's identical to SortHintEQ.
func SortHint(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldSortHint, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotNull(FieldDeletedAt))
}

// MetadataIsNil applies the IsNil predicate on the "metadata" field.
func MetadataIsNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIsNull(FieldMetadata))
}

// MetadataNotNil applies the NotNil predicate on the "metadata" field.
func MetadataNotNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotNull(FieldMetadata))
}

// SubscriptionIDEQ applies the EQ predicate on the "subscription_id" field.
func SubscriptionIDEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldSubscriptionID, v))
}

// SubscriptionIDNEQ applies the NEQ predicate on the "subscription_id" field.
func SubscriptionIDNEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldSubscriptionID, v))
}

// SubscriptionIDIn applies the In predicate on the "subscription_id" field.
func SubscriptionIDIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldSubscriptionID, vs...))
}

// SubscriptionIDNotIn applies the NotIn predicate on the "subscription_id" field.
func SubscriptionIDNotIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldSubscriptionID, vs...))
}

// SubscriptionIDGT applies the GT predicate on the "subscription_id" field.
func SubscriptionIDGT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldSubscriptionID, v))
}

// SubscriptionIDGTE applies the GTE predicate on the "subscription_id" field.
func SubscriptionIDGTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldSubscriptionID, v))
}

// SubscriptionIDLT applies the LT predicate on the "subscription_id" field.
func SubscriptionIDLT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldSubscriptionID, v))
}

// SubscriptionIDLTE applies the LTE predicate on the "subscription_id" field.
func SubscriptionIDLTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldSubscriptionID, v))
}

// SubscriptionIDContains applies the Contains predicate on the "subscription_id" field.
func SubscriptionIDContains(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContains(FieldSubscriptionID, v))
}

// SubscriptionIDHasPrefix applies the HasPrefix predicate on the "subscription_id" field.
func SubscriptionIDHasPrefix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasPrefix(FieldSubscriptionID, v))
}

// SubscriptionIDHasSuffix applies the HasSuffix predicate on the "subscription_id" field.
func SubscriptionIDHasSuffix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasSuffix(FieldSubscriptionID, v))
}

// SubscriptionIDEqualFold applies the EqualFold predicate on the "subscription_id" field.
func SubscriptionIDEqualFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldSubscriptionID, v))
}

// SubscriptionIDContainsFold applies the ContainsFold predicate on the "subscription_id" field.
func SubscriptionIDContainsFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldSubscriptionID, v))
}

// KeyEQ applies the EQ predicate on the "key" field.
func KeyEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldKey, v))
}

// KeyNEQ applies the NEQ predicate on the "key" field.
func KeyNEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldKey, v))
}

// KeyIn applies the In predicate on the "key" field.
func KeyIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldKey, vs...))
}

// KeyNotIn applies the NotIn predicate on the "key" field.
func KeyNotIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldKey, vs...))
}

// KeyGT applies the GT predicate on the "key" field.
func KeyGT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldKey, v))
}

// KeyGTE applies the GTE predicate on the "key" field.
func KeyGTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldKey, v))
}

// KeyLT applies the LT predicate on the "key" field.
func KeyLT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldKey, v))
}

// KeyLTE applies the LTE predicate on the "key" field.
func KeyLTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldKey, v))
}

// KeyContains applies the Contains predicate on the "key" field.
func KeyContains(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContains(FieldKey, v))
}

// KeyHasPrefix applies the HasPrefix predicate on the "key" field.
func KeyHasPrefix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasPrefix(FieldKey, v))
}

// KeyHasSuffix applies the HasSuffix predicate on the "key" field.
func KeyHasSuffix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasSuffix(FieldKey, v))
}

// KeyEqualFold applies the EqualFold predicate on the "key" field.
func KeyEqualFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldKey, v))
}

// KeyContainsFold applies the ContainsFold predicate on the "key" field.
func KeyContainsFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldKey, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldName, v))
}

// DescriptionEQ applies the EQ predicate on the "description" field.
func DescriptionEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldDescription, v))
}

// DescriptionNEQ applies the NEQ predicate on the "description" field.
func DescriptionNEQ(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldDescription, v))
}

// DescriptionIn applies the In predicate on the "description" field.
func DescriptionIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldDescription, vs...))
}

// DescriptionNotIn applies the NotIn predicate on the "description" field.
func DescriptionNotIn(vs ...string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldDescription, vs...))
}

// DescriptionGT applies the GT predicate on the "description" field.
func DescriptionGT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldDescription, v))
}

// DescriptionGTE applies the GTE predicate on the "description" field.
func DescriptionGTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldDescription, v))
}

// DescriptionLT applies the LT predicate on the "description" field.
func DescriptionLT(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldDescription, v))
}

// DescriptionLTE applies the LTE predicate on the "description" field.
func DescriptionLTE(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldDescription, v))
}

// DescriptionContains applies the Contains predicate on the "description" field.
func DescriptionContains(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContains(FieldDescription, v))
}

// DescriptionHasPrefix applies the HasPrefix predicate on the "description" field.
func DescriptionHasPrefix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasPrefix(FieldDescription, v))
}

// DescriptionHasSuffix applies the HasSuffix predicate on the "description" field.
func DescriptionHasSuffix(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldHasSuffix(FieldDescription, v))
}

// DescriptionIsNil applies the IsNil predicate on the "description" field.
func DescriptionIsNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIsNull(FieldDescription))
}

// DescriptionNotNil applies the NotNil predicate on the "description" field.
func DescriptionNotNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotNull(FieldDescription))
}

// DescriptionEqualFold applies the EqualFold predicate on the "description" field.
func DescriptionEqualFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEqualFold(FieldDescription, v))
}

// DescriptionContainsFold applies the ContainsFold predicate on the "description" field.
func DescriptionContainsFold(v string) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldContainsFold(FieldDescription, v))
}

// ActiveFromEQ applies the EQ predicate on the "active_from" field.
func ActiveFromEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldActiveFrom, v))
}

// ActiveFromNEQ applies the NEQ predicate on the "active_from" field.
func ActiveFromNEQ(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldActiveFrom, v))
}

// ActiveFromIn applies the In predicate on the "active_from" field.
func ActiveFromIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldActiveFrom, vs...))
}

// ActiveFromNotIn applies the NotIn predicate on the "active_from" field.
func ActiveFromNotIn(vs ...time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldActiveFrom, vs...))
}

// ActiveFromGT applies the GT predicate on the "active_from" field.
func ActiveFromGT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldActiveFrom, v))
}

// ActiveFromGTE applies the GTE predicate on the "active_from" field.
func ActiveFromGTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldActiveFrom, v))
}

// ActiveFromLT applies the LT predicate on the "active_from" field.
func ActiveFromLT(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldActiveFrom, v))
}

// ActiveFromLTE applies the LTE predicate on the "active_from" field.
func ActiveFromLTE(v time.Time) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldActiveFrom, v))
}

// SortHintEQ applies the EQ predicate on the "sort_hint" field.
func SortHintEQ(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldEQ(FieldSortHint, v))
}

// SortHintNEQ applies the NEQ predicate on the "sort_hint" field.
func SortHintNEQ(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNEQ(FieldSortHint, v))
}

// SortHintIn applies the In predicate on the "sort_hint" field.
func SortHintIn(vs ...uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIn(FieldSortHint, vs...))
}

// SortHintNotIn applies the NotIn predicate on the "sort_hint" field.
func SortHintNotIn(vs ...uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotIn(FieldSortHint, vs...))
}

// SortHintGT applies the GT predicate on the "sort_hint" field.
func SortHintGT(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGT(FieldSortHint, v))
}

// SortHintGTE applies the GTE predicate on the "sort_hint" field.
func SortHintGTE(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldGTE(FieldSortHint, v))
}

// SortHintLT applies the LT predicate on the "sort_hint" field.
func SortHintLT(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLT(FieldSortHint, v))
}

// SortHintLTE applies the LTE predicate on the "sort_hint" field.
func SortHintLTE(v uint8) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldLTE(FieldSortHint, v))
}

// SortHintIsNil applies the IsNil predicate on the "sort_hint" field.
func SortHintIsNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldIsNull(FieldSortHint))
}

// SortHintNotNil applies the NotNil predicate on the "sort_hint" field.
func SortHintNotNil() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.FieldNotNull(FieldSortHint))
}

// HasSubscription applies the HasEdge predicate on the "subscription" edge.
func HasSubscription() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionTable, SubscriptionColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSubscriptionWith applies the HasEdge predicate on the "subscription" edge with a given conditions (other predicates).
func HasSubscriptionWith(preds ...predicate.Subscription) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := newSubscriptionStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasItems applies the HasEdge predicate on the "items" edge.
func HasItems() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, ItemsTable, ItemsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasItemsWith applies the HasEdge predicate on the "items" edge with a given conditions (other predicates).
func HasItemsWith(preds ...predicate.SubscriptionItem) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := newItemsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBillingLines applies the HasEdge predicate on the "billing_lines" edge.
func HasBillingLines() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, BillingLinesTable, BillingLinesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBillingLinesWith applies the HasEdge predicate on the "billing_lines" edge with a given conditions (other predicates).
func HasBillingLinesWith(preds ...predicate.BillingInvoiceLine) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := newBillingLinesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBillingSplitLineGroups applies the HasEdge predicate on the "billing_split_line_groups" edge.
func HasBillingSplitLineGroups() predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, BillingSplitLineGroupsTable, BillingSplitLineGroupsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBillingSplitLineGroupsWith applies the HasEdge predicate on the "billing_split_line_groups" edge with a given conditions (other predicates).
func HasBillingSplitLineGroupsWith(preds ...predicate.BillingInvoiceSplitLineGroup) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(func(s *sql.Selector) {
		step := newBillingSplitLineGroupsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.SubscriptionPhase) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.SubscriptionPhase) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.SubscriptionPhase) predicate.SubscriptionPhase {
	return predicate.SubscriptionPhase(sql.NotPredicates(p))
}
