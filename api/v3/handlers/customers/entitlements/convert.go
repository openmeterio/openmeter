package customersentitlements

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func fromAPICreateEntitlementRequest(customerID customer.CustomerID, body api.CreateEntitlementRequest, now time.Time) (entitlement.CreateCustomerEntitlementInput, error) {
	input := entitlement.CreateCustomerEntitlementInput{
		CustomerID: customerID,
	}

	discriminator, err := body.Discriminator()
	if err != nil {
		return input, fmt.Errorf("invalid entitlement type: %w", err)
	}

	switch discriminator {
	case string(api.CreateEntitlementMeteredRequestTypeMetered):
		metered, err := body.AsCreateEntitlementMeteredRequest()
		if err != nil {
			return input, fmt.Errorf("invalid metered entitlement: %w", err)
		}

		input.Entitlement, input.Grants, err = fromAPICreateEntitlementMeteredRequest(metered, now)

		return input, err
	case string(api.CreateEntitlementStaticRequestTypeStatic):
		static, err := body.AsCreateEntitlementStaticRequest()
		if err != nil {
			return input, fmt.Errorf("invalid static entitlement: %w", err)
		}

		input.Entitlement, err = fromAPICreateEntitlementStaticRequest(static, now)

		return input, err
	case string(api.CreateEntitlementBooleanRequestTypeBoolean):
		boolean, err := body.AsCreateEntitlementBooleanRequest()
		if err != nil {
			return input, fmt.Errorf("invalid boolean entitlement: %w", err)
		}

		input.Entitlement, err = fromAPIEntitlementBase(entitlement.EntitlementTypeBoolean, boolean.Feature, boolean.Labels, boolean.UsagePeriod, now)

		return input, err
	default:
		return input, fmt.Errorf("unsupported entitlement type: %s", discriminator)
	}
}

func fromAPICreateEntitlementMeteredRequest(r api.CreateEntitlementMeteredRequest, now time.Time) (entitlement.CreateEntitlementInputs, []entitlement.CreateEntitlementGrantInputs, error) {
	input, err := fromAPIEntitlementBase(entitlement.EntitlementTypeMetered, r.Feature, r.Labels, &r.UsagePeriod, now)
	if err != nil {
		return input, nil, err
	}

	input.IsSoftLimit = r.IsSoftLimit
	input.PreserveOverageAtReset = r.PreserveOverageAtReset

	// The flat fields are the deprecated v2 shape; the issue object wins when both are sent.
	issueAmount := r.IssueAfterReset
	issuePriority := r.IssueAfterResetPriority
	if r.Issue != nil {
		issueAmount = &r.Issue.Amount
		issuePriority = r.Issue.Priority
	}

	if issueAmount != nil {
		amount, err := parseNumeric(*issueAmount)
		if err != nil {
			return input, nil, fmt.Errorf("invalid issue amount: %w", err)
		}

		input.IssueAfterReset = &amount
		input.IssueAfterResetPriority = issuePriority
	}

	if r.MeasureUsageFrom != nil {
		measureUsageFrom, err := fromAPIMeasureUsageFrom(*r.MeasureUsageFrom, input.UsagePeriod.GetValue(), now)
		if err != nil {
			return input, nil, fmt.Errorf("invalid measure usage from: %w", err)
		}

		input.MeasureUsageFrom = &measureUsageFrom
	}

	grants, err := lo.MapErr(lo.FromPtr(r.Grants), slicesx.WrapMapFn(fromAPIEntitlementGrantCreateRequest))
	if err != nil {
		return input, nil, fmt.Errorf("invalid grants: %w", err)
	}

	return input, grants, nil
}

