package common

import (
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// RTESourcePublisherWithMetrics sends committed facts through the RTE recorder.
// The recorder counts sends, failures and drops on its best-effort queue.
// Business events use the durable outbox and have a separate delivery lifecycle.
type RTESourcePublisherWithMetrics struct {
	EnqueueAfterCommit eventbus.AfterCommitEnqueuer
}

func NewRTESourcePublisherWithMetrics(recorder *shadow.Recorder, logger *slog.Logger) RTESourcePublisherWithMetrics {
	if recorder == nil {
		return RTESourcePublisherWithMetrics{}
	}

	return RTESourcePublisherWithMetrics{
		EnqueueAfterCommit: eventbus.NewAfterCommitEnqueuer(recorder.TryPublishSource, logger),
	}
}
