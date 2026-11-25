package periodicrealization

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

// The realization periods are the same for every namespace: UTC days.
// Everything that needs a period asks these functions, so making the schedule
// configurable later only changes this file.

// PeriodTo returns the end of the period that contains asOf. A period includes
// its start and excludes its end, so a time that is itself a period end belongs
// to the following period.
func PeriodTo(asOf time.Time) time.Time {
	year, month, day := asOf.UTC().Date()

	return time.Date(year, month, day+1, 0, 0, 0, 0, time.UTC)
}

// RealizeAfter returns the earliest time a period ending at periodTo may be
// realized. Usage up to the period end is only safe to read once the collection
// delay has passed.
func RealizeAfter(periodTo time.Time) time.Time {
	return periodTo.Add(usagebased.InternalCollectionPeriod)
}
