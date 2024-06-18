package sink

import "fmt"

type ProcessingState int8

func (c ProcessingState) String() string {
	var state string
	switch c {
	case OK:
		state = "ok"
	case INVALID:
		state = "invalid"
	case DROP:
		state = "drop"
	default:
		state = fmt.Sprintf("unknown(%d)", c)
	}

	return state
}

const (
	OK ProcessingState = iota
	DROP
	INVALID
)

type ProcessingStatus struct {
	State ProcessingState
	Error error
}
