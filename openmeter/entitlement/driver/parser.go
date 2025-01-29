package entitlementdriver

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type parser struct{}

var Parser = parser{}

func (parser) ToMetered(e *entitlement.Entitlement) (*api.EntitlementMetered, error) {
	metered, err := meteredentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	return &api.EntitlementMetered{
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
		SubjectKey:             metered.SubjectKey,
		Type:                   api.EntitlementMeteredType(metered.EntitlementType),
		UpdatedAt:              metered.UpdatedAt,
		UsagePeriod:            *mapUsagePeriod(e.UsagePeriod),
		CurrentUsagePeriod:     *mapPeriod(e.CurrentUsagePeriod),
		LastReset:              metered.LastReset,
		PreserveOverageAtReset: convert.ToPointer(metered.PreserveOverageAtReset),
	}, nil
}

func (parser) ToStatic(e *entitlement.Entitlement) (*api.EntitlementStatic, error) {
	static, err := staticentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	apiRes := &api.EntitlementStatic{
		ActiveFrom:         static.ActiveFromTime(),
		ActiveTo:           static.ActiveToTime(),
		CreatedAt:          static.CreatedAt,
		DeletedAt:          static.DeletedAt,
		FeatureId:          static.FeatureID,
		FeatureKey:         static.FeatureKey,
		Id:                 static.ID,
		Metadata:           convert.MapToPointer(static.Metadata),
		SubjectKey:         static.SubjectKey,
		Type:               api.EntitlementStaticType(static.EntitlementType),
		UpdatedAt:          static.UpdatedAt,
		Config:             static.Config,
		CurrentUsagePeriod: mapPeriod(static.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriod(e.UsagePeriod),
	}

	return apiRes, nil
}

func (parser) ToBoolean(e *entitlement.Entitlement) (*api.EntitlementBoolean, error) {
	boolean, err := booleanentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	apiRes := &api.EntitlementBoolean{
		ActiveFrom:         boolean.ActiveFromTime(),
		ActiveTo:           boolean.ActiveToTime(),
		CreatedAt:          boolean.CreatedAt,
		DeletedAt:          boolean.DeletedAt,
		FeatureId:          boolean.FeatureID,
		FeatureKey:         boolean.FeatureKey,
		Id:                 boolean.ID,
		Metadata:           convert.MapToPointer(boolean.Metadata),
		SubjectKey:         boolean.SubjectKey,
		Type:               string(boolean.EntitlementType),
		UpdatedAt:          boolean.UpdatedAt,
		CurrentUsagePeriod: mapPeriod(boolean.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriod(e.UsagePeriod),
	}

	return apiRes, nil
}

func (p parser) ToAPIGeneric(e *entitlement.Entitlement) (*api.Entitlement, error) {
	res := &api.Entitlement{}
	switch e.EntitlementType {
	case entitlement.EntitlementTypeMetered:
		c, err := p.ToMetered(e)
		if err != nil {
			return nil, err
		}
		err = res.FromEntitlementMetered(*c)
		if err != nil {
			return nil, err
		}
		return res, nil
	case entitlement.EntitlementTypeStatic:
		c, err := p.ToStatic(e)
		if err != nil {
			return nil, err
		}
		err = res.FromEntitlementStatic(*c)
		if err != nil {
			return nil, err
		}
		return res, nil
	case entitlement.EntitlementTypeBoolean:
		c, err := p.ToBoolean(e)
		if err != nil {
			return nil, err
		}
		err = res.FromEntitlementBoolean(*c)
		if err != nil {
			return nil, err
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported entitlement type: %s", e.EntitlementType)
	}
}

func MapEntitlementValueToAPI(entitlementValue entitlement.EntitlementValue) (api.EntitlementValue, error) {
	switch ent := entitlementValue.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		return api.EntitlementValue{
			HasAccess: ent.HasAccess(),
			Balance:   &ent.Balance,
			Usage:     &ent.UsageInPeriod,
			Overage:   &ent.Overage,
		}, nil
	case *staticentitlement.StaticEntitlementValue:
		var config *string
		if len(ent.Config) > 0 {
			config = convert.ToPointer(string(ent.Config))
		}

		return api.EntitlementValue{
			HasAccess: ent.HasAccess(),
			Config:    config,
		}, nil
	case *booleanentitlement.BooleanEntitlementValue:
		return api.EntitlementValue{
			HasAccess: ent.HasAccess(),
		}, nil
	case *entitlement.NoAccessValue:
		return api.EntitlementValue{
			HasAccess: false,
		}, nil
	default:
		return api.EntitlementValue{}, errors.New("unknown entitlement type")
	}
}

func mapUsagePeriod(u *entitlement.UsagePeriod) *api.RecurringPeriod {
	if u == nil {
		return nil
	}
	return &api.RecurringPeriod{
		Anchor:      u.Anchor,
		Interval:    MapRecurrenceToAPI(u.Interval),
		IntervalISO: u.Interval.ISOString().String(),
	}
}

func mapPeriod(u *recurrence.Period) *api.Period {
	if u == nil {
		return nil
	}
	return &api.Period{
		From: u.From,
		To:   u.To,
	}
}

func ParseAPICreateInput(inp *api.EntitlementCreateInputs, ns string, subjectIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
	request := entitlement.CreateEntitlementInputs{}
	if inp == nil {
		return request, errors.New("input is nil")
	}

	value, err := inp.ValueByDiscriminator()
	if err != nil {
		return request, err
	}

	switch v := value.(type) {
	case api.EntitlementMeteredCreateInputs:
		iv, err := MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
		if err != nil {
			return request, fmt.Errorf("failed to map interval: %w", err)
		}

		request = entitlement.CreateEntitlementInputs{
			Namespace:               ns,
			FeatureID:               v.FeatureId,
			FeatureKey:              v.FeatureKey,
			SubjectKey:              subjectIdOrKey,
			EntitlementType:         entitlement.EntitlementTypeMetered,
			IsSoftLimit:             v.IsSoftLimit,
			IssueAfterReset:         v.IssueAfterReset,
			IssueAfterResetPriority: v.IssueAfterResetPriority,
			UsagePeriod: &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			},
			PreserveOverageAtReset: v.PreserveOverageAtReset,
		}
		if v.Metadata != nil {
			request.Metadata = *v.Metadata
		}
		if v.MeasureUsageFrom != nil {
			measureUsageFrom := &entitlement.MeasureUsageFromInput{}
			apiTime, err := v.MeasureUsageFrom.AsMeasureUsageFromTime()
			if err == nil {
				err := measureUsageFrom.FromTime(apiTime)
				if err != nil {
					return request, err
				}
			} else {
				apiEnum, err := v.MeasureUsageFrom.AsMeasureUsageFromPreset()
				if err != nil {
					return request, err
				}

				// sanity check
				if request.UsagePeriod == nil {
					return request, errors.New("usage period is required for enum measure usage from")
				}

				err = measureUsageFrom.FromEnum(entitlement.MeasureUsageFromEnum(apiEnum), *request.UsagePeriod, clock.Now())
				if err != nil {
					return request, err
				}
			}
			request.MeasureUsageFrom = measureUsageFrom
		}
	case api.EntitlementStaticCreateInputs:
		request = entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			FeatureID:       v.FeatureId,
			FeatureKey:      v.FeatureKey,
			SubjectKey:      subjectIdOrKey,
			EntitlementType: entitlement.EntitlementTypeStatic,
			Config:          []byte(v.Config),
		}
		if v.UsagePeriod != nil {
			iv, err := MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
			if err != nil {
				return request, fmt.Errorf("failed to map interval: %w", err)
			}

			request.UsagePeriod = &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			}
		}
		if v.Metadata != nil {
			request.Metadata = *v.Metadata
		}
	case api.EntitlementBooleanCreateInputs:
		request = entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			FeatureID:       v.FeatureId,
			FeatureKey:      v.FeatureKey,
			SubjectKey:      subjectIdOrKey,
			EntitlementType: entitlement.EntitlementTypeBoolean,
		}
		if v.UsagePeriod != nil {
			iv, err := MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
			if err != nil {
				return request, fmt.Errorf("failed to map interval: %w", err)
			}

			request.UsagePeriod = &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			}
		}
		if v.Metadata != nil {
			request.Metadata = *v.Metadata
		}
	default:
		return request, errors.New("unknown entitlement type")
	}

	// We prune activity data explicitly
	request.ActiveFrom = nil
	request.ActiveTo = nil

	return request, nil
}

