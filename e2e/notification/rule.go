package notification

import (
	"context"
	"testing"

	"github.com/huandu/go-clone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

var createRuleInput = notification.CreateRuleInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: TestNamespace,
	},
	Type:     notification.RuleTypeBalanceThreshold,
	Name:     "NotificationRuleTest",
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
	Channels: nil,
}

type RuleTestSuite struct {
	Env TestEnv

	channel notification.Channel
}

func (s *RuleTestSuite) Setup(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input.Name = "NotificationRule"

	channel, err := connector.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	s.channel = *channel

	createRuleInput.Channels = []string{
		s.channel.ID,
	}
}

func (s *RuleTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createRuleInput).(notification.CreateRuleInput)

	rule, err := connector.CreateRule(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")
	assert.NotEmpty(t, rule.ID, "Rule ID must not be empty")
	assert.Equal(t, input.Disabled, rule.Disabled, "Rule must not be disabled")
	assert.Equal(t, input.Type, rule.Type, "Rule type must be the same")
	assert.EqualValues(t, input.Config, rule.Config, "Rule config must be the same")
}

func (s *RuleTestSuite) TestList(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input1 := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	input1.Name = "NotificationListRule1"
	rule1, err := connector.CreateRule(ctx, input1)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule1, "Rule must not be nil")

	input2 := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	input2.Name = "NotificationListRule2"
	rule2, err := connector.CreateRule(ctx, input2)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	list, err := connector.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{
			input1.Namespace,
			input2.Namespace,
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

	assert.EqualValues(t, expectedList, list.Items, "Unexpected items returned by listing rules")
}
func (s *RuleTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input1 := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	input1.Name = "NotificationUpdateRule1"
	rule, err := connector.CreateRule(ctx, input1)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	input2 := notification.UpdateRuleInput{
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

	rule2, err := connector.UpdateRule(ctx, input2)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	assert.Equal(t, input2.Disabled, rule2.Disabled, "Rule must not be disabled")
	assert.Equal(t, input2.Type, rule2.Type, "Rule type must be the same")
	assert.EqualValues(t, input2.Config, rule2.Config, "Rule config must be the same")
}
func (s *RuleTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	input.Name = "NotificationDeleteRule1"

	rule, err := connector.CreateRule(ctx, input)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")
	assert.NotEmpty(t, rule.ID, "Rule ID must not be empty")

	err = connector.DeleteRule(ctx, notification.DeleteRuleInput{
		Namespace: rule.Namespace,
		ID:        rule.ID,
	})
	require.NoError(t, err, "Deleting rule must not return error")
}
func (s *RuleTestSuite) TestGet(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input1 := clone.Clone(createRuleInput).(notification.CreateRuleInput)
	input1.Name = "NotificationGetRule1"

	rule, err := connector.CreateRule(ctx, input1)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule, "Rule must not be nil")

	input2 := notification.GetRuleInput{
		Namespace: rule.Namespace,
		ID:        rule.ID,
	}

	rule2, err := connector.GetRule(ctx, input2)
	require.NoError(t, err, "Creating rule must not return error")
	require.NotNil(t, rule2, "Rule must not be nil")

	assert.Equal(t, rule.Namespace, rule2.Namespace, "Rule namespace must be equal")
	assert.Equal(t, rule.ID, rule2.ID, "Rule ID must be equal")
	assert.Equal(t, rule.Disabled, rule2.Disabled, "Rule must not be disabled")
	assert.Equal(t, rule.Type, rule.Type, "Rule type must be the same")
	assert.Equal(t, rule.Channels, rule.Channels, "Rule channels must be the same")
	assert.EqualValues(t, rule.Config, rule.Config, "Rule config must be the same")
}

func RunRuleTests(ctx context.Context, t *testing.T, env TestEnv) {
	if t == nil {
		panic("t is nil")
	}

	if env == nil {
		panic("env is nil")
	}

}
