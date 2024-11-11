// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/datex"
)

// PlanRateCard is the model entity for the PlanRateCard schema.
type PlanRateCard struct {
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
	// Type holds the value of the "type" field.
	Type model.RateCardType `json:"type,omitempty"`
	// FeatureKey holds the value of the "feature_key" field.
	FeatureKey *string `json:"feature_key,omitempty"`
	// EntitlementTemplate holds the value of the "entitlement_template" field.
	EntitlementTemplate *model.EntitlementTemplate `json:"entitlement_template,omitempty"`
	// TaxConfig holds the value of the "tax_config" field.
	TaxConfig *model.TaxConfig `json:"tax_config,omitempty"`
	// BillingCadence holds the value of the "billing_cadence" field.
	BillingCadence *datex.ISOString `json:"billing_cadence,omitempty"`
	// Price holds the value of the "price" field.
	Price *model.Price `json:"price,omitempty"`
	// The phase identifier the ratecard is assigned to.
	PhaseID string `json:"phase_id,omitempty"`
	// The feature identifier the ratecard is related to.
	FeatureID *string `json:"feature_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the PlanRateCardQuery when eager-loading is set.
	Edges        PlanRateCardEdges `json:"edges"`
	selectValues sql.SelectValues
}

// PlanRateCardEdges holds the relations/edges for other nodes in the graph.
type PlanRateCardEdges struct {
	// Phase holds the value of the phase edge.
	Phase *PlanPhase `json:"phase,omitempty"`
	// Features holds the value of the features edge.
	Features *Feature `json:"features,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// PhaseOrErr returns the Phase value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanRateCardEdges) PhaseOrErr() (*PlanPhase, error) {
	if e.Phase != nil {
		return e.Phase, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: planphase.Label}
	}
	return nil, &NotLoadedError{edge: "phase"}
}

// FeaturesOrErr returns the Features value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanRateCardEdges) FeaturesOrErr() (*Feature, error) {
	if e.Features != nil {
		return e.Features, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: feature.Label}
	}
	return nil, &NotLoadedError{edge: "features"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*PlanRateCard) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case planratecard.FieldMetadata:
			values[i] = new([]byte)
		case planratecard.FieldID, planratecard.FieldNamespace, planratecard.FieldName, planratecard.FieldDescription, planratecard.FieldKey, planratecard.FieldType, planratecard.FieldFeatureKey, planratecard.FieldBillingCadence, planratecard.FieldPhaseID, planratecard.FieldFeatureID:
			values[i] = new(sql.NullString)
		case planratecard.FieldCreatedAt, planratecard.FieldUpdatedAt, planratecard.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		case planratecard.FieldEntitlementTemplate:
			values[i] = planratecard.ValueScanner.EntitlementTemplate.ScanValue()
		case planratecard.FieldTaxConfig:
			values[i] = planratecard.ValueScanner.TaxConfig.ScanValue()
		case planratecard.FieldPrice:
			values[i] = planratecard.ValueScanner.Price.ScanValue()
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the PlanRateCard fields.
func (prc *PlanRateCard) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case planratecard.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				prc.ID = value.String
			}
		case planratecard.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				prc.Namespace = value.String
			}
		case planratecard.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &prc.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case planratecard.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				prc.CreatedAt = value.Time
			}
		case planratecard.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				prc.UpdatedAt = value.Time
			}
		case planratecard.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				prc.DeletedAt = new(time.Time)
				*prc.DeletedAt = value.Time
			}
		case planratecard.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				prc.Name = value.String
			}
		case planratecard.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				prc.Description = new(string)
				*prc.Description = value.String
			}
		case planratecard.FieldKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field key", values[i])
			} else if value.Valid {
				prc.Key = value.String
			}
		case planratecard.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				prc.Type = model.RateCardType(value.String)
			}
		case planratecard.FieldFeatureKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field feature_key", values[i])
			} else if value.Valid {
				prc.FeatureKey = new(string)
				*prc.FeatureKey = value.String
			}
		case planratecard.FieldEntitlementTemplate:
			if value, err := planratecard.ValueScanner.EntitlementTemplate.FromValue(values[i]); err != nil {
				return err
			} else {
				prc.EntitlementTemplate = value
			}
		case planratecard.FieldTaxConfig:
			if value, err := planratecard.ValueScanner.TaxConfig.FromValue(values[i]); err != nil {
				return err
			} else {
				prc.TaxConfig = value
			}
		case planratecard.FieldBillingCadence:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field billing_cadence", values[i])
			} else if value.Valid {
				prc.BillingCadence = new(datex.ISOString)
				*prc.BillingCadence = datex.ISOString(value.String)
			}
		case planratecard.FieldPrice:
			if value, err := planratecard.ValueScanner.Price.FromValue(values[i]); err != nil {
				return err
			} else {
				prc.Price = value
			}
		case planratecard.FieldPhaseID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field phase_id", values[i])
			} else if value.Valid {
				prc.PhaseID = value.String
			}
		case planratecard.FieldFeatureID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field feature_id", values[i])
			} else if value.Valid {
				prc.FeatureID = new(string)
				*prc.FeatureID = value.String
			}
		default:
			prc.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the PlanRateCard.
