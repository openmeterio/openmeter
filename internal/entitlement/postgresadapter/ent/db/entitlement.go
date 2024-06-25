// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/entitlement"
)

// Entitlement is the model entity for the Entitlement schema.
type Entitlement struct {
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
	// EntitlementType holds the value of the "entitlement_type" field.
	EntitlementType entitlement.EntitlementType `json:"entitlement_type,omitempty"`
	// FeatureID holds the value of the "feature_id" field.
	FeatureID string `json:"feature_id,omitempty"`
	// SubjectKey holds the value of the "subject_key" field.
	SubjectKey string `json:"subject_key,omitempty"`
	// MeasureUsageFrom holds the value of the "measure_usage_from" field.
	MeasureUsageFrom *time.Time `json:"measure_usage_from,omitempty"`
	// IssueAfterReset holds the value of the "issue_after_reset" field.
	IssueAfterReset *float64 `json:"issue_after_reset,omitempty"`
	// IsSoftLimit holds the value of the "is_soft_limit" field.
	IsSoftLimit *bool `json:"is_soft_limit,omitempty"`
	// Config holds the value of the "config" field.
	Config map[string]interface{} `json:"config,omitempty"`
	// UsagePeriodInterval holds the value of the "usage_period_interval" field.
	UsagePeriodInterval *entitlement.UsagePeriodInterval `json:"usage_period_interval,omitempty"`
	// UsagePeriodAnchor holds the value of the "usage_period_anchor" field.
	UsagePeriodAnchor *time.Time `json:"usage_period_anchor,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the EntitlementQuery when eager-loading is set.
	Edges        EntitlementEdges `json:"edges"`
	selectValues sql.SelectValues
}

// EntitlementEdges holds the relations/edges for other nodes in the graph.
type EntitlementEdges struct {
	// UsageReset holds the value of the usage_reset edge.
	UsageReset []*UsageReset `json:"usage_reset,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// UsageResetOrErr returns the UsageReset value or an error if the edge
// was not loaded in eager-loading.
func (e EntitlementEdges) UsageResetOrErr() ([]*UsageReset, error) {
	if e.loadedTypes[0] {
		return e.UsageReset, nil
	}
	return nil, &NotLoadedError{edge: "usage_reset"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Entitlement) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case entitlement.FieldMetadata, entitlement.FieldConfig:
			values[i] = new([]byte)
		case entitlement.FieldIsSoftLimit:
			values[i] = new(sql.NullBool)
		case entitlement.FieldIssueAfterReset:
			values[i] = new(sql.NullFloat64)
		case entitlement.FieldID, entitlement.FieldNamespace, entitlement.FieldEntitlementType, entitlement.FieldFeatureID, entitlement.FieldSubjectKey, entitlement.FieldUsagePeriodInterval:
			values[i] = new(sql.NullString)
		case entitlement.FieldCreatedAt, entitlement.FieldUpdatedAt, entitlement.FieldDeletedAt, entitlement.FieldMeasureUsageFrom, entitlement.FieldUsagePeriodAnchor:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Entitlement fields.
func (e *Entitlement) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case entitlement.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				e.ID = value.String
			}
		case entitlement.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				e.Namespace = value.String
			}
		case entitlement.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case entitlement.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				e.CreatedAt = value.Time
			}
		case entitlement.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				e.UpdatedAt = value.Time
			}
		case entitlement.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				e.DeletedAt = new(time.Time)
				*e.DeletedAt = value.Time
			}
		case entitlement.FieldEntitlementType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field entitlement_type", values[i])
			} else if value.Valid {
				e.EntitlementType = entitlement.EntitlementType(value.String)
			}
		case entitlement.FieldFeatureID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field feature_id", values[i])
			} else if value.Valid {
				e.FeatureID = value.String
			}
		case entitlement.FieldSubjectKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field subject_key", values[i])
			} else if value.Valid {
				e.SubjectKey = value.String
			}
		case entitlement.FieldMeasureUsageFrom:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field measure_usage_from", values[i])
			} else if value.Valid {
				e.MeasureUsageFrom = new(time.Time)
				*e.MeasureUsageFrom = value.Time
			}
		case entitlement.FieldIssueAfterReset:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field issue_after_reset", values[i])
			} else if value.Valid {
				e.IssueAfterReset = new(float64)
				*e.IssueAfterReset = value.Float64
			}
		case entitlement.FieldIsSoftLimit:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field is_soft_limit", values[i])
			} else if value.Valid {
				e.IsSoftLimit = new(bool)
				*e.IsSoftLimit = value.Bool
			}
		case entitlement.FieldConfig:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field config", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.Config); err != nil {
					return fmt.Errorf("unmarshal field config: %w", err)
				}
			}
		case entitlement.FieldUsagePeriodInterval:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field usage_period_interval", values[i])
			} else if value.Valid {
				e.UsagePeriodInterval = new(entitlement.UsagePeriodInterval)
				*e.UsagePeriodInterval = entitlement.UsagePeriodInterval(value.String)
			}
		case entitlement.FieldUsagePeriodAnchor:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field usage_period_anchor", values[i])
			} else if value.Valid {
				e.UsagePeriodAnchor = new(time.Time)
				*e.UsagePeriodAnchor = value.Time
			}
		default:
			e.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Entitlement.
