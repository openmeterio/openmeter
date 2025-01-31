// Code generated by ent, DO NOT EDIT.

package billinginvoice

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

const (
	// Label holds the string label denoting the billinginvoice type in the database.
	Label = "billing_invoice"
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
	// FieldSupplierAddressCountry holds the string denoting the supplier_address_country field in the database.
	FieldSupplierAddressCountry = "supplier_address_country"
	// FieldSupplierAddressPostalCode holds the string denoting the supplier_address_postal_code field in the database.
	FieldSupplierAddressPostalCode = "supplier_address_postal_code"
	// FieldSupplierAddressState holds the string denoting the supplier_address_state field in the database.
	FieldSupplierAddressState = "supplier_address_state"
	// FieldSupplierAddressCity holds the string denoting the supplier_address_city field in the database.
	FieldSupplierAddressCity = "supplier_address_city"
	// FieldSupplierAddressLine1 holds the string denoting the supplier_address_line1 field in the database.
	FieldSupplierAddressLine1 = "supplier_address_line1"
	// FieldSupplierAddressLine2 holds the string denoting the supplier_address_line2 field in the database.
	FieldSupplierAddressLine2 = "supplier_address_line2"
	// FieldSupplierAddressPhoneNumber holds the string denoting the supplier_address_phone_number field in the database.
	FieldSupplierAddressPhoneNumber = "supplier_address_phone_number"
	// FieldCustomerAddressCountry holds the string denoting the customer_address_country field in the database.
	FieldCustomerAddressCountry = "customer_address_country"
	// FieldCustomerAddressPostalCode holds the string denoting the customer_address_postal_code field in the database.
	FieldCustomerAddressPostalCode = "customer_address_postal_code"
	// FieldCustomerAddressState holds the string denoting the customer_address_state field in the database.
	FieldCustomerAddressState = "customer_address_state"
	// FieldCustomerAddressCity holds the string denoting the customer_address_city field in the database.
	FieldCustomerAddressCity = "customer_address_city"
	// FieldCustomerAddressLine1 holds the string denoting the customer_address_line1 field in the database.
	FieldCustomerAddressLine1 = "customer_address_line1"
	// FieldCustomerAddressLine2 holds the string denoting the customer_address_line2 field in the database.
	FieldCustomerAddressLine2 = "customer_address_line2"
	// FieldCustomerAddressPhoneNumber holds the string denoting the customer_address_phone_number field in the database.
	FieldCustomerAddressPhoneNumber = "customer_address_phone_number"
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
	// FieldSupplierName holds the string denoting the supplier_name field in the database.
	FieldSupplierName = "supplier_name"
	// FieldSupplierTaxCode holds the string denoting the supplier_tax_code field in the database.
	FieldSupplierTaxCode = "supplier_tax_code"
	// FieldCustomerName holds the string denoting the customer_name field in the database.
	FieldCustomerName = "customer_name"
	// FieldCustomerUsageAttribution holds the string denoting the customer_usage_attribution field in the database.
	FieldCustomerUsageAttribution = "customer_usage_attribution"
	// FieldNumber holds the string denoting the number field in the database.
	FieldNumber = "number"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldCustomerID holds the string denoting the customer_id field in the database.
	FieldCustomerID = "customer_id"
	// FieldSourceBillingProfileID holds the string denoting the source_billing_profile_id field in the database.
	FieldSourceBillingProfileID = "source_billing_profile_id"
	// FieldVoidedAt holds the string denoting the voided_at field in the database.
	FieldVoidedAt = "voided_at"
	// FieldIssuedAt holds the string denoting the issued_at field in the database.
	FieldIssuedAt = "issued_at"
	// FieldSentToCustomerAt holds the string denoting the sent_to_customer_at field in the database.
	FieldSentToCustomerAt = "sent_to_customer_at"
	// FieldDraftUntil holds the string denoting the draft_until field in the database.
	FieldDraftUntil = "draft_until"
	// FieldCurrency holds the string denoting the currency field in the database.
	FieldCurrency = "currency"
	// FieldDueAt holds the string denoting the due_at field in the database.
	FieldDueAt = "due_at"
	// FieldStatus holds the string denoting the status field in the database.
	FieldStatus = "status"
	// FieldWorkflowConfigID holds the string denoting the workflow_config_id field in the database.
	FieldWorkflowConfigID = "workflow_config_id"
	// FieldTaxAppID holds the string denoting the tax_app_id field in the database.
	FieldTaxAppID = "tax_app_id"
	// FieldInvoicingAppID holds the string denoting the invoicing_app_id field in the database.
	FieldInvoicingAppID = "invoicing_app_id"
	// FieldPaymentAppID holds the string denoting the payment_app_id field in the database.
	FieldPaymentAppID = "payment_app_id"
	// FieldInvoicingAppExternalID holds the string denoting the invoicing_app_external_id field in the database.
	FieldInvoicingAppExternalID = "invoicing_app_external_id"
	// FieldPaymentAppExternalID holds the string denoting the payment_app_external_id field in the database.
	FieldPaymentAppExternalID = "payment_app_external_id"
	// FieldPeriodStart holds the string denoting the period_start field in the database.
	FieldPeriodStart = "period_start"
	// FieldPeriodEnd holds the string denoting the period_end field in the database.
	FieldPeriodEnd = "period_end"
	// FieldCollectionAt holds the string denoting the collection_at field in the database.
	FieldCollectionAt = "collection_at"
	// EdgeSourceBillingProfile holds the string denoting the source_billing_profile edge name in mutations.
	EdgeSourceBillingProfile = "source_billing_profile"
	// EdgeBillingWorkflowConfig holds the string denoting the billing_workflow_config edge name in mutations.
	EdgeBillingWorkflowConfig = "billing_workflow_config"
	// EdgeBillingInvoiceLines holds the string denoting the billing_invoice_lines edge name in mutations.
	EdgeBillingInvoiceLines = "billing_invoice_lines"
	// EdgeBillingInvoiceValidationIssues holds the string denoting the billing_invoice_validation_issues edge name in mutations.
	EdgeBillingInvoiceValidationIssues = "billing_invoice_validation_issues"
	// EdgeBillingInvoiceCustomer holds the string denoting the billing_invoice_customer edge name in mutations.
	EdgeBillingInvoiceCustomer = "billing_invoice_customer"
	// EdgeTaxApp holds the string denoting the tax_app edge name in mutations.
	EdgeTaxApp = "tax_app"
	// EdgeInvoicingApp holds the string denoting the invoicing_app edge name in mutations.
	EdgeInvoicingApp = "invoicing_app"
	// EdgePaymentApp holds the string denoting the payment_app edge name in mutations.
	EdgePaymentApp = "payment_app"
	// EdgeInvoiceDiscounts holds the string denoting the invoice_discounts edge name in mutations.
	EdgeInvoiceDiscounts = "invoice_discounts"
	// Table holds the table name of the billinginvoice in the database.
	Table = "billing_invoices"
	// SourceBillingProfileTable is the table that holds the source_billing_profile relation/edge.
	SourceBillingProfileTable = "billing_invoices"
	// SourceBillingProfileInverseTable is the table name for the BillingProfile entity.
	// It exists in this package in order to avoid circular dependency with the "billingprofile" package.
	SourceBillingProfileInverseTable = "billing_profiles"
	// SourceBillingProfileColumn is the table column denoting the source_billing_profile relation/edge.
	SourceBillingProfileColumn = "source_billing_profile_id"
	// BillingWorkflowConfigTable is the table that holds the billing_workflow_config relation/edge.
	BillingWorkflowConfigTable = "billing_invoices"
	// BillingWorkflowConfigInverseTable is the table name for the BillingWorkflowConfig entity.
	// It exists in this package in order to avoid circular dependency with the "billingworkflowconfig" package.
	BillingWorkflowConfigInverseTable = "billing_workflow_configs"
	// BillingWorkflowConfigColumn is the table column denoting the billing_workflow_config relation/edge.
	BillingWorkflowConfigColumn = "workflow_config_id"
	// BillingInvoiceLinesTable is the table that holds the billing_invoice_lines relation/edge.
	BillingInvoiceLinesTable = "billing_invoice_lines"
	// BillingInvoiceLinesInverseTable is the table name for the BillingInvoiceLine entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceline" package.
	BillingInvoiceLinesInverseTable = "billing_invoice_lines"
	// BillingInvoiceLinesColumn is the table column denoting the billing_invoice_lines relation/edge.
	BillingInvoiceLinesColumn = "invoice_id"
	// BillingInvoiceValidationIssuesTable is the table that holds the billing_invoice_validation_issues relation/edge.
	BillingInvoiceValidationIssuesTable = "billing_invoice_validation_issues"
	// BillingInvoiceValidationIssuesInverseTable is the table name for the BillingInvoiceValidationIssue entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoicevalidationissue" package.
	BillingInvoiceValidationIssuesInverseTable = "billing_invoice_validation_issues"
	// BillingInvoiceValidationIssuesColumn is the table column denoting the billing_invoice_validation_issues relation/edge.
	BillingInvoiceValidationIssuesColumn = "invoice_id"
	// BillingInvoiceCustomerTable is the table that holds the billing_invoice_customer relation/edge.
	BillingInvoiceCustomerTable = "billing_invoices"
	// BillingInvoiceCustomerInverseTable is the table name for the Customer entity.
	// It exists in this package in order to avoid circular dependency with the "customer" package.
	BillingInvoiceCustomerInverseTable = "customers"
	// BillingInvoiceCustomerColumn is the table column denoting the billing_invoice_customer relation/edge.
	BillingInvoiceCustomerColumn = "customer_id"
	// TaxAppTable is the table that holds the tax_app relation/edge.
	TaxAppTable = "billing_invoices"
	// TaxAppInverseTable is the table name for the App entity.
	// It exists in this package in order to avoid circular dependency with the "app" package.
	TaxAppInverseTable = "apps"
	// TaxAppColumn is the table column denoting the tax_app relation/edge.
	TaxAppColumn = "tax_app_id"
	// InvoicingAppTable is the table that holds the invoicing_app relation/edge.
	InvoicingAppTable = "billing_invoices"
	// InvoicingAppInverseTable is the table name for the App entity.
	// It exists in this package in order to avoid circular dependency with the "app" package.
	InvoicingAppInverseTable = "apps"
	// InvoicingAppColumn is the table column denoting the invoicing_app relation/edge.
	InvoicingAppColumn = "invoicing_app_id"
	// PaymentAppTable is the table that holds the payment_app relation/edge.
	PaymentAppTable = "billing_invoices"
	// PaymentAppInverseTable is the table name for the App entity.
	// It exists in this package in order to avoid circular dependency with the "app" package.
	PaymentAppInverseTable = "apps"
	// PaymentAppColumn is the table column denoting the payment_app relation/edge.
	PaymentAppColumn = "payment_app_id"
	// InvoiceDiscountsTable is the table that holds the invoice_discounts relation/edge.
	InvoiceDiscountsTable = "billing_invoice_discounts"
	// InvoiceDiscountsInverseTable is the table name for the BillingInvoiceDiscount entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoicediscount" package.
	InvoiceDiscountsInverseTable = "billing_invoice_discounts"
	// InvoiceDiscountsColumn is the table column denoting the invoice_discounts relation/edge.
	InvoiceDiscountsColumn = "invoice_id"
)

