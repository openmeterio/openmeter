package openmeter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient starts an httptest server backed by handler and returns a
// Client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return c
}

func TestMeters_Get(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		if r.URL.Path != "/openmeter/meters/01ABC" {
			t.Errorf("path = %s, want /openmeter/meters/01ABC", r.URL.Path)
		}

		w.Header().Set("Content-Type", contentTypeJSON)

		_, _ = io.WriteString(w, `{"id":"01ABC","key":"tokens","name":"Tokens","aggregation":"sum","event_type":"prompt","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	m, err := c.Meters.Get(t.Context(), "01ABC")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if m.ID != "01ABC" || m.Key != "tokens" || m.Aggregation != MeterAggregationSum {
		t.Fatalf("unexpected meter: %+v", m)
	}
}

func TestMeters_Get_IDEncodedOnce(t *testing.T) {
	// A meter ID with characters that need escaping must be encoded exactly once.
	// The server sees the decoded path; a double-encoded ID would arrive as the
	// literal "%20" segment instead of a space.
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"id":"a b","key":"k","name":"n","aggregation":"count","event_type":"e","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	if _, err := c.Meters.Get(t.Context(), "a b"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if gotPath != "/openmeter/meters/a b" {
		t.Fatalf("server path = %q, want %q (ID double-encoded?)", gotPath, "/openmeter/meters/a b")
	}
}

func TestMeters_List_QueryString(t *testing.T) {
	var gotRawQuery string

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"number":1,"size":10,"total":0}}}`)
	})

	_, err := c.Meters.List(t.Context(), MeterListParams{
		Page:   &PageParams{Size: Int(10), Number: Int(1)},
		Sort:   []string{"created_at desc"},
		Filter: &MeterFilter{Key: &StringFilter{Eq: String("tokens")}},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	want := "filter%5Bkey%5D%5Beq%5D=tokens&page%5Bnumber%5D=1&page%5Bsize%5D=10&sort=created_at+desc"
	if gotRawQuery != want {
		t.Fatalf("raw query\n got: %q\nwant: %q", gotRawQuery, want)
	}
}

func TestMeters_Query_JSON(t *testing.T) {
	var gotBody MeterQueryRequest

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if r.URL.Path != "/openmeter/meters/m1/query" {
			t.Errorf("path = %s", r.URL.Path)
		}

		if ct := r.Header.Get("Content-Type"); ct != contentTypeJSON {
			t.Errorf("Content-Type = %s", ct)
		}

		if ac := r.Header.Get("Accept"); ac != contentTypeJSON {
			t.Errorf("Accept = %s, want %s", ac, contentTypeJSON)
		}

		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"data":[{"value":"12.34","from":"2024-01-01T00:00:00Z","to":"2024-01-02T00:00:00Z","dimensions":{"model":"gpt-4"}}]}`)
	})

	gran := MeterQueryGranularityDay

	res, err := c.Meters.Query(t.Context(), "m1", MeterQueryRequest{
		Granularity:       &gran,
		GroupByDimensions: []string{"model"},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if gotBody.Granularity == nil || *gotBody.Granularity != MeterQueryGranularityDay {
		t.Fatalf("server did not receive granularity: %+v", gotBody)
	}

	if len(res.Data) != 1 || res.Data[0].Value != "12.34" || res.Data[0].Dimensions["model"] != "gpt-4" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestMeters_QueryCSV_ContentNegotiation(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if ac := r.Header.Get("Accept"); ac != contentTypeCSV {
			t.Errorf("Accept = %s, want %s", ac, contentTypeCSV)
		}

		w.Header().Set("Content-Type", contentTypeCSV)
		_, _ = io.WriteString(w, "from,to,value\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,12\n")
	})

	csv, err := c.Meters.QueryCSV(t.Context(), "m1", MeterQueryRequest{})
	if err != nil {
		t.Fatalf("QueryCSV: %v", err)
	}

	if string(csv) != "from,to,value\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,12\n" {
		t.Fatalf("unexpected csv: %q", string(csv))
	}
}

func TestMeters_QueryCSVStream(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if ac := r.Header.Get("Accept"); ac != contentTypeCSV {
			t.Errorf("Accept = %s, want %s", ac, contentTypeCSV)
		}

		w.Header().Set("Content-Type", contentTypeCSV)
		_, _ = io.WriteString(w, "from,to,value\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,12\n")
	})

	rc, err := c.Meters.QueryCSVStream(t.Context(), "m1", MeterQueryRequest{})
	if err != nil {
		t.Fatalf("QueryCSVStream: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	if string(got) != "from,to,value\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,12\n" {
		t.Fatalf("unexpected stream: %q", string(got))
	}
}

func TestDoRaw_BodyCap(t *testing.T) {
	// A body larger than the buffered cap must fail rather than being read
	// unbounded into memory. QueryCSVStream is the escape hatch for such sizes.
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentTypeCSV)

		big := make([]byte, maxBufferedResponse+1)
		_, _ = w.Write(big)
	})

	_, err := c.Meters.QueryCSV(t.Context(), "m1", MeterQueryRequest{})
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}

	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want body-limit error", err)
	}
}

func TestAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"status":404,"title":"Not found","detail":"Meter [x] not found","instance":"kong:trace:abc"}`)
	})

	_, err := c.Meters.Get(t.Context(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}

	if apiErr.StatusCode != 404 || apiErr.Title != "Not found" || apiErr.Instance != "kong:trace:abc" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestAPIError_OversizedBodyPreserved(t *testing.T) {
	// An error body larger than the cap must not be dropped: APIError should
	// still carry the (truncated) diagnostic bytes. Uses a POST so the 5xx isn't
	// retried.
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(make([]byte, maxErrorBody+1024))
	})

	_, err := c.Meters.Query(t.Context(), "m1", MeterQueryRequest{})

	var apiErr *APIError

	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", apiErr.StatusCode)
	}

	if len(apiErr.RawBody) == 0 {
		t.Fatal("oversized error body was dropped")
	}

	if int64(len(apiErr.RawBody)) > maxErrorBody {
		t.Fatalf("error body not capped: %d > %d", len(apiErr.RawBody), maxErrorBody)
	}
}

