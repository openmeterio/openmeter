// Hand-written wire tests for the generated OpenMeter Go SDK. The generator's
// output cleaner preserves *_test.go files, so these survive regeneration.
package openmeter_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	openmeter "github.com/openmeterio/openmeter/api/v3/client"
)

const emptyPageBody = `{"data":[],"meta":{"page":{"number":1,"size":100,"total":0}}}`

// recordedRequest captures the wire-level view of a request as the server saw it.
type recordedRequest struct {
	method     string
	path       string
	requestURI string
	query      url.Values
	header     http.Header
	body       []byte
}

// requestRecorder is a shared test harness: an http.Handler that records every
// request (race-safely) and responds with a fixed status and body.
type requestRecorder struct {
	mu   sync.Mutex
	reqs []recordedRequest
}

func (rr *requestRecorder) handler(status int, respBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rr.mu.Lock()
		rr.reqs = append(rr.reqs, recordedRequest{
			method:     r.Method,
			path:       r.URL.Path,
			requestURI: r.RequestURI,
			query:      r.URL.Query(),
			header:     r.Header.Clone(),
			body:       body,
		})
		rr.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}
}

func (rr *requestRecorder) last(t *testing.T) recordedRequest {
	t.Helper()
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if len(rr.reqs) == 0 {
		t.Fatal("no request recorded")
	}
	return rr.reqs[len(rr.reqs)-1]
}

// newTestClient starts an httptest server around handler and returns a client
// pointed at it. The server is closed via t.Cleanup.
func newTestClient(t *testing.T, handler http.Handler, opts ...openmeter.Option) *openmeter.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := openmeter.New(srv.URL, opts...)
	if err != nil {
		t.Fatalf("openmeter.New(%q): %v", srv.URL, err)
	}
	return c
}

func TestBaseURLJoining(t *testing.T) {
	t.Parallel()

	t.Run("host-only base", func(t *testing.T) {
		rec := &requestRecorder{}
		om := newTestClient(t, rec.handler(http.StatusOK, emptyPageBody))

		if _, err := om.Meters.List(t.Context(), openmeter.MeterListParams{}); err != nil {
			t.Fatalf("Meters.List: %v", err)
		}

		if got := rec.last(t).path; got != "/openmeter/meters" {
			t.Errorf("request path = %q, want %q", got, "/openmeter/meters")
		}
	})

	t.Run("base path and base query preserved", func(t *testing.T) {
		rec := &requestRecorder{}
		srv := httptest.NewServer(rec.handler(http.StatusOK, "{}"))
		t.Cleanup(srv.Close)

		om, err := openmeter.New(srv.URL + "/api/v3?tenant=acme")
		if err != nil {
			t.Fatalf("openmeter.New: %v", err)
		}

		if _, err := om.Meters.Get(t.Context(), "m-1"); err != nil {
			t.Fatalf("Meters.Get: %v", err)
		}

		r := rec.last(t)
		if r.path != "/api/v3/openmeter/meters/m-1" {
			t.Errorf("request path = %q, want %q", r.path, "/api/v3/openmeter/meters/m-1")
		}
		if got := r.query.Get("tenant"); got != "acme" {
			t.Errorf("base query param tenant = %q, want %q", got, "acme")
		}
	})

	t.Run("per-request query merges with and overrides base query", func(t *testing.T) {
		rec := &requestRecorder{}
		srv := httptest.NewServer(rec.handler(http.StatusOK, emptyPageBody))
		t.Cleanup(srv.Close)

		// The base URL carries both an unrelated param and a page[size] that the
		// request-level params must override.
		om, err := openmeter.New(srv.URL + "/api/v3?tenant=acme&page%5Bsize%5D=9")
		if err != nil {
			t.Fatalf("openmeter.New: %v", err)
		}

		params := openmeter.MeterListParams{Page: &openmeter.PageParams{Size: openmeter.Int(5)}}
		if _, err := om.Meters.List(t.Context(), params); err != nil {
			t.Fatalf("Meters.List: %v", err)
		}

		q := rec.last(t).query
		if got := q.Get("tenant"); got != "acme" {
			t.Errorf("merged query tenant = %q, want %q", got, "acme")
		}
		if got := q.Get("page[size]"); got != "5" {
			t.Errorf("merged query page[size] = %q, want %q (request must override base)", got, "5")
		}
		if got := q["page[size]"]; len(got) != 1 {
			t.Errorf("page[size] has %d values %v, want exactly 1", len(got), got)
		}
	})
}

