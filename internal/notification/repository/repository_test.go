package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	featureKey1 = "feature1"
	featureID1  = "featureID1"

	subjectKey1 = "subject1"
	subjectID1  = "subjectID1"

	namespace = "test"
)

type RepositoryTestSuite struct {
	suite.Suite

	dbClient *db.Client
	repo     notification.Repository

	eventWithFeature1          *notification.Event
	eventWithSubject1          *notification.Event
	eventWithFeatureAndSubject *notification.Event
	eventWithoutAnnotations    *notification.Event
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (s *RepositoryTestSuite) SetupSuite() {
	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(s.T())

	// build db clients
	s.dbClient = db.NewClient(db.Driver(driver))

	if err := s.dbClient.Schema.Create(context.Background()); err != nil {
		s.T().Fatalf("failed to migrate database %s", err)
	}

	repo, err := New(Config{
		Client: s.dbClient,
		Logger: testutils.NewLogger(s.T()),
	})
	require.NoError(s.T(), err)
	s.repo = repo

	s.setupTestData()
}

func (s *RepositoryTestSuite) setupTestData() {
	require := require.New(s.T())

	ctx := context.Background()

	channel, err := s.repo.CreateChannel(ctx, notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
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

	require.NoError(err)

	rule, err := s.repo.CreateRule(ctx, notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:     notification.RuleTypeBalanceThreshold,
		Name:     "whatever",
		Disabled: false,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.RuleTypeBalanceThreshold,
			},
		},
		Channels: []string{channel.ID},
	},
	)

	require.NoError(err)

	s.eventWithFeature1, err = s.repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventFeatureKey: featureKey1,
			notification.AnnotationEventFeatureID:  featureID1,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})
	require.NoError(err)

	s.eventWithSubject1, err = s.repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventSubjectID:  subjectID1,
			notification.AnnotationEventSubjectKey: subjectKey1,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})

	require.NoError(err)

	s.eventWithFeatureAndSubject, err = s.repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},

		Annotations: map[string]interface{}{
			notification.AnnotationEventSubjectID:  subjectID1,
			notification.AnnotationEventSubjectKey: subjectKey1,
			notification.AnnotationEventFeatureKey: featureKey1,
			notification.AnnotationEventFeatureID:  featureID1,
		},

		Type: notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
		},

		RuleID: rule.ID,
	})

	require.NoError(err)

	s.eventWithoutAnnotations, err = s.repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
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

	require.NoError(err)
}

func (s *RepositoryTestSuite) TestFilterEventByFeature() {
	require := require.New(s.T())
	ctx := context.Background()

	listedEvents, err := s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Features:   []string{featureID1},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithFeature1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Features:   []string{featureKey1},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithFeature1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Features:   []string{"invalid-feature"},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 0)
}

func (s *RepositoryTestSuite) TestFilterEventBySubject() {
	require := require.New(s.T())
	ctx := context.Background()

	listedEvents, err := s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Subjects:   []string{subjectID1},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithSubject1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Subjects:   []string{subjectID1},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 2)
	require.ElementsMatch(eventIDsFromEventPaginatedResponse(listedEvents), []string{s.eventWithSubject1.ID, s.eventWithFeatureAndSubject.ID})

	listedEvents, err = s.repo.ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{namespace},
		Subjects:   []string{"invalid-subject"},
	})

	require.NoError(err)
	require.Len(listedEvents.Items, 0)
}

func eventIDsFromEventPaginatedResponse(events pagination.PagedResponse[notification.Event]) []string {
	eventIDs := make([]string, len(events.Items))
	for i, event := range events.Items {
		eventIDs[i] = event.ID
	}

	return eventIDs
}

func (s *RepositoryTestSuite) TeardownSuite() {
	require.NoError(s.T(), s.dbClient.Close())
}
