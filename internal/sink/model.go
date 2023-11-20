package sink

type ProcessingControl int32

const (
	DROP    ProcessingControl = 0
	INVALID ProcessingControl = 1
)

type ProcessingError struct {
	Message           string
	ProcessingControl ProcessingControl
}

func (e *ProcessingError) Error() string {
	return e.Message
}

func NewProcessingError(msg string, control ProcessingControl) *ProcessingError {
	return &ProcessingError{
		Message:           msg,
		ProcessingControl: control,
	}
}
