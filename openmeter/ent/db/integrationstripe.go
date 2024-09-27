// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/integrationstripe"
)

// IntegrationStripe is the model entity for the IntegrationStripe schema.
type IntegrationStripe struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// AppID holds the value of the "app_id" field.
	AppID string `json:"app_id,omitempty"`
	// StripeAccountID holds the value of the "stripe_account_id" field.
	StripeAccountID *string `json:"stripe_account_id,omitempty"`
	// StripeLivemode holds the value of the "stripe_livemode" field.
	StripeLivemode *bool `json:"stripe_livemode,omitempty"`
	selectValues   sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*IntegrationStripe) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case integrationstripe.FieldStripeLivemode:
			values[i] = new(sql.NullBool)
		case integrationstripe.FieldID:
			values[i] = new(sql.NullInt64)
		case integrationstripe.FieldNamespace, integrationstripe.FieldAppID, integrationstripe.FieldStripeAccountID:
			values[i] = new(sql.NullString)
		case integrationstripe.FieldCreatedAt, integrationstripe.FieldUpdatedAt, integrationstripe.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the IntegrationStripe fields.
func (is *IntegrationStripe) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case integrationstripe.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			is.ID = int(value.Int64)
		case integrationstripe.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				is.Namespace = value.String
			}
		case integrationstripe.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				is.CreatedAt = value.Time
			}
		case integrationstripe.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				is.UpdatedAt = value.Time
			}
		case integrationstripe.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				is.DeletedAt = new(time.Time)
				*is.DeletedAt = value.Time
			}
		case integrationstripe.FieldAppID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field app_id", values[i])
			} else if value.Valid {
				is.AppID = value.String
			}
		case integrationstripe.FieldStripeAccountID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field stripe_account_id", values[i])
			} else if value.Valid {
				is.StripeAccountID = new(string)
				*is.StripeAccountID = value.String
			}
		case integrationstripe.FieldStripeLivemode:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field stripe_livemode", values[i])
			} else if value.Valid {
				is.StripeLivemode = new(bool)
				*is.StripeLivemode = value.Bool
			}
		default:
			is.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the IntegrationStripe.
// This includes values selected through modifiers, order, etc.
func (is *IntegrationStripe) Value(name string) (ent.Value, error) {
	return is.selectValues.Get(name)
}

// Update returns a builder for updating this IntegrationStripe.
// Note that you need to call IntegrationStripe.Unwrap() before calling this method if this IntegrationStripe
// was returned from a transaction, and the transaction was committed or rolled back.
func (is *IntegrationStripe) Update() *IntegrationStripeUpdateOne {
	return NewIntegrationStripeClient(is.config).UpdateOne(is)
}

// Unwrap unwraps the IntegrationStripe entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (is *IntegrationStripe) Unwrap() *IntegrationStripe {
	_tx, ok := is.config.driver.(*txDriver)
	if !ok {
		panic("db: IntegrationStripe is not a transactional entity")
	}
	is.config.driver = _tx.drv
	return is
}

// String implements the fmt.Stringer.
func (is *IntegrationStripe) String() string {
	var builder strings.Builder
	builder.WriteString("IntegrationStripe(")
	builder.WriteString(fmt.Sprintf("id=%v, ", is.ID))
	builder.WriteString("namespace=")
	builder.WriteString(is.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(is.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(is.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := is.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("app_id=")
	builder.WriteString(is.AppID)
	builder.WriteString(", ")
	if v := is.StripeAccountID; v != nil {
		builder.WriteString("stripe_account_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := is.StripeLivemode; v != nil {
		builder.WriteString("stripe_livemode=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteByte(')')
	return builder.String()
}

// IntegrationStripes is a parsable slice of IntegrationStripe.
type IntegrationStripes []*IntegrationStripe
