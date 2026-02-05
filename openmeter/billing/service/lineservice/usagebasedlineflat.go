package lineservice

import (
	"fmt"
)

var _ Line = (*ubpFlatFeeLine)(nil)

type ubpFlatFeeLine struct {
	lineBase
}

func (l ubpFlatFeeLine) calculateDetailedLines() (newDetailedLinesInput, error) {
	pricer, err := newPricerFor(l.line)
	if err != nil {
		return nil, err
	}

	return pricer.Calculate(PricerCalculateInput(l))
}

func (l ubpFlatFeeLine) CalculateDetailedLines() error {
	newDetailedLinesInput, err := l.calculateDetailedLines()
	if err != nil {
		return err
	}

	if err := mergeDetailedLines(l.line, newDetailedLinesInput); err != nil {
		return fmt.Errorf("merging detailed lines: %w", err)
	}

	return nil
}

func (l *ubpFlatFeeLine) UpdateTotals() error {
	return UpdateTotalsFromDetailedLines(l.line)
}
