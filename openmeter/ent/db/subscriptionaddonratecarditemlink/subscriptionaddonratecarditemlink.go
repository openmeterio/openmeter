// Code generated by ent, DO NOT EDIT.

package subscriptionaddonratecarditemlink

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the subscriptionaddonratecarditemlink type in the database.
	Label = "subscription_addon_rate_card_item_link"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldSubscriptionAddonRateCardID holds the string denoting the subscription_addon_rate_card_id field in the database.
	FieldSubscriptionAddonRateCardID = "subscription_addon_rate_card_id"
	// FieldSubscriptionItemID holds the string denoting the subscription_item_id field in the database.
	FieldSubscriptionItemID = "subscription_item_id"
	// FieldSubscriptionItemThroughID holds the string denoting the subscription_item_through_id field in the database.
	FieldSubscriptionItemThroughID = "subscription_item_through_id"
	// EdgeSubscriptionAddonRateCard holds the string denoting the subscription_addon_rate_card edge name in mutations.
	EdgeSubscriptionAddonRateCard = "subscription_addon_rate_card"
	// EdgeSubscriptionItem holds the string denoting the subscription_item edge name in mutations.
	EdgeSubscriptionItem = "subscription_item"
	// Table holds the table name of the subscriptionaddonratecarditemlink in the database.
	Table = "subscription_addon_rate_card_item_links"
	// SubscriptionAddonRateCardTable is the table that holds the subscription_addon_rate_card relation/edge.
	SubscriptionAddonRateCardTable = "subscription_addon_rate_card_item_links"
	// SubscriptionAddonRateCardInverseTable is the table name for the SubscriptionAddonRateCard entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionaddonratecard" package.
	SubscriptionAddonRateCardInverseTable = "subscription_addon_rate_cards"
	// SubscriptionAddonRateCardColumn is the table column denoting the subscription_addon_rate_card relation/edge.
	SubscriptionAddonRateCardColumn = "subscription_addon_rate_card_id"
	// SubscriptionItemTable is the table that holds the subscription_item relation/edge.
	SubscriptionItemTable = "subscription_addon_rate_card_item_links"
	// SubscriptionItemInverseTable is the table name for the SubscriptionItem entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionitem" package.
	SubscriptionItemInverseTable = "subscription_items"
	// SubscriptionItemColumn is the table column denoting the subscription_item relation/edge.
	SubscriptionItemColumn = "subscription_item_id"
)

// Columns holds all SQL columns for subscriptionaddonratecarditemlink fields.
var Columns = []string{
	FieldID,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldSubscriptionAddonRateCardID,
	FieldSubscriptionItemID,
	FieldSubscriptionItemThroughID,
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
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// SubscriptionAddonRateCardIDValidator is a validator for the "subscription_addon_rate_card_id" field. It is called by the builders before save.
	SubscriptionAddonRateCardIDValidator func(string) error
	// SubscriptionItemIDValidator is a validator for the "subscription_item_id" field. It is called by the builders before save.
	SubscriptionItemIDValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the SubscriptionAddonRateCardItemLink queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
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

// BySubscriptionAddonRateCardID orders the results by the subscription_addon_rate_card_id field.
func BySubscriptionAddonRateCardID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionAddonRateCardID, opts...).ToFunc()
}

// BySubscriptionItemID orders the results by the subscription_item_id field.
func BySubscriptionItemID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionItemID, opts...).ToFunc()
}

// BySubscriptionItemThroughID orders the results by the subscription_item_through_id field.
func BySubscriptionItemThroughID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionItemThroughID, opts...).ToFunc()
}

// BySubscriptionAddonRateCardField orders the results by subscription_addon_rate_card field.
func BySubscriptionAddonRateCardField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionAddonRateCardStep(), sql.OrderByField(field, opts...))
	}
}

// BySubscriptionItemField orders the results by subscription_item field.
func BySubscriptionItemField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionItemStep(), sql.OrderByField(field, opts...))
	}
}
func newSubscriptionAddonRateCardStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionAddonRateCardInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionAddonRateCardTable, SubscriptionAddonRateCardColumn),
	)
}
func newSubscriptionItemStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionItemInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionItemTable, SubscriptionItemColumn),
	)
}