// Columns holds all SQL columns for billinginvoice fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldMetadata,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldSupplierAddressCountry,
	FieldSupplierAddressPostalCode,
	FieldSupplierAddressState,
	FieldSupplierAddressCity,
	FieldSupplierAddressLine1,
	FieldSupplierAddressLine2,
	FieldSupplierAddressPhoneNumber,
	FieldCustomerAddressCountry,
	FieldCustomerAddressPostalCode,
	FieldCustomerAddressState,
	FieldCustomerAddressCity,
	FieldCustomerAddressLine1,
	FieldCustomerAddressLine2,
	FieldCustomerAddressPhoneNumber,
	FieldAmount,
	FieldTaxesTotal,
	FieldTaxesInclusiveTotal,
	FieldTaxesExclusiveTotal,
	FieldChargesTotal,
	FieldDiscountsTotal,
	FieldTotal,
	FieldSupplierName,
	FieldSupplierTaxCode,
	FieldCustomerName,
	FieldCustomerUsageAttribution,
	FieldNumber,
	FieldType,
	FieldDescription,
	FieldCustomerID,
	FieldSourceBillingProfileID,
	FieldVoidedAt,
	FieldIssuedAt,
	FieldSentToCustomerAt,
	FieldDraftUntil,
	FieldCurrency,
	FieldDueAt,
	FieldStatus,
	FieldWorkflowConfigID,
	FieldTaxAppID,
	FieldInvoicingAppID,
	FieldPaymentAppID,
	FieldInvoicingAppExternalID,
	FieldPaymentAppExternalID,
	FieldPeriodStart,
	FieldPeriodEnd,
	FieldCollectionAt,
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
	// SupplierAddressCountryValidator is a validator for the "supplier_address_country" field. It is called by the builders before save.
	SupplierAddressCountryValidator func(string) error
	// CustomerAddressCountryValidator is a validator for the "customer_address_country" field. It is called by the builders before save.
	CustomerAddressCountryValidator func(string) error
	// SupplierNameValidator is a validator for the "supplier_name" field. It is called by the builders before save.
	SupplierNameValidator func(string) error
	// CustomerNameValidator is a validator for the "customer_name" field. It is called by the builders before save.
	CustomerNameValidator func(string) error
	// CustomerIDValidator is a validator for the "customer_id" field. It is called by the builders before save.
	CustomerIDValidator func(string) error
	// SourceBillingProfileIDValidator is a validator for the "source_billing_profile_id" field. It is called by the builders before save.
	SourceBillingProfileIDValidator func(string) error
	// CurrencyValidator is a validator for the "currency" field. It is called by the builders before save.
	CurrencyValidator func(string) error
	// DefaultCollectionAt holds the default value on creation for the "collection_at" field.
	DefaultCollectionAt func() time.Time
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type billing.InvoiceType) error {
	switch _type {
	case "standard", "credit-note":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for type field: %q", _type)
	}
}

