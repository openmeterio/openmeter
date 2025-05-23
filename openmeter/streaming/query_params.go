package streaming

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	Cachable       bool
	ClientID       *string
	From           *time.Time
	To             *time.Time
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
}

// Hash returns a deterministic hash for the QueryParams.
// It implements the hasher.Hasher interface.
func (p *QueryParams) Hash() hasher.Hash {
	h := xxhash.New()

	// Hash From
	if p.From != nil {
		_, _ = h.WriteString(p.From.UTC().Format(time.RFC3339))
	}

	// Hash To
	if p.To != nil {
		_, _ = h.WriteString(p.To.UTC().Format(time.RFC3339))
	}

	// Hash FilterSubject (sort for determinism)
	if len(p.FilterSubject) > 0 {
		sorted := make([]string, len(p.FilterSubject))
		copy(sorted, p.FilterSubject)
		sort.Strings(sorted)
		_, _ = h.WriteString(strings.Join(sorted, ","))
	}

	// Hash FilterGroupBy (sort keys and values for determinism)
	if len(p.FilterGroupBy) > 0 {
		keys := make([]string, 0, len(p.FilterGroupBy))
		for k := range p.FilterGroupBy {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			_, _ = h.WriteString(k)
			values := make([]string, len(p.FilterGroupBy[k]))
			copy(values, p.FilterGroupBy[k])
			sort.Strings(values)
			_, _ = h.WriteString(strings.Join(values, ","))
		}
	}

	// Hash GroupBy (sort for determinism)
	if len(p.GroupBy) > 0 {
		sorted := make([]string, len(p.GroupBy))
		copy(sorted, p.GroupBy)
		sort.Strings(sorted)
		_, _ = h.WriteString(strings.Join(sorted, ","))
	}

	// Hash WindowSize
	if p.WindowSize != nil {
		_, _ = h.WriteString(string(*p.WindowSize))
	}

	// Hash WindowTimeZone
	if p.WindowTimeZone != nil {
		_, _ = h.WriteString(p.WindowTimeZone.String())
	}

	return h.Sum64()
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate() error {
	var errs []error

	if p.ClientID != nil && len(*p.ClientID) == 0 {
		return errors.New("client id cannot be empty")
	}

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return nil
}