func MapAPIPeriodIntervalToRecurrence(interval api.RecurringPeriodInterval) (recurrence.RecurrenceInterval, error) {
	str, err := interval.AsRecurringPeriodInterval0()
	if err != nil {
		return recurrence.RecurrenceInterval{}, err
	}

	switch str {
	case string(api.RecurringPeriodIntervalEnumDAY):
		return recurrence.RecurrencePeriodDaily, nil
	case string(api.RecurringPeriodIntervalEnumWEEK):
		return recurrence.RecurrencePeriodWeek, nil
	case string(api.RecurringPeriodIntervalEnumMONTH):
		return recurrence.RecurrencePeriodMonth, nil
	case string(api.RecurringPeriodIntervalEnumYEAR):
		return recurrence.RecurrencePeriodYear, nil
	default:
		p, err := datex.ISOString(str).Parse()

		return recurrence.RecurrenceInterval{Period: p}, err
	}
}

func MapRecurrenceToAPI(r recurrence.RecurrenceInterval) api.RecurringPeriodInterval {
	// FIXME: due to the facts that
	// 1. not all components of period.Period are normalizable (e.g. 24h != 1d)
	// 2. `Diff(t1, t2 time.Time) period.Period` always calculates in seconds
	// the results of those diff calculations won't match with exact month, year, etc... values
	//
	// Due to that, this attempt at mapping here happens on a best effort basis, as it's only temporary either way. In cases where it cannot be mapped, we return a new (unexpected by the client value) of the ISO string representation.
	normalised := r.Normalise(false)

	apiInt := &api.RecurringPeriodInterval{}

	if d, err := normalised.Subtract(recurrence.RecurrencePeriodDaily.Period); err == nil && d.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumDAY)
	} else if w, err := normalised.Subtract(recurrence.RecurrencePeriodWeek.Period); err == nil && w.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumWEEK)
	} else if m, err := normalised.Subtract(recurrence.RecurrencePeriodMonth.Period); err == nil && m.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH)
	} else if y, err := normalised.Subtract(recurrence.RecurrencePeriodYear.Period); err == nil && y.IsZero() {
		_ = apiInt.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumYEAR)
	} else {
		_ = apiInt.FromRecurringPeriodInterval0(r.ISOString().String())
	}

	return *apiInt
}