func fromAPICreateEntitlementStaticRequest(r api.CreateEntitlementStaticRequest, now time.Time) (entitlement.CreateEntitlementInputs, error) {
	input, err := fromAPIEntitlementBase(entitlement.EntitlementTypeStatic, r.Feature, r.Labels, r.UsagePeriod, now)
	if err != nil {
		return input, err
	}

	// The domain stores the config as JSON text; a missing or null config is left
	// unset so the domain reports it.
	if len(r.Config) > 0 && !bytes.Equal(r.Config, []byte("null")) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, r.Config); err != nil {
			return input, fmt.Errorf("invalid config: %w", err)
		}

		input.Config = lo.ToPtr(compact.String())
	}

	return input, nil
}

func fromAPIEntitlementBase(entitlementType entitlement.EntitlementType, feature api.FeatureReference, apiLabels *api.Labels, usagePeriod *api.RecurringPeriodInput, now time.Time) (entitlement.CreateEntitlementInputs, error) {
	metadata, err := labels.ToMetadata(apiLabels)
	if err != nil {
		return entitlement.CreateEntitlementInputs{}, err
	}

	input := entitlement.CreateEntitlementInputs{
		FeatureID:       lo.EmptyableToPtr(feature.Id),
		EntitlementType: entitlementType,
		Metadata:        metadata,
	}

	if usagePeriod != nil {
		recurrence, err := fromAPIRecurringPeriodInput(*usagePeriod, now)
		if err != nil {
			return input, fmt.Errorf("invalid usage period: %w", err)
		}

		input.UsagePeriod = lo.ToPtr(timeutil.AsTimed(func(r timeutil.Recurrence) time.Time {
			return r.Anchor
		})(recurrence))
	}

	return input, nil
}

func fromAPIRecurringPeriodInput(p api.RecurringPeriodInput, defaultAnchor time.Time) (timeutil.Recurrence, error) {
	interval, err := datetime.ISODurationString(p.Interval).Parse()
	if err != nil {
		return timeutil.Recurrence{}, fmt.Errorf("invalid interval: %w", err)
	}

	return timeutil.NewRecurrenceFromISODuration(interval, lo.FromPtrOr(p.Anchor, defaultAnchor))
}

// fromAPIMeasureUsageFrom resolves the preset variant against the usage period
// the entitlement is being created with, mirroring the v2 API semantics.
func fromAPIMeasureUsageFrom(m api.BillingEntitlementMeasureUsageFrom, usagePeriod timeutil.Recurrence, now time.Time) (entitlement.MeasureUsageFromInput, error) {
	var out entitlement.MeasureUsageFromInput

	if at, err := m.AsDateTime(); err == nil {
		return out, out.FromTime(at)
	}

	preset, err := m.AsBillingEntitlementMeasureUsageFromPreset()
	if err != nil {
		return out, err
	}

	var value entitlement.MeasureUsageFromEnum

	switch preset {
	case api.BillingEntitlementMeasureUsageFromPresetCurrentPeriodStart:
		value = entitlement.MeasureUsageFromCurrentPeriodStart
	case api.BillingEntitlementMeasureUsageFromPresetNow:
		value = entitlement.MeasureUsageFromNow
	default:
		return out, fmt.Errorf("unsupported preset: %s", preset)
	}

	currentPeriod, err := usagePeriod.GetPeriodAt(now)
	if err != nil {
		return out, err
	}

	return out, out.FromEnum(value, currentPeriod, now)
}

