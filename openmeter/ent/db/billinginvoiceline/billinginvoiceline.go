// Code generated by ent, DO NOT EDIT.

package billinginvoiceline

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

const (
	// Label holds the string label denoting the billinginvoiceline type in the database.
	Label = "billing_invoice_line"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldMetadata holds the string denoting the metadata field in the database.
	FieldMetadata = "metadata"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldInvoiceID holds the string denoting the invoice_id field in the database.
	FieldInvoiceID = "invoice_id"
	// FieldParentLineID holds the string denoting the parent_line_id field in the database.
	FieldParentLineID = "parent_line_id"
	// FieldPeriodStart holds the string denoting the period_start field in the database.
	FieldPeriodStart = "period_start"
	// FieldPeriodEnd holds the string denoting the period_end field in the database.
	FieldPeriodEnd = "period_end"
	// FieldInvoiceAt holds the string denoting the invoice_at field in the database.
	FieldInvoiceAt = "invoice_at"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldStatus holds the string denoting the status field in the database.
	FieldStatus = "status"
	// FieldCurrency holds the string denoting the currency field in the database.
	FieldCurrency = "currency"
	// FieldQuantity holds the string denoting the quantity field in the database.
	FieldQuantity = "quantity"
	// FieldTaxConfig holds the string denoting the tax_config field in the database.
	FieldTaxConfig = "tax_config"
	// FieldChildUniqueReferenceID holds the string denoting the child_unique_reference_id field in the database.
	FieldChildUniqueReferenceID = "child_unique_reference_id"
	// EdgeBillingInvoice holds the string denoting the billing_invoice edge name in mutations.
	EdgeBillingInvoice = "billing_invoice"
	// EdgeFlatFeeLine holds the string denoting the flat_fee_line edge name in mutations.
	EdgeFlatFeeLine = "flat_fee_line"
	// EdgeUsageBasedLine holds the string denoting the usage_based_line edge name in mutations.
	EdgeUsageBasedLine = "usage_based_line"
	// EdgeParentLine holds the string denoting the parent_line edge name in mutations.
	EdgeParentLine = "parent_line"
	// EdgeChildLines holds the string denoting the child_lines edge name in mutations.
	EdgeChildLines = "child_lines"
	// Table holds the table name of the billinginvoiceline in the database.
	Table = "billing_invoice_lines"
	// BillingInvoiceTable is the table that holds the billing_invoice relation/edge.
	BillingInvoiceTable = "billing_invoice_lines"
	// BillingInvoiceInverseTable is the table name for the BillingInvoice entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoice" package.
	BillingInvoiceInverseTable = "billing_invoices"
	// BillingInvoiceColumn is the table column denoting the billing_invoice relation/edge.
	BillingInvoiceColumn = "invoice_id"
	// FlatFeeLineTable is the table that holds the flat_fee_line relation/edge.
	FlatFeeLineTable = "billing_invoice_lines"
	// FlatFeeLineInverseTable is the table name for the BillingInvoiceFlatFeeLineConfig entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceflatfeelineconfig" package.
	FlatFeeLineInverseTable = "billing_invoice_flat_fee_line_configs"
	// FlatFeeLineColumn is the table column denoting the flat_fee_line relation/edge.
	FlatFeeLineColumn = "fee_line_config_id"
	// UsageBasedLineTable is the table that holds the usage_based_line relation/edge.
	UsageBasedLineTable = "billing_invoice_lines"
	// UsageBasedLineInverseTable is the table name for the BillingInvoiceUsageBasedLineConfig entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceusagebasedlineconfig" package.
	UsageBasedLineInverseTable = "billing_invoice_usage_based_line_configs"
	// UsageBasedLineColumn is the table column denoting the usage_based_line relation/edge.
	UsageBasedLineColumn = "usage_based_line_config_id"
	// ParentLineTable is the table that holds the parent_line relation/edge.
	ParentLineTable = "billing_invoice_lines"
	// ParentLineColumn is the table column denoting the parent_line relation/edge.
	ParentLineColumn = "parent_line_id"
	// ChildLinesTable is the table that holds the child_lines relation/edge.
	ChildLinesTable = "billing_invoice_lines"
	// ChildLinesColumn is the table column denoting the child_lines relation/edge.
	ChildLinesColumn = "parent_line_id"
)

