package entitlementdriverv2

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// parserV2 mirrors the v1 parser pattern for V2 entitlement mappings
type parserV2 struct{}

var ParserV2 = parserV2{}

// ToAPIGenericV2 maps a generic entitlement to the EntitlementV2 union, augmenting with customer fields
func (parserV2) ToAPIGenericV2(e *entitlement.Entitlement, customerId string, customerKey *string) (*api.EntitlementV2, error) {
	if e == nil {
		return nil, fmt.Errorf("nil entitlement")
	}

	res := &api.EntitlementV2{}

	switch e.EntitlementType {
	case entitlement.EntitlementTypeMetered:
		m, err := meteredentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return nil, err
		}
		v, err := ParserV2.ToMeteredV2(m, e, customerId, customerKey)
		if err != nil {
			return nil, err
		}
		if err := res.FromEntitlementMeteredV2(*v); err != nil {
			return nil, err
		}
		return res, nil
	case entitlement.EntitlementTypeStatic:
		s, err := staticentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return nil, err
		}
		v, err := ParserV2.ToStaticV2(s, e, customerId, customerKey)
		if err != nil {
			return nil, err
		}
		if err := res.FromEntitlementStaticV2(*v); err != nil {
			return nil, err
		}
		return res, nil
	case entitlement.EntitlementTypeBoolean:
		b, err := booleanentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return nil, err
		}
		v, err := ParserV2.ToBooleanV2(b, e, customerId, customerKey)
		if err != nil {
			return nil, err
		}
		if err := res.FromEntitlementBooleanV2(*v); err != nil {
			return nil, err
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported entitlement type: %s", e.EntitlementType)
	}
}

func (parserV2) ToMeteredV2(m *meteredentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*api.EntitlementMeteredV2, error) {
	v := api.EntitlementMeteredV2{
		ActiveFrom:         m.ActiveFromTime(),
		ActiveTo:           m.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(api.Annotations(e.Annotations)),
		CreatedAt:          m.CreatedAt,
		CurrentUsagePeriod: mapPeriodValue(&m.CurrentUsagePeriod),
		CustomerId:         customerId,
		CustomerKey:        customerKey,
		DeletedAt:          m.DeletedAt,
		FeatureId:          m.FeatureID,
		FeatureKey:         m.FeatureKey,
		Id:                 m.ID,
		IsSoftLimit:        convert.ToPointer(m.IsSoftLimit),
		IssueAfterReset:    convert.SafeDeRef(m.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *float64 { return &i.Amount }),
		IssueAfterResetPriority: convert.SafeDeRef(m.IssueAfterReset, func(i meteredentitlement.IssueAfterReset) *uint8 {
			return convert.SafeDeRef(i.Priority, func(p uint8) *uint8 { return &p })
		}),
		Issue: func() *api.IssueAfterReset {
			if m.IssueAfterReset != nil {
				return &api.IssueAfterReset{
					Amount:   m.IssueAfterReset.Amount,
					Priority: m.IssueAfterReset.Priority,
				}
			}
			return nil
		}(),
		LastReset:              m.LastReset,
		MeasureUsageFrom:       m.MeasureUsageFrom,
		Metadata:               convert.MapToPointer(m.Metadata),
		PreserveOverageAtReset: convert.ToPointer(m.PreserveOverageAtReset),
		Type:                   api.EntitlementMeteredV2TypeMetered,
		UpdatedAt:              m.UpdatedAt,
		UsagePeriod:            mapUsagePeriodValue(e.UsagePeriod),
	}
	return &v, nil
}

