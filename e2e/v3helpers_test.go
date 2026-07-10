package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// v3Client embeds the generated v3 SDK so tests call service methods
// directly (e.g. c.Plans.Create(...)). It adds only exact-2xx status pinning
// on top of the SDK and a raw HTTP escape hatch for requests the typed SDK
// cannot represent.
//
// v3 e2e tests deliberately diverge from the older v1 style in this folder:
// fixture keys carry a unique suffix so re-runs against the same DB don't
// collide, and subtests are written to be independent rather than rely on
// ordering.
type v3Client struct {
	t testing.TB
	*v3sdk.Client
	statuses *statusCapturingTransport
	baseURL  string
}

// newV3Client returns a client pointed at $OPENMETER_ADDRESS. Skips the test
// when the variable is unset.
func newV3Client(t testing.TB) *v3Client {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	baseURL := strings.TrimRight(address, "/")
	statuses := &statusCapturingTransport{base: http.DefaultTransport}

	sdk, err := v3sdk.New(baseURL+"/api/v3", v3sdk.WithHTTPClient(&http.Client{Transport: statuses}))
	require.NoError(t, err)

	return &v3Client{
		t:        t,
		Client:   sdk,
		statuses: statuses,
		baseURL:  baseURL + "/api/v3/openmeter",
	}
}

type statusCapturingTransport struct {
	base http.RoundTripper

	mu         sync.Mutex
	lastStatus int
}

func (t *statusCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if resp != nil {
		t.mu.Lock()
		t.lastStatus = resp.StatusCode
		t.mu.Unlock()
	}
	return resp, err
}

func (t *statusCapturingTransport) last() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastStatus
}

// v3Problem is the decoded shape of an application/problem+json response.
// The server uses two shapes interchangeably depending on the error layer:
//   - Schema validation (bad enum, missing required field, bad JSON) returns
//     RFC 7807 with top-level invalid_parameters[] (api/v3/apierrors/errors.go).
//   - Product-catalog validation (publish-time or create-time domain checks)
//     returns RFC 7807 with extensions.validationErrors[]
//     (pkg/framework/commonhttp/errors.go HandleIssueIfHTTPStatusKnown).
//
// Both are parsed so a single response type can drive either assertion.
type v3Problem struct {
	Type              string               `json:"type"`
	Title             string               `json:"title"`
	Status            int                  `json:"status"`
	Detail            string               `json:"detail"`
	Instance          string               `json:"instance"`
	Extensions        map[string]any       `json:"extensions"`
	InvalidParameters []v3InvalidParameter `json:"invalid_parameters"`
}

type v3InvalidParameter struct {
	Field  string `json:"field"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
	Source string `json:"source"`
}

type v3ValidationError struct {
	Code     string `json:"code"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ValidationErrors returns the product-catalog validation errors embedded in
// extensions.validationErrors, or nil. Use this (not InvalidParameters) when
// asserting on domain codes like plan_with_no_phases.
func (p *v3Problem) ValidationErrors() []v3ValidationError {
	if p == nil || p.Extensions == nil {
		return nil
	}
	raw, ok := p.Extensions["validationErrors"]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []v3ValidationError
	_ = json.Unmarshal(b, &out)
	return out
}

// doMalformedRequest is the raw HTTP escape hatch for parser/binder tests that
// intentionally need to send values the typed SDK cannot represent.
func (c *v3Client) doMalformedRequest(method, path string, body any) (int, []byte, *v3Problem) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(c.t.Context(), 30*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err, "%s %s", method, path)

		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	require.NoError(c.t, err, "%s %s", method, path)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err, "%s %s", method, path)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err, "%s %s", method, path)

	var problem *v3Problem
	if resp.StatusCode >= 400 && len(raw) > 0 {
		var p v3Problem
		if err := json.Unmarshal(raw, &p); err == nil && (p.Status != 0 || p.Title != "") {
			problem = &p
		}
	}

	return resp.StatusCode, raw, problem
}

// requireStatus asserts the immediately preceding SDK call on this client
// succeeded with exactly want (e.g. 201 vs 200). Fails c.t — do not use inside
// assert.EventuallyWithT; use require.NoError(collect, err) there instead.
func (c *v3Client) requireStatus(want int, err error) {
	c.t.Helper()
	require.NoError(c.t, err)
	require.Equal(c.t, want, c.statuses.last())
}

