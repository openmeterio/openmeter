// Code generated by ent, DO NOT EDIT.

package billingcustomeroverride

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

const (
	// Label holds the string label denoting the billingcustomeroverride type in the database.
	Label = "billing_customer_override"
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
	// FieldCustomerID holds the string denoting the customer_id field in the database.
	FieldCustomerID = "customer_id"
	// FieldBillingProfileID holds the string denoting the billing_profile_id field in the database.
	FieldBillingProfileID = "billing_profile_id"
	// FieldCollectionAlignment holds the string denoting the collection_alignment field in the database.
	FieldCollectionAlignment = "collection_alignment"
	// FieldItemCollectionPeriod holds the string denoting the item_collection_period field in the database.
	FieldItemCollectionPeriod = "item_collection_period"
	// FieldInvoiceAutoAdvance holds the string denoting the invoice_auto_advance field in the database.
	FieldInvoiceAutoAdvance = "invoice_auto_advance"
	// FieldInvoiceDraftPeriod holds the string denoting the invoice_draft_period field in the database.
	FieldInvoiceDraftPeriod = "invoice_draft_period"
	// FieldInvoiceDueAfter holds the string denoting the invoice_due_after field in the database.
	FieldInvoiceDueAfter = "invoice_due_after"
	// FieldInvoiceCollectionMethod holds the string denoting the invoice_collection_method field in the database.
	FieldInvoiceCollectionMethod = "invoice_collection_method"
	// EdgeCustomer holds the string denoting the customer edge name in mutations.
	EdgeCustomer = "customer"
	// EdgeBillingProfile holds the string denoting the billing_profile edge name in mutations.
	EdgeBillingProfile = "billing_profile"
	// Table holds the table name of the billingcustomeroverride in the database.
	Table = "billing_customer_overrides"
	// CustomerTable is the table that holds the customer relation/edge.
	CustomerTable = "billing_customer_overrides"
	// CustomerInverseTable is the table name for the Customer entity.
	// It exists in this package in order to avoid circular dependency with the "customer" package.
	CustomerInverseTable = "customers"
	// CustomerColumn is the table column denoting the customer relation/edge.
	CustomerColumn = "customer_id"
	// BillingProfileTable is the table that holds the billing_profile relation/edge.
	BillingProfileTable = "billing_customer_overrides"
	// BillingProfileInverseTable is the table name for the BillingProfile entity.
	// It exists in this package in order to avoid circular dependency with the "billingprofile" package.
	BillingProfileInverseTable = "billing_profiles"
	// BillingProfileColumn is the table column denoting the billing_profile relation/edge.
	BillingProfileColumn = "billing_profile_id"
)

// Columns holds all SQL columns for billingcustomeroverride fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldCustomerID,
	FieldBillingProfileID,
	FieldCollectionAlignment,
	FieldItemCollectionPeriod,
	FieldInvoiceAutoAdvance,
	FieldInvoiceDraftPeriod,
	FieldInvoiceDueAfter,
	FieldInvoiceCollectionMethod,
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
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// CollectionAlignmentValidator is a validator for the "collection_alignment" field enum values. It is called by the builders before save.
func CollectionAlignmentValidator(ca billingentity.AlignmentKind) error {
	switch ca {
	case "subscription":
		return nil
	default:
		return fmt.Errorf("billingcustomeroverride: invalid enum value for collection_alignment field: %q", ca)
	}
}

// InvoiceCollectionMethodValidator is a validator for the "invoice_collection_method" field enum values. It is called by the builders before save.
func InvoiceCollectionMethodValidator(icm billingentity.CollectionMethod) error {
	switch icm {
	case "charge_automatically", "send_invoice":
		return nil
	default:
		return fmt.Errorf("billingcustomeroverride: invalid enum value for invoice_collection_method field: %q", icm)
	}
}

// OrderOption defines the ordering options for the BillingCustomerOverride queries.
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

// ByCustomerID orders the results by the customer_id field.
func ByCustomerID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerID, opts...).ToFunc()
}

// ByBillingProfileID orders the results by the billing_profile_id field.
func ByBillingProfileID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBillingProfileID, opts...).ToFunc()
}

// ByCollectionAlignment orders the results by the collection_alignment field.
func ByCollectionAlignment(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCollectionAlignment, opts...).ToFunc()
}

// ByItemCollectionPeriod orders the results by the item_collection_period field.
func ByItemCollectionPeriod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldItemCollectionPeriod, opts...).ToFunc()
}

// ByInvoiceAutoAdvance orders the results by the invoice_auto_advance field.
func ByInvoiceAutoAdvance(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceAutoAdvance, opts...).ToFunc()
}

// ByInvoiceDraftPeriod orders the results by the invoice_draft_period field.
func ByInvoiceDraftPeriod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceDraftPeriod, opts...).ToFunc()
}

// ByInvoiceDueAfter orders the results by the invoice_due_after field.
func ByInvoiceDueAfter(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceDueAfter, opts...).ToFunc()
}

// ByInvoiceCollectionMethod orders the results by the invoice_collection_method field.
func ByInvoiceCollectionMethod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceCollectionMethod, opts...).ToFunc()
}

// ByCustomerField orders the results by customer field.
func ByCustomerField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCustomerStep(), sql.OrderByField(field, opts...))
	}
}

// ByBillingProfileField orders the results by billing_profile field.
func ByBillingProfileField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingProfileStep(), sql.OrderByField(field, opts...))
	}
}
func newCustomerStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CustomerInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, CustomerTable, CustomerColumn),
	)
}
func newBillingProfileStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingProfileInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, BillingProfileTable, BillingProfileColumn),
	)
}