// This includes values selected through modifiers, order, etc.
func (prc *PlanRateCard) Value(name string) (ent.Value, error) {
	return prc.selectValues.Get(name)
}

// QueryPhase queries the "phase" edge of the PlanRateCard entity.
func (prc *PlanRateCard) QueryPhase() *PlanPhaseQuery {
	return NewPlanRateCardClient(prc.config).QueryPhase(prc)
}

// QueryFeatures queries the "features" edge of the PlanRateCard entity.
func (prc *PlanRateCard) QueryFeatures() *FeatureQuery {
	return NewPlanRateCardClient(prc.config).QueryFeatures(prc)
}

// Update returns a builder for updating this PlanRateCard.
// Note that you need to call PlanRateCard.Unwrap() before calling this method if this PlanRateCard
// was returned from a transaction, and the transaction was committed or rolled back.
func (prc *PlanRateCard) Update() *PlanRateCardUpdateOne {
	return NewPlanRateCardClient(prc.config).UpdateOne(prc)
}

// Unwrap unwraps the PlanRateCard entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (prc *PlanRateCard) Unwrap() *PlanRateCard {
	_tx, ok := prc.config.driver.(*txDriver)
	if !ok {
		panic("db: PlanRateCard is not a transactional entity")
	}
	prc.config.driver = _tx.drv
	return prc
}

// String implements the fmt.Stringer.
func (prc *PlanRateCard) String() string {
	var builder strings.Builder
	builder.WriteString("PlanRateCard(")
	builder.WriteString(fmt.Sprintf("id=%v, ", prc.ID))
	builder.WriteString("namespace=")
	builder.WriteString(prc.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", prc.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(prc.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(prc.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := prc.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(prc.Name)
	builder.WriteString(", ")
	if v := prc.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("key=")
	builder.WriteString(prc.Key)
	builder.WriteString(", ")
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", prc.Type))
	builder.WriteString(", ")
	if v := prc.FeatureKey; v != nil {
		builder.WriteString("feature_key=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := prc.EntitlementTemplate; v != nil {
		builder.WriteString("entitlement_template=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := prc.TaxConfig; v != nil {
		builder.WriteString("tax_config=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := prc.BillingCadence; v != nil {
		builder.WriteString("billing_cadence=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := prc.Price; v != nil {
		builder.WriteString("price=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("phase_id=")
	builder.WriteString(prc.PhaseID)
	builder.WriteString(", ")
	if v := prc.FeatureID; v != nil {
		builder.WriteString("feature_id=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// PlanRateCards is a parsable slice of PlanRateCard.
type PlanRateCards []*PlanRateCard
