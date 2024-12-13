// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datex"
)

// SubscriptionItem is the model entity for the SubscriptionItem schema.
type SubscriptionItem struct {
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
	// Metadata holds the value of the "metadata" field.
	Metadata map[string]string `json:"metadata,omitempty"`
	// ActiveFrom holds the value of the "active_from" field.
	ActiveFrom time.Time `json:"active_from,omitempty"`
	// ActiveTo holds the value of the "active_to" field.
	ActiveTo *time.Time `json:"active_to,omitempty"`
	// PhaseID holds the value of the "phase_id" field.
	PhaseID string `json:"phase_id,omitempty"`
	// Key holds the value of the "key" field.
	Key string `json:"key,omitempty"`
	// EntitlementID holds the value of the "entitlement_id" field.
	EntitlementID *string `json:"entitlement_id,omitempty"`
	// ActiveFromOverrideRelativeToPhaseStart holds the value of the "active_from_override_relative_to_phase_start" field.
	ActiveFromOverrideRelativeToPhaseStart *datex.ISOString `json:"active_from_override_relative_to_phase_start,omitempty"`
	// ActiveToOverrideRelativeToPhaseStart holds the value of the "active_to_override_relative_to_phase_start" field.
	ActiveToOverrideRelativeToPhaseStart *datex.ISOString `json:"active_to_override_relative_to_phase_start,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// FeatureKey holds the value of the "feature_key" field.
	FeatureKey *string `json:"feature_key,omitempty"`
	// EntitlementTemplate holds the value of the "entitlement_template" field.
	EntitlementTemplate *productcatalog.EntitlementTemplate `json:"entitlement_template,omitempty"`
	// TaxConfig holds the value of the "tax_config" field.
	TaxConfig *productcatalog.TaxConfig `json:"tax_config,omitempty"`
	// BillingCadence holds the value of the "billing_cadence" field.
	BillingCadence *datex.ISOString `json:"billing_cadence,omitempty"`
	// Price holds the value of the "price" field.
	Price *productcatalog.Price `json:"price,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the SubscriptionItemQuery when eager-loading is set.
	Edges        SubscriptionItemEdges `json:"edges"`
	selectValues sql.SelectValues
}

