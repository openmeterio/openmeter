package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type EventsConfiguration struct {
	Enabled      bool
	SystemEvents EventSubsystemConfiguration
}

func (c EventsConfiguration) Validate() error {
	return c.SystemEvents.Validate()
}

type EventSubsystemConfiguration struct {
	Enabled bool
	Topic   string

	AutoProvision AutoProvisionConfiguration
}

func (c EventSubsystemConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Topic == "" {
		return errors.New("topic name is required")
	}
	return c.AutoProvision.Validate()
}

type AutoProvisionConfiguration struct {
	Enabled    bool
	Partitions int
}

func (c AutoProvisionConfiguration) Validate() error {
	if c.Enabled && c.Partitions < 1 {
		return errors.New("partitions must be greater than 0")
	}
	return nil
}

type PoisionQueueConfiguration struct {
	Enabled       bool
	Topic         string
	AutoProvision AutoProvisionConfiguration
	Throttle      ThrottleConfiguration
}

func (c PoisionQueueConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Topic == "" {
		return errors.New("topic name is required")
	}

	if err := c.Throttle.Validate(); err != nil {
		return fmt.Errorf("throttle: %w", err)
	}

	return nil
}

type ThrottleConfiguration struct {
	Enabled  bool
	Count    int64
	Duration time.Duration
}

func (c ThrottleConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Count <= 0 {
		return errors.New("count must be greater than 0")
	}

	if c.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}

	return nil
}

type RetryConfiguration struct {
	MaxRetries      int
	InitialInterval time.Duration
}

func (c RetryConfiguration) Validate() error {
	if c.MaxRetries <= 0 {
		return errors.New("max retries must be greater than 0")
	}

	if c.InitialInterval <= 0 {
		return errors.New("initial interval must be greater than 0")
	}

	return nil
}

func ConfigureEvents(v *viper.Viper) {
	// TODO: after the system events are fully implemented, we should enable them by default
	v.SetDefault("events.enabled", false)
	v.SetDefault("events.systemEvents.enabled", true)
	v.SetDefault("events.systemEvents.topic", "om_sys.api_events")
	v.SetDefault("events.systemEvents.autoProvision.enabled", true)
	v.SetDefault("events.systemEvents.autoProvision.partitions", 4)
}
