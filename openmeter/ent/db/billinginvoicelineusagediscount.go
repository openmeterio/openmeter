// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelineusagediscount"
)

// BillingInvoiceLineUsageDiscount is the model entity for the BillingInvoiceLineUsageDiscount schema.
type BillingInvoiceLineUsageDiscount struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// LineID holds the value of the "line_id" field.
	LineID string `json:"line_id,omitempty"`
	// ChildUniqueReferenceID holds the value of the "child_unique_reference_id" field.
	ChildUniqueReferenceID *string `json:"child_unique_reference_id,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// Reason holds the value of the "reason" field.
	Reason billing.DiscountReasonType `json:"reason,omitempty"`
	// InvoicingAppExternalID holds the value of the "invoicing_app_external_id" field.
	InvoicingAppExternalID *string `json:"invoicing_app_external_id,omitempty"`
	// Quantity holds the value of the "quantity" field.
	Quantity alpacadecimal.Decimal `json:"quantity,omitempty"`
	// PreLinePeriodQuantity holds the value of the "pre_line_period_quantity" field.
	PreLinePeriodQuantity *alpacadecimal.Decimal `json:"pre_line_period_quantity,omitempty"`
	// ReasonDetails holds the value of the "reason_details" field.
	ReasonDetails *billing.DiscountReason `json:"reason_details,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BillingInvoiceLineUsageDiscountQuery when eager-loading is set.
	Edges        BillingInvoiceLineUsageDiscountEdges `json:"edges"`
	selectValues sql.SelectValues
}

