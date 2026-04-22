package rating

import (
	"fmt"
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
