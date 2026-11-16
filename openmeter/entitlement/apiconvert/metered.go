package apiconvert

import (
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func ToAPIEntitlementMetered(e *entitlement.EntitlementWithCustomer) (*apiv1models.EntitlementMetered, error) {
	metered, err := meteredentitlement.ParseFromGenericEntitlement(&e.Entitlement)
	if err != nil {
		return nil, err
	}

	var subjKey string
	if e.Customer.UsageAttribution != nil {
		subjKey, err = e.Customer.UsageAttribution.GetFirstSubjectKey()
		if err != nil {
			subjKey = ""
		}
	}

	return &apiv1models.EntitlementMetered{
		ActiveFrom:  metered.ActiveFromTime(),
		ActiveTo:    metered.ActiveToTime(),
		CreatedAt:   metered.CreatedAt,
		DeletedAt:   metered.DeletedAt,
		FeatureId:   metered.FeatureID,
		FeatureKey:  metered.FeatureKey,
		Id:          metered.ID,
		IsSoftLimit: convert.ToPointer(metered.IsSoftLimit),
		IsUnlimited: convert.ToPointer(false), // implement
		IssueAfterReset: convert.SafeDeRef(metered.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *float64 {
			return &i.Amount
		}),
		IssueAfterResetPriority: convert.SafeDeRef(metered.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *uint8 {
			return convert.SafeDeRef(i.Priority, func(p uint8) *uint8 {
				return convert.ToPointer(p)
			})
		}),
		MeasureUsageFrom:       metered.MeasureUsageFrom,
		Metadata:               convert.MapToPointer(metered.Metadata),
		Annotations:            lo.EmptyableToPtr(apiv1models.Annotations(metered.Annotations)),
		SubjectKey:             subjKey,
		Type:                   apiv1models.EntitlementMeteredType(metered.EntitlementType),
		UpdatedAt:              metered.UpdatedAt,
		UsagePeriod:            *ToAPIRecurringPeriod(e.UsagePeriod),
		CurrentUsagePeriod:     *ToAPIPeriod(e.CurrentUsagePeriod),
		LastReset:              metered.LastReset,
		PreserveOverageAtReset: convert.ToPointer(metered.PreserveOverageAtReset),
	}, nil
}

func ToAPIRecurringPeriod(u *entitlement.UsagePeriod) *apiv1models.RecurringPeriod {
	if u == nil {
		return nil
	}

	origi := u.GetOriginalValueAsUsagePeriodInput().GetValue()

	return &apiv1models.RecurringPeriod{
		Anchor:      origi.Anchor,
		Interval:    ToAPIRecurringPeriodInterval(origi.Interval),
		IntervalISO: origi.Interval.ISOString().String(),
	}
}

func ToAPIPeriod(u *timeutil.ClosedPeriod) *apiv1models.Period {
	if u == nil {
		return nil
	}

	return &apiv1models.Period{
		From: u.From,
		To:   u.To,
	}
}

func ToAPIRecurringPeriodInterval(r timeutil.RecurrenceInterval) apiv1models.RecurringPeriodInterval {
	// FIXME: due to the facts that
	// 1. not all components of period.Period are normalizable (e.g. 24h != 1d)
	// 2. `Diff(t1, t2 time.Time) period.Period` always calculates in seconds
	// the results of those diff calculations won't match with exact month, year, etc... values
	//
	// Due to that, this attempt at mapping here happens on a best effort basis, as it's only temporary either way. In cases where it cannot be mapped, we return a new (unexpected by the client value) of the ISO string representation.
	normalised := r.Normalise(false)

	apiInt := &apiv1models.RecurringPeriodInterval{}

	if d, err := normalised.Subtract(timeutil.RecurrencePeriodDaily.ISODuration); err == nil && d.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(apiv1models.RecurringPeriodIntervalEnumDAY)
	} else if w, err := normalised.Subtract(timeutil.RecurrencePeriodWeek.ISODuration); err == nil && w.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(apiv1models.RecurringPeriodIntervalEnumWEEK)
	} else if m, err := normalised.Subtract(timeutil.RecurrencePeriodMonth.ISODuration); err == nil && m.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(apiv1models.RecurringPeriodIntervalEnumMONTH)
	} else if y, err := normalised.Subtract(timeutil.RecurrencePeriodYear.ISODuration); err == nil && y.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(apiv1models.RecurringPeriodIntervalEnumYEAR)
	} else {
		_ = apiInt.FromRecurringPeriodInterval0(r.ISOString().String())
	}

	return *apiInt
}
