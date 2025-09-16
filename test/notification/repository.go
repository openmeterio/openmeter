package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type RepositoryTestSuite struct {
	Env TestEnv

	namespace string

	eventWithFeature1          *notification.Event
	eventWithSubject1          *notification.Event
	eventWithFeatureAndSubject *notification.Event
	eventWithoutAnnotations    *notification.Event
}

func (s *RepositoryTestSuite) Setup(ctx context.Context, t *testing.T) {
	s.namespace = s.Env.Namespace()

	repo := s.Env.NotificationRepo()

	channel, err := repo.CreateChannel(ctx, notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},
		Type: notification.ChannelTypeWebhook,
		Name: "whatever",
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: notification.ChannelTypeWebhook,
			},

			WebHook: notification.WebHookChannelConfig{
				URL: "http://localhost",
			},
		},
	})

	require.NoError(t, err)

	rule, err := repo.CreateRule(ctx, notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},
		Type:     notification.EventTypeBalanceThreshold,
		Name:     "whatever",
		Disabled: false,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},
		Channels: []string{channel.ID},
	},
	)

	require.NoError(t, err)

	s.eventWithFeature1, err = repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventFeatureKey: TestFeatureKey,
			notification.AnnotationEventFeatureID:  TestFeatureID,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})
	require.NoError(t, err)

	s.eventWithSubject1, err = repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventSubjectID:  TestSubjectID,
			notification.AnnotationEventSubjectKey: TestSubjectKey,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})

	require.NoError(t, err)

	s.eventWithFeatureAndSubject, err = repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventSubjectID:  TestSubjectID,
			notification.AnnotationEventSubjectKey: TestSubjectKey,
			notification.AnnotationEventFeatureKey: TestFeatureKey,
			notification.AnnotationEventFeatureID:  TestFeatureID,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})

	require.NoError(t, err)

	s.eventWithoutAnnotations, err = repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},

		Annotations: nil,

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})

	require.NoError(t, err)
}

func (s *RepositoryTestSuite) TestFilterEventByFeature(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	repo := s.Env.NotificationRepo()

	listedEvents, err := repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Features:   []string{TestFeatureID},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithFeature1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Features:   []string{TestFeatureKey},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithFeature1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Features:   []string{"invalid-feature"},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 0)
}

func (s *RepositoryTestSuite) TestFilterEventBySubject(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	repo := s.Env.NotificationRepo()

	listedEvents, err := repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Subjects:   []string{TestSubjectID},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithSubject1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Subjects:   []string{TestSubjectID},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithSubject1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		Subjects:   []string{"invalid-subject"},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 0)
}

func eventIDsFromEventPaginatedResponse(events pagination.Result[notification.Event]) []string {
	eventIDs := make([]string, len(events.Items))
	for i, event := range events.Items {
		eventIDs[i] = event.ID
	}

	return eventIDs
}
