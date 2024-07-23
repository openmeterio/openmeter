package notification

import (
	"context"
	"testing"
	"time"

	"github.com/huandu/go-clone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

var createEventInput = notification.CreateEventInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: TestNamespace,
	},
	Type:      notification.EventTypeBalanceThreshold,
	CreatedAt: time.Now(),
	Payload: notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeBalanceThreshold,
		},
		BalanceThreshold: notification.BalanceThresholdPayload{
			Entitlement: api.EntitlementMetered{
				CreatedAt: &time.Time{},
				CurrentUsagePeriod: api.Period{
					From: time.Time{},
					To:   time.Time{},
				},
				DeletedAt:               &time.Time{},
				FeatureId:               "",
				FeatureKey:              "",
				Id:                      nil,
				IsSoftLimit:             nil,
				IsUnlimited:             nil,
				IssueAfterReset:         nil,
				IssueAfterResetPriority: nil,
				LastReset:               time.Time{},
				MeasureUsageFrom:        time.Time{},
				Metadata:                nil,
				SubjectKey:              "",
				Type:                    "",
				UpdatedAt:               &time.Time{},
				UsagePeriod: api.RecurringPeriod{
					Anchor:   time.Time{},
					Interval: "",
				},
			},
			Feature: api.Feature{
				ArchivedAt:          &time.Time{},
				CreatedAt:           &time.Time{},
				DeletedAt:           &time.Time{},
				Id:                  nil,
				Key:                 "",
				Metadata:            nil,
				MeterGroupByFilters: nil,
				MeterSlug:           nil,
				Name:                "",
				UpdatedAt:           &time.Time{},
			},
			Subject: api.Subject{
				CurrentPeriodEnd:   &time.Time{},
				CurrentPeriodStart: &time.Time{},
				DisplayName:        nil,
				Id:                 nil,
				Key:                "",
				Metadata:           nil,
				StripeCustomerId:   nil,
			},
			Balance: api.EntitlementValue{
				Balance:   nil,
				Config:    nil,
				HasAccess: nil,
				Overage:   nil,
				Usage:     nil,
			},
			Threshold: api.NotificationRuleBalanceThresholdValue{
				Type:  notification.BalanceThresholdTypeNumber,
				Value: 50,
			},
		},
	},
	Rule: notification.Rule{},
}

type EventTestSuite struct {
	Env TestEnv

	feature     interface{}
	subject     interface{}
	entitlement interface{}

	channel notification.Channel
	rule    notification.Rule
}

func (s *EventTestSuite) Setup(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	channelIn := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	channelIn.Name = "NotificationEvent"
	channelIn.Config.WebHook.URL = "https://play.svix.com/in/e_vfY684MsprnBfc8IR04tSJH4K1T/"

	channel, err := connector.CreateChannel(ctx, channelIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	s.channel = *channel

	ruleIn := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	ruleIn.Name = "NotificationEvent"
	ruleIn.Channels = []string{
		s.channel.ID,
	}

	rule, err := connector.CreateRule(ctx, ruleIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	s.rule = *rule

	createEventInput.Rule = *rule
}

func (s *EventTestSuite) TestCreateEvent(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createEventInput).(notification.CreateEventInput)
	input.Rule = s.rule

	event, err := connector.CreateEvent(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event, "Rule must not be nil")

	// FIXME: add more assertions
}
func (s *EventTestSuite) TestListEvents(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createEventInput).(notification.CreateEventInput)
	input.Rule = s.rule

	event, err := connector.CreateEvent(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event, "Rule must not be nil")

	input2 := notification.ListEventsInput{
		Namespaces: []string{
			event.Namespace,
		},
		From: event.CreatedAt.Add(-time.Minute),
		To:   event.CreatedAt.Add(time.Minute),
	}

	events, err := connector.ListEvents(ctx, input2)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, event, "Rule must not be nil")

	expectedList := []notification.Event{
		*event,
	}

	assert.EqualValues(t, expectedList, events.Items, "Unexpected items returned by listing events")

	// FIXME: add more assertions

}
func (s *EventTestSuite) TestCreateDeliveryStatus(ctx context.Context, t *testing.T) {
	// FIXME:
}
func (s *EventTestSuite) TestListCreateDeliveryStatus(ctx context.Context, t *testing.T) {
	// FIXME:
}