// requireProblem asserts err is an *v3sdk.APIError with wantStatus and returns
// the decoded RFC 7807 problem for follow-up assertions
// (assertValidationCode / assertProblemDetail / assertInvalidParameterRule).
// Top-level fields are backfilled from the SDK's own parse so Detail/Title
// assertions keep working for bodies that do not decode as v3Problem.
func requireProblem(t testing.TB, err error, wantStatus int) *v3Problem {
	t.Helper()

	apiErr, ok := v3sdk.AsAPIError(err)
	require.True(t, ok, "expected an API error, got: %v", err)

	problem := &v3Problem{}
	if len(apiErr.RawBody) > 0 {
		_ = apiErr.Decode(problem)
	}
	if problem.Status == 0 {
		problem.Status = apiErr.StatusCode
	}
	if problem.Title == "" {
		problem.Title = apiErr.Title
	}
	if problem.Detail == "" {
		problem.Detail = apiErr.Detail
	}
	if problem.Type == "" {
		problem.Type = apiErr.Type
	}
	if problem.Instance == "" {
		problem.Instance = apiErr.Instance
	}

	require.Equal(t, wantStatus, apiErr.StatusCode, "problem: %+v", problem)
	return problem
}

// queryMeterV3 posts a v3 meter query with a fresh client. It returns the
// error instead of failing the test because callers poll it inside
// assert.EventuallyWithT.
func queryMeterV3(t testing.TB, meterID string, body v3sdk.MeterQueryRequest) (*v3sdk.MeterQueryResult, error) {
	t.Helper()
	c := newV3Client(t)
	return c.Meters.Query(t.Context(), meterID, body)
}

// --- Fixture builders ---

// uniqueKey returns a fixture key suffix that survives re-runs against a
// shared database without collision. Format: <prefix>_<millis>_<rand>.
func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), rand.Intn(10_000))
}

// validPlanRequest returns a baseline plan that creates and publishes
// successfully. Tests mutate the returned struct before posting.
func validPlanRequest(keyPrefix string) v3sdk.CreatePlanRequest {
	return v3sdk.CreatePlanRequest{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Plan " + keyPrefix,
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases:         []v3sdk.PlanPhaseInput{validPlanPhase("phase_1", true /* isLast */)},
	}
}

// validPlanPhase returns a single phase with one flat rate card. If isLast is
// false the phase carries a P1M duration (non-last phases must be bounded).
func validPlanPhase(keyPrefix string, isLast bool) v3sdk.PlanPhaseInput {
	phase := v3sdk.PlanPhaseInput{
		Key:       uniqueKey(keyPrefix),
		Name:      "Test Phase " + keyPrefix,
		RateCards: []v3sdk.RateCardInput{validFlatRateCard("fee")},
	}
	if !isLast {
		phase.Duration = lo.ToPtr("P1M")
	}
	return phase
}

// validFlatRateCard returns a flat, in-advance, P1M rate card — the simplest
// shape that passes plan- and addon-level validation.
func validFlatRateCard(keyPrefix string) v3sdk.RateCardInput {
	return v3sdk.RateCardInput{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Rate Card " + keyPrefix,
		Price:          lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: "10"})),
		BillingCadence: lo.ToPtr("P1M"),
		PaymentTerm:    lo.ToPtr(v3sdk.PricePaymentTermInAdvance),
	}
}

// validUnitRateCard returns a usage-based unit-priced rate card. Unit prices
// cannot use payment_term=in_advance (that's flat-only), so this uses
// in_arrears.
func validUnitRateCard(f v3sdk.Feature) v3sdk.RateCardInput {
	return v3sdk.RateCardInput{
		Key:            f.Key,
		Name:           "Test Unit Rate Card " + f.Key,
		Price:          lo.Must(v3sdk.PriceFromPriceUnit(v3sdk.PriceUnit{Amount: "0.10"})),
		BillingCadence: lo.ToPtr("P1M"),
		PaymentTerm:    lo.ToPtr(v3sdk.PricePaymentTermInArrears),
		Feature:        &v3sdk.FeatureReference{ID: f.ID},
	}
}

// validGraduatedRateCard returns a graduated tiered rate card with two tiers:
// 0–100 units at $0.10/unit and 100+ units at $0.05/unit.
func validGraduatedRateCard(f v3sdk.Feature) v3sdk.RateCardInput {
	return v3sdk.RateCardInput{
		Key:  f.Key,
		Name: "Test Graduated Rate Card " + f.Key,
		Price: lo.Must(v3sdk.PriceFromPriceGraduated(v3sdk.PriceGraduated{
			Tiers: []v3sdk.PriceTier{
				{
					UpToAmount: lo.ToPtr(v3sdk.Numeric("100")),
					// Tier-nested prices are not built by a union constructor,
					// so the type discriminator must be set explicitly or the
					// request fails schema validation with a 400.
					UnitPrice: &v3sdk.PriceUnit{Type: v3sdk.PriceTypeUnit, Amount: "0.10"},
				},
				{
					UnitPrice: &v3sdk.PriceUnit{Type: v3sdk.PriceTypeUnit, Amount: "0.05"},
				},
			},
		})),
		BillingCadence: lo.ToPtr("P1M"),
		PaymentTerm:    lo.ToPtr(v3sdk.PricePaymentTermInArrears),
		Feature:        &v3sdk.FeatureReference{ID: f.ID},
	}
}

