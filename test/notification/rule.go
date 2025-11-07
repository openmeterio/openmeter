package notification

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewCreateRuleInput(namespace string, name string, channels ...string) notification.CreateRuleInput {
	return notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:     notification.EventTypeBalanceThreshold,
		Name:     name,
		Disabled: false,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features: nil,
				Thresholds: []notification.BalanceThreshold{
					{
						Type:  notification.BalanceThresholdTypeNumber,
						Value: 1000,
					},
					{
						Type:  notification.BalanceThresholdTypePercent,
						Value: 95,
					},
				},
			},
		},
		Channels: channels,
		Metadata: models.Metadata{
			"namespace": namespace,
			"name":      name,
		},
		Annotations: models.Annotations{
			"namespace": namespace,
			"name":      name,
		},
	}
}

type RuleTestSuite struct {
	Env TestEnv

	channel notification.Channel
	feature feature.Feature
}

func (s *RuleTestSuite) Setup(ctx context.Context, t *testing.T) {
	t.Helper()

	service := s.Env.Notification()

	err := s.Env.Meter().ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: s.Env.Namespace(),
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test Meter",
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

	meter, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
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
			MeterSlug:           convert.ToPointer(meter.Key),
			MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(meter.GroupBy),
		})
	}
	require.NoError(t, err, "Creating feature must not return error")

	input := NewCreateChannelInput(s.Env.Namespace(), "NotificationRuleTest")

	channel, err := service.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	s.channel = *channel
}

func (s *RuleTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	t.Helper()

	service := s.Env.Notification()

	tests := []struct {
		Name     string
		CreateIn notification.CreateRuleInput
	}{
		{
			Name:     "WithoutFeature",
			CreateIn: NewCreateRuleInput(s.Env.Namespace(), "NotificationCreateRuleWithoutFeature", s.channel.ID),
		},
		{
			Name: "WithFeature",
			CreateIn: func() notification.CreateRuleInput {
				createIn := NewCreateRuleInput(s.Env.Namespace(), "NotificationCreateRuleWithFeature", s.channel.ID)
				createIn.Config.BalanceThreshold.Features = []string{s.feature.Key}

				return createIn
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			rule, err := service.CreateRule(ctx, test.CreateIn)
			require.NoError(t, err, "Creating rule must not return error")
			require.NotNil(t, rule, "Rule must not be nil")
			assert.NotEmpty(t, rule.ID, "Rule ID must not be empty")
			assert.Equal(t, test.CreateIn.Disabled, rule.Disabled, "Rule must not be disabled")
			assert.Equal(t, test.CreateIn.Type, rule.Type, "Rule type must be the same")
			assert.EqualValues(t, test.CreateIn.Config, rule.Config, "Rule config must be the same")
			assert.Equalf(t, test.CreateIn.Annotations, rule.Annotations, "Annotations must be the same")
			assert.Equalf(t, test.CreateIn.Metadata, rule.Metadata, "Metadata must be the same")
		})
	}
}

func (s *RuleTestSuite) TestList(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn1 := NewCreateRuleInput(s.Env.Namespace(), "NotificationListRule1", s.channel.ID)
	rule1, err := service.CreateRule(ctx, createIn1)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule1, "Rule must not be nil")

	createIn2 := NewCreateRuleInput(s.Env.Namespace(), "NotificationListRule2", s.channel.ID)
	rule2, err := service.CreateRule(ctx, createIn2)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	list, err := service.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{
			createIn1.Namespace,
			createIn2.Namespace,
		},
		Rules: []string{
			rule1.ID,
			rule2.ID,
		},
		OrderBy:         "id",
		IncludeDisabled: false,
	})
	require.NoError(t, err, "Listing rules must not return error")
	assert.NotEmpty(t, list.Items, "List of rules must not be empty")

	expectedList := []notification.Rule{
		*rule1,
		*rule2,
	}

	require.Len(t, list.Items, len(expectedList), "List of rules must match")

	for idx, r2 := range expectedList {
		r1 := list.Items[idx]

		assert.Equal(t, r1.ID, r2.ID, "Rule IDs must match")
		assert.Equal(t, r1.Namespace, r2.Namespace, "Rule namespaces must match")
	}
}

func (s *RuleTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateRuleInput(s.Env.Namespace(), "NotificationUpdateRule1", s.channel.ID)
	rule, err := service.CreateRule(ctx, createIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	updateIn := notification.UpdateRuleInput{
		NamespacedID: models.NamespacedID{
			Namespace: rule.Namespace,
			ID:        rule.ID,
		},
		Type:     rule.Type,
		Name:     "NotificationUpdateRule2",
		Disabled: true,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: rule.Config.Type,
			},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features: rule.Config.BalanceThreshold.Features,
				Thresholds: append(rule.Config.BalanceThreshold.Thresholds, notification.BalanceThreshold{
					Type:  notification.BalanceThresholdTypeNumber,
					Value: 2000,
				}),
			},
		},
		Channels: []string{s.channel.ID},
		Metadata: models.Metadata{
			"namespace": rule.Namespace,
			"name":      "NotificationUpdateRule2",
		},
		Annotations: models.Annotations{
			"namespace": rule.Namespace,
			"name":      "NotificationUpdateRule2",
		},
	}

	rule2, err := service.UpdateRule(ctx, updateIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	assert.Equal(t, updateIn.Disabled, rule2.Disabled, "Rule must not be disabled")
	assert.Equal(t, updateIn.Type, rule2.Type, "Rule type must be the same")
	assert.EqualValues(t, updateIn.Config, rule2.Config, "Rule config must be the same")
	assert.Equalf(t, updateIn.Annotations, rule2.Annotations, "Annotations must be the same")
	assert.Equalf(t, updateIn.Metadata, rule2.Metadata, "Metadata must be the same")
}

func (s *RuleTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateRuleInput(s.Env.Namespace(), "NotificationDeleteRule1", s.channel.ID)

	rule, err := service.CreateRule(ctx, createIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")
	assert.NotEmpty(t, rule.ID, "Rule ID must not be empty")

	err = service.DeleteRule(ctx, notification.DeleteRuleInput{
		Namespace: rule.Namespace,
		ID:        rule.ID,
	})
	require.NoError(t, err, "Deleting rule must not return error")
}

func (s *RuleTestSuite) TestGet(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateRuleInput(s.Env.Namespace(), "NotificationGetRule1", s.channel.ID)

	rule, err := service.CreateRule(ctx, createIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	getIn := notification.GetRuleInput{
		Namespace: rule.Namespace,
		ID:        rule.ID,
	}

	rule2, err := service.GetRule(ctx, getIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	assert.Equal(t, rule.Namespace, rule2.Namespace, "Rule namespace must be equal")
	assert.Equal(t, rule.ID, rule2.ID, "Rule ID must be equal")
	assert.Equal(t, rule.Disabled, rule2.Disabled, "Rule must not be disabled")
	assert.Equal(t, rule.Type, rule2.Type, "Rule type must be the same")
	assert.Equal(t, rule.Channels, rule2.Channels, "Rule channels must be the same")
	assert.EqualValues(t, rule.Config, rule2.Config, "Rule config must be the same")
	assert.Equalf(t, rule.Annotations, rule2.Annotations, "Annotations must be the same")
	assert.Equalf(t, rule.Metadata, rule2.Metadata, "Metadata must be the same")
}
