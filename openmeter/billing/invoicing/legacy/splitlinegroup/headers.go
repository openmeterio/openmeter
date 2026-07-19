package splitlinegroup

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
	timeutil "github.com/openmeterio/openmeter/pkg/timeutil"
)

type LineHeaderAccessor interface {
	GetDeletedAt() *time.Time
	GetServicePeriod() timeutil.ClosedPeriod
	GetAnnotations() models.Annotations
	GetManagedBy() billing.InvoiceLineManagedBy
	GetInvoiceID() models.NamespacedID
	GetID() billing.LineID
}

var _ LineHeaderAccessor = (*StandardLine)(nil)

type StandardLines []StandardLine

type StandardLine struct {
	ID          billing.LineID
	DeletedAt   *time.Time
	Annotations models.Annotations
	ManagedBy   billing.InvoiceLineManagedBy

	Invoice InvoiceHeader

	ServicePeriod timeutil.ClosedPeriod
	Totals        totals.Totals
	Subscription  *billing.SubscriptionReference
}

func (l StandardLine) GetDeletedAt() *time.Time {
	return l.DeletedAt
}

func (l StandardLine) GetServicePeriod() timeutil.ClosedPeriod {
	return l.ServicePeriod
}

func (l StandardLine) GetAnnotations() models.Annotations {
	return l.Annotations
}

func (l StandardLine) GetManagedBy() billing.InvoiceLineManagedBy {
	return l.ManagedBy
}

func (l StandardLine) GetInvoiceID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: l.ID.Namespace,
		ID:        l.Invoice.ID,
	}
}

func (l StandardLine) GetID() billing.LineID {
	return l.ID
}

func (l StandardLine) Clone() (StandardLine, error) {
	var err error
	l.Annotations, err = l.Annotations.Clone()
	if err != nil {
		return StandardLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	return l, nil
}

var _ LineHeaderAccessor = (*GatheringLine)(nil)

type GatheringLine struct {
	DeletedAt   *time.Time
	ID          billing.LineID
	Annotations models.Annotations
	ManagedBy   billing.InvoiceLineManagedBy

	Invoice InvoiceHeader

	ServicePeriod timeutil.ClosedPeriod
	InvoiceAt     time.Time
	Subscription  *billing.SubscriptionReference
}

func (l GatheringLine) GetDeletedAt() *time.Time {
	return l.DeletedAt
}

func (l GatheringLine) GetServicePeriod() timeutil.ClosedPeriod {
	return l.ServicePeriod
}

func (l GatheringLine) GetAnnotations() models.Annotations {
	return l.Annotations
}

func (l GatheringLine) GetManagedBy() billing.InvoiceLineManagedBy {
	return l.ManagedBy
}

func (l GatheringLine) GetInvoiceID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: l.ID.Namespace,
		ID:        l.Invoice.ID,
	}
}

func (l GatheringLine) GetID() billing.LineID {
	return l.ID
}

func (l *GatheringLine) CloneOrNil() (*GatheringLine, error) {
	if l == nil {
		return nil, nil
	}

	out := *l
	annotations, err := out.Annotations.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning annotations: %w", err)
	}

	out.Annotations = annotations

	return &out, nil
}

type InvoiceHeader struct {
	ID        string
	DeletedAt *time.Time
}
