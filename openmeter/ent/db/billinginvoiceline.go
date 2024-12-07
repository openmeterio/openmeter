// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// BillingInvoiceLine is the model entity for the BillingInvoiceLine schema.
type BillingInvoiceLine struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// Metadata holds the value of the "metadata" field.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// Amount holds the value of the "amount" field.
	Amount alpacadecimal.Decimal `json:"amount,omitempty"`
	// TaxesTotal holds the value of the "taxes_total" field.
	TaxesTotal alpacadecimal.Decimal `json:"taxes_total,omitempty"`
	// TaxesInclusiveTotal holds the value of the "taxes_inclusive_total" field.
	TaxesInclusiveTotal alpacadecimal.Decimal `json:"taxes_inclusive_total,omitempty"`
	// TaxesExclusiveTotal holds the value of the "taxes_exclusive_total" field.
	TaxesExclusiveTotal alpacadecimal.Decimal `json:"taxes_exclusive_total,omitempty"`
	// ChargesTotal holds the value of the "charges_total" field.
	ChargesTotal alpacadecimal.Decimal `json:"charges_total,omitempty"`
	// DiscountsTotal holds the value of the "discounts_total" field.
	DiscountsTotal alpacadecimal.Decimal `json:"discounts_total,omitempty"`
	// Total holds the value of the "total" field.
	Total alpacadecimal.Decimal `json:"total,omitempty"`
	// InvoiceID holds the value of the "invoice_id" field.
	InvoiceID string `json:"invoice_id,omitempty"`
	// ParentLineID holds the value of the "parent_line_id" field.
	ParentLineID *string `json:"parent_line_id,omitempty"`
	// PeriodStart holds the value of the "period_start" field.
	PeriodStart time.Time `json:"period_start,omitempty"`
	// PeriodEnd holds the value of the "period_end" field.
	PeriodEnd time.Time `json:"period_end,omitempty"`
	// InvoiceAt holds the value of the "invoice_at" field.
	InvoiceAt time.Time `json:"invoice_at,omitempty"`
	// Type holds the value of the "type" field.
	Type billingentity.InvoiceLineType `json:"type,omitempty"`
	// Status holds the value of the "status" field.
	Status billingentity.InvoiceLineStatus `json:"status,omitempty"`
	// Currency holds the value of the "currency" field.
	Currency currencyx.Code `json:"currency,omitempty"`
	// Quantity holds the value of the "quantity" field.
	Quantity *alpacadecimal.Decimal `json:"quantity,omitempty"`
	// TaxConfig holds the value of the "tax_config" field.
	TaxConfig productcatalog.TaxConfig `json:"tax_config,omitempty"`
	// InvoicingAppExternalID holds the value of the "invoicing_app_external_id" field.
	InvoicingAppExternalID *string `json:"invoicing_app_external_id,omitempty"`
	// ChildUniqueReferenceID holds the value of the "child_unique_reference_id" field.
	ChildUniqueReferenceID *string `json:"child_unique_reference_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BillingInvoiceLineQuery when eager-loading is set.
	Edges                      BillingInvoiceLineEdges `json:"edges"`
	fee_line_config_id         *string
	usage_based_line_config_id *string
	selectValues               sql.SelectValues
}

// BillingInvoiceLineEdges holds the relations/edges for other nodes in the graph.
type BillingInvoiceLineEdges struct {
	// BillingInvoice holds the value of the billing_invoice edge.
	BillingInvoice *BillingInvoice `json:"billing_invoice,omitempty"`
	// FlatFeeLine holds the value of the flat_fee_line edge.
	FlatFeeLine *BillingInvoiceFlatFeeLineConfig `json:"flat_fee_line,omitempty"`
	// UsageBasedLine holds the value of the usage_based_line edge.
	UsageBasedLine *BillingInvoiceUsageBasedLineConfig `json:"usage_based_line,omitempty"`
	// ParentLine holds the value of the parent_line edge.
	ParentLine *BillingInvoiceLine `json:"parent_line,omitempty"`
	// DetailedLines holds the value of the detailed_lines edge.
	DetailedLines []*BillingInvoiceLine `json:"detailed_lines,omitempty"`
	// LineDiscounts holds the value of the line_discounts edge.
	LineDiscounts []*BillingInvoiceLineDiscount `json:"line_discounts,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [6]bool
}