func TestRequestHeaders(t *testing.T) {
	t.Parallel()

	t.Run("authorization and default user agent", func(t *testing.T) {
		rec := &requestRecorder{}
		om := newTestClient(t, rec.handler(http.StatusOK, "{}"), openmeter.WithToken("test-token"))

		if _, err := om.Meters.Get(t.Context(), "m-1"); err != nil {
			t.Fatalf("Meters.Get: %v", err)
		}

		r := rec.last(t)
		if got := r.header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		wantUA := "openmeter-go-sdk/" + openmeter.Version
		if got := r.header.Get("User-Agent"); got != wantUA {
			t.Errorf("User-Agent = %q, want %q", got, wantUA)
		}
		if got := r.header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
	})

	t.Run("custom user agent", func(t *testing.T) {
		rec := &requestRecorder{}
		om := newTestClient(t, rec.handler(http.StatusOK, "{}"), openmeter.WithUserAgent("acme-billing/1.2"))

		if _, err := om.Meters.Get(t.Context(), "m-1"); err != nil {
			t.Fatalf("Meters.Get: %v", err)
		}

		if got := rec.last(t).header.Get("User-Agent"); got != "acme-billing/1.2" {
			t.Errorf("User-Agent = %q, want %q", got, "acme-billing/1.2")
		}
	})

	t.Run("csv accept header", func(t *testing.T) {
		rec := &requestRecorder{}
		om := newTestClient(t, rec.handler(http.StatusOK, "from,to,value\n"))

		if _, err := om.Meters.QueryCSV(t.Context(), "m-1", openmeter.MeterQueryRequest{}); err != nil {
			t.Fatalf("Meters.QueryCSV: %v", err)
		}

		r := rec.last(t)
		if got := r.header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept = %q, want %q", got, "text/csv")
		}
		if got := r.header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
	})
}

func TestIngestContentTypes(t *testing.T) {
	t.Parallel()

	event := openmeter.EventInput{ID: "evt-1", Source: "svc", Type: "request", Subject: "cust-1"}

	cases := []struct {
		name     string
		send     func(ctx context.Context, om *openmeter.Client) error
		wantCT   string
		wantLead byte
	}{
		{
			name:     "single event uses cloudevents json",
			send:     func(ctx context.Context, om *openmeter.Client) error { return om.Events.IngestEvent(ctx, event) },
			wantCT:   "application/cloudevents+json",
			wantLead: '{',
		},
		{
			name: "batch uses cloudevents batch json",
			send: func(ctx context.Context, om *openmeter.Client) error {
				return om.Events.IngestEvents(ctx, []openmeter.EventInput{event})
			},
			wantCT:   "application/cloudevents-batch+json",
			wantLead: '[',
		},
		{
			name: "plain json single",
			send: func(ctx context.Context, om *openmeter.Client) error {
				return om.Events.IngestEventsJSON(ctx, openmeter.One(event))
			},
			wantCT:   "application/json",
			wantLead: '{',
		},
		{
			name: "plain json many",
			send: func(ctx context.Context, om *openmeter.Client) error {
				return om.Events.IngestEventsJSON(ctx, openmeter.Many([]openmeter.EventInput{event}))
			},
			wantCT:   "application/json",
			wantLead: '[',
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := &requestRecorder{}
			om := newTestClient(t, rec.handler(http.StatusNoContent, ""))

			if err := tc.send(t.Context(), om); err != nil {
				t.Fatalf("ingest call: %v", err)
			}

			r := rec.last(t)
			if got := r.header.Get("Content-Type"); got != tc.wantCT {
				t.Errorf("Content-Type = %q, want %q", got, tc.wantCT)
			}
			if len(r.body) == 0 || r.body[0] != tc.wantLead {
				t.Errorf("request body %q does not start with %q", r.body, string(tc.wantLead))
			}
		})
	}
}

// contextDeadlineRecorder is a RoundTripper that captures the deadline of the
// context the transport actually sees, i.e. after the SDK applied (or skipped)
// its default request deadline.
type contextDeadlineRecorder struct {
	mu       sync.Mutex
	deadline time.Time
	ok       bool
}

func (rec *contextDeadlineRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	rec.mu.Lock()
	rec.deadline, rec.ok = deadline, ok
	rec.mu.Unlock()
	return http.DefaultTransport.RoundTrip(req)
}

func (rec *contextDeadlineRecorder) snapshot() (time.Time, bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.deadline, rec.ok
}

