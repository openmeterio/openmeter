package subject

import (
	"errors"
	"time"
)

type BillingPeriod struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
}

var (
	ErrBillingPeriodNotFound     = errors.New("billing period not found")
	ErrBillingPeriodNotSupported = errors.New("billing period is only supported for OpenMeter cloud")
)

type BillingConnector interface {
	GetBillingPeriodOfSubject(namespace string, subjectKey string) (*BillingPeriod, error)
}

type billingConnector struct{}

func NewBillingConnector() BillingConnector {
	return &billingConnector{}
}

func (c *billingConnector) GetBillingPeriodOfSubject(namespace string, subjectKey string) (*BillingPeriod, error) {
	return nil, ErrBillingPeriodNotSupported
}
