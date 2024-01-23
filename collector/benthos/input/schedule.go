package input

import (
	"context"
	"time"

	"github.com/benthosdev/benthos/v4/public/service"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import kubernetes auth plugins
)

// TODO: add batching config and policy

func scheduleInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Utility").
		Summary("Reads messages from a child input on a schedule").
		Description("").
		Fields(
			service.NewInputField("input").
				Description("The child input to consume from."),
			service.NewDurationField("interval").
				Description("Interval at which the child input should be consumed."),
		)
}

func init() {
	err := service.RegisterBatchInput("schedule", scheduleInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		return newScheduleInput(conf)
	})
	if err != nil {
		panic(err)
	}
}

type scheduleInput struct {
	child    *service.OwnedInput
	timer    *time.Ticker
	interval time.Duration
}

func newScheduleInput(conf *service.ParsedConfig) (*scheduleInput, error) {
	interval, err := conf.FieldDuration("interval")
	if err != nil {
		return nil, err
	}

	child, err := conf.FieldInput("input")
	if err != nil {
		return nil, err
	}

	return &scheduleInput{
		child:    child,
		timer:    time.NewTicker(interval),
		interval: interval,
	}, nil
}

func (in *scheduleInput) Connect(_ context.Context) error {
	return nil
}

func (in *scheduleInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	var t time.Time

	select {
	case v := <-in.timer.C:
		t = v
	default:
		return nil, func(context.Context, error) error { return nil }, nil
	}

	batch, ackFunc, err := in.child.ReadBatch(ctx)
	if err != nil {
		return batch, ackFunc, err
	}

	// This should never error
	_ = batch.WalkWithBatchedErrors(func(_ int, msg *service.Message) error {
		msg.MetaSet("schedule_time", t.Format(time.RFC3339))
		msg.MetaSet("schedule_interval", in.interval.String())

		return nil
	})

	return batch, ackFunc, err
}

func (in *scheduleInput) Close(ctx context.Context) error {
	in.timer.Stop()

	return in.child.Close(ctx)
}