// StatusValidator is a validator for the "status" field enum values. It is called by the builders before save.
func StatusValidator(s billing.InvoiceStatus) error {
	switch s {
	case "gathering", "draft_created", "draft_updating", "draft_manual_approval_needed", "draft_validating", "draft_invalid", "draft_syncing", "draft_sync_failed", "draft_waiting_auto_approval", "draft_ready_to_issue", "delete_in_progress", "delete_syncing", "delete_failed", "deleted", "issuing_syncing", "issuing_sync_failed", "issued":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for status field: %q", s)
	}
}

// OrderOption defines the ordering options for the BillingInvoice queries.
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

// BySupplierAddressCountry orders the results by the supplier_address_country field.
func BySupplierAddressCountry(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressCountry, opts...).ToFunc()
}

// BySupplierAddressPostalCode orders the results by the supplier_address_postal_code field.
func BySupplierAddressPostalCode(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressPostalCode, opts...).ToFunc()
}

// BySupplierAddressState orders the results by the supplier_address_state field.
func BySupplierAddressState(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressState, opts...).ToFunc()
}

// BySupplierAddressCity orders the results by the supplier_address_city field.
func BySupplierAddressCity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressCity, opts...).ToFunc()
}

// BySupplierAddressLine1 orders the results by the supplier_address_line1 field.
func BySupplierAddressLine1(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressLine1, opts...).ToFunc()
}

