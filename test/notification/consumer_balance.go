package notification

import (
	"context"
	"crypto/rand"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type BalanceNotificaiontHandlerTestSuite struct {
	Env TestEnv

	channel   notification.Channel
	rule      notification.Rule
	feature   feature.Feature
	handler   consumer.EntitlementSnapshotHandler
	namespace string
}

var (
	TestEntitlementCurrentUsagePeriod = timeutil.ClosedPeriod{
		From: time.Now().Add(-time.Hour),
		To:   time.Now().Add(24 * time.Hour),
	}
	TestEntitlementUsagePeriod = entitlement.NewUsagePeriodFromRecurrence(timeutil.Recurrence{
		Interval: timeutil.RecurrencePeriodDaily,
		Anchor:   TestEntitlementCurrentUsagePeriod.From,
	})
	TestEntitlementID = "test-entitlement-id"
)

type BalanceSnapshotEventInput struct {
	Feature   feature.Feature
	Value     snapshot.EntitlementValue
	Namespace string
}

func NewBalanceSnapshotEvent(in BalanceSnapshotEventInput) snapshot.SnapshotEvent {
	return snapshot.SnapshotEvent{
		Entitlement: entitlement.Entitlement{
			GenericProperties: entitlement.GenericProperties{
				NamespacedModel: models.NamespacedModel{
					Namespace: in.Namespace,
				},
				ID:              TestEntitlementID,
				FeatureID:       in.Feature.ID,
				FeatureKey:      in.Feature.Key,
				EntitlementType: entitlement.EntitlementTypeMetered,

				UsagePeriod:               &TestEntitlementUsagePeriod,
				OriginalUsagePeriodAnchor: lo.ToPtr(TestEntitlementUsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Anchor),
				CurrentUsagePeriod:        &TestEntitlementCurrentUsagePeriod,

				CustomerID: TestCustomerID,
			},
			MeasureUsageFrom: &TestEntitlementCurrentUsagePeriod.From,
			IsSoftLimit:      convert.ToPointer(true),
			LastReset:        &TestEntitlementCurrentUsagePeriod.From,
		},
		Namespace: eventmodels.NamespaceID{
			ID: in.Namespace,
		},
		Subject: subject.Subject{
			Key: TestSubjectKey,
		},
		Feature:            in.Feature,
		Operation:          snapshot.ValueOperationUpdate,
		CalculatedAt:       convert.ToPointer(time.Now()),
		Value:              &in.Value,
		CurrentUsagePeriod: &TestEntitlementCurrentUsagePeriod,
	}
}

// setupNamespace can be used to set up an independent namespace for testing, it contains a single
// feature and rule with a channel. For more complex scenarios, additional setup might be required.
func (s *BalanceNotificaiontHandlerTestSuite) setupNamespace(ctx context.Context, t *testing.T) {
	t.Helper()

	// Set a new namespace
	s.namespace = ulid.Make().String()

	err := s.Env.Meter().ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: s.namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           TestMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "request",
			ValueProperty: lo.ToPtr("$.duration_ms"),
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
		},
	})
	require.NoError(t, err, "Replacing meters must not return error")

	// Setup dependencies
	service := s.Env.Notification()

	meter, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: s.namespace,
		IDOrSlug:  TestMeterSlug,
	})
	require.NoError(t, err, "Getting meter must not return error")

	s.feature, err = s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:                TestFeatureName,
		Key:                 TestFeatureKey,
		Namespace:           s.namespace,
		MeterSlug:           convert.ToPointer(meter.Key),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(meter.GroupBy),
	})
	require.NoError(t, err, "Creating feature must not return error")

	input := NewCreateChannelInput(s.namespace, "NotificationRuleTest")

	channel, err := service.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	s.channel = *channel

	ruleInput := NewCreateRuleInput(s.namespace, "TestRuleForNotificationWorker", s.channel.ID)

	rule, err := service.CreateRule(ctx, ruleInput)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	s.rule = *rule
	s.rule.CreatedAt = s.rule.CreatedAt.Truncate(time.Microsecond)
	s.rule.UpdatedAt = s.rule.UpdatedAt.Truncate(time.Microsecond)

	s.handler = consumer.EntitlementSnapshotHandler{
		Notification: service,
		Logger:       slog.Default(),
	}
}

