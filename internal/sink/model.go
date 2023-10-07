package sink

import (
	"fmt"
)

type ProcessingControl int32

const (
	CONTINUE   ProcessingControl = 0
	DROP       ProcessingControl = 1
	DEADLETTER ProcessingControl = 2
	RETRY      ProcessingControl = 3
	FATAL      ProcessingControl = 4
)

type ProcessingError struct {
	Message           string
	ProcessingControl ProcessingControl
}

func (e *ProcessingError) Error() string {
	return fmt.Sprintf("processing error: %s", e.Message)
}

func NewProcessingError(msg string, control ProcessingControl) *ProcessingError {
	return &ProcessingError{
		Message:           msg,
		ProcessingControl: control,
	}
}
