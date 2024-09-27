// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// Customer is the model entity for the Customer schema.
type Customer struct {
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
	// BillingAddressCountry holds the value of the "billing_address_country" field.
	BillingAddressCountry *models.CountryCode `json:"billing_address_country,omitempty"`
	// BillingAddressPostalCode holds the value of the "billing_address_postal_code" field.
	BillingAddressPostalCode *string `json:"billing_address_postal_code,omitempty"`
	// BillingAddressState holds the value of the "billing_address_state" field.
	BillingAddressState *string `json:"billing_address_state,omitempty"`
	// BillingAddressCity holds the value of the "billing_address_city" field.
	BillingAddressCity *string `json:"billing_address_city,omitempty"`
	// BillingAddressLine1 holds the value of the "billing_address_line1" field.
	BillingAddressLine1 *string `json:"billing_address_line1,omitempty"`
	// BillingAddressLine2 holds the value of the "billing_address_line2" field.
	BillingAddressLine2 *string `json:"billing_address_line2,omitempty"`
	// BillingAddressPhoneNumber holds the value of the "billing_address_phone_number" field.
	BillingAddressPhoneNumber *string `json:"billing_address_phone_number,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// PrimaryEmail holds the value of the "primary_email" field.
	PrimaryEmail *string `json:"primary_email,omitempty"`
	// Timezone holds the value of the "timezone" field.
	Timezone *timezone.Timezone `json:"timezone,omitempty"`
	// Currency holds the value of the "currency" field.
	Currency *currencyx.Code `json:"currency,omitempty"`
	// ExternalMappingStripeCustomerID holds the value of the "external_mapping_stripe_customer_id" field.
	ExternalMappingStripeCustomerID *string `json:"external_mapping_stripe_customer_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the CustomerQuery when eager-loading is set.
	Edges        CustomerEdges `json:"edges"`
	selectValues sql.SelectValues
}

