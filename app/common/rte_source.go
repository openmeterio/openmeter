package common

import (
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewRTESourcePublisher(recorder *shadow.Recorder, logger *slog.Logger) eventbus.AfterCommitPublisher {
	if recorder == nil {
		return nil
	}

	return eventbus.NewAfterCommitPublisher(recorder.TryPublishSource, logger)
}
