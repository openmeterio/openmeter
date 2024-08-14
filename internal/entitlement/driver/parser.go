package entitlementdriver

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/internal/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/internal/entitlement/static"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
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
		CreatedAt:   &metered.CreatedAt,
		DeletedAt:   metered.DeletedAt,
		FeatureId:   metered.FeatureID,
		FeatureKey:  metered.FeatureKey,
		Id:          &metered.ID,
		IsSoftLimit: convert.ToPointer(metered.IsSoftLimit),
		IsUnlimited: convert.ToPointer(false), // implement
		IssueAfterReset: convert.SafeDeRef(metered.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *float64 {
			return &i.Amount
		}),
		IssueAfterResetPriority: convert.SafeDeRef(metered.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *int {
			return convert.SafeDeRef(i.Priority, func(p uint8) *int {
				return convert.ToPointer(int(p))
			})
		}),
		MeasureUsageFrom:       metered.MeasureUsageFrom,
		Metadata:               convert.MapToPointer(metered.Metadata),
		SubjectKey:             metered.SubjectKey,
		Type:                   api.EntitlementMeteredType(metered.EntitlementType),
		UpdatedAt:              &metered.UpdatedAt,
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
		CreatedAt:          &static.CreatedAt,
		DeletedAt:          static.DeletedAt,
		FeatureId:          static.FeatureID,
		FeatureKey:         static.FeatureKey,
		Id:                 &static.ID,
		Metadata:           convert.MapToPointer(static.Metadata),
		SubjectKey:         static.SubjectKey,
		Type:               api.EntitlementStaticType(static.EntitlementType),
		UpdatedAt:          &static.UpdatedAt,
		Config:             string(static.Config),
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
		CreatedAt:          &boolean.CreatedAt,
		DeletedAt:          boolean.DeletedAt,
		FeatureId:          boolean.FeatureID,
		FeatureKey:         boolean.FeatureKey,
		Id:                 &boolean.ID,
		Metadata:           convert.MapToPointer(boolean.Metadata),
		SubjectKey:         boolean.SubjectKey,
		Type:               api.EntitlementBooleanType(boolean.EntitlementType),
		UpdatedAt:          &boolean.UpdatedAt,
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
			HasAccess: convert.ToPointer(ent.HasAccess()),
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
			HasAccess: convert.ToPointer(ent.HasAccess()),
			Config:    config,
		}, nil
	case *booleanentitlement.BooleanEntitlementValue:
		return api.EntitlementValue{
			HasAccess: convert.ToPointer(ent.HasAccess()),
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
		Anchor:   u.Anchor,
		Interval: api.RecurringPeriodEnum(u.Interval),
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
		request = entitlement.CreateEntitlementInputs{
			Namespace:       ns,
			FeatureID:       v.FeatureId,
			FeatureKey:      v.FeatureKey,
			SubjectKey:      subjectIdOrKey,
			EntitlementType: entitlement.EntitlementTypeMetered,
			IsSoftLimit:     v.IsSoftLimit,
			IssueAfterReset: v.IssueAfterReset,
			IssueAfterResetPriority: convert.SafeDeRef(v.IssueAfterResetPriority, func(i int) *uint8 {
				return convert.ToPointer(uint8(i))
			}),
			UsagePeriod: &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
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
				apiEnum, err := v.MeasureUsageFrom.AsMeasureUsageFromEnum()
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
			request.UsagePeriod = &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
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
			request.UsagePeriod = &entitlement.UsagePeriod{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
			}
		}
		if v.Metadata != nil {
			request.Metadata = *v.Metadata
		}
	default:
		return request, errors.New("unknown entitlement type")
	}

	return request, nil
}
