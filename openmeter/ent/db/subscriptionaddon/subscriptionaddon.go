// Code generated by ent, DO NOT EDIT.

package subscriptionaddon

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the subscriptionaddon type in the database.
	Label = "subscription_addon"
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
	// FieldAddonID holds the string denoting the addon_id field in the database.
	FieldAddonID = "addon_id"
	// FieldSubscriptionID holds the string denoting the subscription_id field in the database.
	FieldSubscriptionID = "subscription_id"
	// EdgeSubscription holds the string denoting the subscription edge name in mutations.
	EdgeSubscription = "subscription"
	// EdgeRateCards holds the string denoting the rate_cards edge name in mutations.
	EdgeRateCards = "rate_cards"
	// EdgeQuantities holds the string denoting the quantities edge name in mutations.
	EdgeQuantities = "quantities"
	// EdgeAddon holds the string denoting the addon edge name in mutations.
	EdgeAddon = "addon"
	// Table holds the table name of the subscriptionaddon in the database.
	Table = "subscription_addons"
	// SubscriptionTable is the table that holds the subscription relation/edge.
	SubscriptionTable = "subscription_addons"
	// SubscriptionInverseTable is the table name for the Subscription entity.
	// It exists in this package in order to avoid circular dependency with the "subscription" package.
	SubscriptionInverseTable = "subscriptions"
	// SubscriptionColumn is the table column denoting the subscription relation/edge.
	SubscriptionColumn = "subscription_id"
	// RateCardsTable is the table that holds the rate_cards relation/edge.
	RateCardsTable = "subscription_addon_rate_cards"
	// RateCardsInverseTable is the table name for the SubscriptionAddonRateCard entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionaddonratecard" package.
	RateCardsInverseTable = "subscription_addon_rate_cards"
	// RateCardsColumn is the table column denoting the rate_cards relation/edge.
	RateCardsColumn = "subscription_addon_id"
	// QuantitiesTable is the table that holds the quantities relation/edge.
	QuantitiesTable = "subscription_addon_quantities"
	// QuantitiesInverseTable is the table name for the SubscriptionAddonQuantity entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionaddonquantity" package.
	QuantitiesInverseTable = "subscription_addon_quantities"
	// QuantitiesColumn is the table column denoting the quantities relation/edge.
	QuantitiesColumn = "subscription_addon_id"
	// AddonTable is the table that holds the addon relation/edge.
	AddonTable = "subscription_addons"
	// AddonInverseTable is the table name for the Addon entity.
	// It exists in this package in order to avoid circular dependency with the "addon" package.
	AddonInverseTable = "addons"
	// AddonColumn is the table column denoting the addon relation/edge.
	AddonColumn = "addon_id"
)

// Columns holds all SQL columns for subscriptionaddon fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldAddonID,
	FieldSubscriptionID,
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
	// AddonIDValidator is a validator for the "addon_id" field. It is called by the builders before save.
	AddonIDValidator func(string) error
	// SubscriptionIDValidator is a validator for the "subscription_id" field. It is called by the builders before save.
	SubscriptionIDValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the SubscriptionAddon queries.
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

// ByAddonID orders the results by the addon_id field.
func ByAddonID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAddonID, opts...).ToFunc()
}

// BySubscriptionID orders the results by the subscription_id field.
func BySubscriptionID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionID, opts...).ToFunc()
}

// BySubscriptionField orders the results by subscription field.
func BySubscriptionField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionStep(), sql.OrderByField(field, opts...))
	}
}

// ByRateCardsCount orders the results by rate_cards count.
func ByRateCardsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newRateCardsStep(), opts...)
	}
}

// ByRateCards orders the results by rate_cards terms.
func ByRateCards(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newRateCardsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByQuantitiesCount orders the results by quantities count.
func ByQuantitiesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newQuantitiesStep(), opts...)
	}
}

// ByQuantities orders the results by quantities terms.
func ByQuantities(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newQuantitiesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByAddonField orders the results by addon field.
func ByAddonField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newAddonStep(), sql.OrderByField(field, opts...))
	}
}
func newSubscriptionStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionTable, SubscriptionColumn),
	)
}
func newRateCardsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(RateCardsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, RateCardsTable, RateCardsColumn),
	)
}
func newQuantitiesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(QuantitiesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, QuantitiesTable, QuantitiesColumn),
	)
}
func newAddonStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(AddonInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, AddonTable, AddonColumn),
	)
}
