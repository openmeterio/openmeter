// Code generated by ent, DO NOT EDIT.

package billinginvoiceline

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
	// FieldAmount holds the string denoting the amount field in the database.
	FieldAmount = "amount"
	// FieldTaxesTotal holds the string denoting the taxes_total field in the database.
	FieldTaxesTotal = "taxes_total"
	// FieldTaxesInclusiveTotal holds the string denoting the taxes_inclusive_total field in the database.
	FieldTaxesInclusiveTotal = "taxes_inclusive_total"
	// FieldTaxesExclusiveTotal holds the string denoting the taxes_exclusive_total field in the database.
	FieldTaxesExclusiveTotal = "taxes_exclusive_total"
	// FieldChargesTotal holds the string denoting the charges_total field in the database.
	FieldChargesTotal = "charges_total"
	// FieldDiscountsTotal holds the string denoting the discounts_total field in the database.
	FieldDiscountsTotal = "discounts_total"
	// FieldTotal holds the string denoting the total field in the database.
	FieldTotal = "total"
	// FieldInvoiceID holds the string denoting the invoice_id field in the database.
	FieldInvoiceID = "invoice_id"
	// FieldManagedBy holds the string denoting the managed_by field in the database.
	FieldManagedBy = "managed_by"
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
	// FieldRatecardDiscounts holds the string denoting the ratecard_discounts field in the database.
	FieldRatecardDiscounts = "ratecard_discounts"
	// FieldInvoicingAppExternalID holds the string denoting the invoicing_app_external_id field in the database.
	FieldInvoicingAppExternalID = "invoicing_app_external_id"
	// FieldChildUniqueReferenceID holds the string denoting the child_unique_reference_id field in the database.
	FieldChildUniqueReferenceID = "child_unique_reference_id"
	// FieldSubscriptionID holds the string denoting the subscription_id field in the database.
	FieldSubscriptionID = "subscription_id"
	// FieldSubscriptionPhaseID holds the string denoting the subscription_phase_id field in the database.
	FieldSubscriptionPhaseID = "subscription_phase_id"
	// FieldSubscriptionItemID holds the string denoting the subscription_item_id field in the database.
	FieldSubscriptionItemID = "subscription_item_id"
	// FieldLineIds holds the string denoting the line_ids field in the database.
	FieldLineIds = "line_ids"
	// EdgeBillingInvoice holds the string denoting the billing_invoice edge name in mutations.
	EdgeBillingInvoice = "billing_invoice"
	// EdgeFlatFeeLine holds the string denoting the flat_fee_line edge name in mutations.
	EdgeFlatFeeLine = "flat_fee_line"
	// EdgeUsageBasedLine holds the string denoting the usage_based_line edge name in mutations.
	EdgeUsageBasedLine = "usage_based_line"
	// EdgeParentLine holds the string denoting the parent_line edge name in mutations.
	EdgeParentLine = "parent_line"
	// EdgeDetailedLines holds the string denoting the detailed_lines edge name in mutations.
	EdgeDetailedLines = "detailed_lines"
	// EdgeLineDiscounts holds the string denoting the line_discounts edge name in mutations.
	EdgeLineDiscounts = "line_discounts"
	// EdgeSubscription holds the string denoting the subscription edge name in mutations.
	EdgeSubscription = "subscription"
	// EdgeSubscriptionPhase holds the string denoting the subscription_phase edge name in mutations.
	EdgeSubscriptionPhase = "subscription_phase"
	// EdgeSubscriptionItem holds the string denoting the subscription_item edge name in mutations.
	EdgeSubscriptionItem = "subscription_item"
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
	// DetailedLinesTable is the table that holds the detailed_lines relation/edge.
	DetailedLinesTable = "billing_invoice_lines"
	// DetailedLinesColumn is the table column denoting the detailed_lines relation/edge.
	DetailedLinesColumn = "parent_line_id"
	// LineDiscountsTable is the table that holds the line_discounts relation/edge.
	LineDiscountsTable = "billing_invoice_line_discounts"
	// LineDiscountsInverseTable is the table name for the BillingInvoiceLineDiscount entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoicelinediscount" package.
	LineDiscountsInverseTable = "billing_invoice_line_discounts"
	// LineDiscountsColumn is the table column denoting the line_discounts relation/edge.
	LineDiscountsColumn = "line_id"
	// SubscriptionTable is the table that holds the subscription relation/edge.
	SubscriptionTable = "billing_invoice_lines"
	// SubscriptionInverseTable is the table name for the Subscription entity.
	// It exists in this package in order to avoid circular dependency with the "subscription" package.
	SubscriptionInverseTable = "subscriptions"
	// SubscriptionColumn is the table column denoting the subscription relation/edge.
	SubscriptionColumn = "subscription_id"
	// SubscriptionPhaseTable is the table that holds the subscription_phase relation/edge.
	SubscriptionPhaseTable = "billing_invoice_lines"
	// SubscriptionPhaseInverseTable is the table name for the SubscriptionPhase entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionphase" package.
	SubscriptionPhaseInverseTable = "subscription_phases"
	// SubscriptionPhaseColumn is the table column denoting the subscription_phase relation/edge.
	SubscriptionPhaseColumn = "subscription_phase_id"
	// SubscriptionItemTable is the table that holds the subscription_item relation/edge.
	SubscriptionItemTable = "billing_invoice_lines"
	// SubscriptionItemInverseTable is the table name for the SubscriptionItem entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionitem" package.
	SubscriptionItemInverseTable = "subscription_items"
	// SubscriptionItemColumn is the table column denoting the subscription_item relation/edge.
	SubscriptionItemColumn = "subscription_item_id"
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
	FieldAmount,
	FieldTaxesTotal,
	FieldTaxesInclusiveTotal,
	FieldTaxesExclusiveTotal,
	FieldChargesTotal,
	FieldDiscountsTotal,
	FieldTotal,
	FieldInvoiceID,
	FieldManagedBy,
	FieldParentLineID,
	FieldPeriodStart,
	FieldPeriodEnd,
	FieldInvoiceAt,
	FieldType,
	FieldStatus,
	FieldCurrency,
	FieldQuantity,
	FieldTaxConfig,
	FieldRatecardDiscounts,
	FieldInvoicingAppExternalID,
	FieldChildUniqueReferenceID,
	FieldSubscriptionID,
	FieldSubscriptionPhaseID,
	FieldSubscriptionItemID,
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
	for _, f := range [...]string{FieldLineIds} {
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
	// CurrencyValidator is a validator for the "currency" field. It is called by the builders before save.
	CurrencyValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
	// ValueScanner of all BillingInvoiceLine fields.
	ValueScanner struct {
		RatecardDiscounts field.TypeValueScanner[*productcatalog.Discounts]
	}
)