func fromAPIEntitlementGrantCreateRequest(g api.BillingEntitlementGrantCreateRequest) (entitlement.CreateEntitlementGrantInputs, error) {
	amount, err := parseNumeric(g.Amount)
	if err != nil {
		return entitlement.CreateEntitlementGrantInputs{}, fmt.Errorf("invalid amount: %w", err)
	}

	maxRollover := amount
	if g.MaxRolloverAmount != nil {
		maxRollover, err = parseNumeric(*g.MaxRolloverAmount)
		if err != nil {
			return entitlement.CreateEntitlementGrantInputs{}, fmt.Errorf("invalid max rollover amount: %w", err)
		}
	}

	var minRollover float64
	if g.MinRolloverAmount != nil {
		minRollover, err = parseNumeric(*g.MinRolloverAmount)
		if err != nil {
			return entitlement.CreateEntitlementGrantInputs{}, fmt.Errorf("invalid min rollover amount: %w", err)
		}
	}

	metadata, err := labels.ToMetadata(g.Labels)
	if err != nil {
		return entitlement.CreateEntitlementGrantInputs{}, err
	}

	input := entitlement.CreateEntitlementGrantInputs{
		CreateGrantInput: credit.CreateGrantInput{
			Amount:           amount,
			Priority:         lo.FromPtr(g.Priority),
			EffectiveAt:      g.EffectiveAt,
			ResetMaxRollover: maxRollover,
			ResetMinRollover: minRollover,
			Metadata:         metadata,
		},
	}

	if g.ExpiresAfter != nil {
		duration, err := datetime.ISODurationString(*g.ExpiresAfter).Parse()
		if err != nil {
			return input, fmt.Errorf("invalid expires after: %w", err)
		}

		expiration, err := expirationPeriodFromISODuration(duration)
		if err != nil {
			return input, fmt.Errorf("invalid expires after: %w", err)
		}

		input.Expiration = &expiration
	}

	if g.Recurrence != nil {
		recurrence, err := fromAPIRecurringPeriodInput(*g.Recurrence, g.EffectiveAt)
		if err != nil {
			return input, fmt.Errorf("invalid recurrence: %w", err)
		}

		input.Recurrence = &recurrence
	}

	return input, nil
}

// expirationPeriodFromISODuration maps a single-unit ISO 8601 duration onto the
// count and unit pair the grant domain stores. Mixed or fractional durations have
// no such representation and are rejected.
func expirationPeriodFromISODuration(d datetime.ISODuration) (grant.ExpirationPeriod, error) {
	candidates := []struct {
		count    int
		duration grant.ExpirationPeriodDuration
		iso      datetime.ISODuration
	}{
		{d.Years(), grant.ExpirationPeriodDurationYear, datetime.NewISODuration(d.Years(), 0, 0, 0, 0, 0, 0)},
		{d.Months(), grant.ExpirationPeriodDurationMonth, datetime.NewISODuration(0, d.Months(), 0, 0, 0, 0, 0)},
		{d.Weeks(), grant.ExpirationPeriodDurationWeek, datetime.NewISODuration(0, 0, d.Weeks(), 0, 0, 0, 0)},
		{d.Days(), grant.ExpirationPeriodDurationDay, datetime.NewISODuration(0, 0, 0, d.Days(), 0, 0, 0)},
		{d.Hours(), grant.ExpirationPeriodDurationHour, datetime.NewISODuration(0, 0, 0, 0, d.Hours(), 0, 0)},
	}

	for _, candidate := range candidates {
		if candidate.count > 0 && candidate.iso.String() == d.String() {
			if candidate.count > math.MaxUint32 {
				return grant.ExpirationPeriod{}, fmt.Errorf("expiration count %d exceeds the supported maximum of %d", candidate.count, math.MaxUint32)
			}

			return grant.ExpirationPeriod{
				Count:    uint32(candidate.count),
				Duration: candidate.duration,
			}, nil
		}
	}

	return grant.ExpirationPeriod{}, fmt.Errorf("expiration must be a whole number of hours, days, weeks, months or years, got %s", d.String())
}

func parseNumeric(v api.Numeric) (float64, error) {
	d, err := alpacadecimal.NewFromString(v)
	if err != nil {
		return 0, err
	}

	return d.InexactFloat64(), nil
}

