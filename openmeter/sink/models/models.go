package models

import "github.com/openmeterio/openmeter/internal/sink/models"

type (
	SinkMessage      = models.SinkMessage
	ProcessingState  = models.ProcessingState
	ProcessingStatus = models.ProcessingStatus
)

const (
	OK      = models.OK
	DROP    = models.DROP
	INVALID = models.INVALID
)