// ManagedByValidator is a validator for the "managed_by" field enum values. It is called by the builders before save.
func ManagedByValidator(mb billing.InvoiceLineManagedBy) error {
	switch mb {
	case "subscription", "system", "manual":
		return nil
	default:
		return fmt.Errorf("billinginvoiceline: invalid enum value for managed_by field: %q", mb)
	}
}

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type billing.InvoiceLineType) error {
	switch _type {
	case "flat_fee", "usage_based":
		return nil
	default:
		return fmt.Errorf("billinginvoiceline: invalid enum value for type field: %q", _type)
	}
}

// StatusValidator is a validator for the "status" field enum values. It is called by the builders before save.
func StatusValidator(s billing.InvoiceLineStatus) error {
	switch s {
	case "valid", "split", "detailed":
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

// ByAmount orders the results by the amount field.
func ByAmount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAmount, opts...).ToFunc()
}

// ByTaxesTotal orders the results by the taxes_total field.
func ByTaxesTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxesTotal, opts...).ToFunc()
}

// ByTaxesInclusiveTotal orders the results by the taxes_inclusive_total field.
func ByTaxesInclusiveTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxesInclusiveTotal, opts...).ToFunc()
}

// ByTaxesExclusiveTotal orders the results by the taxes_exclusive_total field.
func ByTaxesExclusiveTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxesExclusiveTotal, opts...).ToFunc()
}

// ByChargesTotal orders the results by the charges_total field.
func ByChargesTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldChargesTotal, opts...).ToFunc()
}

// ByDiscountsTotal orders the results by the discounts_total field.
func ByDiscountsTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDiscountsTotal, opts...).ToFunc()
}

// ByTotal orders the results by the total field.
func ByTotal(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTotal, opts...).ToFunc()
}

// ByInvoiceID orders the results by the invoice_id field.
func ByInvoiceID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoiceID, opts...).ToFunc()
}

// ByManagedBy orders the results by the managed_by field.
func ByManagedBy(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldManagedBy, opts...).ToFunc()
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

// ByRatecardDiscounts orders the results by the ratecard_discounts field.
func ByRatecardDiscounts(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRatecardDiscounts, opts...).ToFunc()
}

// ByInvoicingAppExternalID orders the results by the invoicing_app_external_id field.
func ByInvoicingAppExternalID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingAppExternalID, opts...).ToFunc()
}

// ByChildUniqueReferenceID orders the results by the child_unique_reference_id field.
func ByChildUniqueReferenceID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldChildUniqueReferenceID, opts...).ToFunc()
}

// BySubscriptionID orders the results by the subscription_id field.
func BySubscriptionID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionID, opts...).ToFunc()
}

// BySubscriptionPhaseID orders the results by the subscription_phase_id field.
func BySubscriptionPhaseID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionPhaseID, opts...).ToFunc()
}

// BySubscriptionItemID orders the results by the subscription_item_id field.
func BySubscriptionItemID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionItemID, opts...).ToFunc()
}

// ByLineIds orders the results by the line_ids field.
func ByLineIds(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLineIds, opts...).ToFunc()
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

// ByDetailedLinesCount orders the results by detailed_lines count.
func ByDetailedLinesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newDetailedLinesStep(), opts...)
	}
}

// ByDetailedLines orders the results by detailed_lines terms.
func ByDetailedLines(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newDetailedLinesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByLineDiscountsCount orders the results by line_discounts count.
func ByLineDiscountsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newLineDiscountsStep(), opts...)
	}
}

// ByLineDiscounts orders the results by line_discounts terms.
func ByLineDiscounts(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newLineDiscountsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// BySubscriptionField orders the results by subscription field.
func BySubscriptionField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionStep(), sql.OrderByField(field, opts...))
	}
}

// BySubscriptionPhaseField orders the results by subscription_phase field.
func BySubscriptionPhaseField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionPhaseStep(), sql.OrderByField(field, opts...))
	}
}

// BySubscriptionItemField orders the results by subscription_item field.
func BySubscriptionItemField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionItemStep(), sql.OrderByField(field, opts...))
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
func newDetailedLinesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(Table, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, DetailedLinesTable, DetailedLinesColumn),
	)
}
func newLineDiscountsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(LineDiscountsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, LineDiscountsTable, LineDiscountsColumn),
	)
}
func newSubscriptionStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionTable, SubscriptionColumn),
	)
}
func newSubscriptionPhaseStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionPhaseInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionPhaseTable, SubscriptionPhaseColumn),
	)
}
func newSubscriptionItemStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionItemInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionItemTable, SubscriptionItemColumn),
	)
}