func toAPIEntitlement(e *entitlement.Entitlement) (api.BillingEntitlement, error) {
	var out api.BillingEntitlement

	if e == nil {
		return out, errors.New("entitlement is nil")
	}

	switch e.EntitlementType {
	case entitlement.EntitlementTypeMetered:
		metered, err := meteredentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return out, err
		}

		return out, out.FromBillingEntitlementMetered(toAPIEntitlementMetered(metered))
	case entitlement.EntitlementTypeStatic:
		static, err := staticentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return out, err
		}

		return out, out.FromBillingEntitlementStatic(toAPIEntitlementStatic(static))
	case entitlement.EntitlementTypeBoolean:
		boolean, err := booleanentitlement.ParseFromGenericEntitlement(e)
		if err != nil {
			return out, err
		}

		return out, out.FromBillingEntitlementBoolean(toAPIEntitlementBoolean(boolean))
	default:
		return out, fmt.Errorf("unsupported entitlement type: %s", e.EntitlementType)
	}
}

func toAPIEntitlementMetered(m *meteredentitlement.Entitlement) api.BillingEntitlementMetered {
	out := api.BillingEntitlementMetered{
		Type:                   api.BillingEntitlementMeteredTypeMetered,
		Id:                     m.ID,
		Feature:                api.FeatureReference{Id: m.FeatureID},
		Customer:               api.CustomerReference{Id: m.CustomerID},
		Labels:                 labels.FromMetadataAnnotations(m.Metadata, m.Annotations),
		ActiveFrom:             m.ActiveFromTime(),
		ActiveTo:               m.ActiveToTime(),
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
		DeletedAt:              m.DeletedAt,
		UsagePeriod:            toAPIRecurringPeriod(&m.UsagePeriod),
		CurrentUsagePeriod:     toAPIClosedPeriod(m.CurrentUsagePeriod),
		IsSoftLimit:            lo.ToPtr(m.IsSoftLimit),
		PreserveOverageAtReset: lo.ToPtr(m.PreserveOverageAtReset),
		MeasureUsageFrom:       m.MeasureUsageFrom,
		LastReset:              m.LastReset,
	}

	// The store keeps a zero amount when no default grant was configured; only a
	// positive amount is a real issue-after-reset setting.
	if m.HasDefaultGrant() {
		amount := alpacadecimal.NewFromFloat(m.IssueAfterReset.Amount).String()

		out.Issue = &api.BillingEntitlementIssueAfterReset{
			Amount:   amount,
			Priority: m.IssueAfterReset.Priority,
		}
		out.IssueAfterReset = &amount
		out.IssueAfterResetPriority = m.IssueAfterReset.Priority
	}

	return out
}

func toAPIEntitlementStatic(s *staticentitlement.Entitlement) api.BillingEntitlementStatic {
	out := api.BillingEntitlementStatic{
		Type:       api.BillingEntitlementStaticTypeStatic,
		Id:         s.ID,
		Feature:    api.FeatureReference{Id: s.FeatureID},
		Customer:   api.CustomerReference{Id: s.CustomerID},
		Labels:     labels.FromMetadataAnnotations(s.Metadata, s.Annotations),
		ActiveFrom: s.ActiveFromTime(),
		ActiveTo:   s.ActiveToTime(),
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
		DeletedAt:  s.DeletedAt,
		Config:     json.RawMessage(s.Config),
	}

	if s.UsagePeriod != nil {
		out.UsagePeriod = lo.ToPtr(toAPIRecurringPeriod(s.UsagePeriod))
	}

	if s.CurrentUsagePeriod != nil {
		out.CurrentUsagePeriod = lo.ToPtr(toAPIClosedPeriod(*s.CurrentUsagePeriod))
	}

	return out
}

func toAPIEntitlementBoolean(b *booleanentitlement.Entitlement) api.BillingEntitlementBoolean {
	out := api.BillingEntitlementBoolean{
		Type:       api.BillingEntitlementBooleanTypeBoolean,
		Id:         b.ID,
		Feature:    api.FeatureReference{Id: b.FeatureID},
		Customer:   api.CustomerReference{Id: b.CustomerID},
		Labels:     labels.FromMetadataAnnotations(b.Metadata, b.Annotations),
		ActiveFrom: b.ActiveFromTime(),
		ActiveTo:   b.ActiveToTime(),
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
		DeletedAt:  b.DeletedAt,
	}

	if b.UsagePeriod != nil {
		out.UsagePeriod = lo.ToPtr(toAPIRecurringPeriod(b.UsagePeriod))
	}

	if b.CurrentUsagePeriod != nil {
		out.CurrentUsagePeriod = lo.ToPtr(toAPIClosedPeriod(*b.CurrentUsagePeriod))
	}

	return out
}

