package entitlementdriverv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// parserV2 mirrors the v1 parser pattern for V2 entitlement mappings
type parserV2 struct{}

var ParserV2 = parserV2{}

// ToAPIGenericV2 maps a generic entitlement to the EntitlementV2 union, augmenting with customer fields
func (parserV2) ToAPIGenericV2(e *entitlement.Entitlement, customerId string, customerKey *string) (*apiv1models.EntitlementV2, error) {
	if e == nil {
		return nil, fmt.Errorf("nil entitlement")
	}

	res := &apiv1models.EntitlementV2{}

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

func (parserV2) ToMeteredV2(m *meteredentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*apiv1models.EntitlementMeteredV2, error) {
	v := apiv1models.EntitlementMeteredV2{
		ActiveFrom:         m.ActiveFromTime(),
		ActiveTo:           m.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(apiv1models.Annotations(e.Annotations)),
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
		Issue: func() *apiv1models.IssueAfterReset {
			if m.IssueAfterReset != nil {
				return &apiv1models.IssueAfterReset{
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
		Type:                   apiv1models.EntitlementMeteredV2TypeMetered,
		UpdatedAt:              m.UpdatedAt,
		UsagePeriod:            mapUsagePeriodValue(e.UsagePeriod),
	}

	return &v, nil
}

func (parserV2) ToStaticV2(s *staticentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*apiv1models.EntitlementStaticV2, error) {
	v := apiv1models.EntitlementStaticV2{
		ActiveFrom:         s.ActiveFromTime(),
		ActiveTo:           s.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(apiv1models.Annotations(e.Annotations)),
		CreatedAt:          s.CreatedAt,
		CustomerId:         customerId,
		CustomerKey:        customerKey,
		DeletedAt:          s.DeletedAt,
		FeatureId:          s.FeatureID,
		FeatureKey:         s.FeatureKey,
		Id:                 s.ID,
		Metadata:           convert.MapToPointer(s.Metadata),
		Type:               apiv1models.EntitlementStaticV2TypeStatic,
		UpdatedAt:          s.UpdatedAt,
		CurrentUsagePeriod: mapPeriodPtr(s.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriodPtr(e.UsagePeriod),
	}

	var err error

	v.Config, err = json.Marshal(s.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal static entitlement config: %w", err)
	}

	return &v, nil
}

func (parserV2) ToBooleanV2(b *booleanentitlement.Entitlement, e *entitlement.Entitlement, customerId string, customerKey *string) (*apiv1models.EntitlementBooleanV2, error) {
	v := apiv1models.EntitlementBooleanV2{
		ActiveFrom:         b.ActiveFromTime(),
		ActiveTo:           b.ActiveToTime(),
		Annotations:        lo.EmptyableToPtr(apiv1models.Annotations(e.Annotations)),
		CreatedAt:          b.CreatedAt,
		CustomerId:         customerId,
		CustomerKey:        customerKey,
		DeletedAt:          b.DeletedAt,
		FeatureId:          b.FeatureID,
		FeatureKey:         b.FeatureKey,
		Id:                 b.ID,
		Type:               apiv1models.EntitlementBooleanV2TypeBoolean,
		UpdatedAt:          b.UpdatedAt,
		CurrentUsagePeriod: mapPeriodPtr(b.CurrentUsagePeriod),
		UsagePeriod:        mapUsagePeriodPtr(e.UsagePeriod),
	}

	return &v, nil
}

func mapUsagePeriodValue(u *entitlement.UsagePeriod) apiv1models.RecurringPeriod {
	// If nil, return zero value; API type is non-pointer in V2 variants for metered, optional for others
	if u == nil {
		return apiv1models.RecurringPeriod{}
	}

	origi := u.GetOriginalValueAsUsagePeriodInput().GetValue()

	return apiv1models.RecurringPeriod{
		Anchor:      origi.Anchor,
		Interval:    entitlementdriver.MapRecurrenceToAPI(origi.Interval),
		IntervalISO: origi.Interval.ISOString().String(),
	}
}

func mapUsagePeriodPtr(u *entitlement.UsagePeriod) *apiv1models.RecurringPeriod {
	if u == nil {
		return nil
	}

	v := mapUsagePeriodValue(u)

	return &v
}

func mapPeriodPtr(p *timeutil.ClosedPeriod) *apiv1models.Period {
	if p == nil {
		return nil
	}

	return &apiv1models.Period{From: p.From, To: p.To}
}

func mapPeriodValue(p *timeutil.ClosedPeriod) apiv1models.Period {
	if p == nil {
		return apiv1models.Period{}
	}

	return apiv1models.Period{From: p.From, To: p.To}
}

func MapEntitlementGrantToAPIV2(grant *meteredentitlement.EntitlementGrant) apiv1models.EntitlementGrantV2 {
	apiGrant := apiv1models.EntitlementGrantV2{
		Amount:            grant.Amount,
		CreatedAt:         grant.CreatedAt,
		EffectiveAt:       grant.EffectiveAt,
		Id:                grant.ID,
		Annotations:       (*apiv1models.Annotations)(convert.MapToPointer(grant.Annotations)),
		Metadata:          convert.MapToPointer(grant.Metadata),
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
		apiGrant.Expiration = &apiv1models.ExpirationPeriod{
			Count:    grant.Expiration.Count,
			Duration: apiv1models.ExpirationDuration(grant.Expiration.Duration),
		}
		apiGrant.ExpiresAt = grant.ExpiresAt
	}

	if grant.Recurrence != nil {
		apiGrant.Recurrence = &apiv1models.RecurringPeriod{
			Anchor:      grant.Recurrence.Anchor,
			Interval:    entitlementdriver.MapRecurrenceToAPI(grant.Recurrence.Interval),
			IntervalISO: grant.Recurrence.Interval.ISOString().String(),
		}
	}

	return apiGrant
}

func ParseAPICreateInputV2(inp *apiv1models.EntitlementV2CreateInputs, ns string, usageAttribution streaming.CustomerUsageAttribution) (entitlement.CreateEntitlementInputs, []entitlement.CreateEntitlementGrantInputs, error) {
	entCreateInp := entitlement.CreateEntitlementInputs{}
	grantsInp := []entitlement.CreateEntitlementGrantInputs{}
	if inp == nil {
		return entCreateInp, grantsInp, errors.New("input is nil")
	}

	value, err := inp.ValueByDiscriminator()
	if err != nil {
		return entCreateInp, grantsInp, err
	}

	switch v := value.(type) {
	case apiv1models.EntitlementMeteredV2CreateInputs:
		iv, err := entitlementdriver.MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
		if err != nil {
			return entCreateInp, grantsInp, fmt.Errorf("failed to map interval: %w", err)
		}

		entCreateInp = entitlement.CreateEntitlementInputs{
			Namespace:        ns,
			FeatureID:        v.FeatureId,
			FeatureKey:       v.FeatureKey,
			UsageAttribution: usageAttribution,
			EntitlementType:  entitlement.EntitlementTypeMetered,
			IsSoftLimit:      v.IsSoftLimit,
			// IssueAfterReset:         v.IssueAfterReset,
			// IssueAfterResetPriority: v.IssueAfterResetPriority,
			UsagePeriod: lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
				return defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now())
			})(timeutil.Recurrence{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			})),
			PreserveOverageAtReset: v.PreserveOverageAtReset,
		}

		// Let's handle the default grants
		{
			issueAmount := v.IssueAfterReset
			issuePriority := v.IssueAfterResetPriority

			if v.Issue != nil {
				issueAmount = &v.Issue.Amount
				issuePriority = v.Issue.Priority
			}

			switch {
			case issueAmount != nil && len(lo.FromPtr(v.Grants)) != 0:
				return entCreateInp, grantsInp, models.NewGenericValidationError(
					errors.New("issueAfterReset and grants cannot be used together"),
				)
			case issueAmount != nil:
				entCreateInp.IssueAfterReset = issueAmount
				entCreateInp.IssueAfterResetPriority = issuePriority
			case len(lo.FromPtr(v.Grants)) != 0:
				gs, err := slicesx.MapWithErr(lo.FromPtr(v.Grants), func(g apiv1models.EntitlementGrantCreateInputV2) (meteredentitlement.CreateEntitlementGrantInputs, error) {
					return MapAPIGrantV2ToCreateGrantInput(g)
				})
				if err != nil {
					return entCreateInp, grantsInp, err
				}

				grantsInp = gs
			}
		}

		if v.Metadata != nil {
			entCreateInp.Metadata = *v.Metadata
		}

		if v.MeasureUsageFrom != nil {
			measureUsageFrom := &entitlement.MeasureUsageFromInput{}
			apiTime, err := v.MeasureUsageFrom.AsMeasureUsageFromTime()
			if err == nil {
				err := measureUsageFrom.FromTime(apiTime)
				if err != nil {
					return entCreateInp, grantsInp, err
				}
			} else {
				apiEnum, err := v.MeasureUsageFrom.AsMeasureUsageFromPreset()
				if err != nil {
					return entCreateInp, grantsInp, err
				}

				// sanity check
				if entCreateInp.UsagePeriod == nil {
					return entCreateInp, grantsInp, errors.New("usage period is required for enum measure usage from")
				}

				cPer, err := entCreateInp.UsagePeriod.GetValue().GetPeriodAt(clock.Now())
				if err != nil {
					return entCreateInp, grantsInp, err
				}

				err = measureUsageFrom.FromEnum(entitlement.MeasureUsageFromEnum(apiEnum), cPer, clock.Now())
				if err != nil {
					return entCreateInp, grantsInp, err
				}
			}

			entCreateInp.MeasureUsageFrom = measureUsageFrom
		}
	case apiv1models.EntitlementStaticCreateInputs:
		entCreateInp = entitlement.CreateEntitlementInputs{
			Namespace:        ns,
			FeatureID:        v.FeatureId,
			FeatureKey:       v.FeatureKey,
			UsageAttribution: usageAttribution,
			EntitlementType:  entitlement.EntitlementTypeStatic,
		}

		if len(v.Config) > 0 {
			var config string

			err = json.Unmarshal(v.Config, &config)
			if err != nil {
				return entCreateInp, grantsInp, fmt.Errorf("failed to unmarshal static entitlement config: %w", err)
			}

			entCreateInp.Config = lo.ToPtr(config)
		}

		if v.UsagePeriod != nil {
			iv, err := entitlementdriver.MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
			if err != nil {
				return entCreateInp, grantsInp, fmt.Errorf("failed to map interval: %w", err)
			}

			entCreateInp.UsagePeriod = lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
				return defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now())
			})(timeutil.Recurrence{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			}))
		}

		if v.Metadata != nil {
			entCreateInp.Metadata = *v.Metadata
		}
	case apiv1models.EntitlementBooleanCreateInputs:
		entCreateInp = entitlement.CreateEntitlementInputs{
			Namespace:        ns,
			FeatureID:        v.FeatureId,
			FeatureKey:       v.FeatureKey,
			UsageAttribution: usageAttribution,
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}
		if v.UsagePeriod != nil {
			iv, err := entitlementdriver.MapAPIPeriodIntervalToRecurrence(v.UsagePeriod.Interval)
			if err != nil {
				return entCreateInp, grantsInp, fmt.Errorf("failed to map interval: %w", err)
			}

			entCreateInp.UsagePeriod = lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
				return defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now())
			})(timeutil.Recurrence{
				Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
				Interval: iv,
			}))
		}

		if v.Metadata != nil {
			entCreateInp.Metadata = *v.Metadata
		}
	default:
		return entCreateInp, grantsInp, errors.New("unknown entitlement type")
	}

	// We prune activity data explicitly
	entCreateInp.ActiveFrom = nil
	entCreateInp.ActiveTo = nil

	return entCreateInp, grantsInp, nil
}

