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
	IngestEvents EventSubsystemConfiguration
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
	Enabled      bool
	Partitions   int
	DLQRetention time.Duration
}

func (c AutoProvisionConfiguration) Validate() error {
	if c.Enabled && c.Partitions < 1 {
		return errors.New("partitions must be greater than 0")
	}
	return nil
}

type ConsumerConfiguration struct {
	// ProcessingTimeout is the maximum time a message is allowed to be processed before it is considered failed. 0 disables the timeout.
	ProcessingTimeout time.Duration

	// Retry specifies how many times a message should be retried before it is sent to the DLQ.
	Retry RetryConfiguration

	// ConsumerGroupName is the name of the consumer group that the consumer belongs to.
	ConsumerGroupName string

	// DLQ specifies the configuration for the Dead Letter Queue.
	DLQ DLQConfiguration
}

func (c ConsumerConfiguration) Validate() error {
	if c.ProcessingTimeout < 0 {
		return errors.New("processing timeout must be positive or 0")
	}

	if c.ConsumerGroupName == "" {
		return errors.New("consumer group name is required")
	}

	if err := c.Retry.Validate(); err != nil {
		return fmt.Errorf("retry configuration is invalid: %w", err)
	}

	if err := c.DLQ.Validate(); err != nil {
		return fmt.Errorf("dlq configuration is invalid: %w", err)
	}

	return nil
}

type DLQConfiguration struct {
	Enabled       bool
	Topic         string
	AutoProvision DLQAutoProvisionConfiguration
}

func (c DLQConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Topic == "" {
		return errors.New("topic name is required")
	}

	if err := c.AutoProvision.Validate(); err != nil {
		return fmt.Errorf("auto provision configuration is invalid: %w", err)
	}

	return nil
}

type DLQAutoProvisionConfiguration struct {
	Enabled    bool
	Partitions int
	Retention  time.Duration
}

func (c DLQAutoProvisionConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Partitions < 1 {
		return errors.New("partitions must be greater than 0")
	}

	if c.Retention <= 0 {
		return errors.New("retention must be greater than 0")
	}
	return nil
}

type RetryConfiguration struct {
	// MaxRetries is maximum number of times a retry will be attempted. Disabled if 0
	MaxRetries int
	// InitialInterval is the first interval between retries. Subsequent intervals will be scaled by Multiplier.
	InitialInterval time.Duration
	// MaxInterval sets the limit for the exponential backoff of retries. The interval will not be increased beyond MaxInterval.
	MaxInterval time.Duration
	// MaxElapsedTime sets the time limit of how long retries will be attempted. Disabled if 0.
	MaxElapsedTime time.Duration
}

func (c RetryConfiguration) Validate() error {
	if c.MaxRetries < 0 {
		return errors.New("max retries must be positive or 0")
	}

	if c.MaxElapsedTime < 0 {
		return errors.New("max elapsed time must be positive or 0")
	}

	if c.InitialInterval <= 0 {
		return errors.New("initial interval must be greater than 0")
	}

	if c.MaxInterval <= 0 {
		return errors.New("max interval must be greater than 0")
	}

	return nil
}

func ConfigureConsumer(v *viper.Viper, prefix string) {
	v.SetDefault(prefix+".processingTimeout", 30*time.Second)

	v.SetDefault(prefix+".retry.maxRetries", 0)
	v.SetDefault(prefix+".retry.initialInterval", 10*time.Millisecond)
	v.SetDefault(prefix+".retry.maxInterval", time.Second)
	v.SetDefault(prefix+".retry.maxElapsedTime", time.Minute)

	v.SetDefault(prefix+".dlq.enabled", true)
	v.SetDefault(prefix+".dlq.autoProvision.enabled", true)
	v.SetDefault(prefix+".dlq.autoProvision.partitions", 1)
	v.SetDefault(prefix+".dlq.autoProvision.retention", 90*24*time.Hour)
}

func ConfigureEvents(v *viper.Viper) {
	// TODO: after the system events are fully implemented, we should enable them by default
	v.SetDefault("events.enabled", false)
	v.SetDefault("events.systemEvents.enabled", true)
	v.SetDefault("events.systemEvents.topic", "om_sys.api_events")
	v.SetDefault("events.systemEvents.autoProvision.enabled", true)
	v.SetDefault("events.systemEvents.autoProvision.partitions", 4)

	v.SetDefault("events.ingestEvents.enabled", true)
	v.SetDefault("events.ingestEvents.topic", "om_sys.ingest_events")
	v.SetDefault("events.ingestEvents.autoProvision.enabled", true)
	v.SetDefault("events.ingestEvents.autoProvision.partitions", 8)
}