func toAPIRecurringPeriod(u *entitlement.UsagePeriod) api.RecurringPeriod {
	original := u.GetOriginalValueAsUsagePeriodInput().GetValue()

	return api.RecurringPeriod{
		Anchor:   original.Anchor,
		Interval: original.Interval.ISOString().String(),
	}
}

func toAPIClosedPeriod(p timeutil.ClosedPeriod) api.ClosedPeriod {
	return api.ClosedPeriod{
		From: p.From,
		To:   p.To,
	}
}

func mapHistoryWindowSize(size api.BillingEntitlementHistoryWindowSize) (meter.WindowSize, error) {
	switch size {
	case api.BillingEntitlementHistoryWindowSizePT1H:
		return meter.WindowSizeHour, nil
	case api.BillingEntitlementHistoryWindowSizeP1D:
		return meter.WindowSizeDay, nil
	default:
		return "", fmt.Errorf("unsupported window size %q", size)
	}
}

func mapHistoryToAPI(history entitlement.CustomerEntitlementHistory) api.BillingEntitlementHistory {
	return api.BillingEntitlementHistory{
		WindowedHistory: lo.Map(history.Windows, func(window entitlement.BalanceHistoryWindow, _ int) api.BillingEntitlementHistoryWindow {
			return api.BillingEntitlementHistoryWindow{
				Period:         mapPeriodToAPI(window.From, window.To),
				Usage:          alpacadecimal.NewFromFloat(window.UsageInPeriod).String(),
				BalanceAtStart: alpacadecimal.NewFromFloat(window.BalanceAtStart).String(),
			}
		}),
		BurndownHistory: lo.Map(history.Burndown.Segments(), func(segment engine.GrantBurnDownHistorySegment, _ int) api.BillingEntitlementBurndownSegment {
			return mapBurndownSegmentToAPI(segment)
		}),
	}
}

func mapBurndownSegmentToAPI(segment engine.GrantBurnDownHistorySegment) api.BillingEntitlementBurndownSegment {
	balancesAtEnd := segment.ApplyUsage()
	numeric := func(balance float64, _ string) api.Numeric {
		return alpacadecimal.NewFromFloat(balance).String()
	}

	return api.BillingEntitlementBurndownSegment{
		Period:  mapPeriodToAPI(segment.From, segment.To),
		Usage:   alpacadecimal.NewFromFloat(segment.TotalUsage).String(),
		Overage: alpacadecimal.NewFromFloat(segment.Overage).String(),
		Balance: api.BillingEntitlementBurndownBalance{
			Start: alpacadecimal.NewFromFloat(segment.BalanceAtStart.Balance()).String(),
			End:   alpacadecimal.NewFromFloat(balancesAtEnd.Balance()).String(),
		},
		GrantBalances: api.BillingEntitlementBurndownGrantBalances{
			Start: lo.MapValues(segment.BalanceAtStart, numeric),
			End:   lo.MapValues(balancesAtEnd, numeric),
		},
		GrantUsages: lo.Map(segment.GrantUsages, func(usage engine.GrantUsage, _ int) api.BillingEntitlementGrantUsage {
			return api.BillingEntitlementGrantUsage{
				GrantId: usage.GrantID,
				Usage:   alpacadecimal.NewFromFloat(usage.Usage).String(),
			}
		}),
	}
}

// mapPeriodToAPI renders period bounds in UTC as AIP-142 requires, regardless of
// the location the calculation ran in.
func mapPeriodToAPI(from, to time.Time) api.ClosedPeriod {
	return api.ClosedPeriod{From: from.UTC(), To: to.UTC()}
}
