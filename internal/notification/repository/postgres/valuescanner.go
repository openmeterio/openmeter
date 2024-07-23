package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/internal/notification"
)

type channelConfigSerde[T any] struct {
	notification.ChannelConfigMeta

	Data T `json:"data"`
}

var ChannelConfigValueScanner = field.ValueScannerFunc[notification.ChannelConfig, *sql.NullString]{
	V: func(config notification.ChannelConfig) (driver.Value, error) {
		switch config.Type {
		case notification.ChannelTypeWebhook:
			serde := channelConfigSerde[notification.WebHookChannelConfig]{
				ChannelConfigMeta: notification.ChannelConfigMeta{
					Type: config.Type,
				},
				Data: config.WebHook,
			}
			return json.Marshal(serde)
		default:
			return nil, fmt.Errorf("unknown channel type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (notification.ChannelConfig, error) {
		var channelConfig notification.ChannelConfig
		if ns == nil || !ns.Valid {
			return channelConfig, nil
		}

		data := []byte(ns.String)

		var meta notification.ChannelConfigMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return channelConfig, err
		}

		switch meta.Type {
		case notification.ChannelTypeWebhook:
			serde := channelConfigSerde[notification.WebHookChannelConfig]{
				ChannelConfigMeta: notification.ChannelConfigMeta{
					Type: meta.Type,
				},
				Data: notification.WebHookChannelConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return channelConfig, err
			}

			channelConfig = notification.ChannelConfig{
				ChannelConfigMeta: serde.ChannelConfigMeta,
				WebHook:           serde.Data,
			}

		default:
			return channelConfig, fmt.Errorf("unknown channel type: %s", meta.Type)
		}

		return channelConfig, nil
	},
}

type ruleConfigSerde[T any] struct {
	notification.RuleConfigMeta

	Data T `json:"data"`
}

var RuleConfigValueScanner = field.ValueScannerFunc[notification.RuleConfig, *sql.NullString]{
	V: func(config notification.RuleConfig) (driver.Value, error) {
		switch config.Type {
		case notification.RuleTypeBalanceThreshold:
			serde := ruleConfigSerde[notification.BalanceThresholdRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: config.Type,
				},
				Data: config.BalanceThreshold,
			}
			return json.Marshal(serde)
		default:
			return nil, fmt.Errorf("unknown channel type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (notification.RuleConfig, error) {
		var ruleConfig notification.RuleConfig
		if ns == nil || !ns.Valid {
			return ruleConfig, nil
		}

		data := []byte(ns.String)

		var meta notification.RuleConfigMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return ruleConfig, err
		}

		switch meta.Type {
		case notification.RuleTypeBalanceThreshold:
			serde := ruleConfigSerde[notification.BalanceThresholdRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: meta.Type,
				},
				Data: notification.BalanceThresholdRuleConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return ruleConfig, err
			}

			ruleConfig = notification.RuleConfig{
				RuleConfigMeta:   serde.RuleConfigMeta,
				BalanceThreshold: serde.Data,
			}

		default:
			return ruleConfig, fmt.Errorf("unknown rule type: %s", meta.Type)
		}

		return ruleConfig, nil
	},
}
