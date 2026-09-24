package customersentitlements

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// entitlementGrantSortFields are the grant order by values exposed on the API. The
// owner is fixed by the path and the ID carries no meaning to sort by.
var entitlementGrantSortFields = []grant.OrderBy{
	grant.OrderByCreatedAt,
	grant.OrderByUpdatedAt,
	grant.OrderByEffectiveAt,
	grant.OrderByExpiresAt,
}

func fromAPIEntitlementGrantSortField(ctx context.Context, field string) (grant.OrderBy, error) {
	orderBy := grant.OrderBy(field)
	if !slices.Contains(entitlementGrantSortFields, orderBy) {
		supported := lo.Map(entitlementGrantSortFields, func(f grant.OrderBy, _ int) string { return string(f) })

		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, supported...)
	}

	return orderBy, nil
}

// ToAPIEntitlementGrant maps a grant owned by an entitlement. The next recurrence
// is calculated at now, the same way the legacy grant listing reports it.
func ToAPIEntitlementGrant(g grant.Grant, now time.Time) (api.BillingEntitlementGrant, error) {
	out := api.BillingEntitlementGrant{
		Id:                g.ID,
		EntitlementId:     g.OwnerID,
		Amount:            alpacadecimal.NewFromFloat(g.Amount).String(),
		Priority:          g.Priority,
		EffectiveAt:       g.EffectiveAt,
		ExpiresAt:         g.ExpiresAt,
		MaxRolloverAmount: alpacadecimal.NewFromFloat(g.ResetMaxRollover).String(),
		MinRolloverAmount: alpacadecimal.NewFromFloat(g.ResetMinRollover).String(),
		VoidedAt:          g.VoidedAt,
		Labels:            labels.FromMetadataAnnotations(g.Metadata, g.Annotations),
		CreatedAt:         g.CreatedAt,
		UpdatedAt:         g.UpdatedAt,
		DeletedAt:         g.DeletedAt,
	}

	if g.Expiration != nil {
		expiresAfter, err := toAPIEntitlementGrantExpiresAfter(*g.Expiration)
		if err != nil {
			return out, err
		}

		out.ExpiresAfter = &expiresAfter
	}

	if g.Recurrence != nil {
		out.Recurrence = &api.RecurringPeriod{
			Anchor:   g.Recurrence.Anchor,
			Interval: g.Recurrence.Interval.ISOString().String(),
		}

		next, err := g.Recurrence.NextAfter(now, timeutil.Inclusive)
		if err != nil {
			return out, fmt.Errorf("calculating next recurrence: %w", err)
		}

		out.NextRecurrence = &next
	}

	return out, nil
}

// toAPIEntitlementGrantExpiresAfter renders the count and unit pair the grant
// domain stores as the single-unit ISO 8601 duration the API exposes.
func toAPIEntitlementGrantExpiresAfter(p grant.ExpirationPeriod) (string, error) {
	count := int(p.Count)

	var d datetime.ISODuration

	switch p.Duration {
	case grant.ExpirationPeriodDurationHour:
		d = datetime.NewISODuration(0, 0, 0, 0, count, 0, 0)
	case grant.ExpirationPeriodDurationDay:
		d = datetime.NewISODuration(0, 0, 0, count, 0, 0, 0)
	case grant.ExpirationPeriodDurationWeek:
		d = datetime.NewISODuration(0, 0, count, 0, 0, 0, 0)
	case grant.ExpirationPeriodDurationMonth:
		d = datetime.NewISODuration(0, count, 0, 0, 0, 0, 0)
	case grant.ExpirationPeriodDurationYear:
		d = datetime.NewISODuration(count, 0, 0, 0, 0, 0, 0)
	default:
		return "", fmt.Errorf("unsupported expiration duration: %s", p.Duration)
	}

	return d.String(), nil
}
