package recognizer

import "context"

// NoopService returns zero results for all operations.
// Useful for tests that don't exercise revenue recognition.
type NoopService struct{}

var _ Service = NoopService{}

func (NoopService) RecognizeEarnings(context.Context, RecognizeEarningsInput) (RecognizeEarningsResult, error) {
	return RecognizeEarningsResult{}, nil
}
