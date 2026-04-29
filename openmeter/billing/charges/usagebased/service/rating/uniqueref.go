package rating

import (
	"fmt"
	"strings"
	"time"

	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// formatDetailedLineChildUniqueReferenceID turns a rating-local detailed line key
// into the persisted child reference by appending the rated service period in UTC.
func formatDetailedLineChildUniqueReferenceID(id string, servicePeriod timeutil.ClosedPeriod) string {
	return fmt.Sprintf(
		"%s@[%s..%s]",
		id,
		servicePeriod.From.UTC().Format(time.RFC3339),
		servicePeriod.To.UTC().Format(time.RFC3339),
	)
}

// withServicePeriodInDetailedLineChildUniqueReferenceIDs rewrites the generated
// detailed lines so each child unique reference carries the charge service period.
func withServicePeriodInDetailedLineChildUniqueReferenceIDs(lines billingrating.DetailedLines, servicePeriod timeutil.ClosedPeriod) billingrating.DetailedLines {
	out := make(billingrating.DetailedLines, len(lines))

	for idx, line := range lines {
		line.ChildUniqueReferenceID = formatDetailedLineChildUniqueReferenceID(line.ChildUniqueReferenceID, servicePeriod)
		out[idx] = line
	}

	return out
}

type childUniqueReferenceIDParts struct {
	ReferenceID string
	Period      timeutil.ClosedPeriod
}

func parseChildUniqueReferenceID(id string) (childUniqueReferenceIDParts, error) {
	idx := strings.LastIndex(id, "@")
	if idx < 0 {
		return childUniqueReferenceIDParts{}, fmt.Errorf("child unique reference id %q is missing @ period suffix", id)
	}

	base := id[:idx]
	suffix := id[idx:]

	if !strings.HasPrefix(suffix, "@[") || !strings.HasSuffix(suffix, "]") {
		return childUniqueReferenceIDParts{}, fmt.Errorf("child unique reference id %q has invalid period suffix", id)
	}

	periodRange := strings.TrimSuffix(strings.TrimPrefix(suffix, "@["), "]")
	fromRaw, toRaw, ok := strings.Cut(periodRange, "..")
	if !ok {
		return childUniqueReferenceIDParts{}, fmt.Errorf("child unique reference id %q has invalid [RFC3339..RFC3339] period suffix", id)
	}

	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return childUniqueReferenceIDParts{}, fmt.Errorf("child unique reference id %q has invalid period start: %w", id, err)
	}

	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return childUniqueReferenceIDParts{}, fmt.Errorf("child unique reference id %q has invalid period end: %w", id, err)
	}

	return childUniqueReferenceIDParts{
		ReferenceID: base,
		Period: timeutil.ClosedPeriod{
			From: from,
			To:   to,
		},
	}, nil
}
