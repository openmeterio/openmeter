// Code generated by ent, DO NOT EDIT.

package balancesnapshot

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the balancesnapshot type in the database.
	Label = "balance_snapshot"
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
	// FieldOwnerID holds the string denoting the owner_id field in the database.
	FieldOwnerID = "owner_id"
	// FieldGrantBalances holds the string denoting the grant_balances field in the database.
	FieldGrantBalances = "grant_balances"
	// FieldBalance holds the string denoting the balance field in the database.
	FieldBalance = "balance"
	// FieldOverage holds the string denoting the overage field in the database.
	FieldOverage = "overage"
	// FieldAt holds the string denoting the at field in the database.
	FieldAt = "at"
	// EdgeEntitlement holds the string denoting the entitlement edge name in mutations.
	EdgeEntitlement = "entitlement"
	// Table holds the table name of the balancesnapshot in the database.
	Table = "balance_snapshots"
	// EntitlementTable is the table that holds the entitlement relation/edge.
	EntitlementTable = "balance_snapshots"
	// EntitlementInverseTable is the table name for the Entitlement entity.
	// It exists in this package in order to avoid circular dependency with the "entitlement" package.
	EntitlementInverseTable = "entitlements"
	// EntitlementColumn is the table column denoting the entitlement relation/edge.
	EntitlementColumn = "owner_id"
)

// Columns holds all SQL columns for balancesnapshot fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldOwnerID,
	FieldGrantBalances,
	FieldBalance,
	FieldOverage,
	FieldAt,
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
)

// OrderOption defines the ordering options for the BalanceSnapshot queries.
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

// ByOwnerID orders the results by the owner_id field.
func ByOwnerID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldOwnerID, opts...).ToFunc()
}

// ByBalance orders the results by the balance field.
func ByBalance(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBalance, opts...).ToFunc()
}

// ByOverage orders the results by the overage field.
func ByOverage(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldOverage, opts...).ToFunc()
}

// ByAt orders the results by the at field.
func ByAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAt, opts...).ToFunc()
}

// ByEntitlementField orders the results by entitlement field.
func ByEntitlementField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEntitlementStep(), sql.OrderByField(field, opts...))
	}
}
func newEntitlementStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EntitlementInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, EntitlementTable, EntitlementColumn),
	)
}