func TestDefaultRequestDeadline(t *testing.T) {
	t.Parallel()

	newDeadlineClient := func(t *testing.T) (*openmeter.Client, *contextDeadlineRecorder) {
		t.Helper()
		rec := &contextDeadlineRecorder{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "{}")
		}))
		t.Cleanup(srv.Close)

		om, err := openmeter.New(srv.URL, openmeter.WithHTTPClient(&http.Client{Transport: rec}))
		if err != nil {
			t.Fatalf("openmeter.New: %v", err)
		}
		return om, rec
	}

	t.Run("buffered call without caller deadline gets the default", func(t *testing.T) {
		om, rec := newDeadlineClient(t)

		before := time.Now()
		if _, err := om.Meters.Get(t.Context(), "m-1"); err != nil {
			t.Fatalf("Meters.Get: %v", err)
		}

		deadline, ok := rec.snapshot()
		if !ok {
			t.Fatal("transport saw no context deadline, want the 30s default applied")
		}
		if until := deadline.Sub(before); until < 29*time.Second || until > 40*time.Second {
			t.Errorf("default deadline is %v away, want roughly 30s", until)
		}
	})

	t.Run("caller deadline beyond the default is honored unchanged", func(t *testing.T) {
		om, rec := newDeadlineClient(t)

		// A 90s caller deadline discriminates pass-through from re-wrapping: if
		// the SDK wrongly layered its 30s default on top, the captured deadline
		// would shrink to ~30s.
		want := time.Now().Add(90 * time.Second)
		ctx, cancel := context.WithDeadline(t.Context(), want)
		defer cancel()

		if _, err := om.Meters.Get(ctx, "m-1"); err != nil {
			t.Fatalf("Meters.Get: %v", err)
		}

		deadline, ok := rec.snapshot()
		if !ok {
			t.Fatal("transport saw no context deadline, want the caller's deadline")
		}
		if !deadline.Equal(want) {
			t.Errorf("transport deadline = %v, want caller deadline %v unchanged", deadline, want)
		}
	})

	t.Run("streaming call gets no default deadline", func(t *testing.T) {
		om, rec := newDeadlineClient(t)

		stream, err := om.Meters.QueryCSVStream(t.Context(), "m-1", openmeter.MeterQueryRequest{})
		if err != nil {
			t.Fatalf("Meters.QueryCSVStream: %v", err)
		}
		defer stream.Close()

		if _, ok := rec.snapshot(); ok {
			t.Error("streaming request carries a deadline, want none unless the caller sets one")
		}
	})
}

func TestBufferedResponseCap(t *testing.T) {
	t.Parallel()

	const bufferedCap = 10 << 20

	t.Run("response over the cap errors and points at streaming", func(t *testing.T) {
		om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write(bytes.Repeat([]byte("a"), bufferedCap+1))
		}))

		_, err := om.Meters.QueryCSV(t.Context(), "m-1", openmeter.MeterQueryRequest{})
		if err == nil {
			t.Fatal("QueryCSV returned nil error for a response over the buffered cap")
		}
		if !strings.Contains(err.Error(), "streaming") {
			t.Errorf("error %q does not mention the streaming alternative", err)
		}
	})

	t.Run("response exactly at the cap is returned whole", func(t *testing.T) {
		om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write(bytes.Repeat([]byte("a"), bufferedCap))
		}))

		body, err := om.Meters.QueryCSV(t.Context(), "m-1", openmeter.MeterQueryRequest{})
		if err != nil {
			t.Fatalf("QueryCSV: %v", err)
		}
		if len(body) != bufferedCap {
			t.Errorf("len(body) = %d, want %d", len(body), bufferedCap)
		}
	})
}

func TestErrorBodyCap(t *testing.T) {
	t.Parallel()

	const errorBodyCap = 1 << 20
	oversized := strings.Repeat("x", errorBodyCap+4096)

	om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, oversized)
	}))

	_, err := om.Meters.Get(t.Context(), "m-1")
	apiErr, ok := openmeter.AsAPIError(err)
	if !ok {
		t.Fatalf("error %v is not an *APIError", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusInternalServerError)
	}
	if len(apiErr.RawBody) != errorBodyCap {
		t.Errorf("len(RawBody) = %d, want error bodies capped at %d", len(apiErr.RawBody), errorBodyCap)
	}
}

func TestStream(t *testing.T) {
	t.Parallel()

	t.Run("returns a live body readable past the buffered cap", func(t *testing.T) {
		const size = (10 << 20) + 1
		om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write(bytes.Repeat([]byte("a"), size))
		}))

		stream, err := om.Meters.QueryCSVStream(t.Context(), "m-1", openmeter.MeterQueryRequest{})
		if err != nil {
			t.Fatalf("QueryCSVStream: %v", err)
		}
		defer stream.Close()

		body, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("reading stream: %v", err)
		}
		if len(body) != size {
			t.Errorf("len(body) = %d, want %d (streams must not be capped)", len(body), size)
		}
	})

	t.Run("non-2xx stream response returns an APIError", func(t *testing.T) {
		rec := &requestRecorder{}
		om := newTestClient(t, rec.handler(http.StatusForbidden, `{"status":403,"title":"Forbidden","detail":"no access"}`))

		stream, err := om.Meters.QueryCSVStream(t.Context(), "m-1", openmeter.MeterQueryRequest{})
		if err == nil {
			stream.Close()
			t.Fatal("QueryCSVStream returned nil error for a 403 response")
		}

		apiErr, ok := openmeter.AsAPIError(err)
		if !ok {
			t.Fatalf("error %v is not an *APIError", err)
		}
		if apiErr.StatusCode != http.StatusForbidden || apiErr.Title != "Forbidden" {
			t.Errorf("APIError = status %d title %q, want 403 %q", apiErr.StatusCode, apiErr.Title, "Forbidden")
		}
	})
}