// Columns holds all SQL columns for billinginvoiceline fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldMetadata,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldName,
	FieldDescription,
	FieldInvoiceID,
	FieldParentLineID,
	FieldPeriodStart,
	FieldPeriodEnd,
	FieldInvoiceAt,
	FieldType,
	FieldStatus,
	FieldCurrency,
	FieldQuantity,
	FieldTaxConfig,
	FieldChildUniqueReferenceID,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "billing_invoice_lines"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"fee_line_config_id",
	"usage_based_line_config_id",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
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
	// CurrencyValidator is a validator for the "currency" field. It is called by the builders before save.
	CurrencyValidator func(string) error
	// ChildUniqueReferenceIDValidator is a validator for the "child_unique_reference_id" field. It is called by the builders before save.
	ChildUniqueReferenceIDValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type billingentity.InvoiceLineType) error {
	switch _type {
	case "flat_fee", "usage_based":
		return nil
	default:
		return fmt.Errorf("billinginvoiceline: invalid enum value for type field: %q", _type)
	}
}

// StatusValidator is a validator for the "status" field enum values. It is called by the builders before save.
func StatusValidator(s billingentity.InvoiceLineStatus) error {
	switch s {
	case "valid", "split":
		return nil
	default:
		return fmt.Errorf("billinginvoiceline: invalid enum value for status field: %q", s)
	}
}

// OrderOption defines the ordering options for the BillingInvoiceLine queries.
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

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByInvoiceID orders the results by the invoice_id field.
func ByInvoiceID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceID, opts...).ToFunc()
}

// ByParentLineID orders the results by the parent_line_id field.
func ByParentLineID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldParentLineID, opts...).ToFunc()
}

// ByPeriodStart orders the results by the period_start field.
func ByPeriodStart(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodStart, opts...).ToFunc()
}

// ByPeriodEnd orders the results by the period_end field.
func ByPeriodEnd(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodEnd, opts...).ToFunc()
}

// ByInvoiceAt orders the results by the invoice_at field.
func ByInvoiceAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceAt, opts...).ToFunc()
}

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByStatus orders the results by the status field.
func ByStatus(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStatus, opts...).ToFunc()
}

// ByCurrency orders the results by the currency field.
func ByCurrency(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCurrency, opts...).ToFunc()
}

// ByQuantity orders the results by the quantity field.
func ByQuantity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldQuantity, opts...).ToFunc()
}

// ByChildUniqueReferenceID orders the results by the child_unique_reference_id field.
func ByChildUniqueReferenceID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldChildUniqueReferenceID, opts...).ToFunc()
}

// ByBillingInvoiceField orders the results by billing_invoice field.
func ByBillingInvoiceField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceStep(), sql.OrderByField(field, opts...))
	}
}

// ByFlatFeeLineField orders the results by flat_fee_line field.
func ByFlatFeeLineField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newFlatFeeLineStep(), sql.OrderByField(field, opts...))
	}
}

// ByUsageBasedLineField orders the results by usage_based_line field.
func ByUsageBasedLineField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newUsageBasedLineStep(), sql.OrderByField(field, opts...))
	}
}

// ByParentLineField orders the results by parent_line field.
func ByParentLineField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newParentLineStep(), sql.OrderByField(field, opts...))
	}
}

// ByChildLinesCount orders the results by child_lines count.
func ByChildLinesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newChildLinesStep(), opts...)
	}
}

// ByChildLines orders the results by child_lines terms.
func ByChildLines(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newChildLinesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newBillingInvoiceStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, BillingInvoiceTable, BillingInvoiceColumn),
	)
}
func newFlatFeeLineStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(FlatFeeLineInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, FlatFeeLineTable, FlatFeeLineColumn),
	)
}
func newUsageBasedLineStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(UsageBasedLineInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, UsageBasedLineTable, UsageBasedLineColumn),
	)
}
func newParentLineStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(Table, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, ParentLineTable, ParentLineColumn),
	)
}
func newChildLinesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(Table, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, ChildLinesTable, ChildLinesColumn),
	)
}
