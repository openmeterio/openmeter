// Code generated by ent, DO NOT EDIT.

package billinginvoicelinediscount

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

const (
	// Label holds the string label denoting the billinginvoicelinediscount type in the database.
	Label = "billing_invoice_line_discount"
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
	// FieldLineID holds the string denoting the line_id field in the database.
	FieldLineID = "line_id"
	// FieldChildUniqueReferenceID holds the string denoting the child_unique_reference_id field in the database.
	FieldChildUniqueReferenceID = "child_unique_reference_id"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldReason holds the string denoting the reason field in the database.
	FieldReason = "reason"
	// FieldInvoicingAppExternalID holds the string denoting the invoicing_app_external_id field in the database.
	FieldInvoicingAppExternalID = "invoicing_app_external_id"
	// FieldAmount holds the string denoting the amount field in the database.
	FieldAmount = "amount"
	// FieldRoundingAmount holds the string denoting the rounding_amount field in the database.
	FieldRoundingAmount = "rounding_amount"
	// FieldSourceDiscount holds the string denoting the source_discount field in the database.
	FieldSourceDiscount = "source_discount"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldQuantity holds the string denoting the quantity field in the database.
	FieldQuantity = "quantity"
	// FieldPreLinePeriodQuantity holds the string denoting the pre_line_period_quantity field in the database.
	FieldPreLinePeriodQuantity = "pre_line_period_quantity"
	// EdgeBillingInvoiceLine holds the string denoting the billing_invoice_line edge name in mutations.
	EdgeBillingInvoiceLine = "billing_invoice_line"
	// Table holds the table name of the billinginvoicelinediscount in the database.
	Table = "billing_invoice_line_discounts"
	// BillingInvoiceLineTable is the table that holds the billing_invoice_line relation/edge.
	BillingInvoiceLineTable = "billing_invoice_line_discounts"
	// BillingInvoiceLineInverseTable is the table name for the BillingInvoiceLine entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceline" package.
	BillingInvoiceLineInverseTable = "billing_invoice_lines"
	// BillingInvoiceLineColumn is the table column denoting the billing_invoice_line relation/edge.
	BillingInvoiceLineColumn = "line_id"
)

// Columns holds all SQL columns for billinginvoicelinediscount fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldLineID,
	FieldChildUniqueReferenceID,
	FieldDescription,
	FieldReason,
	FieldInvoicingAppExternalID,
	FieldAmount,
	FieldRoundingAmount,
	FieldSourceDiscount,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for _, f := range [...]string{FieldType, FieldQuantity, FieldPreLinePeriodQuantity} {
		if column == f {
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
	// ValueScanner of all BillingInvoiceLineDiscount fields.
	ValueScanner struct {
		SourceDiscount field.TypeValueScanner[*billing.DiscountReason]
	}
)

// ReasonValidator is a validator for the "reason" field enum values. It is called by the builders before save.
func ReasonValidator(r billing.DiscountReasonType) error {
	switch r {
	case "maximum_spend", "ratecard_percentage", "ratecard_usage":
		return nil
	default:
		return fmt.Errorf("billinginvoicelinediscount: invalid enum value for reason field: %q", r)
	}
}

// OrderOption defines the ordering options for the BillingInvoiceLineDiscount queries.
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

// ByLineID orders the results by the line_id field.
func ByLineID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLineID, opts...).ToFunc()
}

// ByChildUniqueReferenceID orders the results by the child_unique_reference_id field.
func ByChildUniqueReferenceID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldChildUniqueReferenceID, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByReason orders the results by the reason field.
func ByReason(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldReason, opts...).ToFunc()
}

// ByInvoicingAppExternalID orders the results by the invoicing_app_external_id field.
func ByInvoicingAppExternalID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingAppExternalID, opts...).ToFunc()
}

// ByAmount orders the results by the amount field.
func ByAmount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAmount, opts...).ToFunc()
}

// ByRoundingAmount orders the results by the rounding_amount field.
func ByRoundingAmount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRoundingAmount, opts...).ToFunc()
}

// BySourceDiscount orders the results by the source_discount field.
func BySourceDiscount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSourceDiscount, opts...).ToFunc()
}

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByQuantity orders the results by the quantity field.
func ByQuantity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldQuantity, opts...).ToFunc()
}

// ByPreLinePeriodQuantity orders the results by the pre_line_period_quantity field.
func ByPreLinePeriodQuantity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPreLinePeriodQuantity, opts...).ToFunc()
}

// ByBillingInvoiceLineField orders the results by billing_invoice_line field.
func ByBillingInvoiceLineField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceLineStep(), sql.OrderByField(field, opts...))
	}
}
func newBillingInvoiceLineStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceLineInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, BillingInvoiceLineTable, BillingInvoiceLineColumn),
	)
}