// validAddonRequest returns a baseline addon that publishes successfully.
func validAddonRequest(keyPrefix string) v3sdk.CreateAddonRequest {
	return v3sdk.CreateAddonRequest{
		Key:          uniqueKey(keyPrefix),
		Name:         "Test Addon " + keyPrefix,
		Currency:     "USD",
		InstanceType: v3sdk.AddonInstanceTypeSingle,
		RateCards:    []v3sdk.RateCardInput{validFlatRateCard("addon_fee")},
	}
}

// validPlanAddonRequest returns a baseline attach request. The addon must be
// published and its currency/cadence compatible with the plan.
func validPlanAddonRequest(phaseKey, addonID string) v3sdk.CreatePlanAddonRequest {
	return v3sdk.CreatePlanAddonRequest{
		Name:          "Test Plan Addon",
		Addon:         v3sdk.AddonReference{ID: addonID},
		FromPlanPhase: phaseKey,
	}
}

// --- Assertion helpers ---

// assertValidationCode asserts the problem response carries a
// ProductCatalogValidationError with the given code under
// extensions.validationErrors. Use for publish-time and create-time domain
// validation assertions. The server may return multiple errors in one
// response — this helper matches any.
func assertValidationCode(t *testing.T, problem *v3Problem, code string) {
	t.Helper()
	require.NotNil(t, problem, "expected problem response")
	errs := problem.ValidationErrors()
	require.NotEmpty(t, errs, "no validation errors in problem response: %+v", problem)
	for _, e := range errs {
		if e.Code == code {
			return
		}
	}
	codes := make([]string, 0, len(errs))
	for _, e := range errs {
		codes = append(codes, e.Code)
	}
	assert.Failf(t, "validation code not found", "expected %q, got %v", code, codes)
}

// assertProblemDetail asserts the problem's Detail field contains the given
// substring. Use for errors surfaced via BaseAPIError with a plain Detail
// message rather than extensions.validationErrors — for example, the
// attach-status mismatches in planaddon ("invalid <status> status, allowed
// statuses: [...]").
func assertProblemDetail(t *testing.T, problem *v3Problem, substring string) {
	t.Helper()
	require.NotNil(t, problem, "expected problem response")
	assert.Contains(t, problem.Detail, substring, "problem: %+v", problem)
}

// assertInvalidParameterRule asserts the problem response carries a schema
// InvalidParameter whose Rule matches. Use for TypeSpec/JSON-schema-layer
// rejections (min_items, required, enum) that the request fails before
// reaching the product-catalog validator.
func assertInvalidParameterRule(t *testing.T, problem *v3Problem, rule string) {
	t.Helper()
	require.NotNil(t, problem, "expected problem response")
	require.NotEmpty(t, problem.InvalidParameters, "no invalid_parameters in problem response: %+v", problem)
	for _, p := range problem.InvalidParameters {
		if p.Rule == rule {
			return
		}
	}
	rules := make([]string, 0, len(problem.InvalidParameters))
	for _, p := range problem.InvalidParameters {
		rules = append(rules, p.Rule)
	}
	assert.Failf(t, "invalid parameter rule not found", "expected %q, got %v", rule, rules)
}

// findRateCardByKey looks up a rate card by key across all phases of a plan.
// Fails the test if no match is found.
func findRateCardByKey(t *testing.T, plan *v3sdk.Plan, key string) *v3sdk.RateCard {
	t.Helper()

	for i := range plan.Phases {
		for j := range plan.Phases[i].RateCards {
			rc := &plan.Phases[i].RateCards[j]
			if rc.Key == key {
				return rc
			}
		}
	}

	require.FailNow(t, "rate card not found", "key=%s", key)
	return nil
}

// assertUnitPriceAmount asserts the rate card's price discriminates as "unit"
// and carries the given amount. Used to verify the synthesized unit price that
// replaces v1 dynamic and package prices on the v3 read path.
func assertUnitPriceAmount(t *testing.T, rc *v3sdk.RateCard, want string) {
	t.Helper()

	require.Equal(t, string(v3sdk.PriceTypeUnit), rc.Price.Type, "expected synthesized unit price")

	unit, err := rc.Price.AsPriceUnit()
	require.NoError(t, err)
	assert.Equal(t, want, unit.Amount)
}
