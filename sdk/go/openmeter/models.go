package openmeter

import "time"

// Numeric represents an arbitrary-precision number. The API encodes it as a
// decimal string (e.g. "12.3456") to avoid float precision loss, so the SDK
// surfaces it as a string. Parse with a decimal library when arithmetic is needed.
type Numeric = string

// MeterAggregation is the aggregation type a meter applies to matched events.
type MeterAggregation string

const (
	MeterAggregationSum         MeterAggregation = "sum"
	MeterAggregationCount       MeterAggregation = "count"
	MeterAggregationUniqueCount MeterAggregation = "unique_count"
	MeterAggregationAvg         MeterAggregation = "avg"
	MeterAggregationMin         MeterAggregation = "min"
	MeterAggregationMax         MeterAggregation = "max"
	MeterAggregationLatest      MeterAggregation = "latest"
)

// Meter is a configuration that defines how to match and aggregate events.
type Meter struct {
	ID          string           `json:"id"`
	Key         string           `json:"key"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Aggregation MeterAggregation `json:"aggregation"`
	EventType   string           `json:"event_type"`
	// EventsFrom, when set, is the date from which the meter includes events.
	EventsFrom *time.Time `json:"events_from,omitempty"`
	// ValueProperty is a JSONPath expression extracting the aggregated value from
	// the event data. Ignored for count aggregation.
	ValueProperty *string `json:"value_property,omitempty"`
	// Dimensions maps group-by dimension names to JSONPath expressions.
	Dimensions map[string]string `json:"dimensions,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	DeletedAt  *time.Time        `json:"deleted_at,omitempty"`
}

// PageMeta carries the pagination query parameters echoed back plus the total
// count. The API types these as JSON numbers; the SDK exposes them as ints.
type PageMeta struct {
	Number int `json:"number"`
	Size   int `json:"size"`
	Total  int `json:"total"`
}

// PaginatedMeta wraps the pagination information of a page-paginated response.
type PaginatedMeta struct {
	Page PageMeta `json:"page"`
}

// MeterPagePaginatedResponse is a page of meters plus pagination metadata.
type MeterPagePaginatedResponse struct {
	Data []Meter       `json:"data"`
	Meta PaginatedMeta `json:"meta"`
}

// MeterQueryGranularity is the size of the time buckets a query groups usage into.
type MeterQueryGranularity string

const (
	MeterQueryGranularityMinute MeterQueryGranularity = "PT1M"
	MeterQueryGranularityHour   MeterQueryGranularity = "PT1H"
	MeterQueryGranularityDay    MeterQueryGranularity = "P1D"
	MeterQueryGranularityMonth  MeterQueryGranularity = "P1M"
)

// QueryFilterStringMapItem is a per-dimension filter in a meter query. For the
// reserved subject and customer_id dimensions only Eq/In are supported.
type QueryFilterStringMapItem struct {
	Eq        *string  `json:"eq,omitempty"`
	Neq       *string  `json:"neq,omitempty"`
	In        []string `json:"in,omitempty"`
	Nin       []string `json:"nin,omitempty"`
	Contains  *string  `json:"contains,omitempty"`
	Ncontains *string  `json:"ncontains,omitempty"`
	Exists    *bool    `json:"exists,omitempty"`
}

// MeterQueryFilters filters a meter query by dimension values.
type MeterQueryFilters struct {
	Dimensions map[string]QueryFilterStringMapItem `json:"dimensions,omitempty"`
}

// MeterQueryRequest is the POST body for querying a meter for usage.
type MeterQueryRequest struct {
	From        *time.Time             `json:"from,omitempty"`
	To          *time.Time             `json:"to,omitempty"`
	Granularity *MeterQueryGranularity `json:"granularity,omitempty"`
	// TimeZone is an IANA Time Zone Database name used to align time buckets.
	// Defaults to UTC when unset.
	TimeZone          *string            `json:"time_zone,omitempty"`
	GroupByDimensions []string           `json:"group_by_dimensions,omitempty"`
	Filters           *MeterQueryFilters `json:"filters,omitempty"`
}

// MeterQueryRow is one aggregated bucket of a meter query result.
type MeterQueryRow struct {
	Value Numeric   `json:"value"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	// Dimensions holds the group-by values for this row. subject and customer_id
	// are reserved dimension keys.
	Dimensions map[string]string `json:"dimensions"`
}

// MeterQueryResult is the JSON result of a meter query.
type MeterQueryResult struct {
	From *time.Time      `json:"from,omitempty"`
	To   *time.Time      `json:"to,omitempty"`
	Data []MeterQueryRow `json:"data"`
}