func (parserV2) ToStaticV2(s *staticentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*api.EntitlementStaticV2, error) {
	v := api.EntitlementStaticV2{
		ActiveFrom:         s.ActiveFromTime(),
		ActiveTo:           s.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(api.Annotations(e.Annotations)),
		CreatedAt:          s.CreatedAt,
		CustomerId:         customerId,
		CustomerKey:        customerKey,
		DeletedAt:          s.DeletedAt,
		FeatureId:          s.FeatureID,
		FeatureKey:         s.FeatureKey,
		Id:                 s.ID,
		Metadata:           convert.MapToPointer(s.Metadata),
		Type:               api.EntitlementStaticV2TypeStatic,
		UpdatedAt:          s.UpdatedAt,
		CurrentUsagePeriod: mapPeriodPtr(s.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriodPtr(e.UsagePeriod),
	}
	return &v, nil
}

func (parserV2) ToBooleanV2(b *booleanentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*api.EntitlementBooleanV2, error) {
	v := api.EntitlementBooleanV2{
		ActiveFrom:         b.ActiveFromTime(),
		ActiveTo:           b.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(api.Annotations(e.Annotations)),
		CreatedAt:          b.CreatedAt,
		CustomerId:         customerId,
		CustomerKey:        customerKey,
		DeletedAt:          b.DeletedAt,
		FeatureId:          b.FeatureID,
		FeatureKey:         b.FeatureKey,
		Id:                 b.ID,
		Type:               api.EntitlementBooleanV2TypeBoolean,
		UpdatedAt:          b.UpdatedAt,
		CurrentUsagePeriod: mapPeriodPtr(b.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriodPtr(e.UsagePeriod),
	}
	return &v, nil
}

func mapUsagePeriodValue(u *entitlement.UsagePeriod) api.RecurringPeriod {
	// If nil, return zero value; API type is non-pointer in V2 variants for metered, optional for others
	if u == nil {
		return api.RecurringPeriod{}
	}
	origi := u.GetOriginalValueAsUsagePeriodInput().GetValue()
	return api.RecurringPeriod{
		Anchor:      origi.Anchor,
		Interval:    entitlementdriver.MapRecurrenceToAPI(origi.Interval),
		IntervalISO: origi.Interval.ISOString().String(),
	}
}

func mapUsagePeriodPtr(u *entitlement.UsagePeriod) *api.RecurringPeriod {
	if u == nil {
		return nil
	}
	v := mapUsagePeriodValue(u)
	return &v
}

func mapPeriodPtr(p *timeutil.ClosedPeriod) *api.Period {
	if p == nil {
		return nil
	}
	return &api.Period{From: p.From, To: p.To}
}

func mapPeriodValue(p *timeutil.ClosedPeriod) api.Period {
	if p == nil {
		return api.Period{}
	}
	return api.Period{From: p.From, To: p.To}
}

func MapEntitlementGrantToAPIV2(grant *meteredentitlement.EntitlementGrant) api.EntitlementGrantV2 {
	apiGrant := api.EntitlementGrantV2{
		Amount:            grant.Amount,
		CreatedAt:         grant.CreatedAt,
		EffectiveAt:       grant.EffectiveAt,
		Id:                grant.ID,
		Annotations:       lo.ToPtr(api.Annotations(grant.Annotations)),
		Priority:          convert.ToPointer(grant.Priority),
		UpdatedAt:         grant.UpdatedAt,
		DeletedAt:         grant.DeletedAt,
		EntitlementId:     grant.EntitlementID,
		MaxRolloverAmount: &grant.MaxRolloverAmount,
		MinRolloverAmount: &grant.MinRolloverAmount,
		NextRecurrence:    grant.NextRecurrence,
		VoidedAt:          grant.VoidedAt,
	}

	if grant.Expiration != nil {
		apiGrant.Expiration = &api.ExpirationPeriod{
			Count:    grant.Expiration.Count,
			Duration: api.ExpirationDuration(grant.Expiration.Duration),
		}
		apiGrant.ExpiresAt = grant.ExpiresAt
	}

	if grant.Recurrence != nil {
		apiGrant.Recurrence = &api.RecurringPeriod{
			Anchor:      grant.Recurrence.Anchor,
			Interval:    entitlementdriver.MapRecurrenceToAPI(grant.Recurrence.Interval),
			IntervalISO: grant.Recurrence.Interval.ISOString().String(),
		}
	}

	return apiGrant
}
