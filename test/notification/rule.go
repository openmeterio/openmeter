package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewCreateRuleInput(name string, channels ...string) notification.CreateRuleInput {
	return notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: TestNamespace,
		},
		Type:     notification.RuleTypeBalanceThreshold,
		Name:     name,
		Disabled: false,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.RuleTypeBalanceThreshold,
			},
			BalanceThreshold: notification.BalanceThresholdRuleConfig{
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

	meter, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, TestNamespace, TestMeterSlug)
	require.NoError(t, err, "Getting meter must not return error")

	feat, err := s.Env.Feature().GetFeature(ctx, TestNamespace, TestFeatureKey, false)
	if _, ok := errorsx.ErrorAs[*feature.FeatureNotFoundError](err); !ok {
		require.NoError(t, err, "Getting feature must not return error")
	}
	if feat != nil {
		s.feature = *feat
	} else {
		s.feature, err = s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
			Name:                TestFeatureName,
			Key:                 TestFeatureKey,
			Namespace:           TestNamespace,
			MeterSlug:           convert.ToPointer(meter.Slug),
			MeterGroupByFilters: meter.GroupBy,
		})
	}
	require.NoError(t, err, "Creating feature must not return error")

	input := NewCreateChannelInput("NotificationRuleTest")

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
			CreateIn: NewCreateRuleInput("NotificationCreateRuleWithoutFeature", s.channel.ID),
		},
		{
			Name: "WithFeature",
			CreateIn: func() notification.CreateRuleInput {
				createIn := NewCreateRuleInput("NotificationCreateRuleWithFeature", s.channel.ID)
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
		})
	}
}

func (s *RuleTestSuite) TestList(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn1 := NewCreateRuleInput("NotificationListRule1", s.channel.ID)
	rule1, err := service.CreateRule(ctx, createIn1)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule1, "Rule must not be nil")

	createIn2 := NewCreateRuleInput("NotificationListRule2", s.channel.ID)
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

	createIn := NewCreateRuleInput("NotificationUpdateRule1", s.channel.ID)
	rule, err := service.CreateRule(ctx, createIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	updateIn := notification.UpdateRuleInput{
		NamespacedModel: rule.NamespacedModel,
		Type:            rule.Type,
		Name:            "NotificationUpdateRule2",
		Disabled:        true,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: rule.Config.Type,
			},
			BalanceThreshold: notification.BalanceThresholdRuleConfig{
				Features: rule.Config.BalanceThreshold.Features,
				Thresholds: append(rule.Config.BalanceThreshold.Thresholds, notification.BalanceThreshold{
					Type:  notification.BalanceThresholdTypeNumber,
					Value: 2000,
				}),
			},
		},
		ID: rule.ID,
	}

	rule2, err := service.UpdateRule(ctx, updateIn)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	assert.Equal(t, updateIn.Disabled, rule2.Disabled, "Rule must not be disabled")
	assert.Equal(t, updateIn.Type, rule2.Type, "Rule type must be the same")
	assert.EqualValues(t, updateIn.Config, rule2.Config, "Rule config must be the same")
}

func (s *RuleTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateRuleInput("NotificationDeleteRule1", s.channel.ID)

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

	createIn := NewCreateRuleInput("NotificationGetRule1", s.channel.ID)

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
	assert.Equal(t, rule.Type, rule.Type, "Rule type must be the same")
	assert.Equal(t, rule.Channels, rule.Channels, "Rule channels must be the same")
	assert.EqualValues(t, rule.Config, rule.Config, "Rule config must be the same")
}