func (s *BalanceNotificaiontHandlerTestSuite) TestGrantingFlow(ctx context.Context, t *testing.T) {
	s.setupNamespace(ctx, t)

	service := s.Env.Notification()

	// Step 1: The current usage is less than the thresholds
	snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(100.0),
			Usage:   convert.ToPointer(50.0),
		},
		Namespace: s.namespace,
	})

	err := s.handler.Handle(ctx, snapshotEvent)
	require.NoError(t, err)

	events, err := service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Empty(t, events.Items, "No events should be created")

	// The rule has the following thresholds:
	// - 95% of the balance
	// - 1000 units

	// Step 2: The current usage is greater than the balance threshold 95% (balance = 4, usage = 96)

	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(4.0),
			Usage:   convert.ToPointer(96.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 1, "One event should be created")

	// Let's sanity check the resulting event
	event := events.Items[0]
	require.Equal(t, s.rule.ID, event.Rule.ID, "Event must be associated with the rule")
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.Equal(t, TestEntitlementID, event.Payload.BalanceThreshold.Entitlement.Id, "Event must be associated with the entitlement")
	require.NotEmpty(t, event.Annotations[notification.AnnotationBalanceEventDedupeHash], "Event must have a deduplication hash")
	require.NoError(t, event.Payload.BalanceThreshold.Validate(), "Event must be valid")
	require.Equal(t, api.NotificationRuleBalanceThresholdValue{
		Value: 95,
		Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
	}, event.Payload.BalanceThreshold.Threshold)

	// Step 3: Additional events hitting the same 95% threshold should not create new events
	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(3.0),
			Usage:   convert.ToPointer(97.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 1, "One event should be created")

	// Step 4: The user receives +2000 credits, given that current usage doesn't exceed any threshold
	// we are not creating additional notifications

	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(2004.0),
			Usage:   convert.ToPointer(96.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 1, "One event should be created")

	// Step 5: The user spends 1000 credits, hitting the 1000 units threshold
	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(1004.0),
			Usage:   convert.ToPointer(1096.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 2, "Two events should be created")

	// Let's sanity check the resulting event
	event = events.Items[0]
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.NotEmpty(t, event.Annotations[notification.AnnotationBalanceEventDedupeHash], "Event must have a deduplication hash")
	require.Equal(t, api.NotificationRuleBalanceThresholdValue{
		Value: 1000,
		Type:  api.NotificationRuleBalanceThresholdValueTypeNumber,
	}, event.Payload.BalanceThreshold.Threshold)

	// Step 6: The user hits the 95% threshold again
	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(4.0),
			Usage:   convert.ToPointer(2096.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")
	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})
	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 3, "Three events should be created")

	// Let's sanity check the resulting event
	event = events.Items[0]
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.Equal(t, api.NotificationRuleBalanceThresholdValue{
		Value: 95,
		Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
	}, event.Payload.BalanceThreshold.Threshold)

	// Step 7: The user gets +1000 credits, given that the 95% threshold is no longer valid
	// a new event should not be created for the 1000 units threshold
	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(1004.0),
			Usage:   convert.ToPointer(2096.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 4, "Four events should be created")

	// Let's sanity check the resulting event
	event = events.Items[0]
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.Equal(t, api.NotificationRuleBalanceThresholdValue{
		Value: 1000,
		Type:  api.NotificationRuleBalanceThresholdValueTypeNumber,
	}, event.Payload.BalanceThreshold.Threshold)

	// Step 8: The entitlement gets reset, no events should be created

	newUsagePeriod := timeutil.ClosedPeriod{
		From: TestEntitlementCurrentUsagePeriod.To,
		To:   TestEntitlementCurrentUsagePeriod.To.Add(24 * time.Hour),
	}

	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(100.0),
			Usage:   convert.ToPointer(0.0),
		},
		Namespace: s.namespace,
	})
	snapshotEvent.Entitlement.CurrentUsagePeriod = &newUsagePeriod
	snapshotEvent.Entitlement.LastReset = &newUsagePeriod.From
	snapshotEvent.CurrentUsagePeriod = &newUsagePeriod

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 4, "Four events should be created")

	// Step 9: The user hits the 95% threshold again after the reset, new event should be created
	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(1.0),
			Usage:   convert.ToPointer(99.0),
		},
		Namespace: s.namespace,
	})
	snapshotEvent.Entitlement.CurrentUsagePeriod = &newUsagePeriod
	snapshotEvent.Entitlement.LastReset = &newUsagePeriod.From
	snapshotEvent.CurrentUsagePeriod = &newUsagePeriod

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 5, "Five events should be created")

	// Let's sanity check the resulting event
	event = events.Items[0]
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.Equal(t, api.NotificationRuleBalanceThresholdValue{
		Value: 95,
		Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
	}, event.Payload.BalanceThreshold.Threshold)
}

