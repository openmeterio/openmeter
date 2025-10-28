package notification

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewBalanceThresholdPayload() notification.EventPayload {
	month := &api.RecurringPeriodInterval{}
	_ = month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH)
	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeBalanceThreshold,
		},
		BalanceThreshold: &notification.BalanceThresholdPayload{
			EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase{
				Entitlement: api.EntitlementMetered{
					CreatedAt: time.Now().Add(-10 * 24 * time.Hour).UTC(),
					CurrentUsagePeriod: api.Period{
						From: time.Now().Add(-24 * time.Hour).UTC(),
						To:   time.Now().UTC(),
					},
					DeletedAt:               nil,
					FeatureId:               "01J4VCZKH5QAF85GE501M8637W",
					FeatureKey:              "feature-1",
					Id:                      "01J4VCTKG06VJ0H78GD0MZBE49",
					IsSoftLimit:             nil,
					IsUnlimited:             nil,
					IssueAfterReset:         nil,
					IssueAfterResetPriority: nil,
					LastReset:               time.Time{},
					MeasureUsageFrom:        time.Time{},
					Metadata:                nil,
					SubjectKey:              "customer-1",
					Type:                    "",
					UpdatedAt:               time.Now().Add(-2 * time.Hour).UTC(),
					UsagePeriod: api.RecurringPeriod{
						Anchor:      time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
						Interval:    *month,
						IntervalISO: "P1M",
					},
				},
				Feature: api.Feature{
					ArchivedAt:          nil,
					CreatedAt:           time.Now().Add(-10 * 24 * time.Hour).UTC(),
					DeletedAt:           nil,
					Id:                  "01J4VCZKH5QAF85GE501M8637W",
					Key:                 "feature-1",
					Metadata:            nil,
					MeterGroupByFilters: nil,
					MeterSlug:           nil,
					Name:                "feature-1",
					UpdatedAt:           time.Now().Add(-24 * time.Hour).UTC(),
				},
				Subject: api.Subject{
					CurrentPeriodEnd:   &time.Time{},
					CurrentPeriodStart: &time.Time{},
					DisplayName:        nil,
					Id:                 "01J4VD1XZH5HM705DCPB8XD5QD",
					Key:                "customer-1",
					Metadata:           nil,
					StripeCustomerId:   nil,
				},
				Value: api.EntitlementValue{
					Balance:   convert.ToPointer(10000.0),
					Config:    nil,
					HasAccess: true,
					Overage:   convert.ToPointer(500.0),
					Usage:     convert.ToPointer(50000.0),
				},
			},
			Threshold: api.NotificationRuleBalanceThresholdValue{
				Type:  notification.BalanceThresholdTypePercent,
				Value: 50,
			},
		},
	}
}

