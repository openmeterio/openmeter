package httpdriver

import (
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/internal/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/internal/entitlement/static"
	"github.com/openmeterio/openmeter/pkg/convert"
)

type parser struct{}

var Parser = parser{}

func (parser) ToMetered(e *entitlement.Entitlement) (*api.EntitlementMetered, error) {
	metered, err := meteredentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	return &api.EntitlementMetered{
		CreatedAt:       &metered.CreatedAt,
		DeletedAt:       metered.DeletedAt,
		FeatureId:       metered.FeatureID,
		Id:              &metered.ID,
		IsUnlimited:     convert.ToPointer(false), // implement
		IssueAfterReset: metered.IssuesAfterReset,
		Metadata:        &metered.Metadata,
		SubjectKey:      metered.SubjectKey,
		Type:            api.EntitlementMeteredType(metered.EntitlementType),
		UpdatedAt:       &metered.UpdatedAt,
		UsagePeriod: api.RecurringPeriod{
			Anchor:   metered.UsagePeriod.Anchor,
			Interval: api.RecurringPeriodEnum(metered.UsagePeriod.Interval),
		},
	}, nil
}

func (parser) ToStatic(e *entitlement.Entitlement) (*api.EntitlementStatic, error) {
	static, err := staticentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	apiRes := &api.EntitlementStatic{
		CreatedAt:  &static.CreatedAt,
		DeletedAt:  static.DeletedAt,
		FeatureId:  static.FeatureID,
		Id:         &static.ID,
		Metadata:   &static.Metadata,
		SubjectKey: static.SubjectKey,
		Type:       api.EntitlementStaticType(static.EntitlementType),
		UpdatedAt:  &static.UpdatedAt,
		Config:     static.Config,
	}

	if static.UsagePeriod != nil {
		apiRes.UsagePeriod = &api.RecurringPeriod{
			Anchor:   static.UsagePeriod.Anchor,
			Interval: api.RecurringPeriodEnum(static.UsagePeriod.Interval),
		}
	}

	return apiRes, nil
}

func (parser) ToBoolean(e *entitlement.Entitlement) (*api.EntitlementBoolean, error) {
	boolean, err := booleanentitlement.ParseFromGenericEntitlement(e)
	if err != nil {
		return nil, err
	}

	apiRes := &api.EntitlementBoolean{
		CreatedAt:  &boolean.CreatedAt,
		DeletedAt:  boolean.DeletedAt,
		FeatureId:  boolean.FeatureID,
		Id:         &boolean.ID,
		Metadata:   &boolean.Metadata,
		SubjectKey: boolean.SubjectKey,
		Type:       api.EntitlementBooleanType(boolean.EntitlementType),
		UpdatedAt:  &boolean.UpdatedAt,
	}

	if boolean.UsagePeriod != nil {
		apiRes.UsagePeriod = &api.RecurringPeriod{
			Anchor:   boolean.UsagePeriod.Anchor,
			Interval: api.RecurringPeriodEnum(boolean.UsagePeriod.Interval),
		}
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