// BillingInvoiceOrErr returns the BillingInvoice value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingInvoiceLineEdges) BillingInvoiceOrErr() (*BillingInvoice, error) {
	if e.BillingInvoice != nil {
		return e.BillingInvoice, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: billinginvoice.Label}
	}
	return nil, &NotLoadedError{edge: "billing_invoice"}
}

// FlatFeeLineOrErr returns the FlatFeeLine value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingInvoiceLineEdges) FlatFeeLineOrErr() (*BillingInvoiceFlatFeeLineConfig, error) {
	if e.FlatFeeLine != nil {
		return e.FlatFeeLine, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: billinginvoiceflatfeelineconfig.Label}
	}
	return nil, &NotLoadedError{edge: "flat_fee_line"}
}

// UsageBasedLineOrErr returns the UsageBasedLine value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingInvoiceLineEdges) UsageBasedLineOrErr() (*BillingInvoiceUsageBasedLineConfig, error) {
	if e.UsageBasedLine != nil {
		return e.UsageBasedLine, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: billinginvoiceusagebasedlineconfig.Label}
	}
	return nil, &NotLoadedError{edge: "usage_based_line"}
}

// ParentLineOrErr returns the ParentLine value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingInvoiceLineEdges) ParentLineOrErr() (*BillingInvoiceLine, error) {
	if e.ParentLine != nil {
		return e.ParentLine, nil
	} else if e.loadedTypes[3] {
		return nil, &NotFoundError{label: billinginvoiceline.Label}
	}
	return nil, &NotLoadedError{edge: "parent_line"}
}

// DetailedLinesOrErr returns the DetailedLines value or an error if the edge
// was not loaded in eager-loading.
func (e BillingInvoiceLineEdges) DetailedLinesOrErr() ([]*BillingInvoiceLine, error) {
	if e.loadedTypes[4] {
		return e.DetailedLines, nil
	}
	return nil, &NotLoadedError{edge: "detailed_lines"}
}

