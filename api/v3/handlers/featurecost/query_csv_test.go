package featurecost

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestFeatureCostCSVRecords(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	row := cost.CostQueryRow{
		WindowStart: from,
		WindowEnd:   to,
		Usage:       alpacadecimal.RequireFromString("123456789012345.123456789"),
		Cost:        lo.ToPtr(alpacadecimal.RequireFromString("0.000000000123456789")),
		Currency:    "USD",
		Detail:      "Partially priced, missing \"model\"\npricing",
		Subject:     lo.ToPtr("subject-1"),
		CustomerID:  lo.ToPtr("customer-1"),
		GroupBy:     map[string]*string{"model": lo.ToPtr("model,one"), "type": nil},
	}
	missing := row
	missing.Cost = nil
	missing.Currency = ""
	missing.Detail = "Pricing unavailable"
	missing.Subject = nil
	missing.CustomerID = lo.ToPtr("unknown-customer")
	missing.GroupBy = nil
	c := customer.Customer{Key: lo.ToPtr("example")}
	c.Name = "Example, Inc."

	for _, tt := range []struct {
		name    string
		groupBy []string
		rows    []cost.CostQueryRow
		want    [][]string
	}{
		{
			name: "ungrouped preserves decimal precision and partial pricing detail",
			rows: []cost.CostQueryRow{row},
			want: [][]string{
				{"from", "to", "usage", "cost", "currency", "detail"},
				{"2024-01-01T00:00:00Z", "2024-01-01T01:00:00Z", row.Usage.String(), row.Cost.String(), "USD", row.Detail},
			},
		},
		{
			name:    "grouping keeps dimension order and missing values",
			groupBy: []string{"type", "customer_id", "model", "subject"},
			rows:    []cost.CostQueryRow{row, missing},
			want: [][]string{
				{"from", "to", "subject", "customer_id", "customer_key", "customer_name", "type", "model", "usage", "cost", "currency", "detail"},
				{"2024-01-01T00:00:00Z", "2024-01-01T01:00:00Z", "subject-1", "customer-1", "example", "Example, Inc.", "", "model,one", row.Usage.String(), row.Cost.String(), "USD", row.Detail},
				{"2024-01-01T00:00:00Z", "2024-01-01T01:00:00Z", "", "unknown-customer", "", "", "", "", row.Usage.String(), "", "", "Pricing unavailable"},
			},
		},
		{
			name:    "empty grouped result retains headers",
			groupBy: []string{"model", "customer_id"},
			want:    [][]string{{"from", "to", "customer_id", "customer_key", "customer_name", "model", "usage", "cost", "currency", "detail"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := &queryFeatureCostCSVResult{
				featureKey:    "tokens",
				groupBy:       tt.groupBy,
				rows:          tt.rows,
				customersByID: map[string]customer.Customer{"customer-1": c},
			}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			require.NoError(t, commonhttp.CSVResponseEncoder(t.Context(), recorder, req, response))
			require.Equal(t, "text/csv", recorder.Header().Get("Content-Type"))
			require.Equal(t, "attachment; filename=tokens-cost.csv", recorder.Header().Get("Content-Disposition"))
			records, err := csv.NewReader(recorder.Body).ReadAll()
			require.NoError(t, err)
			require.Equal(t, tt.want, records)
		})
	}
}

func TestQueryFeatureCostRepresentations(t *testing.T) {
	for _, representation := range []string{"json", "csv"} {
		t.Run(representation, func(t *testing.T) {
			// Given the same query and a cost row whose customer can be enriched.
			customers := &costQueryCustomerService{list: func(input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
				require.Equal(t, "test", input.Namespace)
				require.Equal(t, []string{"customer-1"}, input.CustomerIDs)
				c := customer.Customer{Key: lo.ToPtr("example")}
				c.ID = "customer-1"
				c.Name = "Example"
				return pagination.Result[customer.Customer]{Items: []customer.Customer{c}}, nil
			}}
			calls := 0
			h := New(
				func(context.Context) (string, error) { return "test", nil },
				&costQueryService{query: func(input cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
					calls++
					require.Equal(t, "test", input.Namespace)
					require.Equal(t, "feature-1", input.FeatureID)
					// Subject filtering implies grouping; duplicate dimensions are removed.
					require.Equal(t, []string{"model", "customer_id", "subject"}, input.QueryParams.GroupBy)
					require.Equal(t, []string{"subject-1"}, input.QueryParams.FilterSubject)
					return &cost.CostQueryResult{Rows: []cost.CostQueryRow{{
						WindowStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						WindowEnd:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
						Usage:       alpacadecimal.NewFromInt(100),
						Cost:        lo.ToPtr(alpacadecimal.RequireFromString("0.001")),
						Currency:    "USD",
						Subject:     lo.ToPtr("subject-1"),
						CustomerID:  lo.ToPtr("customer-1"),
						GroupBy:     map[string]*string{"model": lo.ToPtr("model-1")},
					}}}, nil
				}},
				&costQueryFeatureConnector{},
				&costQueryMeterService{},
				customers,
			)

			// When either representation executes through the HTTP handler.
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"group_by_dimensions":["model","customer_id","model"],"filters":{"dimensions":{"subject":{"eq":"subject-1"}}}}`))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			if representation == "csv" {
				h.QueryFeatureCostCSV().With("feature-1").ServeHTTP(recorder, req)
			} else {
				h.QueryFeatureCost().With("feature-1").ServeHTTP(recorder, req)
			}

			// Then both use the same validated service input and preserve the cost.
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.Equal(t, 1, calls)
			if representation == "csv" {
				records, err := csv.NewReader(recorder.Body).ReadAll()
				require.NoError(t, err)
				require.Equal(t, []string{"from", "to", "subject", "customer_id", "customer_key", "customer_name", "model", "usage", "cost", "currency", "detail"}, records[0])
				require.Equal(t, []string{"2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", "subject-1", "customer-1", "example", "Example", "model-1", "100", "0.001", "USD", ""}, records[1])
			} else {
				var result api.FeatureCostQueryResult
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
				require.Len(t, result.Data, 1)
				require.Equal(t, "100", result.Data[0].Usage)
				require.Equal(t, "0.001", result.Data[0].Cost.MustGet())
			}
		})
	}
}

func TestQueryFeatureCostCSVOptionalBodyAndValidation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		status int
	}{
		{name: "omitted body", status: http.StatusOK},
		{name: "empty object", body: `{}`, status: http.StatusOK},
		{name: "invalid dimension", body: `{"group_by_dimensions":["unknown"]}`, status: http.StatusBadRequest},
		{name: "malformed body", body: `{`, status: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			h := New(
				func(context.Context) (string, error) { return "test", nil },
				&costQueryService{query: func(cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
					calls++
					return nil, nil
				}},
				&costQueryFeatureConnector{}, &costQueryMeterService{}, nil,
			)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			h.QueryFeatureCostCSV().With("feature-1").ServeHTTP(recorder, req)
			require.Equal(t, tt.status, recorder.Code, recorder.Body.String())
			if tt.status == http.StatusOK {
				require.Equal(t, 1, calls)
				require.Equal(t, "from,to,usage,cost,currency,detail\n", recorder.Body.String())
			} else {
				require.Zero(t, calls)
				require.Contains(t, recorder.Header().Get("Content-Type"), "application/problem+json")
			}
		})
	}
}

type costQueryService struct {
	cost.Service
	query func(cost.QueryFeatureCostInput) (*cost.CostQueryResult, error)
}

func (s *costQueryService) QueryFeatureCost(_ context.Context, input cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
	return s.query(input)
}

type costQueryFeatureConnector struct{ feature.FeatureConnector }

func (*costQueryFeatureConnector) GetFeature(context.Context, string, string, feature.IncludeArchivedFeature) (*feature.Feature, error) {
	return &feature.Feature{ID: "feature-1", Key: "tokens", MeterID: lo.ToPtr("meter-1")}, nil
}

type costQueryMeterService struct{ meter.Service }

func (*costQueryMeterService) GetMeterByIDOrSlug(context.Context, meter.GetMeterInput) (meter.Meter, error) {
	return meter.Meter{Key: "tokens", GroupBy: map[string]string{"model": "$.model"}}, nil
}

type costQueryCustomerService struct {
	customer.Service
	list func(customer.ListCustomersInput) (pagination.Result[customer.Customer], error)
}

func (s *costQueryCustomerService) ListCustomers(_ context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	return s.list(input)
}
