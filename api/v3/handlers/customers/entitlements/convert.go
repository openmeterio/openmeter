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

	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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

func fromAPICreateEntitlementRequest(customerID customer.CustomerID, body apiv3models.CreateEntitlementRequest, now time.Time) (entitlement.CreateCustomerEntitlementInput, error) {
	input := entitlement.CreateCustomerEntitlementInput{
		CustomerID: customerID,
	}

	discriminator, err := body.Discriminator()
	if err != nil {
		return input, fmt.Errorf("invalid entitlement type: %w", err)
	}

	switch discriminator {
	case string(apiv3models.CreateEntitlementMeteredRequestTypeMetered):
		metered, err := body.AsCreateEntitlementMeteredRequest()
		if err != nil {
			return input, fmt.Errorf("invalid metered entitlement: %w", err)
		}

		input.Entitlement, input.Grants, err = fromAPICreateEntitlementMeteredRequest(metered, now)

		return input, err
	case string(apiv3models.CreateEntitlementStaticRequestTypeStatic):
		static, err := body.AsCreateEntitlementStaticRequest()
		if err != nil {
			return input, fmt.Errorf("invalid static entitlement: %w", err)
		}

		input.Entitlement, err = fromAPICreateEntitlementStaticRequest(static, now)

		return input, err
	case string(apiv3models.CreateEntitlementBooleanRequestTypeBoolean):
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

func fromAPICreateEntitlementMeteredRequest(r apiv3models.CreateEntitlementMeteredRequest, now time.Time) (entitlement.CreateEntitlementInputs, []entitlement.CreateEntitlementGrantInputs, error) {
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

func fromAPICreateEntitlementStaticRequest(r apiv3models.CreateEntitlementStaticRequest, now time.Time) (entitlement.CreateEntitlementInputs, error) {
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

func fromAPIEntitlementBase(entitlementType entitlement.EntitlementType, feature apiv3models.FeatureReference, apiLabels *apiv3models.Labels, usagePeriod *apiv3models.RecurringPeriodInput, now time.Time) (entitlement.CreateEntitlementInputs, error) {
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

func fromAPIRecurringPeriodInput(p apiv3models.RecurringPeriodInput, defaultAnchor time.Time) (timeutil.Recurrence, error) {
	interval, err := datetime.ISODurationString(p.Interval).Parse()
	if err != nil {
		return timeutil.Recurrence{}, fmt.Errorf("invalid interval: %w", err)
	}

	return timeutil.NewRecurrenceFromISODuration(interval, lo.FromPtrOr(p.Anchor, defaultAnchor))
}

// fromAPIMeasureUsageFrom resolves the preset variant against the usage period
// the entitlement is being created with, mirroring the v2 API semantics.
func fromAPIMeasureUsageFrom(m apiv3models.BillingEntitlementMeasureUsageFrom, usagePeriod timeutil.Recurrence, now time.Time) (entitlement.MeasureUsageFromInput, error) {
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
	case apiv3models.BillingEntitlementMeasureUsageFromPresetCurrentPeriodStart:
		value = entitlement.MeasureUsageFromCurrentPeriodStart
	case apiv3models.BillingEntitlementMeasureUsageFromPresetNow:
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

func fromAPIEntitlementGrantCreateRequest(g apiv3models.BillingEntitlementGrantCreateRequest) (entitlement.CreateEntitlementGrantInputs, error) {
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

func parseNumeric(v apiv3models.Numeric) (float64, error) {
	d, err := alpacadecimal.NewFromString(v)
	if err != nil {
		return 0, err
	}

	return d.InexactFloat64(), nil
}

func ToAPIBillingEntitlement(e *entitlement.Entitlement) (apiv3models.BillingEntitlement, error) {
	var out apiv3models.BillingEntitlement

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

func toAPIEntitlementMetered(m *meteredentitlement.Entitlement) apiv3models.BillingEntitlementMetered {
	out := apiv3models.BillingEntitlementMetered{
		Type:                   apiv3models.BillingEntitlementMeteredTypeMetered,
		Id:                     m.ID,
		Feature:                apiv3models.FeatureReference{Id: m.FeatureID},
		Customer:               apiv3models.CustomerReference{Id: m.CustomerID},
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

		out.Issue = &apiv3models.BillingEntitlementIssueAfterReset{
			Amount:   amount,
			Priority: m.IssueAfterReset.Priority,
		}
		out.IssueAfterReset = &amount
		out.IssueAfterResetPriority = m.IssueAfterReset.Priority
	}

	return out
}

func toAPIEntitlementStatic(s *staticentitlement.Entitlement) apiv3models.BillingEntitlementStatic {
	out := apiv3models.BillingEntitlementStatic{
		Type:       apiv3models.BillingEntitlementStaticTypeStatic,
		Id:         s.ID,
		Feature:    apiv3models.FeatureReference{Id: s.FeatureID},
		Customer:   apiv3models.CustomerReference{Id: s.CustomerID},
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

func toAPIEntitlementBoolean(b *booleanentitlement.Entitlement) apiv3models.BillingEntitlementBoolean {
	out := apiv3models.BillingEntitlementBoolean{
		Type:       apiv3models.BillingEntitlementBooleanTypeBoolean,
		Id:         b.ID,
		Feature:    apiv3models.FeatureReference{Id: b.FeatureID},
		Customer:   apiv3models.CustomerReference{Id: b.CustomerID},
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

func toAPIRecurringPeriod(u *entitlement.UsagePeriod) apiv3models.RecurringPeriod {
	original := u.GetOriginalValueAsUsagePeriodInput().GetValue()

	return apiv3models.RecurringPeriod{
		Anchor:   original.Anchor,
		Interval: original.Interval.ISOString().String(),
	}
}

func toAPIClosedPeriod(p timeutil.ClosedPeriod) apiv3models.ClosedPeriod {
	return apiv3models.ClosedPeriod{
		From: p.From,
		To:   p.To,
	}
}

func mapHistoryWindowSize(size apiv3models.BillingEntitlementHistoryWindowSize) (meter.WindowSize, error) {
	switch size {
	case apiv3models.BillingEntitlementHistoryWindowSizePT1H:
		return meter.WindowSizeHour, nil
	case apiv3models.BillingEntitlementHistoryWindowSizeP1D:
		return meter.WindowSizeDay, nil
	default:
		return "", fmt.Errorf("unsupported window size %q", size)
	}
}

func mapHistoryToAPI(history entitlement.CustomerEntitlementHistory) apiv3models.BillingEntitlementHistory {
	return apiv3models.BillingEntitlementHistory{
		WindowedHistory: lo.Map(history.Windows, func(window entitlement.BalanceHistoryWindow, _ int) apiv3models.BillingEntitlementHistoryWindow {
			return apiv3models.BillingEntitlementHistoryWindow{
				Period:         mapPeriodToAPI(window.From, window.To),
				Usage:          alpacadecimal.NewFromFloat(window.UsageInPeriod).String(),
				BalanceAtStart: alpacadecimal.NewFromFloat(window.BalanceAtStart).String(),
			}
		}),
		BurndownHistory: lo.Map(history.Burndown.Segments(), func(segment engine.GrantBurnDownHistorySegment, _ int) apiv3models.BillingEntitlementBurndownSegment {
			return mapBurndownSegmentToAPI(segment)
		}),
	}
}

func mapBurndownSegmentToAPI(segment engine.GrantBurnDownHistorySegment) apiv3models.BillingEntitlementBurndownSegment {
	balancesAtEnd := segment.ApplyUsage()
	numeric := func(balance float64, _ string) apiv3models.Numeric {
		return alpacadecimal.NewFromFloat(balance).String()
	}

	return apiv3models.BillingEntitlementBurndownSegment{
		Period:  mapPeriodToAPI(segment.From, segment.To),
		Usage:   alpacadecimal.NewFromFloat(segment.TotalUsage).String(),
		Overage: alpacadecimal.NewFromFloat(segment.Overage).String(),
		Balance: apiv3models.BillingEntitlementBurndownBalance{
			Start: alpacadecimal.NewFromFloat(segment.BalanceAtStart.Balance()).String(),
			End:   alpacadecimal.NewFromFloat(balancesAtEnd.Balance()).String(),
		},
		GrantBalances: apiv3models.BillingEntitlementBurndownGrantBalances{
			Start: lo.MapValues(segment.BalanceAtStart, numeric),
			End:   lo.MapValues(balancesAtEnd, numeric),
		},
		GrantUsages: lo.Map(segment.GrantUsages, func(usage engine.GrantUsage, _ int) apiv3models.BillingEntitlementGrantUsage {
			return apiv3models.BillingEntitlementGrantUsage{
				GrantId: usage.GrantID,
				Usage:   alpacadecimal.NewFromFloat(usage.Usage).String(),
			}
		}),
	}
}

// mapPeriodToAPI renders period bounds in UTC as AIP-142 requires, regardless of
// the location the calculation ran in.
func mapPeriodToAPI(from, to time.Time) apiv3models.ClosedPeriod {
	return apiv3models.ClosedPeriod{From: from.UTC(), To: to.UTC()}
}