// SubscriptionItemEdges holds the relations/edges for other nodes in the graph.
type SubscriptionItemEdges struct {
	// Phase holds the value of the phase edge.
	Phase *SubscriptionPhase `json:"phase,omitempty"`
	// Entitlement holds the value of the entitlement edge.
	Entitlement *Entitlement `json:"entitlement,omitempty"`
	// BillingLines holds the value of the billing_lines edge.
	BillingLines []*BillingInvoiceLine `json:"billing_lines,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// PhaseOrErr returns the Phase value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionItemEdges) PhaseOrErr() (*SubscriptionPhase, error) {
	if e.Phase != nil {
		return e.Phase, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: subscriptionphase.Label}
	}
	return nil, &NotLoadedError{edge: "phase"}
}

// EntitlementOrErr returns the Entitlement value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionItemEdges) EntitlementOrErr() (*Entitlement, error) {
	if e.Entitlement != nil {
		return e.Entitlement, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: entitlement.Label}
	}
	return nil, &NotLoadedError{edge: "entitlement"}
}

// BillingLinesOrErr returns the BillingLines value or an error if the edge
// was not loaded in eager-loading.
func (e SubscriptionItemEdges) BillingLinesOrErr() ([]*BillingInvoiceLine, error) {
	if e.loadedTypes[2] {
		return e.BillingLines, nil
	}
	return nil, &NotLoadedError{edge: "billing_lines"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*SubscriptionItem) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case subscriptionitem.FieldMetadata:
			values[i] = new([]byte)
		case subscriptionitem.FieldID, subscriptionitem.FieldNamespace, subscriptionitem.FieldPhaseID, subscriptionitem.FieldKey, subscriptionitem.FieldEntitlementID, subscriptionitem.FieldActiveFromOverrideRelativeToPhaseStart, subscriptionitem.FieldActiveToOverrideRelativeToPhaseStart, subscriptionitem.FieldName, subscriptionitem.FieldDescription, subscriptionitem.FieldFeatureKey, subscriptionitem.FieldBillingCadence:
			values[i] = new(sql.NullString)
		case subscriptionitem.FieldCreatedAt, subscriptionitem.FieldUpdatedAt, subscriptionitem.FieldDeletedAt, subscriptionitem.FieldActiveFrom, subscriptionitem.FieldActiveTo:
			values[i] = new(sql.NullTime)
		case subscriptionitem.FieldEntitlementTemplate:
			values[i] = subscriptionitem.ValueScanner.EntitlementTemplate.ScanValue()
		case subscriptionitem.FieldTaxConfig:
			values[i] = subscriptionitem.ValueScanner.TaxConfig.ScanValue()
		case subscriptionitem.FieldPrice:
			values[i] = subscriptionitem.ValueScanner.Price.ScanValue()
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the SubscriptionItem fields.
func (si *SubscriptionItem) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case subscriptionitem.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				si.ID = value.String
			}
		case subscriptionitem.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				si.Namespace = value.String
			}
		case subscriptionitem.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				si.CreatedAt = value.Time
			}
		case subscriptionitem.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				si.UpdatedAt = value.Time
			}
		case subscriptionitem.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				si.DeletedAt = new(time.Time)
				*si.DeletedAt = value.Time
			}
		case subscriptionitem.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &si.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case subscriptionitem.FieldActiveFrom:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field active_from", values[i])
			} else if value.Valid {
				si.ActiveFrom = value.Time
			}
		case subscriptionitem.FieldActiveTo:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field active_to", values[i])
			} else if value.Valid {
				si.ActiveTo = new(time.Time)
				*si.ActiveTo = value.Time
			}
		case subscriptionitem.FieldPhaseID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field phase_id", values[i])
			} else if value.Valid {
				si.PhaseID = value.String
			}
		case subscriptionitem.FieldKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field key", values[i])
			} else if value.Valid {
				si.Key = value.String
			}
		case subscriptionitem.FieldEntitlementID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field entitlement_id", values[i])
			} else if value.Valid {
				si.EntitlementID = new(string)
				*si.EntitlementID = value.String
			}
		case subscriptionitem.FieldActiveFromOverrideRelativeToPhaseStart:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field active_from_override_relative_to_phase_start", values[i])
			} else if value.Valid {
				si.ActiveFromOverrideRelativeToPhaseStart = new(datex.ISOString)
				*si.ActiveFromOverrideRelativeToPhaseStart = datex.ISOString(value.String)
			}
		case subscriptionitem.FieldActiveToOverrideRelativeToPhaseStart:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field active_to_override_relative_to_phase_start", values[i])
			} else if value.Valid {
				si.ActiveToOverrideRelativeToPhaseStart = new(datex.ISOString)
				*si.ActiveToOverrideRelativeToPhaseStart = datex.ISOString(value.String)
			}
		case subscriptionitem.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				si.Name = value.String
			}
		case subscriptionitem.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				si.Description = new(string)
				*si.Description = value.String
			}
		case subscriptionitem.FieldFeatureKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field feature_key", values[i])
			} else if value.Valid {
				si.FeatureKey = new(string)
				*si.FeatureKey = value.String
			}
		case subscriptionitem.FieldEntitlementTemplate:
			if value, err := subscriptionitem.ValueScanner.EntitlementTemplate.FromValue(values[i]); err != nil {
				return err
			} else {
				si.EntitlementTemplate = value
			}
		case subscriptionitem.FieldTaxConfig:
			if value, err := subscriptionitem.ValueScanner.TaxConfig.FromValue(values[i]); err != nil {
				return err
			} else {
				si.TaxConfig = value
			}
		case subscriptionitem.FieldBillingCadence:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_cadence", values[i])
			} else if value.Valid {
				si.BillingCadence = new(datex.ISOString)
				*si.BillingCadence = datex.ISOString(value.String)
			}
		case subscriptionitem.FieldPrice:
			if value, err := subscriptionitem.ValueScanner.Price.FromValue(values[i]); err != nil {
				return err
			} else {
				si.Price = value
			}
		default:
			si.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the SubscriptionItem.
// This includes values selected through modifiers, order, etc.
func (si *SubscriptionItem) Value(name string) (ent.Value, error) {
	return si.selectValues.Get(name)
}

// QueryPhase queries the "phase" edge of the SubscriptionItem entity.
func (si *SubscriptionItem) QueryPhase() *SubscriptionPhaseQuery {
	return NewSubscriptionItemClient(si.config).QueryPhase(si)
}

// QueryEntitlement queries the "entitlement" edge of the SubscriptionItem entity.
func (si *SubscriptionItem) QueryEntitlement() *EntitlementQuery {
	return NewSubscriptionItemClient(si.config).QueryEntitlement(si)
}

// QueryBillingLines queries the "billing_lines" edge of the SubscriptionItem entity.
func (si *SubscriptionItem) QueryBillingLines() *BillingInvoiceLineQuery {
	return NewSubscriptionItemClient(si.config).QueryBillingLines(si)
}

// Update returns a builder for updating this SubscriptionItem.
// Note that you need to call SubscriptionItem.Unwrap() before calling this method if this SubscriptionItem
// was returned from a transaction, and the transaction was committed or rolled back.
func (si *SubscriptionItem) Update() *SubscriptionItemUpdateOne {
	return NewSubscriptionItemClient(si.config).UpdateOne(si)
}

// Unwrap unwraps the SubscriptionItem entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (si *SubscriptionItem) Unwrap() *SubscriptionItem {
	_tx, ok := si.config.driver.(*txDriver)
	if !ok {
		panic("db: SubscriptionItem is not a transactional entity")
	}
	si.config.driver = _tx.drv
	return si
}

// String implements the fmt.Stringer.
func (si *SubscriptionItem) String() string {
	var builder strings.Builder
	builder.WriteString("SubscriptionItem(")
	builder.WriteString(fmt.Sprintf("id=%v, ", si.ID))
	builder.WriteString("namespace=")
	builder.WriteString(si.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(si.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(si.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := si.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", si.Metadata))
	builder.WriteString(", ")
	builder.WriteString("active_from=")
	builder.WriteString(si.ActiveFrom.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := si.ActiveTo; v != nil {
		builder.WriteString("active_to=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("phase_id=")
	builder.WriteString(si.PhaseID)
	builder.WriteString(", ")
	builder.WriteString("key=")
	builder.WriteString(si.Key)
	builder.WriteString(", ")
	if v := si.EntitlementID; v != nil {
		builder.WriteString("entitlement_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := si.ActiveFromOverrideRelativeToPhaseStart; v != nil {
		builder.WriteString("active_from_override_relative_to_phase_start=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := si.ActiveToOverrideRelativeToPhaseStart; v != nil {
		builder.WriteString("active_to_override_relative_to_phase_start=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(si.Name)
	builder.WriteString(", ")
	if v := si.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := si.FeatureKey; v != nil {
		builder.WriteString("feature_key=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := si.EntitlementTemplate; v != nil {
		builder.WriteString("entitlement_template=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := si.TaxConfig; v != nil {
		builder.WriteString("tax_config=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := si.BillingCadence; v != nil {
		builder.WriteString("billing_cadence=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := si.Price; v != nil {
		builder.WriteString("price=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteByte(')')
	return builder.String()
}

// SubscriptionItems is a parsable slice of SubscriptionItem.
type SubscriptionItems []*SubscriptionItem
