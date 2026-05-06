package test_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
	apiv3test "github.com/openmeterio/openmeter/api/v3/test"
)

// fieldFiltersTarget mirrors the FieldFilters schema in openapi.Test.yaml so
// filters.Parse can populate it from incoming query strings.
type fieldFiltersTarget struct {
	Boolean     *filters.FilterBoolean     `json:"boolean,omitempty"`
	Numeric     *filters.FilterNumeric     `json:"numeric,omitempty"`
	String      *filters.FilterString      `json:"string,omitempty"`
	StringExact *filters.FilterStringExact `json:"string_exact,omitempty"`
	ULID        *filters.FilterULID        `json:"ulid,omitempty"`
	DateTime    *filters.FilterDateTime    `json:"datetime,omitempty"`
	Labels      *filters.FilterString      `json:"labels,omitempty"`
}

// validatorErrorResponse mirrors the AIP-style error body produced by
// oasmiddleware.OasValidationErrorHook so tests can assert on individual fields.
type validatorErrorResponse struct {
	Type              string                      `json:"type"`
	Status            int                         `json:"status"`
	Title             string                      `json:"title"`
	Detail            string                      `json:"detail"`
	InvalidParameters []validatorInvalidParameter `json:"invalid_parameters"`
}

type validatorInvalidParameter struct {
	Field  string `json:"field"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
	Source string `json:"source"`
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	doc, err := openapi3.NewLoader().LoadFromData(apiv3test.OpenAPITestSpec)
	require.NoError(t, err)

	validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
		RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
		RouteValidationErrorHook: func(err error, w http.ResponseWriter, req *http.Request) bool {
			return oasmiddleware.OasValidationErrorHook(req.Context(), err, w, req)
		},
		FilterOptions: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			MultiError:         true,
		},
	}))
	r.Get("/field-filters", handler)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return srv
}

func noopHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func parseHandler(w http.ResponseWriter, r *http.Request) {
	var target fieldFiltersTarget
	if err := filters.Parse(r.URL.Query(), &target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(target)
}

// TestFieldFilterValidation exercises the kin-openapi validator against the
// generated FieldFilters schemas. Every filter type is sent in both short
// (filter[field]=value) and object (filter[field][op]=value) form.
func TestFieldFilterValidation(t *testing.T) {
	srv := newTestServer(t, noopHandler)

	cases := []struct {
		name string
		// query is the raw query string sent to /field-filters.
		query string
		// wantStatus is 204 for accepted requests, 400 for rejected.
		wantStatus int
		// wantField, wantRule, wantReasonSubstr are asserted on the validator's
		// invalid_parameters[0] entry when wantStatus == 400. Empty string skips
		// the assertion for that field.
		wantField        string
		wantRule         string
		wantReasonSubstr string
	}{
		// Empty query — no filter parameter is supplied; the schema marks
		// filter as optional, so the request must be accepted.
		{name: "empty filter", query: "", wantStatus: http.StatusNoContent},

		// BooleanFieldFilter
		{name: "boolean valid short", query: "filter[boolean]=true", wantStatus: http.StatusNoContent},
		{name: "boolean valid short false", query: "filter[boolean]=false", wantStatus: http.StatusNoContent},
		{name: "boolean valid eq true", query: "filter[boolean][eq]=true", wantStatus: http.StatusNoContent},
		{name: "boolean valid eq false", query: "filter[boolean][eq]=false", wantStatus: http.StatusNoContent},

		// NumericFieldFilter — every documented operator
		{name: "numeric valid short", query: "filter[numeric]=42", wantStatus: http.StatusNoContent},
		{name: "numeric valid eq", query: "filter[numeric][eq]=42", wantStatus: http.StatusNoContent},
		{name: "numeric valid neq", query: "filter[numeric][neq]=7", wantStatus: http.StatusNoContent},
		{name: "numeric valid lt", query: "filter[numeric][lt]=10.5", wantStatus: http.StatusNoContent},
		{name: "numeric valid lte", query: "filter[numeric][lte]=10.5", wantStatus: http.StatusNoContent},
		{name: "numeric valid gt", query: "filter[numeric][gt]=0", wantStatus: http.StatusNoContent},
		{name: "numeric valid gte", query: "filter[numeric][gte]=0", wantStatus: http.StatusNoContent},
		{name: "numeric valid oeq", query: "filter[numeric][oeq]=1,2,3", wantStatus: http.StatusNoContent},
		{name: "numeric valid negative", query: "filter[numeric][eq]=-3.14", wantStatus: http.StatusNoContent},

		// StringFieldFilter — every documented operator
		{name: "string valid short", query: "filter[string]=hello", wantStatus: http.StatusNoContent},
		{name: "string valid eq", query: "filter[string][eq]=hello", wantStatus: http.StatusNoContent},
		{name: "string valid neq", query: "filter[string][neq]=hello", wantStatus: http.StatusNoContent},
		{name: "string valid contains", query: "filter[string][contains]=foo", wantStatus: http.StatusNoContent},
		{name: "string valid ocontains", query: "filter[string][ocontains]=a,b", wantStatus: http.StatusNoContent},
		{name: "string valid oeq", query: "filter[string][oeq]=a,b,c", wantStatus: http.StatusNoContent},
		{name: "string valid gt", query: "filter[string][gt]=alpha", wantStatus: http.StatusNoContent},
		{name: "string valid gte", query: "filter[string][gte]=alpha", wantStatus: http.StatusNoContent},
		{name: "string valid lt", query: "filter[string][lt]=zeta", wantStatus: http.StatusNoContent},
		{name: "string valid lte", query: "filter[string][lte]=zeta", wantStatus: http.StatusNoContent},
		{name: "string valid exists true", query: "filter[string][exists]=true", wantStatus: http.StatusNoContent},
		{name: "string valid exists false", query: "filter[string][exists]=false", wantStatus: http.StatusNoContent},

		// StringFieldFilterExact — limited operator set
		{name: "string_exact valid short", query: "filter[string_exact]=hello", wantStatus: http.StatusNoContent},
		{name: "string_exact valid eq", query: "filter[string_exact][eq]=hello", wantStatus: http.StatusNoContent},
		{name: "string_exact valid neq", query: "filter[string_exact][neq]=hello", wantStatus: http.StatusNoContent},
		{name: "string_exact valid oeq", query: "filter[string_exact][oeq]=a,b,c", wantStatus: http.StatusNoContent},

		// ULIDFieldFilter
		{name: "ulid valid short", query: "filter[ulid]=01G65Z755AFWAKHE12NY0CQ9FH", wantStatus: http.StatusNoContent},
		{name: "ulid valid eq", query: "filter[ulid][eq]=01G65Z755AFWAKHE12NY0CQ9FH", wantStatus: http.StatusNoContent},
		{name: "ulid valid neq", query: "filter[ulid][neq]=01G65Z755AFWAKHE12NY0CQ9FH", wantStatus: http.StatusNoContent},
		{name: "ulid valid oeq", query: "filter[ulid][oeq]=01G65Z755AFWAKHE12NY0CQ9FH,01G65Z755AFWAKHE12NY0CQ9FJ", wantStatus: http.StatusNoContent},
		// Pattern violation — ULID schema enforces ^[0-7][0-9A-HJKMNP-TV-Z]{25}$.
		// kin-openapi collapses the underlying pattern miss into a generic
		// oneOf-failure entry on the parent union; we still get the right
		// field name back, just not the per-branch rule.
		{
			name:             "ulid invalid short pattern",
			query:            "filter[ulid]=not-a-ulid",
			wantStatus:       http.StatusBadRequest,
			wantField:        "ulid",
			wantRule:         "oneOf",
			wantReasonSubstr: "doesn't match any schema",
		},
		{
			name:             "ulid invalid eq pattern",
			query:            "filter[ulid][eq]=not-a-ulid",
			wantStatus:       http.StatusBadRequest,
			wantField:        "ulid",
			wantRule:         "oneOf",
			wantReasonSubstr: "doesn't match any schema",
		},

		// DateTimeFieldFilter
		{name: "datetime valid short", query: "filter[datetime]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		{name: "datetime valid eq", query: "filter[datetime][eq]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		{name: "datetime valid lt", query: "filter[datetime][lt]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		{name: "datetime valid lte", query: "filter[datetime][lte]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		{name: "datetime valid gt", query: "filter[datetime][gt]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		{name: "datetime valid gte", query: "filter[datetime][gte]=2024-01-01T00:00:00Z", wantStatus: http.StatusNoContent},
		// Format violation — DateTime schema enforces RFC-3339; like the ULID
		// pattern miss, the per-branch failure surfaces as a oneOf collapse.
		{
			name:             "datetime invalid short format",
			query:            "filter[datetime]=not-a-datetime",
			wantStatus:       http.StatusBadRequest,
			wantField:        "datetime",
			wantRule:         "oneOf",
			wantReasonSubstr: "doesn't match any schema",
		},
		{
			name:             "datetime invalid eq format",
			query:            "filter[datetime][eq]=not-a-datetime",
			wantStatus:       http.StatusBadRequest,
			wantField:        "datetime",
			wantRule:         "oneOf",
			wantReasonSubstr: "doesn't match any schema",
		},

		// LabelsFieldFilter
		{name: "labels valid short", query: "filter[labels]=team-a", wantStatus: http.StatusNoContent},
		{name: "labels valid eq", query: "filter[labels][eq]=team-a", wantStatus: http.StatusNoContent},
		{name: "labels valid contains", query: "filter[labels][contains]=team", wantStatus: http.StatusNoContent},
		{name: "labels valid ocontains", query: "filter[labels][ocontains]=a,b", wantStatus: http.StatusNoContent},
		{name: "labels valid oeq", query: "filter[labels][oeq]=a,b", wantStatus: http.StatusNoContent},
		{name: "labels valid neq", query: "filter[labels][neq]=team-a", wantStatus: http.StatusNoContent},

		// Multiple filters in one request — independent fields can be combined.
		{
			name:       "combined boolean+numeric+string",
			query:      "filter[boolean][eq]=true&filter[numeric][gt]=5&filter[string][contains]=foo",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "combined ulid+datetime",
			query:      "filter[ulid][eq]=01G65Z755AFWAKHE12NY0CQ9FH&filter[datetime][lt]=2024-01-01T00:00:00Z",
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := srv.URL + "/field-filters"
			if tc.query != "" {
				url += "?" + tc.query
			}

			resp, err := http.Get(url)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			require.Equalf(t, tc.wantStatus, resp.StatusCode, "query=%q body=%s", tc.query, string(body))

			if tc.wantStatus != http.StatusBadRequest {
				return
			}

			var got validatorErrorResponse
			require.NoError(t, json.Unmarshal(body, &got), "validator error body must be JSON, got %q", string(body))
			require.Equal(t, http.StatusBadRequest, got.Status)
			require.Equal(t, "Bad Request", got.Title)
			require.NotEmpty(t, got.InvalidParameters, "expected at least one invalid_parameters entry")

			ip := got.InvalidParameters[0]
			assert.Equal(t, "query", ip.Source)
			if tc.wantField != "" {
				assert.Equal(t, tc.wantField, ip.Field)
			}
			if tc.wantRule != "" {
				assert.Equal(t, tc.wantRule, ip.Rule)
			}
			if tc.wantReasonSubstr != "" {
				assert.Contains(t, ip.Reason, tc.wantReasonSubstr)
			}
		})
	}
}

// TestFieldFilterParse runs each request through the OAS validator and then
// through filters.Parse, asserting both that the request validates and that
// the parser produces the expected typed Go struct. Negative cases assert on
// the parser's error message body.
func TestFieldFilterParse(t *testing.T) {
	srv := newTestServer(t, parseHandler)

	dt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	ulid1 := "01G65Z755AFWAKHE12NY0CQ9FH"
	ulid2 := "01G65Z755AFWAKHE12NY0CQ9FJ"

	cases := []struct {
		name      string
		query     string
		wantParse fieldFiltersTarget
		// wantErr indicates the parser is expected to reject the request with 400.
		wantErr bool
		// wantBodySubstr, when set, must appear in the response body. For success
		// cases this checks the JSON output; for error cases it checks the parser
		// error message.
		wantBodySubstr string
	}{
		// BooleanFieldFilter
		{
			name:      "boolean short",
			query:     "filter[boolean]=true",
			wantParse: fieldFiltersTarget{Boolean: &filters.FilterBoolean{Eq: lo.ToPtr(true)}},
		},
		{
			name:      "boolean eq false",
			query:     "filter[boolean][eq]=false",
			wantParse: fieldFiltersTarget{Boolean: &filters.FilterBoolean{Eq: lo.ToPtr(false)}},
		},

		// NumericFieldFilter
		{
			name:      "numeric short",
			query:     "filter[numeric]=42",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Eq: lo.ToPtr(42.0)}},
		},
		{
			name:      "numeric eq decimal",
			query:     "filter[numeric][eq]=12.5",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Eq: lo.ToPtr(12.5)}},
		},
		{
			name:      "numeric neq",
			query:     "filter[numeric][neq]=0",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Neq: lo.ToPtr(0.0)}},
		},
		{
			name:      "numeric lt",
			query:     "filter[numeric][lt]=10",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Lt: lo.ToPtr(10.0)}},
		},
		{
			name:      "numeric lte",
			query:     "filter[numeric][lte]=10",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Lte: lo.ToPtr(10.0)}},
		},
		{
			name:      "numeric gt",
			query:     "filter[numeric][gt]=10",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Gt: lo.ToPtr(10.0)}},
		},
		{
			name:      "numeric gte",
			query:     "filter[numeric][gte]=10",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Gte: lo.ToPtr(10.0)}},
		},
		{
			name:      "numeric oeq comma list",
			query:     "filter[numeric][oeq]=1,2,3",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Oeq: []float64{1, 2, 3}}},
		},
		{
			name:      "numeric negative",
			query:     "filter[numeric][eq]=-3.14",
			wantParse: fieldFiltersTarget{Numeric: &filters.FilterNumeric{Eq: lo.ToPtr(-3.14)}},
		},

		// StringFieldFilter
		{
			name:      "string short",
			query:     "filter[string]=hello",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Eq: lo.ToPtr("hello")}},
		},
		{
			name:      "string eq",
			query:     "filter[string][eq]=hello",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Eq: lo.ToPtr("hello")}},
		},
		{
			name:      "string neq",
			query:     "filter[string][neq]=hello",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Neq: lo.ToPtr("hello")}},
		},
		{
			name:      "string contains",
			query:     "filter[string][contains]=foo",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Contains: lo.ToPtr("foo")}},
		},
		{
			name:      "string ocontains comma list",
			query:     "filter[string][ocontains]=a,b,c",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Ocontains: []string{"a", "b", "c"}}},
		},
		{
			name:      "string oeq comma list",
			query:     "filter[string][oeq]=a,b,c",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Oeq: []string{"a", "b", "c"}}},
		},
		{
			// Spaces are URL-encoded so the URL parser doesn't truncate the
			// query value at the first literal space. parseCommaSeparated trims
			// surrounding whitespace from each item.
			name:      "string oeq trims spaces",
			query:     "filter[string][oeq]=foo%20,%20bar,baz",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Oeq: []string{"foo", "bar", "baz"}}},
		},
		{
			name:      "string gt",
			query:     "filter[string][gt]=alpha",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Gt: lo.ToPtr("alpha")}},
		},
		{
			name:      "string lte",
			query:     "filter[string][lte]=zeta",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Lte: lo.ToPtr("zeta")}},
		},
		{
			// The parser uses key-name semantics for existence checks: the
			// presence of [exists] sets Exists=true regardless of value, while
			// [nexists] sets Exists=false.
			name:      "string exists",
			query:     "filter[string][exists]=true",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Exists: lo.ToPtr(true)}},
		},
		{
			name:      "string exists value is ignored",
			query:     "filter[string][exists]=anything",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Exists: lo.ToPtr(true)}},
		},
		{
			name:      "string nexists",
			query:     "filter[string][nexists]=true",
			wantParse: fieldFiltersTarget{String: &filters.FilterString{Exists: lo.ToPtr(false)}},
		},

		// StringFieldFilterExact
		{
			name:      "string_exact short",
			query:     "filter[string_exact]=hello",
			wantParse: fieldFiltersTarget{StringExact: &filters.FilterStringExact{Eq: lo.ToPtr("hello")}},
		},
		{
			name:      "string_exact neq",
			query:     "filter[string_exact][neq]=excluded",
			wantParse: fieldFiltersTarget{StringExact: &filters.FilterStringExact{Neq: lo.ToPtr("excluded")}},
		},
		{
			name:      "string_exact oeq",
			query:     "filter[string_exact][oeq]=a,b",
			wantParse: fieldFiltersTarget{StringExact: &filters.FilterStringExact{Oeq: []string{"a", "b"}}},
		},

		// ULIDFieldFilter
		{
			name:      "ulid short",
			query:     "filter[ulid]=" + ulid1,
			wantParse: fieldFiltersTarget{ULID: &filters.FilterULID{Eq: &ulid1}},
		},
		{
			name:      "ulid eq",
			query:     "filter[ulid][eq]=" + ulid1,
			wantParse: fieldFiltersTarget{ULID: &filters.FilterULID{Eq: &ulid1}},
		},
		{
			name:      "ulid neq",
			query:     "filter[ulid][neq]=" + ulid1,
			wantParse: fieldFiltersTarget{ULID: &filters.FilterULID{Neq: &ulid1}},
		},
		{
			name:      "ulid oeq",
			query:     "filter[ulid][oeq]=" + ulid1 + "," + ulid2,
			wantParse: fieldFiltersTarget{ULID: &filters.FilterULID{Oeq: []string{ulid1, ulid2}}},
		},

		// DateTimeFieldFilter
		{
			name:      "datetime short",
			query:     "filter[datetime]=2024-01-02T03:04:05Z",
			wantParse: fieldFiltersTarget{DateTime: &filters.FilterDateTime{Eq: &dt}},
		},
		{
			name:      "datetime eq",
			query:     "filter[datetime][eq]=2024-01-02T03:04:05Z",
			wantParse: fieldFiltersTarget{DateTime: &filters.FilterDateTime{Eq: &dt}},
		},
		{
			name:      "datetime gt",
			query:     "filter[datetime][gt]=2024-01-02T03:04:05Z",
			wantParse: fieldFiltersTarget{DateTime: &filters.FilterDateTime{Gt: &dt}},
		},
		{
			name:      "datetime lte",
			query:     "filter[datetime][lte]=2024-01-02T03:04:05Z",
			wantParse: fieldFiltersTarget{DateTime: &filters.FilterDateTime{Lte: &dt}},
		},

		// LabelsFieldFilter — handled here as *FilterString since the test
		// fixture wires the labels field as a single filter, not a labels map.
		{
			name:      "labels short",
			query:     "filter[labels]=team-a",
			wantParse: fieldFiltersTarget{Labels: &filters.FilterString{Eq: lo.ToPtr("team-a")}},
		},
		{
			name:      "labels contains",
			query:     "filter[labels][contains]=team",
			wantParse: fieldFiltersTarget{Labels: &filters.FilterString{Contains: lo.ToPtr("team")}},
		},

		// Multiple independent filters in one request.
		{
			name:  "combined boolean+numeric+string",
			query: "filter[boolean][eq]=true&filter[numeric][gt]=5&filter[string][contains]=foo",
			wantParse: fieldFiltersTarget{
				Boolean: &filters.FilterBoolean{Eq: lo.ToPtr(true)},
				Numeric: &filters.FilterNumeric{Gt: lo.ToPtr(5.0)},
				String:  &filters.FilterString{Contains: lo.ToPtr("foo")},
			},
		},
		{
			name:  "combined ulid+datetime+labels",
			query: "filter[ulid][eq]=" + ulid1 + "&filter[datetime][lt]=2024-01-02T03:04:05Z&filter[labels]=team-a",
			wantParse: fieldFiltersTarget{
				ULID:     &filters.FilterULID{Eq: &ulid1},
				DateTime: &filters.FilterDateTime{Lt: &dt},
				Labels:   &filters.FilterString{Eq: lo.ToPtr("team-a")},
			},
		},

		// Negative cases — the OAS validator's deepObject scalar coercion is
		// permissive, but filters.Parse is strict and rejects malformed values
		// or unknown operators with a descriptive message.
		{
			name:           "boolean rejects non-bool",
			query:          "filter[boolean][eq]=notabool",
			wantErr:        true,
			wantBodySubstr: "filter[boolean][eq]: invalid boolean",
		},
		{
			name:           "numeric rejects non-number",
			query:          "filter[numeric][eq]=not-a-number",
			wantErr:        true,
			wantBodySubstr: "filter[numeric][eq]: invalid number",
		},
		// Note: malformed datetime values are caught by the OAS validator
		// before reaching the parser (the date-time format check in oneOf
		// fires first), so we don't have a parser-side negative case for
		// datetime here.
		{
			name:           "string_exact rejects unsupported operator",
			query:          "filter[string_exact][contains]=foo",
			wantErr:        true,
			wantBodySubstr: "unsupported operator",
		},
		{
			name:           "boolean rejects unsupported operator",
			query:          "filter[boolean][gt]=true",
			wantErr:        true,
			wantBodySubstr: "unsupported operator",
		},
		{
			name:           "ulid rejects unsupported operator",
			query:          "filter[ulid][contains]=anything",
			wantErr:        true,
			wantBodySubstr: "unsupported operator",
		},
		{
			name:           "datetime rejects unsupported operator",
			query:          "filter[datetime][contains]=anything",
			wantErr:        true,
			wantBodySubstr: "unsupported operator",
		},
		{
			name:           "unknown filter field rejected",
			query:          "filter[unknown][eq]=foo",
			wantErr:        true,
			wantBodySubstr: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := srv.URL + "/field-filters"
			if tc.query != "" {
				url += "?" + tc.query
			}

			resp, err := http.Get(url)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if tc.wantErr {
				require.Equalf(t, http.StatusBadRequest, resp.StatusCode, "query=%q body=%s", tc.query, string(body))
				if tc.wantBodySubstr != "" {
					assert.Contains(t, string(body), tc.wantBodySubstr)
				}
				return
			}

			require.Equalf(t, http.StatusOK, resp.StatusCode, "query=%q body=%s", tc.query, string(body))

			var got fieldFiltersTarget
			require.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, tc.wantParse, got)

			if tc.wantBodySubstr != "" {
				assert.Contains(t, string(body), tc.wantBodySubstr)
			}
		})
	}
}