// BySupplierAddressLine2 orders the results by the supplier_address_line2 field.
func BySupplierAddressLine2(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressLine2, opts...).ToFunc()
}

// BySupplierAddressPhoneNumber orders the results by the supplier_address_phone_number field.
func BySupplierAddressPhoneNumber(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierAddressPhoneNumber, opts...).ToFunc()
}

// ByCustomerAddressCountry orders the results by the customer_address_country field.
func ByCustomerAddressCountry(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressCountry, opts...).ToFunc()
}

// ByCustomerAddressPostalCode orders the results by the customer_address_postal_code field.
func ByCustomerAddressPostalCode(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressPostalCode, opts...).ToFunc()
}

// ByCustomerAddressState orders the results by the customer_address_state field.
func ByCustomerAddressState(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressState, opts...).ToFunc()
}

// ByCustomerAddressCity orders the results by the customer_address_city field.
func ByCustomerAddressCity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressCity, opts...).ToFunc()
}

// ByCustomerAddressLine1 orders the results by the customer_address_line1 field.
func ByCustomerAddressLine1(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressLine1, opts...).ToFunc()
}

// ByCustomerAddressLine2 orders the results by the customer_address_line2 field.
func ByCustomerAddressLine2(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressLine2, opts...).ToFunc()
}

