package internal

import (
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/convert"
)

var (
	now       = time.Now()
	createdAt = now.Add(-24 * time.Hour)
	updatedAt = now.Add(-12 * time.Hour)
	from      = now.Add(-24 * time.Hour)
	to        = now.Add(-12 * time.Hour)
)

func NewTestEventPayload(eventType notification.EventType) notification.EventPayload {
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
				IsSoftLimit:             convert.ToPointer(false),
				IsUnlimited:             convert.ToPointer(true),
				IssueAfterReset:         convert.ToPointer(10.0),
				IssueAfterResetPriority: convert.ToPointer[uint8](5),
				LastReset:               time.Time{},
				MeasureUsageFrom:        time.Time{},
				Metadata: &map[string]string{
					"test-metadata-key": "test-metadata-value",
				},
				PreserveOverageAtReset: convert.ToPointer(true),
				SubjectKey:             "test-subject-1",
				Type:                   api.EntitlementMeteredTypeMetered,
				UpdatedAt:              updatedAt,
				UsagePeriod: &api.RecurringPeriod{
					Anchor:   from,
					Interval: api.RecurringPeriodIntervalDAY,
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
				MeterSlug:           convert.ToPointer("test-meter-1"),
				Name:                "test-meter-1",
				UpdatedAt:           updatedAt,
			},
			Subject: api.Subject{
				CurrentPeriodEnd:   convert.ToPointer(from),
				CurrentPeriodStart: convert.ToPointer(to),
				DisplayName:        convert.ToPointer("Test Subject 1"),
				Id:                 "01J5AW0ZD6T8624PCK0Q5TYX71",
				Key:                "test-subject-1",
				Metadata: &map[string]interface{}{
					"test-metadata-key": "test-metadata-value",
				},
				StripeCustomerId: convert.ToPointer("01J5AW2XS6DYHH7E9PNJSQJ341"),
			},
			Threshold: api.NotificationRuleBalanceThresholdValue{
				Type:  api.PERCENT,
				Value: 50,
			},
			Value: api.EntitlementValue{
				Balance:   convert.ToPointer(10_000.0),
				HasAccess: true,
				Overage:   convert.ToPointer(99.0),
				Usage:     convert.ToPointer(5_001.0),
			},
		},
	}
}
