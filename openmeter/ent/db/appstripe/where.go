// Code generated by ent, DO NOT EDIT.

package appstripe

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldDeletedAt, v))
}

// StripeAccountID applies equality check predicate on the "stripe_account_id" field. It's identical to StripeAccountIDEQ.
func StripeAccountID(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeAccountID, v))
}

// StripeLivemode applies equality check predicate on the "stripe_livemode" field. It's identical to StripeLivemodeEQ.
func StripeLivemode(v bool) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeLivemode, v))
}

// APIKey applies equality check predicate on the "api_key" field. It's identical to APIKeyEQ.
func APIKey(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldAPIKey, v))
}

// StripeWebhookID applies equality check predicate on the "stripe_webhook_id" field. It's identical to StripeWebhookIDEQ.
func StripeWebhookID(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeWebhookID, v))
}

// WebhookSecret applies equality check predicate on the "webhook_secret" field. It's identical to WebhookSecretEQ.
func WebhookSecret(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldWebhookSecret, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotNull(FieldDeletedAt))
}

// StripeAccountIDEQ applies the EQ predicate on the "stripe_account_id" field.
func StripeAccountIDEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeAccountID, v))
}

// StripeAccountIDNEQ applies the NEQ predicate on the "stripe_account_id" field.
func StripeAccountIDNEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldStripeAccountID, v))
}

// StripeAccountIDIn applies the In predicate on the "stripe_account_id" field.
func StripeAccountIDIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldStripeAccountID, vs...))
}

// StripeAccountIDNotIn applies the NotIn predicate on the "stripe_account_id" field.
func StripeAccountIDNotIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldStripeAccountID, vs...))
}

// StripeAccountIDGT applies the GT predicate on the "stripe_account_id" field.
func StripeAccountIDGT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldStripeAccountID, v))
}

// StripeAccountIDGTE applies the GTE predicate on the "stripe_account_id" field.
func StripeAccountIDGTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldStripeAccountID, v))
}

// StripeAccountIDLT applies the LT predicate on the "stripe_account_id" field.
func StripeAccountIDLT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldStripeAccountID, v))
}

// StripeAccountIDLTE applies the LTE predicate on the "stripe_account_id" field.
func StripeAccountIDLTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldStripeAccountID, v))
}

// StripeAccountIDContains applies the Contains predicate on the "stripe_account_id" field.
func StripeAccountIDContains(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContains(FieldStripeAccountID, v))
}

// StripeAccountIDHasPrefix applies the HasPrefix predicate on the "stripe_account_id" field.
func StripeAccountIDHasPrefix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasPrefix(FieldStripeAccountID, v))
}

// StripeAccountIDHasSuffix applies the HasSuffix predicate on the "stripe_account_id" field.
func StripeAccountIDHasSuffix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasSuffix(FieldStripeAccountID, v))
}

// StripeAccountIDEqualFold applies the EqualFold predicate on the "stripe_account_id" field.
func StripeAccountIDEqualFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldStripeAccountID, v))
}

// StripeAccountIDContainsFold applies the ContainsFold predicate on the "stripe_account_id" field.
func StripeAccountIDContainsFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldStripeAccountID, v))
}

// StripeLivemodeEQ applies the EQ predicate on the "stripe_livemode" field.
func StripeLivemodeEQ(v bool) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeLivemode, v))
}

// StripeLivemodeNEQ applies the NEQ predicate on the "stripe_livemode" field.
func StripeLivemodeNEQ(v bool) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldStripeLivemode, v))
}

// APIKeyEQ applies the EQ predicate on the "api_key" field.
func APIKeyEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldAPIKey, v))
}

// APIKeyNEQ applies the NEQ predicate on the "api_key" field.
func APIKeyNEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldAPIKey, v))
}

// APIKeyIn applies the In predicate on the "api_key" field.
func APIKeyIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldAPIKey, vs...))
}

// APIKeyNotIn applies the NotIn predicate on the "api_key" field.
func APIKeyNotIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldAPIKey, vs...))
}

// APIKeyGT applies the GT predicate on the "api_key" field.
func APIKeyGT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldAPIKey, v))
}

// APIKeyGTE applies the GTE predicate on the "api_key" field.
func APIKeyGTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldAPIKey, v))
}

// APIKeyLT applies the LT predicate on the "api_key" field.
func APIKeyLT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldAPIKey, v))
}

// APIKeyLTE applies the LTE predicate on the "api_key" field.
func APIKeyLTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldAPIKey, v))
}

// APIKeyContains applies the Contains predicate on the "api_key" field.
func APIKeyContains(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContains(FieldAPIKey, v))
}

// APIKeyHasPrefix applies the HasPrefix predicate on the "api_key" field.
func APIKeyHasPrefix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasPrefix(FieldAPIKey, v))
}

// APIKeyHasSuffix applies the HasSuffix predicate on the "api_key" field.
func APIKeyHasSuffix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasSuffix(FieldAPIKey, v))
}

// APIKeyEqualFold applies the EqualFold predicate on the "api_key" field.
func APIKeyEqualFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldAPIKey, v))
}

// APIKeyContainsFold applies the ContainsFold predicate on the "api_key" field.
func APIKeyContainsFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldAPIKey, v))
}

// StripeWebhookIDEQ applies the EQ predicate on the "stripe_webhook_id" field.
func StripeWebhookIDEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldStripeWebhookID, v))
}

// StripeWebhookIDNEQ applies the NEQ predicate on the "stripe_webhook_id" field.
func StripeWebhookIDNEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldStripeWebhookID, v))
}

