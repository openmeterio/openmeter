// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_adapter/ent/db/grant"
)

// Grant is the model entity for the Grant schema.
type Grant struct {
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
	// OwnerID holds the value of the "owner_id" field.
	OwnerID credit.GrantOwner `json:"owner_id,omitempty"`
	// Amount holds the value of the "amount" field.
	Amount float64 `json:"amount,omitempty"`
	// Priority holds the value of the "priority" field.
	Priority uint8 `json:"priority,omitempty"`
	// EffectiveAt holds the value of the "effective_at" field.
	EffectiveAt time.Time `json:"effective_at,omitempty"`
	// Expiration holds the value of the "expiration" field.
	Expiration credit.ExpirationPeriod `json:"expiration,omitempty"`
	// ExpiresAt holds the value of the "expires_at" field.
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	// VoidedAt holds the value of the "voided_at" field.
	VoidedAt *time.Time `json:"voided_at,omitempty"`
	// ResetMaxRollover holds the value of the "reset_max_rollover" field.
	ResetMaxRollover float64 `json:"reset_max_rollover,omitempty"`
	// RecurrenceMaxRollover holds the value of the "recurrence_max_rollover" field.
	RecurrenceMaxRollover *float64 `json:"recurrence_max_rollover,omitempty"`
	// RecurrencePeriod holds the value of the "recurrence_period" field.
	RecurrencePeriod *credit.RecurrencePeriod `json:"recurrence_period,omitempty"`
	// RecurrenceAnchor holds the value of the "recurrence_anchor" field.
	RecurrenceAnchor *time.Time `json:"recurrence_anchor,omitempty"`
	selectValues     sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Grant) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case grant.FieldMetadata, grant.FieldExpiration:
			values[i] = new([]byte)
		case grant.FieldAmount, grant.FieldResetMaxRollover, grant.FieldRecurrenceMaxRollover:
			values[i] = new(sql.NullFloat64)
		case grant.FieldPriority:
			values[i] = new(sql.NullInt64)
		case grant.FieldID, grant.FieldNamespace, grant.FieldOwnerID, grant.FieldRecurrencePeriod:
			values[i] = new(sql.NullString)
		case grant.FieldCreatedAt, grant.FieldUpdatedAt, grant.FieldDeletedAt, grant.FieldEffectiveAt, grant.FieldExpiresAt, grant.FieldVoidedAt, grant.FieldRecurrenceAnchor:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Grant fields.
func (gr *Grant) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case grant.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				gr.ID = value.String
			}
		case grant.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				gr.Namespace = value.String
			}
		case grant.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gr.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case grant.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				gr.CreatedAt = value.Time
			}
		case grant.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				gr.UpdatedAt = value.Time
			}
		case grant.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				gr.DeletedAt = new(time.Time)
				*gr.DeletedAt = value.Time
			}
		case grant.FieldOwnerID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field owner_id", values[i])
			} else if value.Valid {
				gr.OwnerID = credit.GrantOwner(value.String)
			}
		case grant.FieldAmount:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field amount", values[i])
			} else if value.Valid {
				gr.Amount = value.Float64
			}
		case grant.FieldPriority:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field priority", values[i])
			} else if value.Valid {
				gr.Priority = uint8(value.Int64)
			}
		case grant.FieldEffectiveAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field effective_at", values[i])
			} else if value.Valid {
				gr.EffectiveAt = value.Time
			}
		case grant.FieldExpiration:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field expiration", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gr.Expiration); err != nil {
					return fmt.Errorf("unmarshal field expiration: %w", err)
				}
			}
		case grant.FieldExpiresAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field expires_at", values[i])
			} else if value.Valid {
				gr.ExpiresAt = value.Time
			}
		case grant.FieldVoidedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field voided_at", values[i])
			} else if value.Valid {
				gr.VoidedAt = new(time.Time)
				*gr.VoidedAt = value.Time
			}
		case grant.FieldResetMaxRollover:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field reset_max_rollover", values[i])
			} else if value.Valid {
				gr.ResetMaxRollover = value.Float64
			}
		case grant.FieldRecurrenceMaxRollover:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field recurrence_max_rollover", values[i])
			} else if value.Valid {
				gr.RecurrenceMaxRollover = new(float64)
				*gr.RecurrenceMaxRollover = value.Float64
			}
		case grant.FieldRecurrencePeriod:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field recurrence_period", values[i])
			} else if value.Valid {
				gr.RecurrencePeriod = new(credit.RecurrencePeriod)
				*gr.RecurrencePeriod = credit.RecurrencePeriod(value.String)
			}
		case grant.FieldRecurrenceAnchor:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field recurrence_anchor", values[i])
			} else if value.Valid {
				gr.RecurrenceAnchor = new(time.Time)
				*gr.RecurrenceAnchor = value.Time
			}
		default:
			gr.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Grant.
// This includes values selected through modifiers, order, etc.
func (gr *Grant) Value(name string) (ent.Value, error) {
	return gr.selectValues.Get(name)
}

// Update returns a builder for updating this Grant.
// Note that you need to call Grant.Unwrap() before calling this method if this Grant
// was returned from a transaction, and the transaction was committed or rolled back.
func (gr *Grant) Update() *GrantUpdateOne {
	return NewGrantClient(gr.config).UpdateOne(gr)
}

// Unwrap unwraps the Grant entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (gr *Grant) Unwrap() *Grant {
	_tx, ok := gr.config.driver.(*txDriver)
	if !ok {
		panic("db: Grant is not a transactional entity")
	}
	gr.config.driver = _tx.drv
	return gr
}

// String implements the fmt.Stringer.
func (gr *Grant) String() string {
	var builder strings.Builder
	builder.WriteString("Grant(")
	builder.WriteString(fmt.Sprintf("id=%v, ", gr.ID))
	builder.WriteString("namespace=")
	builder.WriteString(gr.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", gr.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(gr.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(gr.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := gr.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("owner_id=")
	builder.WriteString(fmt.Sprintf("%v", gr.OwnerID))
	builder.WriteString(", ")
	builder.WriteString("amount=")
	builder.WriteString(fmt.Sprintf("%v", gr.Amount))
	builder.WriteString(", ")
	builder.WriteString("priority=")
	builder.WriteString(fmt.Sprintf("%v", gr.Priority))
	builder.WriteString(", ")
	builder.WriteString("effective_at=")
	builder.WriteString(gr.EffectiveAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("expiration=")
	builder.WriteString(fmt.Sprintf("%v", gr.Expiration))
	builder.WriteString(", ")
	builder.WriteString("expires_at=")
	builder.WriteString(gr.ExpiresAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := gr.VoidedAt; v != nil {
		builder.WriteString("voided_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("reset_max_rollover=")
	builder.WriteString(fmt.Sprintf("%v", gr.ResetMaxRollover))
	builder.WriteString(", ")
	if v := gr.RecurrenceMaxRollover; v != nil {
		builder.WriteString("recurrence_max_rollover=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := gr.RecurrencePeriod; v != nil {
		builder.WriteString("recurrence_period=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := gr.RecurrenceAnchor; v != nil {
		builder.WriteString("recurrence_anchor=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteByte(')')
	return builder.String()
}

// Grants is a parsable slice of Grant.
type Grants []*Grant
