package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type BalanceWorkerConfiguration struct {
	DLQ               DLQConfiguration
	Retry             RetryConfiguration
	ConsumerGroupName string
}

func (c BalanceWorkerConfiguration) Validate() error {
	if err := c.DLQ.Validate(); err != nil {
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
	v.SetDefault("balanceWorker.dlq.enabled", true)
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.dlq.autoProvision.enabled", true)
	v.SetDefault("balanceWorker.dlq.autoProvision.partitions", 1)

	v.SetDefault("balanceWorker.dlq.throttle.enabled", true)
	// Let's throttle poision queue to 10 messages per second
	v.SetDefault("balanceWorker.dlq.throttle.count", 10)
	v.SetDefault("balanceWorker.dlq.throttle.duration", time.Second)

	v.SetDefault("balanceWorker.retry.maxRetries", 5)
	v.SetDefault("balanceWorker.retry.initialInterval", 100*time.Millisecond)

	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")
}
