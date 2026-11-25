package periodicrealization

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

// The realization periods are the same for every namespace: UTC days.
// Everything that needs a period asks these functions, so making the schedule
// configurable later only changes this file.

// PeriodTo returns the end of the period that usage before asOf falls into: the
// first period end at or after asOf. Both are exclusive bounds, so an asOf that
// is itself a period end gets that period, not the following one.
func PeriodTo(asOf time.Time) time.Time {
	year, month, day := asOf.UTC().Date()

	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if asOf.Equal(startOfDay) {
		return startOfDay
	}

	return startOfDay.AddDate(0, 0, 1)
}

// RealizeAfter returns the earliest time a period ending at periodTo may be
// realized. Usage up to the period end is only safe to read once the collection
// delay has passed.
func RealizeAfter(periodTo time.Time) time.Time {
	return periodTo.Add(usagebased.InternalCollectionPeriod)
}
