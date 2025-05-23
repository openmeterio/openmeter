// Code generated by ent, DO NOT EDIT.

package notificationrule

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

const (
	// Label holds the string label denoting the notificationrule type in the database.
	Label = "notification_rule"
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
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDisabled holds the string denoting the disabled field in the database.
	FieldDisabled = "disabled"
	// FieldConfig holds the string denoting the config field in the database.
	FieldConfig = "config"
	// EdgeChannels holds the string denoting the channels edge name in mutations.
	EdgeChannels = "channels"
	// EdgeEvents holds the string denoting the events edge name in mutations.
	EdgeEvents = "events"
	// Table holds the table name of the notificationrule in the database.
	Table = "notification_rules"
	// ChannelsTable is the table that holds the channels relation/edge. The primary key declared below.
	ChannelsTable = "notification_channel_rules"
	// ChannelsInverseTable is the table name for the NotificationChannel entity.
	// It exists in this package in order to avoid circular dependency with the "notificationchannel" package.
	ChannelsInverseTable = "notification_channels"
	// EventsTable is the table that holds the events relation/edge.
	EventsTable = "notification_events"
	// EventsInverseTable is the table name for the NotificationEvent entity.
	// It exists in this package in order to avoid circular dependency with the "notificationevent" package.
	EventsInverseTable = "notification_events"
	// EventsColumn is the table column denoting the events relation/edge.
	EventsColumn = "rule_id"
)

// Columns holds all SQL columns for notificationrule fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldType,
	FieldName,
	FieldDisabled,
	FieldConfig,
}

var (
	// ChannelsPrimaryKey and ChannelsColumn2 are the table columns denoting the
	// primary key for the channels relation (M2M).
	ChannelsPrimaryKey = []string{"notification_channel_id", "notification_rule_id"}
)

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
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
	// DefaultDisabled holds the default value on creation for the "disabled" field.
	DefaultDisabled bool
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
	// ValueScanner of all NotificationRule fields.
	ValueScanner struct {
		Config field.TypeValueScanner[notification.RuleConfig]
	}
)

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type notification.EventType) error {
	switch _type {
	case "entitlements.balance.threshold", "entitlements.reset", "invoice.created", "invoice.updated":
		return nil
	default:
		return fmt.Errorf("notificationrule: invalid enum value for type field: %q", _type)
	}
}

// OrderOption defines the ordering options for the NotificationRule queries.
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

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDisabled orders the results by the disabled field.
func ByDisabled(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDisabled, opts...).ToFunc()
}

// ByConfig orders the results by the config field.
func ByConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldConfig, opts...).ToFunc()
}

// ByChannelsCount orders the results by channels count.
func ByChannelsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newChannelsStep(), opts...)
	}
}

// ByChannels orders the results by channels terms.
func ByChannels(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newChannelsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByEventsCount orders the results by events count.
func ByEventsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newEventsStep(), opts...)
	}
}

// ByEvents orders the results by events terms.
func ByEvents(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEventsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newChannelsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ChannelsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, ChannelsTable, ChannelsPrimaryKey...),
	)
}
func newEventsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EventsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, EventsTable, EventsColumn),
	)
}