func NewEntitlementResetPayload() notification.EventPayload {
	month := &api.RecurringPeriodInterval{}
	_ = month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH)
	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeEntitlementReset,
		},
		EntitlementReset: &notification.EntitlementResetPayload{
			Entitlement: api.EntitlementMetered{
				CreatedAt: time.Now().Add(-10 * 24 * time.Hour).UTC(),
				CurrentUsagePeriod: api.Period{
					From: time.Now().Add(-24 * time.Hour).UTC(),
					To:   time.Now().UTC(),
				},
				DeletedAt:               nil,
				FeatureId:               "01J4VCZKH5QAF85GE501M8637W",
				FeatureKey:              "feature-1",
				Id:                      "01J4VCTKG06VJ0H78GD0MZBE49",
				IsSoftLimit:             nil,
				IsUnlimited:             nil,
				IssueAfterReset:         nil,
				IssueAfterResetPriority: nil,
				LastReset:               time.Time{},
				MeasureUsageFrom:        time.Time{},
				Metadata:                nil,
				SubjectKey:              "customer-1",
				Type:                    "",
				UpdatedAt:               time.Now().Add(-2 * time.Hour).UTC(),
				UsagePeriod: api.RecurringPeriod{
					Anchor:      time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
					Interval:    *month,
					IntervalISO: "P1M",
				},
			},
			Feature: api.Feature{
				ArchivedAt:          nil,
				CreatedAt:           time.Now().Add(-10 * 24 * time.Hour).UTC(),
				DeletedAt:           nil,
				Id:                  "01J4VCZKH5QAF85GE501M8637W",
				Key:                 "feature-1",
				Metadata:            nil,
				MeterGroupByFilters: nil,
				MeterSlug:           nil,
				Name:                "feature-1",
				UpdatedAt:           time.Now().Add(-24 * time.Hour).UTC(),
			},
			Subject: api.Subject{
				CurrentPeriodEnd:   &time.Time{},
				CurrentPeriodStart: &time.Time{},
				DisplayName:        nil,
				Id:                 "01J4VD1XZH5HM705DCPB8XD5QD",
				Key:                "customer-1",
				Metadata:           nil,
				StripeCustomerId:   nil,
			},
			Value: api.EntitlementValue{
				Balance:   convert.ToPointer(10000.0),
				Config:    nil,
				HasAccess: true,
				Overage:   convert.ToPointer(500.0),
				Usage:     convert.ToPointer(50000.0),
			},
		},
	}
}

func NewCreateEventInput(namespace string, t notification.EventType, ruleID string, payload notification.EventPayload) notification.CreateEventInput {
	return notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:    t,
		Payload: payload,
		RuleID:  ruleID,
	}
}

type EventTestSuite struct {
	Env TestEnv

	channel    notification.Channel
	rule       notification.Rule
	subjectKey string
	feature    feature.Feature
}

func (s *EventTestSuite) Setup(ctx context.Context, t *testing.T) {
	t.Helper()

	m, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: s.Env.Namespace(),
		IDOrSlug:  TestMeterSlug,
	})
	require.NoError(t, err, "Getting meter must not return error")

	feat, err := s.Env.Feature().GetFeature(ctx, s.Env.Namespace(), TestFeatureKey, false)
	if _, ok := lo.ErrorsAs[*feature.FeatureNotFoundError](err); !ok {
		require.NoError(t, err, "Getting feature must not return error")
	}
	if feat != nil {
		s.feature = *feat
	} else {
		s.feature, err = s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
			Name:                TestFeatureName,
			Key:                 TestFeatureKey,
			Namespace:           s.Env.Namespace(),
			MeterSlug:           convert.ToPointer(m.Key),
			MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(m.GroupBy),
		})
	}
	require.NoError(t, err, "Creating feature must not return error")

	s.subjectKey = TestSubjectKey

	service := s.Env.Notification()

	channelIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationEventTest")

	channel, err := service.CreateChannel(ctx, channelIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	s.channel = *channel

	ruleIn := NewCreateRuleInput(s.Env.Namespace(), "NotificationEvent", s.channel.ID)

	rule, err := service.CreateRule(ctx, ruleIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	s.rule = *rule
}

func (s *EventTestSuite) TestCreateEvent(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	input := NewCreateEventInput(s.Env.Namespace(), notification.EventTypeBalanceThreshold, s.rule.ID, NewBalanceThresholdPayload())

	event, err := service.CreateEvent(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event, "Rule must not be nil")

	assert.Equal(t, float64(50), event.Payload.BalanceThreshold.Threshold.Value)
}

func (s *EventTestSuite) TestGetEvent(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	input := NewCreateEventInput(s.Env.Namespace(), notification.EventTypeBalanceThreshold, s.rule.ID, NewBalanceThresholdPayload())

	event, err := service.CreateEvent(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event, "Rule must not be nil")

	event2, err := service.GetEvent(ctx, notification.GetEventInput{
		Namespace: event.Namespace,
		ID:        event.ID,
	})
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event2, "Rule must not be nil")
}

