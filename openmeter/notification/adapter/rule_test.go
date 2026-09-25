package adapter_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// createRule seeds an invoice.created rule via the adapter directly, mirroring
// createChannel: service-level CreateRule also registers the rule on the channels'
// webhooks, which the noop handler cannot do.
func createRule(t *testing.T, env *channelTestEnv, ns string, at time.Time, mutate func(*notification.CreateRuleInput)) notification.Rule {
	t.Helper()

	clock.FreezeTime(at)
	defer clock.UnFreeze()

	in := notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.EventTypeInvoiceCreated,
		Name:            "rule-" + ulid.Make().String(),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
			Invoice:        &notification.InvoiceRuleConfig{},
		},
	}
	if mutate != nil {
		mutate(&in)
	}

	rule, err := env.adapter.CreateRule(t.Context(), in)
	require.NoError(t, err, "seeding rule must not fail")
	require.NotNil(t, rule)

	return *rule
}

func TestListRules_Filters(t *testing.T) {
	env := newChannelTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()
	t0 := frozen.Add(-2 * time.Hour)
	t1 := frozen.Add(-1 * time.Hour)

	channelA := createChannel(t, env, ns, t0, nil)
	channelB := createChannel(t, env, ns, t0, nil)
	disabledChannel := createChannel(t, env, ns, t0, func(in *notification.CreateChannelInput) {
		in.Disabled = true
	})

	first := createRule(t, env, ns, t0, func(in *notification.CreateRuleInput) {
		in.Name = "first-rule"
		in.Channels = []string{channelA.ID}
	})

	second := createRule(t, env, ns, t1, func(in *notification.CreateRuleInput) {
		in.Name = "second-rule"
		in.Type = notification.EventTypeInvoiceUpdated
		in.Config.Type = notification.EventTypeInvoiceUpdated
		in.Channels = []string{channelB.ID, disabledChannel.ID}
	})

	disabled := createRule(t, env, ns, frozen, func(in *notification.CreateRuleInput) {
		in.Name = "disabled-rule"
		in.Disabled = true
		in.Channels = []string{channelA.ID}
	})

	testCases := []struct {
		name    string
		input   notification.ListRulesInput
		wantIDs []string
	}{
		{
			name:    "no filter returns every rule, disabled included",
			wantIDs: []string{first.ID, second.ID, disabled.ID},
		},
		{
			name: "ID eq matches exactly one rule",
			input: notification.ListRulesInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(first.ID)}},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "ID in matches the selected rules",
			input: notification.ListRulesInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr([]string{first.ID, second.ID})}},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Name eq matches exactly one rule",
			input: notification.ListRulesInput{
				Name: &filter.FilterString{Eq: lo.ToPtr("second-rule")},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "Name contains matches by substring",
			input: notification.ListRulesInput{
				Name: &filter.FilterString{Contains: lo.ToPtr("first")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "Type eq matches rules of that type only",
			input: notification.ListRulesInput{
				Type: &filter.FilterString{Eq: lo.ToPtr(string(notification.EventTypeInvoiceCreated))},
			},
			wantIDs: []string{first.ID, disabled.ID},
		},
		{
			name: "Disabled eq false excludes the disabled rule",
			input: notification.ListRulesInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(false)},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Disabled eq true matches only the disabled rule",
			input: notification.ListRulesInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(true)},
			},
			wantIDs: []string{disabled.ID},
		},
		{
			name: "ChannelID eq matches every rule targeting the channel",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{Eq: lo.ToPtr(channelA.ID)},
			},
			wantIDs: []string{first.ID, disabled.ID},
		},
		{
			name: "ChannelID in matches rules targeting any of the channels",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{In: lo.ToPtr([]string{channelA.ID, channelB.ID})},
			},
			wantIDs: []string{first.ID, second.ID, disabled.ID},
		},
		{
			// DeleteChannel refuses to delete a channel still assigned to a rule, so the
			// assignment must stay visible even once the channel is disabled.
			name: "ChannelID matches a disabled channel assignment",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{Eq: lo.ToPtr(disabledChannel.ID)},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "CreatedAt range matches only the rule created inside the window",
			input: notification.ListRulesInput{
				CreatedAt: &filter.FilterTime{
					And: &[]filter.FilterTime{
						{Gte: lo.ToPtr(t0.Add(-time.Minute))},
						{Lte: lo.ToPtr(t0.Add(time.Minute))},
					},
				},
			},
			wantIDs: []string{first.ID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			result, err := env.adapter.ListRules(t.Context(), input)
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}