// BillingInvoiceLineUsageDiscountEdges holds the relations/edges for other nodes in the graph.
type BillingInvoiceLineUsageDiscountEdges struct {
	// BillingInvoiceLine holds the value of the billing_invoice_line edge.
	BillingInvoiceLine *BillingInvoiceLine `json:"billing_invoice_line,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// BillingInvoiceLineOrErr returns the BillingInvoiceLine value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingInvoiceLineUsageDiscountEdges) BillingInvoiceLineOrErr() (*BillingInvoiceLine, error) {
	if e.BillingInvoiceLine != nil {
		return e.BillingInvoiceLine, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: billinginvoiceline.Label}
	}
	return nil, &NotLoadedError{edge: "billing_invoice_line"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BillingInvoiceLineUsageDiscount) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case billinginvoicelineusagediscount.FieldPreLinePeriodQuantity:
			values[i] = &sql.NullScanner{S: new(alpacadecimal.Decimal)}
		case billinginvoicelineusagediscount.FieldQuantity:
			values[i] = new(alpacadecimal.Decimal)
		case billinginvoicelineusagediscount.FieldID, billinginvoicelineusagediscount.FieldNamespace, billinginvoicelineusagediscount.FieldLineID, billinginvoicelineusagediscount.FieldChildUniqueReferenceID, billinginvoicelineusagediscount.FieldDescription, billinginvoicelineusagediscount.FieldReason, billinginvoicelineusagediscount.FieldInvoicingAppExternalID:
			values[i] = new(sql.NullString)
		case billinginvoicelineusagediscount.FieldCreatedAt, billinginvoicelineusagediscount.FieldUpdatedAt, billinginvoicelineusagediscount.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		case billinginvoicelineusagediscount.FieldReasonDetails:
			values[i] = billinginvoicelineusagediscount.ValueScanner.ReasonDetails.ScanValue()
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BillingInvoiceLineUsageDiscount fields.
func (_m *BillingInvoiceLineUsageDiscount) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case billinginvoicelineusagediscount.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				_m.ID = value.String
			}
		case billinginvoicelineusagediscount.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				_m.Namespace = value.String
			}
		case billinginvoicelineusagediscount.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				_m.CreatedAt = value.Time
			}
		case billinginvoicelineusagediscount.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				_m.UpdatedAt = value.Time
			}
		case billinginvoicelineusagediscount.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				_m.DeletedAt = new(time.Time)
				*_m.DeletedAt = value.Time
			}
		case billinginvoicelineusagediscount.FieldLineID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field line_id", values[i])
			} else if value.Valid {
				_m.LineID = value.String
			}
		case billinginvoicelineusagediscount.FieldChildUniqueReferenceID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field child_unique_reference_id", values[i])
			} else if value.Valid {
				_m.ChildUniqueReferenceID = new(string)
				*_m.ChildUniqueReferenceID = value.String
			}
		case billinginvoicelineusagediscount.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				_m.Description = new(string)
				*_m.Description = value.String
			}
		case billinginvoicelineusagediscount.FieldReason:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field reason", values[i])
			} else if value.Valid {
				_m.Reason = billing.DiscountReasonType(value.String)
			}
		case billinginvoicelineusagediscount.FieldInvoicingAppExternalID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoicing_app_external_id", values[i])
			} else if value.Valid {
				_m.InvoicingAppExternalID = new(string)
				*_m.InvoicingAppExternalID = value.String
			}
		case billinginvoicelineusagediscount.FieldQuantity:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field quantity", values[i])
			} else if value != nil {
				_m.Quantity = *value
			}
		case billinginvoicelineusagediscount.FieldPreLinePeriodQuantity:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field pre_line_period_quantity", values[i])
			} else if value.Valid {
				_m.PreLinePeriodQuantity = new(alpacadecimal.Decimal)
				*_m.PreLinePeriodQuantity = *value.S.(*alpacadecimal.Decimal)
			}
		case billinginvoicelineusagediscount.FieldReasonDetails:
			if value, err := billinginvoicelineusagediscount.ValueScanner.ReasonDetails.FromValue(values[i]); err != nil {
				return err
			} else {
				_m.ReasonDetails = value
			}
		default:
			_m.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BillingInvoiceLineUsageDiscount.
// This includes values selected through modifiers, order, etc.
func (_m *BillingInvoiceLineUsageDiscount) Value(name string) (ent.Value, error) {
	return _m.selectValues.Get(name)
}

// QueryBillingInvoiceLine queries the "billing_invoice_line" edge of the BillingInvoiceLineUsageDiscount entity.
func (_m *BillingInvoiceLineUsageDiscount) QueryBillingInvoiceLine() *BillingInvoiceLineQuery {
	return NewBillingInvoiceLineUsageDiscountClient(_m.config).QueryBillingInvoiceLine(_m)
}

// Update returns a builder for updating this BillingInvoiceLineUsageDiscount.
// Note that you need to call BillingInvoiceLineUsageDiscount.Unwrap() before calling this method if this BillingInvoiceLineUsageDiscount
// was returned from a transaction, and the transaction was committed or rolled back.
func (_m *BillingInvoiceLineUsageDiscount) Update() *BillingInvoiceLineUsageDiscountUpdateOne {
	return NewBillingInvoiceLineUsageDiscountClient(_m.config).UpdateOne(_m)
}

// Unwrap unwraps the BillingInvoiceLineUsageDiscount entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (_m *BillingInvoiceLineUsageDiscount) Unwrap() *BillingInvoiceLineUsageDiscount {
	_tx, ok := _m.config.driver.(*txDriver)
	if !ok {
		panic("db: BillingInvoiceLineUsageDiscount is not a transactional entity")
	}
	_m.config.driver = _tx.drv
	return _m
}

// String implements the fmt.Stringer.
func (_m *BillingInvoiceLineUsageDiscount) String() string {
	var builder strings.Builder
	builder.WriteString("BillingInvoiceLineUsageDiscount(")
	builder.WriteString(fmt.Sprintf("id=%v, ", _m.ID))
	builder.WriteString("namespace=")
	builder.WriteString(_m.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(_m.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(_m.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := _m.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("line_id=")
	builder.WriteString(_m.LineID)
	builder.WriteString(", ")
	if v := _m.ChildUniqueReferenceID; v != nil {
		builder.WriteString("child_unique_reference_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := _m.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("reason=")
	builder.WriteString(fmt.Sprintf("%v", _m.Reason))
	builder.WriteString(", ")
	if v := _m.InvoicingAppExternalID; v != nil {
		builder.WriteString("invoicing_app_external_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("quantity=")
	builder.WriteString(fmt.Sprintf("%v", _m.Quantity))
	builder.WriteString(", ")
	if v := _m.PreLinePeriodQuantity; v != nil {
		builder.WriteString("pre_line_period_quantity=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := _m.ReasonDetails; v != nil {
		builder.WriteString("reason_details=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteByte(')')
	return builder.String()
}

// BillingInvoiceLineUsageDiscounts is a parsable slice of BillingInvoiceLineUsageDiscount.
type BillingInvoiceLineUsageDiscounts []*BillingInvoiceLineUsageDiscount