// LineDiscountsOrErr returns the LineDiscounts value or an error if the edge
// was not loaded in eager-loading.
func (e BillingInvoiceLineEdges) LineDiscountsOrErr() ([]*BillingInvoiceLineDiscount, error) {
	if e.loadedTypes[5] {
		return e.LineDiscounts, nil
	}
	return nil, &NotLoadedError{edge: "line_discounts"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BillingInvoiceLine) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case billinginvoiceline.FieldQuantity:
			values[i] = &sql.NullScanner{S: new(alpacadecimal.Decimal)}
		case billinginvoiceline.FieldMetadata, billinginvoiceline.FieldTaxConfig:
			values[i] = new([]byte)
		case billinginvoiceline.FieldAmount, billinginvoiceline.FieldTaxesTotal, billinginvoiceline.FieldTaxesInclusiveTotal, billinginvoiceline.FieldTaxesExclusiveTotal, billinginvoiceline.FieldChargesTotal, billinginvoiceline.FieldDiscountsTotal, billinginvoiceline.FieldTotal:
			values[i] = new(alpacadecimal.Decimal)
		case billinginvoiceline.FieldID, billinginvoiceline.FieldNamespace, billinginvoiceline.FieldName, billinginvoiceline.FieldDescription, billinginvoiceline.FieldInvoiceID, billinginvoiceline.FieldParentLineID, billinginvoiceline.FieldType, billinginvoiceline.FieldStatus, billinginvoiceline.FieldCurrency, billinginvoiceline.FieldInvoicingAppExternalID, billinginvoiceline.FieldChildUniqueReferenceID:
			values[i] = new(sql.NullString)
		case billinginvoiceline.FieldCreatedAt, billinginvoiceline.FieldUpdatedAt, billinginvoiceline.FieldDeletedAt, billinginvoiceline.FieldPeriodStart, billinginvoiceline.FieldPeriodEnd, billinginvoiceline.FieldInvoiceAt:
			values[i] = new(sql.NullTime)
		case billinginvoiceline.ForeignKeys[0]: // fee_line_config_id
			values[i] = new(sql.NullString)
		case billinginvoiceline.ForeignKeys[1]: // usage_based_line_config_id
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BillingInvoiceLine fields.
func (bil *BillingInvoiceLine) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case billinginvoiceline.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				bil.ID = value.String
			}
		case billinginvoiceline.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				bil.Namespace = value.String
			}
		case billinginvoiceline.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &bil.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case billinginvoiceline.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				bil.CreatedAt = value.Time
			}
		case billinginvoiceline.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				bil.UpdatedAt = value.Time
			}
		case billinginvoiceline.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				bil.DeletedAt = new(time.Time)
				*bil.DeletedAt = value.Time
			}
		case billinginvoiceline.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				bil.Name = value.String
			}
		case billinginvoiceline.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				bil.Description = new(string)
				*bil.Description = value.String
			}
		case billinginvoiceline.FieldAmount:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field amount", values[i])
			} else if value != nil {
				bil.Amount = *value
			}
		case billinginvoiceline.FieldTaxesTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field taxes_total", values[i])
			} else if value != nil {
				bil.TaxesTotal = *value
			}
		case billinginvoiceline.FieldTaxesInclusiveTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field taxes_inclusive_total", values[i])
			} else if value != nil {
				bil.TaxesInclusiveTotal = *value
			}
		case billinginvoiceline.FieldTaxesExclusiveTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field taxes_exclusive_total", values[i])
			} else if value != nil {
				bil.TaxesExclusiveTotal = *value
			}
		case billinginvoiceline.FieldChargesTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field charges_total", values[i])
			} else if value != nil {
				bil.ChargesTotal = *value
			}
		case billinginvoiceline.FieldDiscountsTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field discounts_total", values[i])
			} else if value != nil {
				bil.DiscountsTotal = *value
			}
		case billinginvoiceline.FieldTotal:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field total", values[i])
			} else if value != nil {
				bil.Total = *value
			}
		case billinginvoiceline.FieldInvoiceID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_id", values[i])
			} else if value.Valid {
				bil.InvoiceID = value.String
			}
		case billinginvoiceline.FieldParentLineID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field parent_line_id", values[i])
			} else if value.Valid {
				bil.ParentLineID = new(string)
				*bil.ParentLineID = value.String
			}
		case billinginvoiceline.FieldPeriodStart:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field period_start", values[i])
			} else if value.Valid {
				bil.PeriodStart = value.Time
			}
		case billinginvoiceline.FieldPeriodEnd:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field period_end", values[i])
			} else if value.Valid {
				bil.PeriodEnd = value.Time
			}
		case billinginvoiceline.FieldInvoiceAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_at", values[i])
			} else if value.Valid {
				bil.InvoiceAt = value.Time
			}
		case billinginvoiceline.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				bil.Type = billingentity.InvoiceLineType(value.String)
			}
		case billinginvoiceline.FieldStatus:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field status", values[i])
			} else if value.Valid {
				bil.Status = billingentity.InvoiceLineStatus(value.String)
			}
		case billinginvoiceline.FieldCurrency:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field currency", values[i])
			} else if value.Valid {
				bil.Currency = currencyx.Code(value.String)
			}
		case billinginvoiceline.FieldQuantity:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field quantity", values[i])
			} else if value.Valid {
				bil.Quantity = new(alpacadecimal.Decimal)
				*bil.Quantity = *value.S.(*alpacadecimal.Decimal)
			}
		case billinginvoiceline.FieldTaxConfig:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tax_config", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &bil.TaxConfig); err != nil {
					return fmt.Errorf("unmarshal field tax_config: %w", err)
				}
			}
		case billinginvoiceline.FieldInvoicingAppExternalID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoicing_app_external_id", values[i])
			} else if value.Valid {
				bil.InvoicingAppExternalID = new(string)
				*bil.InvoicingAppExternalID = value.String
			}
		case billinginvoiceline.FieldChildUniqueReferenceID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field child_unique_reference_id", values[i])
			} else if value.Valid {
				bil.ChildUniqueReferenceID = new(string)
				*bil.ChildUniqueReferenceID = value.String
			}
		case billinginvoiceline.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field fee_line_config_id", values[i])
			} else if value.Valid {
				bil.fee_line_config_id = new(string)
				*bil.fee_line_config_id = value.String
			}
		case billinginvoiceline.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field usage_based_line_config_id", values[i])
			} else if value.Valid {
				bil.usage_based_line_config_id = new(string)
				*bil.usage_based_line_config_id = value.String
			}
		default:
			bil.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BillingInvoiceLine.
// This includes values selected through modifiers, order, etc.
func (bil *BillingInvoiceLine) Value(name string) (ent.Value, error) {
	return bil.selectValues.Get(name)
}

// QueryBillingInvoice queries the "billing_invoice" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryBillingInvoice() *BillingInvoiceQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryBillingInvoice(bil)
}

// QueryFlatFeeLine queries the "flat_fee_line" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryFlatFeeLine() *BillingInvoiceFlatFeeLineConfigQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryFlatFeeLine(bil)
}

