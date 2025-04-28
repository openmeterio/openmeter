package internal

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

var (
	now       = time.Now()
	createdAt = now.Add(-24 * time.Hour)
	updatedAt = now.Add(-12 * time.Hour)
	from      = now.Add(-24 * time.Hour)
	to        = now.Add(-12 * time.Hour)
)

func NewTestEventPayload(eventType notification.EventType) notification.EventPayload {
	day := &api.RecurringPeriodInterval{}
	_ = day.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumDAY)
	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: eventType,
		},
		BalanceThreshold: notification.BalanceThresholdPayload{
			Entitlement: api.EntitlementMetered{
				CreatedAt: createdAt,
				CurrentUsagePeriod: api.Period{
					From: from,
					To:   to,
				},
				DeletedAt:               nil,
				FeatureId:               "01J5AVN2T6S0RDGJHVNN0BW3F5",
				FeatureKey:              "test-feature-1",
				Id:                      "01J5AVNM7H1PT65RDFWGXXPT47",
				IsSoftLimit:             lo.ToPtr(false),
				IsUnlimited:             lo.ToPtr(true),
				IssueAfterReset:         lo.ToPtr(10.0),
				IssueAfterResetPriority: lo.ToPtr[uint8](5),
				LastReset:               time.Time{},
				MeasureUsageFrom:        time.Time{},
				Metadata: &map[string]string{
					"test-metadata-key": "test-metadata-value",
				},
				PreserveOverageAtReset: lo.ToPtr(true),
				SubjectKey:             "test-subject-1",
				Type:                   api.EntitlementMeteredTypeMetered,
				UpdatedAt:              updatedAt,
				UsagePeriod: api.RecurringPeriod{
					Anchor:      from,
					Interval:    *day,
					IntervalISO: "P1D",
				},
			},
			Feature: api.Feature{
				ArchivedAt: nil,
				CreatedAt:  createdAt,
				DeletedAt:  nil,
				Id:         "01J5AVN2T6S0RDGJHVNN0BW3F5",
				Key:        "test-feature-1",
				Metadata: &map[string]string{
					"test-metadata-key": "test-metadata-value",
				},
				MeterGroupByFilters: nil,
				MeterSlug:           lo.ToPtr("test-meter-1"),
				Name:                "test-meter-1",
				UpdatedAt:           updatedAt,
			},
			Subject: api.Subject{
				CurrentPeriodEnd:   lo.ToPtr(from),
				CurrentPeriodStart: lo.ToPtr(to),
				DisplayName:        lo.ToPtr("Test Subject 1"),
				Id:                 "01J5AW0ZD6T8624PCK0Q5TYX71",
				Key:                "test-subject-1",
				Metadata: &map[string]interface{}{
					"test-metadata-key": "test-metadata-value",
				},
				StripeCustomerId: lo.ToPtr("01J5AW2XS6DYHH7E9PNJSQJ341"),
			},
			Threshold: api.NotificationRuleBalanceThresholdValue{
				Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
				Value: 50,
			},
			Value: api.EntitlementValue{
				Balance:   lo.ToPtr(10_000.0),
				HasAccess: true,
				Overage:   lo.ToPtr(99.0),
				Usage:     lo.ToPtr(5_001.0),
			},
		},
	}
}