const (
	TestFeature2Name = "TestFeature2"
	TestFeature2Key  = "test-feature-2"

	TestFeature3Name = "TestFeature3"
	TestFeature3Key  = "test-feature-3"
)

func (s *BalanceNotificaiontHandlerTestSuite) TestFeatureFiltering(ctx context.Context, t *testing.T) {
	s.setupNamespace(ctx, t)

	service := s.Env.Notification()

	meter, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: s.namespace,
		IDOrSlug:  TestMeterSlug,
	})
	require.NoError(t, err, "Getting meter must not return error")

	// let's setup two more features (we should use different meters but for the sake of simplicity we are using the same one)
	feature1 := s.feature
	require.NotNil(t, feature1, "Feature must not be nil")

	feature2, err := s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:                TestFeature2Name,
		Key:                 TestFeature2Key,
		Namespace:           s.namespace,
		MeterSlug:           convert.ToPointer(meter.Key),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(meter.GroupBy),
	})
	require.NoError(t, err, "Creating feature must not return error")

	feature3, err := s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:                TestFeature3Name,
		Key:                 TestFeature3Key,
		Namespace:           s.namespace,
		MeterSlug:           convert.ToPointer(meter.Key),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(meter.GroupBy),
	})
	require.NoError(t, err, "Creating feature must not return error")
	require.NotNil(t, feature3, "Feature must not be nil")

	// Let's create a few rules to test feature filtering
	// wildcard rule without feature filtering
	rule1 := s.rule
	require.NotNil(t, rule1, "Rule must not be nil")
	rule1.CreatedAt = rule1.CreatedAt.Truncate(time.Microsecond)

	// rule with feature filtering using feature key
	ruleInput := NewCreateRuleInput(s.namespace, "TestRule2ForNotificationWorker", s.channel.ID)
	ruleInput.Config.BalanceThreshold.Features = []string{feature2.Key}

	rule2, err := service.CreateRule(ctx, ruleInput)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")
	rule2.CreatedAt = rule2.CreatedAt.Truncate(time.Microsecond)
	rule2.UpdatedAt = rule2.UpdatedAt.Truncate(time.Microsecond)

	// rule with feature filtering using feature key
	ruleInput = NewCreateRuleInput(s.namespace, "TestRule3ForNotificationWorker", s.channel.ID)
	ruleInput.Config.BalanceThreshold.Features = []string{feature2.ID}

	rule3, err := service.CreateRule(ctx, ruleInput)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule3, "Rule must not be nil")
	rule3.CreatedAt = rule3.CreatedAt.Truncate(time.Microsecond)
	rule3.UpdatedAt = rule3.UpdatedAt.Truncate(time.Microsecond)

	// Step 1: A new event is created for feature 3, which should be only matched by
	// rule 1 (wildcard rule)
	snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: feature3,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(1.0),
			Usage:   convert.ToPointer(10001.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err := service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 1, "Event is created")

	// Let's sanity check the resulting event
	event := events.Items[0]
	require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")
	require.Equal(t, rule1, event.Rule, "Event must be associated with the rule")

	// Step 2: A new event is created for feature 2, which should be matched by all rules:
	// - rule 1 (wildcard rule)
	// - rule 2 (feature key filtering)
	// - rule 3 (feature ID filtering)

	snapshotEvent = NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: feature2,
		Value: snapshot.EntitlementValue{
			Balance: convert.ToPointer(1.0),
			Usage:   convert.ToPointer(10001.0),
		},
		Namespace: s.namespace,
	})

	require.NoError(t, s.handler.Handle(ctx, snapshotEvent), "Handling event must not return error")

	events, err = service.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing events must not return error")
	require.Len(t, events.Items, 4, "Events are created")

	// Let's sanity check the resulting events
	eventsCreated := events.Items[0:3]
	affectedRules := []notification.Rule{}
	for _, event := range eventsCreated {
		require.Equal(t, notification.EventTypeBalanceThreshold, event.Payload.Type, "Event must be of type balance threshold")

		affectedRules = append(affectedRules, event.Rule)
	}

	require.Contains(t, affectedRules, rule1, "Event must be associated with the rule1")
	require.Contains(t, affectedRules, *rule2, "Event must be associated with the rule2")
	require.Contains(t, affectedRules, *rule3, "Event must be associated with the rule3")
}