func MapAPIGrantV2ToCreateGrantInput(g apiv1models.EntitlementGrantCreateInputV2) (meteredentitlement.CreateEntitlementGrantInputs, error) {
	grantInput := meteredentitlement.CreateEntitlementGrantInputs{
		CreateGrantInput: credit.CreateGrantInput{
			Amount:           g.Amount,
			Priority:         defaultx.WithDefault(g.Priority, 0),
			EffectiveAt:      g.EffectiveAt,
			ResetMaxRollover: defaultx.WithDefault(g.MaxRolloverAmount, g.Amount),
			ResetMinRollover: defaultx.WithDefault(g.MinRolloverAmount, 0),
		},
	}

	if g.Expiration != nil {
		grantInput.Expiration = &grant.ExpirationPeriod{
			Count:    g.Expiration.Count,
			Duration: grant.ExpirationPeriodDuration(g.Expiration.Duration),
		}
	}

	if g.Annotations != nil && len(lo.FromPtr(g.Annotations)) > 0 {
		grantInput.Annotations = make(models.Annotations)

		for k, v := range lo.FromPtr(g.Annotations) {
			grantInput.Annotations[k] = v
		}
	}

	if g.Metadata != nil && len(lo.FromPtr(g.Metadata)) > 0 {
		grantInput.Metadata = make(map[string]string)
		for k, v := range lo.FromPtr(g.Metadata) {
			grantInput.Metadata[k] = v
		}
	}

	if g.Recurrence != nil {
		iv, err := entitlementdriver.MapAPIPeriodIntervalToRecurrence(g.Recurrence.Interval)
		if err != nil {
			return grantInput, err
		}

		grantInput.Recurrence = &timeutil.Recurrence{
			Interval: iv,
			Anchor:   defaultx.WithDefault(g.Recurrence.Anchor, g.EffectiveAt),
		}
	}

	return grantInput, nil
}
