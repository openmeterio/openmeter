package sink

import (
	"fmt"
)

type ProcessingControl int32

const (
	DROP       ProcessingControl = 0
	DEADLETTER ProcessingControl = 1
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