func TestAuthHeader(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", auth)
		}

		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"data":[],"meta":{"page":{"number":1,"size":10,"total":0}}}`)
	}, WithToken("secret-token"))

	if _, err := c.Meters.List(t.Context(), MeterListParams{}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestMeters_EmptyMeterID(t *testing.T) {
	// No request should be made for an empty meter ID; the guard fails fast.
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s", r.URL.Path)
	})

	ctx := t.Context()

	if _, err := c.Meters.Get(ctx, ""); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Get(\"\") error = %v, want ErrEmptyID", err)
	}

	if _, err := c.Meters.Query(ctx, "", MeterQueryRequest{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("Query(\"\") error = %v, want ErrEmptyID", err)
	}

	if _, err := c.Meters.QueryCSV(ctx, "", MeterQueryRequest{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("QueryCSV(\"\") error = %v, want ErrEmptyID", err)
	}

	if _, err := c.Meters.QueryCSVStream(ctx, "", MeterQueryRequest{}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("QueryCSVStream(\"\") error = %v, want ErrEmptyID", err)
	}
}

func TestDefaultDeadline(t *testing.T) {
	// Inspect whether the request reaching the transport carries a deadline.
	// context.Background() (not t.Context()) is used deliberately: it is
	// guaranteed to have no deadline, so it exercises the default-deadline branch
	// deterministically.
	var hadDeadline bool
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		_, hadDeadline = r.Context().Deadline()

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {contentTypeJSON}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"m1","key":"k","name":"n","aggregation":"count","event_type":"e","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)),
		}, nil
	})

	c, err := New("https://example.invalid", WithHTTPClient(&http.Client{Transport: rt}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Buffered call, no caller deadline -> SDK applies its default deadline.
	hadDeadline = false
	if _, err := c.Meters.Get(context.Background(), "m1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hadDeadline {
		t.Error("buffered call without a caller deadline should get the default deadline")
	}

	// Streaming call, no caller deadline -> no default deadline imposed, so a long
	// stream is bounded only by the caller's context.
	hadDeadline = false
	rc, err := c.Meters.QueryCSVStream(context.Background(), "m1", MeterQueryRequest{})
	if err != nil {
		t.Fatalf("QueryCSVStream: %v", err)
	}
	_ = rc.Close()
	if hadDeadline {
		t.Error("streaming call should not receive the default deadline")
	}

	// Buffered call with a caller deadline -> preserved (still present downstream).
	hadDeadline = false
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	if _, err := c.Meters.Get(ctx, "m1"); err != nil {
		t.Fatalf("Get with deadline: %v", err)
	}
	if !hadDeadline {
		t.Error("caller-provided deadline should be present on the request context")
	}
}

// roundTripFunc adapts a function to http.RoundTripper for transport injection.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWithHTTPClient_InjectedTransport(t *testing.T) {
	called := false

	injected := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {contentTypeJSON}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"m1","key":"k","name":"n","aggregation":"count","event_type":"e","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)),
		}, nil
	})}

	c, err := New("https://example.invalid", WithHTTPClient(injected))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.Meters.Get(t.Context(), "m1"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !called {
		t.Fatal("injected transport was not used")
	}
}