// StripeWebhookIDIn applies the In predicate on the "stripe_webhook_id" field.
func StripeWebhookIDIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldStripeWebhookID, vs...))
}

// StripeWebhookIDNotIn applies the NotIn predicate on the "stripe_webhook_id" field.
func StripeWebhookIDNotIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldStripeWebhookID, vs...))
}

// StripeWebhookIDGT applies the GT predicate on the "stripe_webhook_id" field.
func StripeWebhookIDGT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldStripeWebhookID, v))
}

// StripeWebhookIDGTE applies the GTE predicate on the "stripe_webhook_id" field.
func StripeWebhookIDGTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldStripeWebhookID, v))
}

// StripeWebhookIDLT applies the LT predicate on the "stripe_webhook_id" field.
func StripeWebhookIDLT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldStripeWebhookID, v))
}

// StripeWebhookIDLTE applies the LTE predicate on the "stripe_webhook_id" field.
func StripeWebhookIDLTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldStripeWebhookID, v))
}

// StripeWebhookIDContains applies the Contains predicate on the "stripe_webhook_id" field.
func StripeWebhookIDContains(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContains(FieldStripeWebhookID, v))
}

// StripeWebhookIDHasPrefix applies the HasPrefix predicate on the "stripe_webhook_id" field.
func StripeWebhookIDHasPrefix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasPrefix(FieldStripeWebhookID, v))
}

// StripeWebhookIDHasSuffix applies the HasSuffix predicate on the "stripe_webhook_id" field.
func StripeWebhookIDHasSuffix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasSuffix(FieldStripeWebhookID, v))
}

// StripeWebhookIDEqualFold applies the EqualFold predicate on the "stripe_webhook_id" field.
func StripeWebhookIDEqualFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldStripeWebhookID, v))
}

// StripeWebhookIDContainsFold applies the ContainsFold predicate on the "stripe_webhook_id" field.
func StripeWebhookIDContainsFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldStripeWebhookID, v))
}

// WebhookSecretEQ applies the EQ predicate on the "webhook_secret" field.
func WebhookSecretEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEQ(FieldWebhookSecret, v))
}

// WebhookSecretNEQ applies the NEQ predicate on the "webhook_secret" field.
func WebhookSecretNEQ(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNEQ(FieldWebhookSecret, v))
}

// WebhookSecretIn applies the In predicate on the "webhook_secret" field.
func WebhookSecretIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldIn(FieldWebhookSecret, vs...))
}

// WebhookSecretNotIn applies the NotIn predicate on the "webhook_secret" field.
func WebhookSecretNotIn(vs ...string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldNotIn(FieldWebhookSecret, vs...))
}

// WebhookSecretGT applies the GT predicate on the "webhook_secret" field.
func WebhookSecretGT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGT(FieldWebhookSecret, v))
}

// WebhookSecretGTE applies the GTE predicate on the "webhook_secret" field.
func WebhookSecretGTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldGTE(FieldWebhookSecret, v))
}

// WebhookSecretLT applies the LT predicate on the "webhook_secret" field.
func WebhookSecretLT(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLT(FieldWebhookSecret, v))
}

// WebhookSecretLTE applies the LTE predicate on the "webhook_secret" field.
func WebhookSecretLTE(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldLTE(FieldWebhookSecret, v))
}

// WebhookSecretContains applies the Contains predicate on the "webhook_secret" field.
func WebhookSecretContains(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContains(FieldWebhookSecret, v))
}

// WebhookSecretHasPrefix applies the HasPrefix predicate on the "webhook_secret" field.
func WebhookSecretHasPrefix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasPrefix(FieldWebhookSecret, v))
}

// WebhookSecretHasSuffix applies the HasSuffix predicate on the "webhook_secret" field.
func WebhookSecretHasSuffix(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldHasSuffix(FieldWebhookSecret, v))
}

// WebhookSecretEqualFold applies the EqualFold predicate on the "webhook_secret" field.
func WebhookSecretEqualFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldEqualFold(FieldWebhookSecret, v))
}

// WebhookSecretContainsFold applies the ContainsFold predicate on the "webhook_secret" field.
func WebhookSecretContainsFold(v string) predicate.AppStripe {
	return predicate.AppStripe(sql.FieldContainsFold(FieldWebhookSecret, v))
}

// HasCustomerApps applies the HasEdge predicate on the "customer_apps" edge.
func HasCustomerApps() predicate.AppStripe {
	return predicate.AppStripe(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, CustomerAppsTable, CustomerAppsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCustomerAppsWith applies the HasEdge predicate on the "customer_apps" edge with a given conditions (other predicates).
func HasCustomerAppsWith(preds ...predicate.AppStripeCustomer) predicate.AppStripe {
	return predicate.AppStripe(func(s *sql.Selector) {
		step := newCustomerAppsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasApp applies the HasEdge predicate on the "app" edge.
func HasApp() predicate.AppStripe {
	return predicate.AppStripe(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AppTable, AppColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAppWith applies the HasEdge predicate on the "app" edge with a given conditions (other predicates).
func HasAppWith(preds ...predicate.App) predicate.AppStripe {
	return predicate.AppStripe(func(s *sql.Selector) {
		step := newAppStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.AppStripe) predicate.AppStripe {
	return predicate.AppStripe(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.AppStripe) predicate.AppStripe {
	return predicate.AppStripe(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.AppStripe) predicate.AppStripe {
	return predicate.AppStripe(sql.NotPredicates(p))
}
