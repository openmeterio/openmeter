package config

import (
	"errors"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type EventsConfiguration struct {
	SystemEvents        EventSubsystemConfiguration
	IngestEvents        EventSubsystemConfiguration
	BalanceWorkerEvents EventSubsystemConfiguration
}

func (c EventsConfiguration) Validate() error {
	var errs []error

	if err := c.SystemEvents.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "system events"))
	}

	if err := c.IngestEvents.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "ingest events"))
	}

	if err := c.BalanceWorkerEvents.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "balance worker events"))
	}

	// Validate topic uniqueness
	uniqueTopics := lo.Uniq([]string{c.SystemEvents.Topic, c.IngestEvents.Topic, c.BalanceWorkerEvents.Topic})
	if len(uniqueTopics) != 3 {
		errs = append(errs, errors.New("topic names must be unique"))
	}

	return errors.Join(errs...)
}

func (c EventsConfiguration) EventBusTopicMapping() eventbus.TopicMapping {
	return eventbus.TopicMapping{
		IngestEventsTopic:        c.IngestEvents.Topic,
		SystemEventsTopic:        c.SystemEvents.Topic,
		BalanceWorkerEventsTopic: c.BalanceWorkerEvents.Topic,
	}
}

type EventSubsystemConfiguration struct {
	Topic string

	AutoProvision AutoProvisionConfiguration
}

func (c EventSubsystemConfiguration) Validate() error {
	var errs []error

	if c.Topic == "" {
		errs = append(errs, errors.New("topic name is required"))
	}

	if err := c.AutoProvision.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "auto provision"))
	}

	return errors.Join(errs...)
}

type AutoProvisionConfiguration struct {
	Enabled      bool
	Partitions   int
	DLQRetention time.Duration
}

func (c AutoProvisionConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error

	if c.Partitions < 1 {
		errs = append(errs, errors.New("partitions must be greater than 0"))
	}

	return errors.Join(errs...)
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
	var errs []error

	if c.ProcessingTimeout < 0 {
		errs = append(errs, errors.New("processing timeout must be positive or 0"))
	}

	if c.ConsumerGroupName == "" {
		errs = append(errs, errors.New("consumer group name is required"))
	}

	if err := c.Retry.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "retry"))
	}

	if err := c.DLQ.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "dlq"))
	}

	return errors.Join(errs...)
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

	var errs []error

	if c.Topic == "" {
		errs = append(errs, errors.New("topic name is required"))
	}

	if err := c.AutoProvision.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "auto provision"))
	}

	return errors.Join(errs...)
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

	var errs []error

	if c.Partitions < 1 {
		errs = append(errs, errors.New("partitions must be greater than 0"))
	}

	if c.Retention <= 0 {
		errs = append(errs, errors.New("retention must be greater than 0"))
	}

	return errors.Join(errs...)
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
	var errs []error

	if c.MaxRetries < 0 {
		errs = append(errs, errors.New("max retries must be positive or 0"))
	}

	if c.MaxElapsedTime < 0 {
		errs = append(errs, errors.New("max elapsed time must be positive or 0"))
	}

	if c.InitialInterval <= 0 {
		errs = append(errs, errors.New("initial interval must be greater than 0"))
	}

	if c.MaxInterval <= 0 {
		errs = append(errs, errors.New("max interval must be greater than 0"))
	}

	if c.MaxElapsedTime > 0 && c.MaxRetries == 0 {
		errs = append(errs, errors.New("max elapsed time is set but max retries is disabled, set max retries to enable retries"))
	}

	return errors.Join(errs...)
}

func ConfigureConsumer(v *viper.Viper, prefix string) {
	v.SetDefault(prefix+".processingTimeout", 30*time.Second)

	v.SetDefault(prefix+".retry.maxRetries", 10)
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
	v.SetDefault("events.systemEvents.topic", "om_sys.api_events")
	v.SetDefault("events.systemEvents.autoProvision.enabled", true)
	v.SetDefault("events.systemEvents.autoProvision.partitions", 4)

	v.SetDefault("events.ingestEvents.topic", "om_sys.ingest_events")
	v.SetDefault("events.ingestEvents.autoProvision.enabled", true)
	v.SetDefault("events.ingestEvents.autoProvision.partitions", 8)

	v.SetDefault("events.balanceWorkerEvents.topic", "om_sys.balance_worker_events")
	v.SetDefault("events.balanceWorkerEvents.autoProvision.enabled", true)
	v.SetDefault("events.balanceWorkerEvents.autoProvision.partitions", 4)
}