// ByCustomerAddressPhoneNumber orders the results by the customer_address_phone_number field.
func ByCustomerAddressPhoneNumber(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerAddressPhoneNumber, opts...).ToFunc()
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

// BySupplierName orders the results by the supplier_name field.
func BySupplierName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierName, opts...).ToFunc()
}

// BySupplierTaxCode orders the results by the supplier_tax_code field.
func BySupplierTaxCode(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSupplierTaxCode, opts...).ToFunc()
}

// ByCustomerName orders the results by the customer_name field.
func ByCustomerName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerName, opts...).ToFunc()
}

// ByNumber orders the results by the number field.
func ByNumber(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNumber, opts...).ToFunc()
}

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByCustomerID orders the results by the customer_id field.
func ByCustomerID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerID, opts...).ToFunc()
}

// BySourceBillingProfileID orders the results by the source_billing_profile_id field.
func BySourceBillingProfileID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSourceBillingProfileID, opts...).ToFunc()
}

// ByVoidedAt orders the results by the voided_at field.
func ByVoidedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldVoidedAt, opts...).ToFunc()
}

// ByIssuedAt orders the results by the issued_at field.
func ByIssuedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIssuedAt, opts...).ToFunc()
}

// BySentToCustomerAt orders the results by the sent_to_customer_at field.
func BySentToCustomerAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSentToCustomerAt, opts...).ToFunc()
}

// ByDraftUntil orders the results by the draft_until field.
func ByDraftUntil(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDraftUntil, opts...).ToFunc()
}

// ByCurrency orders the results by the currency field.
func ByCurrency(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCurrency, opts...).ToFunc()
}

// ByDueAt orders the results by the due_at field.
func ByDueAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDueAt, opts...).ToFunc()
}

// ByStatus orders the results by the status field.
func ByStatus(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStatus, opts...).ToFunc()
}

// ByWorkflowConfigID orders the results by the workflow_config_id field.
func ByWorkflowConfigID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWorkflowConfigID, opts...).ToFunc()
}

// ByTaxAppID orders the results by the tax_app_id field.
func ByTaxAppID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxAppID, opts...).ToFunc()
}

// ByInvoicingAppID orders the results by the invoicing_app_id field.
func ByInvoicingAppID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingAppID, opts...).ToFunc()
}

// ByPaymentAppID orders the results by the payment_app_id field.
func ByPaymentAppID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPaymentAppID, opts...).ToFunc()
}

// ByInvoicingAppExternalID orders the results by the invoicing_app_external_id field.
func ByInvoicingAppExternalID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingAppExternalID, opts...).ToFunc()
}