// CustomerEdges holds the relations/edges for other nodes in the graph.
type CustomerEdges struct {
	// Subjects holds the value of the subjects edge.
	Subjects []*CustomerSubjects `json:"subjects,omitempty"`
	// BillingCustomerOverride holds the value of the billing_customer_override edge.
	BillingCustomerOverride *BillingCustomerOverride `json:"billing_customer_override,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// SubjectsOrErr returns the Subjects value or an error if the edge
// was not loaded in eager-loading.
func (e CustomerEdges) SubjectsOrErr() ([]*CustomerSubjects, error) {
	if e.loadedTypes[0] {
		return e.Subjects, nil
	}
	return nil, &NotLoadedError{edge: "subjects"}
}

// BillingCustomerOverrideOrErr returns the BillingCustomerOverride value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomerEdges) BillingCustomerOverrideOrErr() (*BillingCustomerOverride, error) {
	if e.BillingCustomerOverride != nil {
		return e.BillingCustomerOverride, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: billingcustomeroverride.Label}
	}
	return nil, &NotLoadedError{edge: "billing_customer_override"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Customer) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case customer.FieldMetadata:
			values[i] = new([]byte)
		case customer.FieldID, customer.FieldNamespace, customer.FieldBillingAddressCountry, customer.FieldBillingAddressPostalCode, customer.FieldBillingAddressState, customer.FieldBillingAddressCity, customer.FieldBillingAddressLine1, customer.FieldBillingAddressLine2, customer.FieldBillingAddressPhoneNumber, customer.FieldName, customer.FieldPrimaryEmail, customer.FieldTimezone, customer.FieldCurrency, customer.FieldExternalMappingStripeCustomerID:
			values[i] = new(sql.NullString)
		case customer.FieldCreatedAt, customer.FieldUpdatedAt, customer.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Customer fields.
func (c *Customer) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case customer.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				c.ID = value.String
			}
		case customer.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				c.Namespace = value.String
			}
		case customer.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &c.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case customer.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				c.CreatedAt = value.Time
			}
		case customer.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				c.UpdatedAt = value.Time
			}
		case customer.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				c.DeletedAt = new(time.Time)
				*c.DeletedAt = value.Time
			}
		case customer.FieldBillingAddressCountry:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_country", values[i])
			} else if value.Valid {
				c.BillingAddressCountry = new(models.CountryCode)
				*c.BillingAddressCountry = models.CountryCode(value.String)
			}
		case customer.FieldBillingAddressPostalCode:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_postal_code", values[i])
			} else if value.Valid {
				c.BillingAddressPostalCode = new(string)
				*c.BillingAddressPostalCode = value.String
			}
		case customer.FieldBillingAddressState:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_state", values[i])
			} else if value.Valid {
				c.BillingAddressState = new(string)
				*c.BillingAddressState = value.String
			}
		case customer.FieldBillingAddressCity:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_city", values[i])
			} else if value.Valid {
				c.BillingAddressCity = new(string)
				*c.BillingAddressCity = value.String
			}
		case customer.FieldBillingAddressLine1:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_line1", values[i])
			} else if value.Valid {
				c.BillingAddressLine1 = new(string)
				*c.BillingAddressLine1 = value.String
			}
		case customer.FieldBillingAddressLine2:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_line2", values[i])
			} else if value.Valid {
				c.BillingAddressLine2 = new(string)
				*c.BillingAddressLine2 = value.String
			}
		case customer.FieldBillingAddressPhoneNumber:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_address_phone_number", values[i])
			} else if value.Valid {
				c.BillingAddressPhoneNumber = new(string)
				*c.BillingAddressPhoneNumber = value.String
			}
		case customer.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				c.Name = value.String
			}
		case customer.FieldPrimaryEmail:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field primary_email", values[i])
			} else if value.Valid {
				c.PrimaryEmail = new(string)
				*c.PrimaryEmail = value.String
			}
		case customer.FieldTimezone:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field timezone", values[i])
			} else if value.Valid {
				c.Timezone = new(timezone.Timezone)
				*c.Timezone = timezone.Timezone(value.String)
			}
		case customer.FieldCurrency:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field currency", values[i])
			} else if value.Valid {
				c.Currency = new(currencyx.Code)
				*c.Currency = currencyx.Code(value.String)
			}
		case customer.FieldExternalMappingStripeCustomerID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field external_mapping_stripe_customer_id", values[i])
			} else if value.Valid {
				c.ExternalMappingStripeCustomerID = new(string)
				*c.ExternalMappingStripeCustomerID = value.String
			}
		default:
			c.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Customer.
// This includes values selected through modifiers, order, etc.
func (c *Customer) Value(name string) (ent.Value, error) {
	return c.selectValues.Get(name)
}

// QuerySubjects queries the "subjects" edge of the Customer entity.
func (c *Customer) QuerySubjects() *CustomerSubjectsQuery {
	return NewCustomerClient(c.config).QuerySubjects(c)
}

// QueryBillingCustomerOverride queries the "billing_customer_override" edge of the Customer entity.
func (c *Customer) QueryBillingCustomerOverride() *BillingCustomerOverrideQuery {
	return NewCustomerClient(c.config).QueryBillingCustomerOverride(c)
}

// Update returns a builder for updating this Customer.
// Note that you need to call Customer.Unwrap() before calling this method if this Customer
// was returned from a transaction, and the transaction was committed or rolled back.
func (c *Customer) Update() *CustomerUpdateOne {
	return NewCustomerClient(c.config).UpdateOne(c)
}

// Unwrap unwraps the Customer entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (c *Customer) Unwrap() *Customer {
	_tx, ok := c.config.driver.(*txDriver)
	if !ok {
		panic("db: Customer is not a transactional entity")
	}
	c.config.driver = _tx.drv
	return c
}

// String implements the fmt.Stringer.
func (c *Customer) String() string {
	var builder strings.Builder
	builder.WriteString("Customer(")
	builder.WriteString(fmt.Sprintf("id=%v, ", c.ID))
	builder.WriteString("namespace=")
	builder.WriteString(c.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", c.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(c.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(c.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := c.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := c.BillingAddressCountry; v != nil {
		builder.WriteString("billing_address_country=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := c.BillingAddressPostalCode; v != nil {
		builder.WriteString("billing_address_postal_code=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.BillingAddressState; v != nil {
		builder.WriteString("billing_address_state=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.BillingAddressCity; v != nil {
		builder.WriteString("billing_address_city=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.BillingAddressLine1; v != nil {
		builder.WriteString("billing_address_line1=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.BillingAddressLine2; v != nil {
		builder.WriteString("billing_address_line2=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.BillingAddressPhoneNumber; v != nil {
		builder.WriteString("billing_address_phone_number=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(c.Name)
	builder.WriteString(", ")
	if v := c.PrimaryEmail; v != nil {
		builder.WriteString("primary_email=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := c.Timezone; v != nil {
		builder.WriteString("timezone=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := c.Currency; v != nil {
		builder.WriteString("currency=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := c.ExternalMappingStripeCustomerID; v != nil {
		builder.WriteString("external_mapping_stripe_customer_id=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// Customers is a parsable slice of Customer.
type Customers []*Customer
