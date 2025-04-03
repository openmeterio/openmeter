// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// Subscription is the model entity for the Subscription schema.
type Subscription struct {
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
	// BillablesMustAlign holds the value of the "billables_must_align" field.
	BillablesMustAlign bool `json:"billables_must_align,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// PlanID holds the value of the "plan_id" field.
	PlanID *string `json:"plan_id,omitempty"`
	// CustomerID holds the value of the "customer_id" field.
	CustomerID string `json:"customer_id,omitempty"`
	// Currency holds the value of the "currency" field.
	Currency currencyx.Code `json:"currency,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the SubscriptionQuery when eager-loading is set.
	Edges        SubscriptionEdges `json:"edges"`
	selectValues sql.SelectValues
}

// SubscriptionEdges holds the relations/edges for other nodes in the graph.
type SubscriptionEdges struct {
	// Plan holds the value of the plan edge.
	Plan *Plan `json:"plan,omitempty"`
	// Customer holds the value of the customer edge.
	Customer *Customer `json:"customer,omitempty"`
	// Phases holds the value of the phases edge.
	Phases []*SubscriptionPhase `json:"phases,omitempty"`
	// BillingLines holds the value of the billing_lines edge.
	BillingLines []*BillingInvoiceLine `json:"billing_lines,omitempty"`
	// Addons holds the value of the addons edge.
	Addons []*SubscriptionAddon `json:"addons,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [5]bool
}

// PlanOrErr returns the Plan value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionEdges) PlanOrErr() (*Plan, error) {
	if e.Plan != nil {
		return e.Plan, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: plan.Label}
	}
	return nil, &NotLoadedError{edge: "plan"}
}

// CustomerOrErr returns the Customer value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionEdges) CustomerOrErr() (*Customer, error) {
	if e.Customer != nil {
		return e.Customer, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: customer.Label}
	}
	return nil, &NotLoadedError{edge: "customer"}
}

// PhasesOrErr returns the Phases value or an error if the edge
// was not loaded in eager-loading.
func (e SubscriptionEdges) PhasesOrErr() ([]*SubscriptionPhase, error) {
	if e.loadedTypes[2] {
		return e.Phases, nil
	}
	return nil, &NotLoadedError{edge: "phases"}
}

// BillingLinesOrErr returns the BillingLines value or an error if the edge
// was not loaded in eager-loading.
func (e SubscriptionEdges) BillingLinesOrErr() ([]*BillingInvoiceLine, error) {
	if e.loadedTypes[3] {
		return e.BillingLines, nil
	}
	return nil, &NotLoadedError{edge: "billing_lines"}
}

// AddonsOrErr returns the Addons value or an error if the edge
// was not loaded in eager-loading.
func (e SubscriptionEdges) AddonsOrErr() ([]*SubscriptionAddon, error) {
	if e.loadedTypes[4] {
		return e.Addons, nil
	}
	return nil, &NotLoadedError{edge: "addons"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Subscription) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case subscription.FieldMetadata:
			values[i] = new([]byte)
		case subscription.FieldBillablesMustAlign:
			values[i] = new(sql.NullBool)
		case subscription.FieldID, subscription.FieldNamespace, subscription.FieldName, subscription.FieldDescription, subscription.FieldPlanID, subscription.FieldCustomerID, subscription.FieldCurrency:
			values[i] = new(sql.NullString)
		case subscription.FieldCreatedAt, subscription.FieldUpdatedAt, subscription.FieldDeletedAt, subscription.FieldActiveFrom, subscription.FieldActiveTo:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Subscription fields.
func (s *Subscription) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case subscription.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				s.ID = value.String
			}
		case subscription.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				s.Namespace = value.String
			}
		case subscription.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				s.CreatedAt = value.Time
			}
		case subscription.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				s.UpdatedAt = value.Time
			}
		case subscription.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				s.DeletedAt = new(time.Time)
				*s.DeletedAt = value.Time
			}
		case subscription.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &s.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case subscription.FieldActiveFrom:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field active_from", values[i])
			} else if value.Valid {
				s.ActiveFrom = value.Time
			}
		case subscription.FieldActiveTo:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field active_to", values[i])
			} else if value.Valid {
				s.ActiveTo = new(time.Time)
				*s.ActiveTo = value.Time
			}
		case subscription.FieldBillablesMustAlign:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field billables_must_align", values[i])
			} else if value.Valid {
				s.BillablesMustAlign = value.Bool
			}
		case subscription.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				s.Name = value.String
			}
		case subscription.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				s.Description = new(string)
				*s.Description = value.String
			}
		case subscription.FieldPlanID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field plan_id", values[i])
			} else if value.Valid {
				s.PlanID = new(string)
				*s.PlanID = value.String
			}
		case subscription.FieldCustomerID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field customer_id", values[i])
			} else if value.Valid {
				s.CustomerID = value.String
			}
		case subscription.FieldCurrency:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field currency", values[i])
			} else if value.Valid {
				s.Currency = currencyx.Code(value.String)
			}
		default:
			s.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Subscription.
// This includes values selected through modifiers, order, etc.
func (s *Subscription) Value(name string) (ent.Value, error) {
	return s.selectValues.Get(name)
}

// QueryPlan queries the "plan" edge of the Subscription entity.
func (s *Subscription) QueryPlan() *PlanQuery {
	return NewSubscriptionClient(s.config).QueryPlan(s)
}

// QueryCustomer queries the "customer" edge of the Subscription entity.
func (s *Subscription) QueryCustomer() *CustomerQuery {
	return NewSubscriptionClient(s.config).QueryCustomer(s)
}

// QueryPhases queries the "phases" edge of the Subscription entity.
func (s *Subscription) QueryPhases() *SubscriptionPhaseQuery {
	return NewSubscriptionClient(s.config).QueryPhases(s)
}

// QueryBillingLines queries the "billing_lines" edge of the Subscription entity.
func (s *Subscription) QueryBillingLines() *BillingInvoiceLineQuery {
	return NewSubscriptionClient(s.config).QueryBillingLines(s)
}

// QueryAddons queries the "addons" edge of the Subscription entity.
func (s *Subscription) QueryAddons() *SubscriptionAddonQuery {
	return NewSubscriptionClient(s.config).QueryAddons(s)
}

// Update returns a builder for updating this Subscription.
// Note that you need to call Subscription.Unwrap() before calling this method if this Subscription
// was returned from a transaction, and the transaction was committed or rolled back.
func (s *Subscription) Update() *SubscriptionUpdateOne {
	return NewSubscriptionClient(s.config).UpdateOne(s)
}

// Unwrap unwraps the Subscription entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (s *Subscription) Unwrap() *Subscription {
	_tx, ok := s.config.driver.(*txDriver)
	if !ok {
		panic("db: Subscription is not a transactional entity")
	}
	s.config.driver = _tx.drv
	return s
}

// String implements the fmt.Stringer.
func (s *Subscription) String() string {
	var builder strings.Builder
	builder.WriteString("Subscription(")
	builder.WriteString(fmt.Sprintf("id=%v, ", s.ID))
	builder.WriteString("namespace=")
	builder.WriteString(s.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(s.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(s.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := s.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", s.Metadata))
	builder.WriteString(", ")
	builder.WriteString("active_from=")
	builder.WriteString(s.ActiveFrom.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := s.ActiveTo; v != nil {
		builder.WriteString("active_to=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("billables_must_align=")
	builder.WriteString(fmt.Sprintf("%v", s.BillablesMustAlign))
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(s.Name)
	builder.WriteString(", ")
	if v := s.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := s.PlanID; v != nil {
		builder.WriteString("plan_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("customer_id=")
	builder.WriteString(s.CustomerID)
	builder.WriteString(", ")
	builder.WriteString("currency=")
	builder.WriteString(fmt.Sprintf("%v", s.Currency))
	builder.WriteByte(')')
	return builder.String()
}

// Subscriptions is a parsable slice of Subscription.
type Subscriptions []*Subscription
