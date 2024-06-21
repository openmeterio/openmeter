package subject

import "github.com/openmeterio/openmeter/internal/subject"

var (
	ErrBillingPeriodNotFound     = subject.ErrBillingPeriodNotFound
	ErrBillingPeriodNotSupported = subject.ErrBillingPeriodNotSupported
)

type BillingPeriod = subject.BillingPeriod
type BillingConnector = subject.BillingConnector