// This includes values selected through modifiers, order, etc.
func (e *Entitlement) Value(name string) (ent.Value, error) {
	return e.selectValues.Get(name)
}

// QueryUsageReset queries the "usage_reset" edge of the Entitlement entity.
func (e *Entitlement) QueryUsageReset() *UsageResetQuery {
	return NewEntitlementClient(e.config).QueryUsageReset(e)
}

// Update returns a builder for updating this Entitlement.
// Note that you need to call Entitlement.Unwrap() before calling this method if this Entitlement
// was returned from a transaction, and the transaction was committed or rolled back.
func (e *Entitlement) Update() *EntitlementUpdateOne {
	return NewEntitlementClient(e.config).UpdateOne(e)
}

// Unwrap unwraps the Entitlement entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (e *Entitlement) Unwrap() *Entitlement {
	_tx, ok := e.config.driver.(*txDriver)
	if !ok {
		panic("db: Entitlement is not a transactional entity")
	}
	e.config.driver = _tx.drv
	return e
}

// String implements the fmt.Stringer.
func (e *Entitlement) String() string {
	var builder strings.Builder
	builder.WriteString("Entitlement(")
	builder.WriteString(fmt.Sprintf("id=%v, ", e.ID))
	builder.WriteString("namespace=")
	builder.WriteString(e.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", e.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(e.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(e.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := e.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("entitlement_type=")
	builder.WriteString(fmt.Sprintf("%v", e.EntitlementType))
	builder.WriteString(", ")
	builder.WriteString("feature_id=")
	builder.WriteString(e.FeatureID)
	builder.WriteString(", ")
	builder.WriteString("subject_key=")
	builder.WriteString(e.SubjectKey)
	builder.WriteString(", ")
	if v := e.MeasureUsageFrom; v != nil {
		builder.WriteString("measure_usage_from=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := e.IssueAfterReset; v != nil {
		builder.WriteString("issue_after_reset=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := e.IsSoftLimit; v != nil {
		builder.WriteString("is_soft_limit=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("config=")
	builder.WriteString(fmt.Sprintf("%v", e.Config))
	builder.WriteString(", ")
	if v := e.UsagePeriodInterval; v != nil {
		builder.WriteString("usage_period_interval=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := e.UsagePeriodAnchor; v != nil {
		builder.WriteString("usage_period_anchor=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteByte(')')
	return builder.String()
}

// Entitlements is a parsable slice of Entitlement.
type Entitlements []*Entitlement
