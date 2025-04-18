// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	dbapp "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicing"
)

// AppCustomInvoicing is the model entity for the AppCustomInvoicing schema.
type AppCustomInvoicing struct {
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
	// SkipDraftSyncHook holds the value of the "skip_draft_sync_hook" field.
	SkipDraftSyncHook bool `json:"skip_draft_sync_hook,omitempty"`
	// SkipIssuingSyncHook holds the value of the "skip_issuing_sync_hook" field.
	SkipIssuingSyncHook bool `json:"skip_issuing_sync_hook,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the AppCustomInvoicingQuery when eager-loading is set.
	Edges        AppCustomInvoicingEdges `json:"edges"`
	selectValues sql.SelectValues
}

// AppCustomInvoicingEdges holds the relations/edges for other nodes in the graph.
type AppCustomInvoicingEdges struct {
	// CustomerApps holds the value of the customer_apps edge.
	CustomerApps []*AppCustomInvoicingCustomer `json:"customer_apps,omitempty"`
	// App holds the value of the app edge.
	App *App `json:"app,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// CustomerAppsOrErr returns the CustomerApps value or an error if the edge
// was not loaded in eager-loading.
func (e AppCustomInvoicingEdges) CustomerAppsOrErr() ([]*AppCustomInvoicingCustomer, error) {
	if e.loadedTypes[0] {
		return e.CustomerApps, nil
	}
	return nil, &NotLoadedError{edge: "customer_apps"}
}

// AppOrErr returns the App value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AppCustomInvoicingEdges) AppOrErr() (*App, error) {
	if e.App != nil {
		return e.App, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: dbapp.Label}
	}
	return nil, &NotLoadedError{edge: "app"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*AppCustomInvoicing) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case appcustominvoicing.FieldSkipDraftSyncHook, appcustominvoicing.FieldSkipIssuingSyncHook:
			values[i] = new(sql.NullBool)
		case appcustominvoicing.FieldID, appcustominvoicing.FieldNamespace:
			values[i] = new(sql.NullString)
		case appcustominvoicing.FieldCreatedAt, appcustominvoicing.FieldUpdatedAt, appcustominvoicing.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the AppCustomInvoicing fields.
func (aci *AppCustomInvoicing) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case appcustominvoicing.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				aci.ID = value.String
			}
		case appcustominvoicing.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				aci.Namespace = value.String
			}
		case appcustominvoicing.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				aci.CreatedAt = value.Time
			}
		case appcustominvoicing.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				aci.UpdatedAt = value.Time
			}
		case appcustominvoicing.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				aci.DeletedAt = new(time.Time)
				*aci.DeletedAt = value.Time
			}
		case appcustominvoicing.FieldSkipDraftSyncHook:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field skip_draft_sync_hook", values[i])
			} else if value.Valid {
				aci.SkipDraftSyncHook = value.Bool
			}
		case appcustominvoicing.FieldSkipIssuingSyncHook:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field skip_issuing_sync_hook", values[i])
			} else if value.Valid {
				aci.SkipIssuingSyncHook = value.Bool
			}
		default:
			aci.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the AppCustomInvoicing.
// This includes values selected through modifiers, order, etc.
func (aci *AppCustomInvoicing) Value(name string) (ent.Value, error) {
	return aci.selectValues.Get(name)
}

// QueryCustomerApps queries the "customer_apps" edge of the AppCustomInvoicing entity.
func (aci *AppCustomInvoicing) QueryCustomerApps() *AppCustomInvoicingCustomerQuery {
	return NewAppCustomInvoicingClient(aci.config).QueryCustomerApps(aci)
}

// QueryApp queries the "app" edge of the AppCustomInvoicing entity.
func (aci *AppCustomInvoicing) QueryApp() *AppQuery {
	return NewAppCustomInvoicingClient(aci.config).QueryApp(aci)
}

// Update returns a builder for updating this AppCustomInvoicing.
// Note that you need to call AppCustomInvoicing.Unwrap() before calling this method if this AppCustomInvoicing
// was returned from a transaction, and the transaction was committed or rolled back.
func (aci *AppCustomInvoicing) Update() *AppCustomInvoicingUpdateOne {
	return NewAppCustomInvoicingClient(aci.config).UpdateOne(aci)
}

// Unwrap unwraps the AppCustomInvoicing entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (aci *AppCustomInvoicing) Unwrap() *AppCustomInvoicing {
	_tx, ok := aci.config.driver.(*txDriver)
	if !ok {
		panic("db: AppCustomInvoicing is not a transactional entity")
	}
	aci.config.driver = _tx.drv
	return aci
}

// String implements the fmt.Stringer.
func (aci *AppCustomInvoicing) String() string {
	var builder strings.Builder
	builder.WriteString("AppCustomInvoicing(")
	builder.WriteString(fmt.Sprintf("id=%v, ", aci.ID))
	builder.WriteString("namespace=")
	builder.WriteString(aci.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(aci.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(aci.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := aci.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("skip_draft_sync_hook=")
	builder.WriteString(fmt.Sprintf("%v", aci.SkipDraftSyncHook))
	builder.WriteString(", ")
	builder.WriteString("skip_issuing_sync_hook=")
	builder.WriteString(fmt.Sprintf("%v", aci.SkipIssuingSyncHook))
	builder.WriteByte(')')
	return builder.String()
}

// AppCustomInvoicings is a parsable slice of AppCustomInvoicing.
type AppCustomInvoicings []*AppCustomInvoicing
