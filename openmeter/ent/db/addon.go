// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// Addon is the model entity for the Addon schema.
type Addon struct {
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
	// Key holds the value of the "key" field.
	Key string `json:"key,omitempty"`
	// Version holds the value of the "version" field.
	Version int `json:"version,omitempty"`
	// Currency holds the value of the "currency" field.
	Currency string `json:"currency,omitempty"`
	// InstanceType holds the value of the "instance_type" field.
	InstanceType productcatalog.AddonInstanceType `json:"instance_type,omitempty"`
	// EffectiveFrom holds the value of the "effective_from" field.
	EffectiveFrom *time.Time `json:"effective_from,omitempty"`
	// EffectiveTo holds the value of the "effective_to" field.
	EffectiveTo *time.Time `json:"effective_to,omitempty"`
	// Annotations holds the value of the "annotations" field.
	Annotations map[string]interface{} `json:"annotations,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the AddonQuery when eager-loading is set.
	Edges        AddonEdges `json:"edges"`
	selectValues sql.SelectValues
}

// AddonEdges holds the relations/edges for other nodes in the graph.
type AddonEdges struct {
	// Ratecards holds the value of the ratecards edge.
	Ratecards []*AddonRateCard `json:"ratecards,omitempty"`
	// Plans holds the value of the plans edge.
	Plans []*PlanAddon `json:"plans,omitempty"`
	// SubscriptionAddons holds the value of the subscription_addons edge.
	SubscriptionAddons []*SubscriptionAddon `json:"subscription_addons,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// RatecardsOrErr returns the Ratecards value or an error if the edge
// was not loaded in eager-loading.
func (e AddonEdges) RatecardsOrErr() ([]*AddonRateCard, error) {
	if e.loadedTypes[0] {
		return e.Ratecards, nil
	}
	return nil, &NotLoadedError{edge: "ratecards"}
}

// PlansOrErr returns the Plans value or an error if the edge
// was not loaded in eager-loading.
func (e AddonEdges) PlansOrErr() ([]*PlanAddon, error) {
	if e.loadedTypes[1] {
		return e.Plans, nil
	}
	return nil, &NotLoadedError{edge: "plans"}
}

// SubscriptionAddonsOrErr returns the SubscriptionAddons value or an error if the edge
// was not loaded in eager-loading.
func (e AddonEdges) SubscriptionAddonsOrErr() ([]*SubscriptionAddon, error) {
	if e.loadedTypes[2] {
		return e.SubscriptionAddons, nil
	}
	return nil, &NotLoadedError{edge: "subscription_addons"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Addon) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case addon.FieldMetadata:
			values[i] = new([]byte)
		case addon.FieldVersion:
			values[i] = new(sql.NullInt64)
		case addon.FieldID, addon.FieldNamespace, addon.FieldName, addon.FieldDescription, addon.FieldKey, addon.FieldCurrency, addon.FieldInstanceType:
			values[i] = new(sql.NullString)
		case addon.FieldCreatedAt, addon.FieldUpdatedAt, addon.FieldDeletedAt, addon.FieldEffectiveFrom, addon.FieldEffectiveTo:
			values[i] = new(sql.NullTime)
		case addon.FieldAnnotations:
			values[i] = addon.ValueScanner.Annotations.ScanValue()
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Addon fields.
func (a *Addon) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case addon.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				a.ID = value.String
			}
		case addon.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				a.Namespace = value.String
			}
		case addon.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &a.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case addon.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				a.CreatedAt = value.Time
			}
		case addon.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				a.UpdatedAt = value.Time
			}
		case addon.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				a.DeletedAt = new(time.Time)
				*a.DeletedAt = value.Time
			}
		case addon.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				a.Name = value.String
			}
		case addon.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				a.Description = new(string)
				*a.Description = value.String
			}
		case addon.FieldKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field key", values[i])
			} else if value.Valid {
				a.Key = value.String
			}
		case addon.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				a.Version = int(value.Int64)
			}
		case addon.FieldCurrency:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field currency", values[i])
			} else if value.Valid {
				a.Currency = value.String
			}
		case addon.FieldInstanceType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field instance_type", values[i])
			} else if value.Valid {
				a.InstanceType = productcatalog.AddonInstanceType(value.String)
			}
		case addon.FieldEffectiveFrom:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field effective_from", values[i])
			} else if value.Valid {
				a.EffectiveFrom = new(time.Time)
				*a.EffectiveFrom = value.Time
			}
		case addon.FieldEffectiveTo:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field effective_to", values[i])
			} else if value.Valid {
				a.EffectiveTo = new(time.Time)
				*a.EffectiveTo = value.Time
			}
		case addon.FieldAnnotations:
			if value, err := addon.ValueScanner.Annotations.FromValue(values[i]); err != nil {
				return err
			} else {
				a.Annotations = value
			}
		default:
			a.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Addon.
// This includes values selected through modifiers, order, etc.
func (a *Addon) Value(name string) (ent.Value, error) {
	return a.selectValues.Get(name)
}

// QueryRatecards queries the "ratecards" edge of the Addon entity.
func (a *Addon) QueryRatecards() *AddonRateCardQuery {
	return NewAddonClient(a.config).QueryRatecards(a)
}

// QueryPlans queries the "plans" edge of the Addon entity.
func (a *Addon) QueryPlans() *PlanAddonQuery {
	return NewAddonClient(a.config).QueryPlans(a)
}

// QuerySubscriptionAddons queries the "subscription_addons" edge of the Addon entity.
func (a *Addon) QuerySubscriptionAddons() *SubscriptionAddonQuery {
	return NewAddonClient(a.config).QuerySubscriptionAddons(a)
}

// Update returns a builder for updating this Addon.
// Note that you need to call Addon.Unwrap() before calling this method if this Addon
// was returned from a transaction, and the transaction was committed or rolled back.
func (a *Addon) Update() *AddonUpdateOne {
	return NewAddonClient(a.config).UpdateOne(a)
}

// Unwrap unwraps the Addon entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (a *Addon) Unwrap() *Addon {
	_tx, ok := a.config.driver.(*txDriver)
	if !ok {
		panic("db: Addon is not a transactional entity")
	}
	a.config.driver = _tx.drv
	return a
}

// String implements the fmt.Stringer.
func (a *Addon) String() string {
	var builder strings.Builder
	builder.WriteString("Addon(")
	builder.WriteString(fmt.Sprintf("id=%v, ", a.ID))
	builder.WriteString("namespace=")
	builder.WriteString(a.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", a.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(a.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(a.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := a.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(a.Name)
	builder.WriteString(", ")
	if v := a.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("key=")
	builder.WriteString(a.Key)
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", a.Version))
	builder.WriteString(", ")
	builder.WriteString("currency=")
	builder.WriteString(a.Currency)
	builder.WriteString(", ")
	builder.WriteString("instance_type=")
	builder.WriteString(fmt.Sprintf("%v", a.InstanceType))
	builder.WriteString(", ")
	if v := a.EffectiveFrom; v != nil {
		builder.WriteString("effective_from=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := a.EffectiveTo; v != nil {
		builder.WriteString("effective_to=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("annotations=")
	builder.WriteString(fmt.Sprintf("%v", a.Annotations))
	builder.WriteByte(')')
	return builder.String()
}

// Addons is a parsable slice of Addon.
type Addons []*Addon