func (s *EventTestSuite) TestListEvents(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateEventInput(s.Env.Namespace(), notification.EventTypeBalanceThreshold, s.rule.ID, NewBalanceThresholdPayload())

	event, err := service.CreateEvent(ctx, createIn)
	require.NoError(t, err, "Creating notification event must not return error")
	require.NotNil(t, event, "Notification event must not be nil")

	listIn := notification.ListEventsInput{
		Namespaces: []string{
			event.Namespace,
		},
		Events: []string{event.ID},
		From:   event.CreatedAt.Add(-time.Minute),
		To:     event.CreatedAt.Add(time.Minute),
	}

	events, err := service.ListEvents(ctx, listIn)
	require.NoError(t, err, "Listing notification events must not return error")
	require.NotNil(t, event, "Notification events must not be nil")

	expectedList := []notification.Event{
		*event,
	}

	require.Len(t, events.Items, len(expectedList), "List of events must match")

	for idx, e2 := range expectedList {
		e1 := events.Items[idx]

		assert.Equal(t, e1.ID, e2.ID, "Event IDs must match")
		assert.Equal(t, e1.Namespace, e2.Namespace, "Event namespaces must match")
	}
}

func (s *EventTestSuite) TestListDeliveryStatus(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateEventInput(s.Env.Namespace(), notification.EventTypeBalanceThreshold, s.rule.ID, NewBalanceThresholdPayload())

	event, err := service.CreateEvent(ctx, createIn)
	require.NoError(t, err, "Creating notification event must not return error")
	require.NotNil(t, event, "Notification event must not be nil")

	listIn := notification.ListEventsDeliveryStatusInput{
		Namespaces: []string{
			event.Namespace,
		},
		Events: []string{event.ID},
	}

	statuses, err := service.ListEventsDeliveryStatus(ctx, listIn)
	require.NoError(t, err, "Listing notification event delivery statuses must not return error")
	require.NotNil(t, event, "Notification event delivery statuses must not be nil")

	assert.Equal(t, statuses.TotalCount, len(s.rule.Channels), "Unexpected number of delivery statuses returned by listing events")

	channelsIDs := func() []string {
		var channelIDs []string
		for _, channel := range s.rule.Channels {
			channelIDs = append(channelIDs, channel.ID)
		}

		return channelIDs
	}()

	for _, status := range statuses.Items {
		assert.Equal(t, status.EventID, event.ID, "Unexpected event ID returned by listing events")
		assert.Truef(t, slices.Contains(channelsIDs, status.ChannelID), "Unexpected channel ID")
	}
}

func (s *EventTestSuite) TestUpdateDeliveryStatus(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateEventInput(s.Env.Namespace(), notification.EventTypeBalanceThreshold, s.rule.ID, NewBalanceThresholdPayload())

	event, err := service.CreateEvent(ctx, createIn)
	require.NoError(t, err, "Creating notification event must not return error")
	require.NotNil(t, event, "Notification event must not be nil")

	subTests := []struct {
		Name  string
		Input notification.UpdateEventDeliveryStatusInput
	}{
		{
			Name: "WithID",
			Input: notification.UpdateEventDeliveryStatusInput{
				NamespacedID: models.NamespacedID{
					Namespace: event.Namespace,
					ID:        event.DeliveryStatus[0].ID,
				},
				State: notification.EventDeliveryStatusStateFailed,
				Annotations: models.Annotations{
					"state": string(notification.EventDeliveryStatusStateFailed),
				},
			},
		},
	}

	for _, test := range subTests {
		t.Run(test.Name, func(t *testing.T) {
			status, err := service.UpdateEventDeliveryStatus(ctx, test.Input)
			require.NoError(t, err, "Updating notification event delivery status must not return error")
			require.NotNil(t, status, "Notification event must not be nil")

			assert.Equalf(t, test.Input.State, status.State, "Unexpected state returned by updating notification event delivery status")
			assert.Equalf(t, test.Input.Annotations, status.Annotations, "Unexpected annotations returned by updating notification event delivery status")
		})
	}
}
