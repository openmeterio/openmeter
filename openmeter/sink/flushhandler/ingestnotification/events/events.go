package events

import "github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification/events"

const (
	EventSubsystem = events.EventSubsystem
)

var EventVersionSubsystem = events.EventVersionSubsystem

type (
	IngestEventData    = events.IngestEventData
	EventBatchedIngest = events.EventBatchedIngest
)