// QueryUsageBasedLine queries the "usage_based_line" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryUsageBasedLine() *BillingInvoiceUsageBasedLineConfigQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryUsageBasedLine(bil)
}

// QueryParentLine queries the "parent_line" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryParentLine() *BillingInvoiceLineQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryParentLine(bil)
}

// QueryDetailedLines queries the "detailed_lines" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryDetailedLines() *BillingInvoiceLineQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryDetailedLines(bil)
}

// QueryLineDiscounts queries the "line_discounts" edge of the BillingInvoiceLine entity.
func (bil *BillingInvoiceLine) QueryLineDiscounts() *BillingInvoiceLineDiscountQuery {
	return NewBillingInvoiceLineClient(bil.config).QueryLineDiscounts(bil)
}

// Update returns a builder for updating this BillingInvoiceLine.
// Note that you need to call BillingInvoiceLine.Unwrap() before calling this method if this BillingInvoiceLine
// was returned from a transaction, and the transaction was committed or rolled back.
func (bil *BillingInvoiceLine) Update() *BillingInvoiceLineUpdateOne {
	return NewBillingInvoiceLineClient(bil.config).UpdateOne(bil)
}

// Unwrap unwraps the BillingInvoiceLine entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bil *BillingInvoiceLine) Unwrap() *BillingInvoiceLine {
	_tx, ok := bil.config.driver.(*txDriver)
	if !ok {
		panic("db: BillingInvoiceLine is not a transactional entity")
	}
	bil.config.driver = _tx.drv
	return bil
}

// String implements the fmt.Stringer.
func (bil *BillingInvoiceLine) String() string {
	var builder strings.Builder
	builder.WriteString("BillingInvoiceLine(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bil.ID))
	builder.WriteString("namespace=")
	builder.WriteString(bil.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", bil.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(bil.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(bil.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := bil.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(bil.Name)
	builder.WriteString(", ")
	if v := bil.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("amount=")
	builder.WriteString(fmt.Sprintf("%v", bil.Amount))
	builder.WriteString(", ")
	builder.WriteString("taxes_total=")
	builder.WriteString(fmt.Sprintf("%v", bil.TaxesTotal))
	builder.WriteString(", ")
	builder.WriteString("taxes_inclusive_total=")
	builder.WriteString(fmt.Sprintf("%v", bil.TaxesInclusiveTotal))
	builder.WriteString(", ")
	builder.WriteString("taxes_exclusive_total=")
	builder.WriteString(fmt.Sprintf("%v", bil.TaxesExclusiveTotal))
	builder.WriteString(", ")
	builder.WriteString("charges_total=")
	builder.WriteString(fmt.Sprintf("%v", bil.ChargesTotal))
	builder.WriteString(", ")
	builder.WriteString("discounts_total=")
	builder.WriteString(fmt.Sprintf("%v", bil.DiscountsTotal))
	builder.WriteString(", ")
	builder.WriteString("total=")
	builder.WriteString(fmt.Sprintf("%v", bil.Total))
	builder.WriteString(", ")
	builder.WriteString("invoice_id=")
	builder.WriteString(bil.InvoiceID)
	builder.WriteString(", ")
	if v := bil.ParentLineID; v != nil {
		builder.WriteString("parent_line_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("period_start=")
	builder.WriteString(bil.PeriodStart.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("period_end=")
	builder.WriteString(bil.PeriodEnd.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("invoice_at=")
	builder.WriteString(bil.InvoiceAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", bil.Type))
	builder.WriteString(", ")
	builder.WriteString("status=")
	builder.WriteString(fmt.Sprintf("%v", bil.Status))
	builder.WriteString(", ")
	builder.WriteString("currency=")
	builder.WriteString(fmt.Sprintf("%v", bil.Currency))
	builder.WriteString(", ")
	if v := bil.Quantity; v != nil {
		builder.WriteString("quantity=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("tax_config=")
	builder.WriteString(fmt.Sprintf("%v", bil.TaxConfig))
	builder.WriteString(", ")
	if v := bil.InvoicingAppExternalID; v != nil {
		builder.WriteString("invoicing_app_external_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := bil.ChildUniqueReferenceID; v != nil {
		builder.WriteString("child_unique_reference_id=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// BillingInvoiceLines is a parsable slice of BillingInvoiceLine.
type BillingInvoiceLines []*BillingInvoiceLine
