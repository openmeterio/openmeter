package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type BillingWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
}

func (c BillingWorkerConfiguration) Validate() error {
	var errs []error

	if err := c.ConsumerConfiguration.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "consumer"))
	}

	return errors.Join(errs...)
}

func ConfigureBillingWorker(v *viper.Viper) {
	v.SetDefault("billing.worker.dlq.topic", "om_sys.billing_worker_dlq")
	v.SetDefault("billing.worker.consumerGroupName", "om_billing_worker")

	ConfigureConsumer(v, "billing.worker")
}
