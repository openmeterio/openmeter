package usagebased

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// StripServicePeriodFromUniqueReferenceID returns cloned detailed lines where
// ChildUniqueReferenceID is stripped at the first service-period suffix marker.
func (l DetailedLines) StripServicePeriodFromUniqueReferenceID() (DetailedLines, error) {
	return slicesx.MapWithErr(l, func(line DetailedLine) (DetailedLine, error) {
		referenceID, err := stripServicePeriodFromUniqueReferenceID(line.ChildUniqueReferenceID)
		if err != nil {
			return DetailedLine{}, fmt.Errorf("stripping service period from unique reference id: %w", err)
		}

		line = line.Clone()
		line.ChildUniqueReferenceID = referenceID
		return line, nil
	})
}

// WithServicePeriodFromUniqueReferenceID returns cloned detailed lines where
// ChildUniqueReferenceID contains the line's ServicePeriod persistence suffix.
func (l DetailedLines) WithServicePeriodFromUniqueReferenceID() (DetailedLines, error) {
	lines, err := l.StripServicePeriodFromUniqueReferenceID()
	if err != nil {
		return nil, err
	}

	return lo.Map(lines, func(line DetailedLine, _ int) DetailedLine {
		line.ChildUniqueReferenceID = formatServicePeriodInUniqueReferenceID(line.ChildUniqueReferenceID, line.ServicePeriod)
		return line
	}), nil
}

func stripServicePeriodFromUniqueReferenceID(id string) (string, error) {
	idx := strings.Index(id, "@[")
	if idx < 0 {
		return id, nil
	}

	base := id[:idx]
	if base == "" {
		return "", fmt.Errorf("child unique reference id %q is missing base reference id", id)
	}

	return base, nil
}

func formatServicePeriodInUniqueReferenceID(id string, servicePeriod timeutil.ClosedPeriod) string {
	return fmt.Sprintf(
		"%s@[%s..%s]",
		id,
		servicePeriod.From.UTC().Format(time.RFC3339),
		servicePeriod.To.UTC().Format(time.RFC3339),
	)
}
