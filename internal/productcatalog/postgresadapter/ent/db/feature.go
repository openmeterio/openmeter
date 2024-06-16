// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db/feature"
)

// Feature is the model entity for the Feature schema.
type Feature struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// MeterSlug holds the value of the "meter_slug" field.
	MeterSlug string `json:"meter_slug,omitempty"`
	// MeterGroupByFilters holds the value of the "meter_group_by_filters" field.
	MeterGroupByFilters map[string]string `json:"meter_group_by_filters,omitempty"`
	// ArchivedAt holds the value of the "archived_at" field.
	ArchivedAt   *time.Time `json:"archived_at,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Feature) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case feature.FieldMeterGroupByFilters:
			values[i] = new([]byte)
		case feature.FieldID, feature.FieldNamespace, feature.FieldName, feature.FieldMeterSlug:
			values[i] = new(sql.NullString)
		case feature.FieldCreatedAt, feature.FieldUpdatedAt, feature.FieldDeletedAt, feature.FieldArchivedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Feature fields.
func (f *Feature) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case feature.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				f.ID = value.String
			}
		case feature.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				f.CreatedAt = value.Time
			}
		case feature.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				f.UpdatedAt = value.Time
			}
		case feature.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				f.DeletedAt = new(time.Time)
				*f.DeletedAt = value.Time
			}
		case feature.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				f.Namespace = value.String
			}
		case feature.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				f.Name = value.String
			}
		case feature.FieldMeterSlug:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field meter_slug", values[i])
			} else if value.Valid {
				f.MeterSlug = value.String
			}
		case feature.FieldMeterGroupByFilters:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field meter_group_by_filters", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &f.MeterGroupByFilters); err != nil {
					return fmt.Errorf("unmarshal field meter_group_by_filters: %w", err)
				}
			}
		case feature.FieldArchivedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field archived_at", values[i])
			} else if value.Valid {
				f.ArchivedAt = new(time.Time)
				*f.ArchivedAt = value.Time
			}
		default:
			f.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Feature.
// This includes values selected through modifiers, order, etc.
func (f *Feature) Value(name string) (ent.Value, error) {
	return f.selectValues.Get(name)
}

// Update returns a builder for updating this Feature.
// Note that you need to call Feature.Unwrap() before calling this method if this Feature
// was returned from a transaction, and the transaction was committed or rolled back.
func (f *Feature) Update() *FeatureUpdateOne {
	return NewFeatureClient(f.config).UpdateOne(f)
}

// Unwrap unwraps the Feature entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (f *Feature) Unwrap() *Feature {
	_tx, ok := f.config.driver.(*txDriver)
	if !ok {
		panic("db: Feature is not a transactional entity")
	}
	f.config.driver = _tx.drv
	return f
}

// String implements the fmt.Stringer.
func (f *Feature) String() string {
	var builder strings.Builder
	builder.WriteString("Feature(")
	builder.WriteString(fmt.Sprintf("id=%v, ", f.ID))
	builder.WriteString("created_at=")
	builder.WriteString(f.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(f.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := f.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("namespace=")
	builder.WriteString(f.Namespace)
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(f.Name)
	builder.WriteString(", ")
	builder.WriteString("meter_slug=")
	builder.WriteString(f.MeterSlug)
	builder.WriteString(", ")
	builder.WriteString("meter_group_by_filters=")
	builder.WriteString(fmt.Sprintf("%v", f.MeterGroupByFilters))
	builder.WriteString(", ")
	if v := f.ArchivedAt; v != nil {
		builder.WriteString("archived_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteByte(')')
	return builder.String()
}

// Features is a parsable slice of Feature.
type Features []*Feature