// ByPaymentAppExternalID orders the results by the payment_app_external_id field.
func ByPaymentAppExternalID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPaymentAppExternalID, opts...).ToFunc()
}

// ByPeriodStart orders the results by the period_start field.
func ByPeriodStart(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodStart, opts...).ToFunc()
}

// ByPeriodEnd orders the results by the period_end field.
func ByPeriodEnd(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodEnd, opts...).ToFunc()
}

// ByCollectionAt orders the results by the collection_at field.
func ByCollectionAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCollectionAt, opts...).ToFunc()
}

// BySourceBillingProfileField orders the results by source_billing_profile field.
func BySourceBillingProfileField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSourceBillingProfileStep(), sql.OrderByField(field, opts...))
	}
}

// ByBillingWorkflowConfigField orders the results by billing_workflow_config field.
func ByBillingWorkflowConfigField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingWorkflowConfigStep(), sql.OrderByField(field, opts...))
	}
}

// ByBillingInvoiceLinesCount orders the results by billing_invoice_lines count.
func ByBillingInvoiceLinesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newBillingInvoiceLinesStep(), opts...)
	}
}

// ByBillingInvoiceLines orders the results by billing_invoice_lines terms.
func ByBillingInvoiceLines(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceLinesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByBillingInvoiceValidationIssuesCount orders the results by billing_invoice_validation_issues count.
func ByBillingInvoiceValidationIssuesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newBillingInvoiceValidationIssuesStep(), opts...)
	}
}

// ByBillingInvoiceValidationIssues orders the results by billing_invoice_validation_issues terms.
func ByBillingInvoiceValidationIssues(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceValidationIssuesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByBillingInvoiceCustomerField orders the results by billing_invoice_customer field.
func ByBillingInvoiceCustomerField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceCustomerStep(), sql.OrderByField(field, opts...))
	}
}

// ByTaxAppField orders the results by tax_app field.
func ByTaxAppField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newTaxAppStep(), sql.OrderByField(field, opts...))
	}
}

// ByInvoicingAppField orders the results by invoicing_app field.
func ByInvoicingAppField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newInvoicingAppStep(), sql.OrderByField(field, opts...))
	}
}

// ByPaymentAppField orders the results by payment_app field.
func ByPaymentAppField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPaymentAppStep(), sql.OrderByField(field, opts...))
	}
}

// ByInvoiceDiscountsCount orders the results by invoice_discounts count.
func ByInvoiceDiscountsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newInvoiceDiscountsStep(), opts...)
	}
}

// ByInvoiceDiscounts orders the results by invoice_discounts terms.
func ByInvoiceDiscounts(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newInvoiceDiscountsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newSourceBillingProfileStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SourceBillingProfileInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, SourceBillingProfileTable, SourceBillingProfileColumn),
	)
}
func newBillingWorkflowConfigStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingWorkflowConfigInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, BillingWorkflowConfigTable, BillingWorkflowConfigColumn),
	)
}
func newBillingInvoiceLinesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceLinesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingInvoiceLinesTable, BillingInvoiceLinesColumn),
	)
}
func newBillingInvoiceValidationIssuesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceValidationIssuesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingInvoiceValidationIssuesTable, BillingInvoiceValidationIssuesColumn),
	)
}
func newBillingInvoiceCustomerStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceCustomerInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, BillingInvoiceCustomerTable, BillingInvoiceCustomerColumn),
	)
}
func newTaxAppStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(TaxAppInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, TaxAppTable, TaxAppColumn),
	)
}
func newInvoicingAppStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(InvoicingAppInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, InvoicingAppTable, InvoicingAppColumn),
	)
}
func newPaymentAppStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PaymentAppInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, PaymentAppTable, PaymentAppColumn),
	)
}
func newInvoiceDiscountsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(InvoiceDiscountsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, InvoiceDiscountsTable, InvoiceDiscountsColumn),
	)
}
