package billing

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type LineID models.NamespacedID

func (i LineID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type InvoiceLineManagedBy string

const (
	// SubscriptionManagedLine is a line that is managed by a subscription.
	SubscriptionManagedLine InvoiceLineManagedBy = "subscription"
	// SystemManagedLine is a line that is managed by the system (non editable, detailed lines)
	SystemManagedLine InvoiceLineManagedBy = "system"
	// ManuallyManagedLine is a line that is managed manually (e.g. overridden by our API users)
	ManuallyManagedLine InvoiceLineManagedBy = "manual"
)

func (InvoiceLineManagedBy) Values() []string {
	return []string{
		string(SubscriptionManagedLine),
		string(SystemManagedLine),
		string(ManuallyManagedLine),
	}
}

// Period represents a time period, in billing the time period is always interpreted as
// [from, to) (i.e. from is inclusive, to is exclusive).
// TODO: Lets merge this with recurrence.Period
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (p Period) Validate() error {
	if p.Start.IsZero() {
		return errors.New("start is required")
	}

	if p.End.IsZero() {
		return errors.New("end is required")
	}

	if p.Start.After(p.End) {
		return errors.New("start must be before end")
	}

	return nil
}

func (p Period) Truncate(resolution time.Duration) Period {
	return Period{
		Start: p.Start.Truncate(resolution),
		End:   p.End.Truncate(resolution),
	}
}

func (p Period) Equal(other Period) bool {
	return p.Start.Equal(other.Start) && p.End.Equal(other.End)
}

func (p Period) IsEmpty() bool {
	return !p.End.After(p.Start)
}

func (p Period) Contains(t time.Time) bool {
	return t.After(p.Start) && t.Before(p.End)
}

func (p Period) Duration() time.Duration {
	return p.End.Sub(p.Start)
}

type GetLinesForSubscriptionInput struct {
	Namespace      string
	SubscriptionID string
	CustomerID     string
}

func (i GetLinesForSubscriptionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.SubscriptionID == "" {
		return errors.New("subscription id is required")
	}

	if i.CustomerID == "" {
		return errors.New("customer id is required")
	}

	return nil
}
