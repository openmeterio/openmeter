package consumer

import "github.com/openmeterio/openmeter/internal/notification/consumer"

type (
	Options  = consumer.Options
	Consumer = consumer.Consumer
)

func New(opts Options) (*Consumer, error) {
	return consumer.New(opts)
}
