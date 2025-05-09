// Code generated by ent, DO NOT EDIT.

package appstripe

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the appstripe type in the database.
	Label = "app_stripe"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldStripeAccountID holds the string denoting the stripe_account_id field in the database.
	FieldStripeAccountID = "stripe_account_id"
	// FieldStripeLivemode holds the string denoting the stripe_livemode field in the database.
	FieldStripeLivemode = "stripe_livemode"
	// FieldAPIKey holds the string denoting the api_key field in the database.
	FieldAPIKey = "api_key"
	// FieldMaskedAPIKey holds the string denoting the masked_api_key field in the database.
	FieldMaskedAPIKey = "masked_api_key"
	// FieldStripeWebhookID holds the string denoting the stripe_webhook_id field in the database.
	FieldStripeWebhookID = "stripe_webhook_id"
	// FieldWebhookSecret holds the string denoting the webhook_secret field in the database.
	FieldWebhookSecret = "webhook_secret"
	// EdgeCustomerApps holds the string denoting the customer_apps edge name in mutations.
	EdgeCustomerApps = "customer_apps"
	// EdgeApp holds the string denoting the app edge name in mutations.
	EdgeApp = "app"
	// Table holds the table name of the appstripe in the database.
	Table = "app_stripes"
	// CustomerAppsTable is the table that holds the customer_apps relation/edge.
	CustomerAppsTable = "app_stripe_customers"
	// CustomerAppsInverseTable is the table name for the AppStripeCustomer entity.
	// It exists in this package in order to avoid circular dependency with the "appstripecustomer" package.
	CustomerAppsInverseTable = "app_stripe_customers"
	// CustomerAppsColumn is the table column denoting the customer_apps relation/edge.
	CustomerAppsColumn = "app_id"
	// AppTable is the table that holds the app relation/edge.
	AppTable = "app_stripes"
	// AppInverseTable is the table name for the App entity.
	// It exists in this package in order to avoid circular dependency with the "dbapp" package.
	AppInverseTable = "apps"
	// AppColumn is the table column denoting the app relation/edge.
	AppColumn = "id"
)

// Columns holds all SQL columns for appstripe fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldStripeAccountID,
	FieldStripeLivemode,
	FieldAPIKey,
	FieldMaskedAPIKey,
	FieldStripeWebhookID,
	FieldWebhookSecret,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	NamespaceValidator func(string) error
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// APIKeyValidator is a validator for the "api_key" field. It is called by the builders before save.
	APIKeyValidator func(string) error
	// MaskedAPIKeyValidator is a validator for the "masked_api_key" field. It is called by the builders before save.
	MaskedAPIKeyValidator func(string) error
	// StripeWebhookIDValidator is a validator for the "stripe_webhook_id" field. It is called by the builders before save.
	StripeWebhookIDValidator func(string) error
	// WebhookSecretValidator is a validator for the "webhook_secret" field. It is called by the builders before save.
	WebhookSecretValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the AppStripe queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByNamespace orders the results by the namespace field.
func ByNamespace(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNamespace, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByDeletedAt orders the results by the deleted_at field.
func ByDeletedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDeletedAt, opts...).ToFunc()
}

// ByStripeAccountID orders the results by the stripe_account_id field.
func ByStripeAccountID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStripeAccountID, opts...).ToFunc()
}

// ByStripeLivemode orders the results by the stripe_livemode field.
func ByStripeLivemode(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStripeLivemode, opts...).ToFunc()
}

// ByAPIKey orders the results by the api_key field.
func ByAPIKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAPIKey, opts...).ToFunc()
}

// ByMaskedAPIKey orders the results by the masked_api_key field.
func ByMaskedAPIKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMaskedAPIKey, opts...).ToFunc()
}

// ByStripeWebhookID orders the results by the stripe_webhook_id field.
func ByStripeWebhookID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStripeWebhookID, opts...).ToFunc()
}

// ByWebhookSecret orders the results by the webhook_secret field.
func ByWebhookSecret(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWebhookSecret, opts...).ToFunc()
}

// ByCustomerAppsCount orders the results by customer_apps count.
func ByCustomerAppsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newCustomerAppsStep(), opts...)
	}
}

// ByCustomerApps orders the results by customer_apps terms.
func ByCustomerApps(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCustomerAppsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByAppField orders the results by app field.
func ByAppField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newAppStep(), sql.OrderByField(field, opts...))
	}
}
func newCustomerAppsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CustomerAppsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, CustomerAppsTable, CustomerAppsColumn),
	)
}
func newAppStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(AppInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, AppTable, AppColumn),
	)
}
