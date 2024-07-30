package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type BalanceWorkerConfiguration struct {
	PoisionQueue      PoisionQueueConfiguration
	Retry             RetryConfiguration
	ConsumerGroupName string
}

func (c BalanceWorkerConfiguration) Validate() error {
	if err := c.PoisionQueue.Validate(); err != nil {
		return fmt.Errorf("poision queue: %w", err)
	}

	if err := c.Retry.Validate(); err != nil {
		return fmt.Errorf("retry: %w", err)
	}

	if c.ConsumerGroupName == "" {
		return errors.New("consumer group name is required")
	}
	return nil
}

func ConfigureBalanceWorker(v *viper.Viper) {
	v.SetDefault("balanceWorker.poisionQueue.enabled", true)
	v.SetDefault("balanceWorker.poisionQueue.topic", "om_sys.balance_worker_poision")
	v.SetDefault("balanceWorker.poisionQueue.autoProvision.enabled", true)
	v.SetDefault("balanceWorker.poisionQueue.autoProvision.partitions", 1)

	v.SetDefault("balanceWorker.poisionQueue.throttle.enabled", true)
	// Let's throttle poision queue to 10 messages per second
	v.SetDefault("balanceWorker.poisionQueue.throttle.count", 10)
	v.SetDefault("balanceWorker.poisionQueue.throttle.duration", time.Second)

	v.SetDefault("balanceWorker.retry.maxRetries", 5)
	v.SetDefault("balanceWorker.retry.initialInterval", 100*time.Millisecond)

	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")
}
