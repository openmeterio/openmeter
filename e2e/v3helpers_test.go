package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// v3RequestTimeout bounds each individual HTTP request against the e2e server
// so that server-side hangs surface as test failures in seconds instead of
// waiting for the default `go test` 10-minute deadline.
const v3RequestTimeout = 30 * time.Second

// v3Client is a minimal HTTP client for exercising v3 product-catalog
// endpoints over the live e2e server. The v3 Go SDK is not yet generated, so
// callers build requests from apiv3.* structs directly and decode success
// bodies themselves.
//
// v3 e2e tests deliberately diverge from the older v1 style in this folder:
// fixture keys carry a unique suffix so re-runs against the same DB don't
// collide, and subtests are written to be independent rather than rely on
// ordering.
type v3Client struct {
	t       *testing.T
	baseURL string
}

// newV3Client returns a client pointed at $OPENMETER_ADDRESS. Skips the test
// when the variable is unset.
func newV3Client(t *testing.T) *v3Client {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	return &v3Client{
		t:       t,
		baseURL: strings.TrimRight(address, "/") + "/api/v3/openmeter",
	}
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

// do marshals body (if non-nil), issues the request, and returns
// (status, rawBody, problem). problem is populated only when the response is
// non-2xx and parses as problem+json.
func (c *v3Client) do(method, path string, body any) (int, []byte, *v3Problem) {
	c.t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		bodyReader = bytes.NewReader(b)
	}

	ctx, cancel := context.WithTimeout(c.t.Context(), v3RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	require.NoError(c.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err, "%s %s", method, path)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	var problem *v3Problem
	if resp.StatusCode >= 400 && len(raw) > 0 {
		var p v3Problem
		if err := json.Unmarshal(raw, &p); err == nil && (p.Status != 0 || p.Title != "") {
			problem = &p
		}
	}

	return resp.StatusCode, raw, problem
}

// decodeTyped is a shared helper used by the typed wrappers below. It returns
// the parsed body on success, or (status, nil, problem) on any non-expected
// status.
func decodeTyped[T any](c *v3Client, status int, raw []byte, problem *v3Problem, want int) (int, *T, *v3Problem) {
	c.t.Helper()
	if status != want {
		return status, nil, problem
	}
	var v T
	require.NoError(c.t, json.Unmarshal(raw, &v), "decode response: %s", raw)
	return status, &v, nil
}

// --- Plans ---

func (c *v3Client) CreatePlan(body apiv3.CreatePlanRequest) (int, *apiv3.BillingPlan, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/plans", body)
	return decodeTyped[apiv3.BillingPlan](c, status, raw, problem, http.StatusCreated)
}

func (c *v3Client) GetPlan(id string) (int, *apiv3.BillingPlan, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/plans/"+id, nil)
	return decodeTyped[apiv3.BillingPlan](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) UpdatePlan(id string, body apiv3.UpsertPlanRequest) (int, *apiv3.BillingPlan, *v3Problem) {
	status, raw, problem := c.do(http.MethodPut, "/plans/"+id, body)
	return decodeTyped[apiv3.BillingPlan](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) PublishPlan(id string) (int, *apiv3.BillingPlan, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/plans/"+id+"/publish", nil)
	return decodeTyped[apiv3.BillingPlan](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) ArchivePlan(id string) (int, *apiv3.BillingPlan, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/plans/"+id+"/archive", nil)
	return decodeTyped[apiv3.BillingPlan](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) DeletePlan(id string) (int, *v3Problem) {
	status, _, problem := c.do(http.MethodDelete, "/plans/"+id, nil)
	return status, problem
}

func (c *v3Client) ListPlans(opts ...listOption) (int, *apiv3.PlanPagePaginatedResponse, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/plans"+buildPageQuery(opts), nil)
	return decodeTyped[apiv3.PlanPagePaginatedResponse](c, status, raw, problem, http.StatusOK)
}

// --- Addons ---

func (c *v3Client) CreateAddon(body apiv3.CreateAddonRequest) (int, *apiv3.Addon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/addons", body)
	return decodeTyped[apiv3.Addon](c, status, raw, problem, http.StatusCreated)
}

func (c *v3Client) GetAddon(id string) (int, *apiv3.Addon, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/addons/"+id, nil)
	return decodeTyped[apiv3.Addon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) UpdateAddon(id string, body apiv3.UpsertAddonRequest) (int, *apiv3.Addon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPut, "/addons/"+id, body)
	return decodeTyped[apiv3.Addon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) PublishAddon(id string) (int, *apiv3.Addon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/addons/"+id+"/publish", nil)
	return decodeTyped[apiv3.Addon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) ArchiveAddon(id string) (int, *apiv3.Addon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/addons/"+id+"/archive", nil)
	return decodeTyped[apiv3.Addon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) DeleteAddon(id string) (int, *v3Problem) {
	status, _, problem := c.do(http.MethodDelete, "/addons/"+id, nil)
	return status, problem
}

func (c *v3Client) ListAddons(opts ...listOption) (int, *apiv3.AddonPagePaginatedResponse, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/addons"+buildPageQuery(opts), nil)
	return decodeTyped[apiv3.AddonPagePaginatedResponse](c, status, raw, problem, http.StatusOK)
}

// --- Plan-addons ---

func (c *v3Client) AttachAddon(planID string, body apiv3.CreatePlanAddonRequest) (int, *apiv3.PlanAddon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPost, "/plans/"+planID+"/addons", body)
	return decodeTyped[apiv3.PlanAddon](c, status, raw, problem, http.StatusCreated)
}

func (c *v3Client) GetPlanAddon(planID, planAddonID string) (int, *apiv3.PlanAddon, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/plans/"+planID+"/addons/"+planAddonID, nil)
	return decodeTyped[apiv3.PlanAddon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) UpdatePlanAddon(planID, planAddonID string, body apiv3.UpsertPlanAddonRequest) (int, *apiv3.PlanAddon, *v3Problem) {
	status, raw, problem := c.do(http.MethodPut, "/plans/"+planID+"/addons/"+planAddonID, body)
	return decodeTyped[apiv3.PlanAddon](c, status, raw, problem, http.StatusOK)
}

func (c *v3Client) DetachAddon(planID, planAddonID string) (int, *v3Problem) {
	status, _, problem := c.do(http.MethodDelete, "/plans/"+planID+"/addons/"+planAddonID, nil)
	return status, problem
}

func (c *v3Client) ListPlanAddons(planID string, opts ...listOption) (int, *apiv3.PlanAddonPagePaginatedResponse, *v3Problem) {
	status, raw, problem := c.do(http.MethodGet, "/plans/"+planID+"/addons"+buildPageQuery(opts), nil)
	return decodeTyped[apiv3.PlanAddonPagePaginatedResponse](c, status, raw, problem, http.StatusOK)
}

// --- List pagination options ---

// listOptions controls pagination query params for list endpoints. The server
// serializes PagePaginationQuery as `?page[number]=N&page[size]=M` (deepObject
// style, explode=true — see ServerInterfaceWrapper in api.gen.go).
type listOptions struct {
	pageNumber int
	pageSize   int
}

type listOption func(*listOptions)

// withPageSize bumps the default page size (20). Useful when a test needs to
// find its own fixture in a list call against a shared DB with many rows.
func withPageSize(n int) listOption { return func(o *listOptions) { o.pageSize = n } }

// withPageNumber selects a specific page. Pages are 1-indexed.
func withPageNumber(n int) listOption { return func(o *listOptions) { o.pageNumber = n } }

func buildPageQuery(opts []listOption) string {
	var o listOptions
	for _, opt := range opts {
		opt(&o)
	}
	vals := url.Values{}
	if o.pageNumber > 0 {
		vals.Set("page[number]", strconv.Itoa(o.pageNumber))
	}
	if o.pageSize > 0 {
		vals.Set("page[size]", strconv.Itoa(o.pageSize))
	}
	s := vals.Encode()
	if s == "" {
		return ""
	}
	return "?" + s
}

// --- Fixture builders ---

// uniqueKey returns a fixture key suffix that survives re-runs against a
// shared database without collision. Format: <prefix>_<millis>_<rand>.
func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), rand.Intn(10_000))
}

// validPlanRequest returns a baseline plan that creates and publishes
// successfully. Tests mutate the returned struct before posting.
func validPlanRequest(keyPrefix string) apiv3.CreatePlanRequest {
	return apiv3.CreatePlanRequest{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Plan " + keyPrefix,
		Currency:       "USD",
		BillingCadence: apiv3.ISO8601Duration("P1M"),
		Phases:         []apiv3.BillingPlanPhase{validPlanPhase("phase_1", true /* isLast */)},
	}
}

// validPlanPhase returns a single phase with one flat rate card. If isLast is
// false the phase carries a P1M duration (non-last phases must be bounded).
func validPlanPhase(keyPrefix string, isLast bool) apiv3.BillingPlanPhase {
	phase := apiv3.BillingPlanPhase{
		Key:       uniqueKey(keyPrefix),
		Name:      "Test Phase " + keyPrefix,
		RateCards: []apiv3.BillingRateCard{validFlatRateCard("fee")},
	}
	if !isLast {
		duration := apiv3.ISO8601Duration("P1M")
		phase.Duration = &duration
	}
	return phase
}

// validFlatRateCard returns a flat, in-advance, P1M rate card — the simplest
// shape that passes plan- and addon-level validation.
func validFlatRateCard(keyPrefix string) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInAdvance

	price := apiv3.BillingPrice{}
	if err := price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: "10",
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Rate Card " + keyPrefix,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}
}

// addonMeterSlug is the pre-configured meter used for addon metered features.
// It must exist in the e2e server config (e2e/config.yaml).
const addonMeterSlug = "addon_meter"

// createTestFeature creates a metered feature via the v1 API and returns its
// (id, key). The feature references the pre-configured addon_meter so that
// non-flat rate cards referencing it pass the "feature must be metered"
// validation. The rate card Key must equal the feature Key, so callers should
// use the returned key as the rate card key.
func createTestFeature(t *testing.T, keyPrefix string) (id, key string) {
	t.Helper()

	client := initClient(t)

	ctx, cancel := context.WithTimeout(t.Context(), v3RequestTimeout)
	defer cancel()

	key = uniqueKey(keyPrefix + "_feature")
	meterSlug := addonMeterSlug
	resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
		Key:       key,
		Name:      "Test Feature " + keyPrefix,
		MeterSlug: &meterSlug,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "create feature: %s", resp.Body)
	require.NotNil(t, resp.JSON201)

	return resp.JSON201.Id, key
}

// validUnitRateCard returns a usage-based unit-priced rate card. Unit prices
// cannot use payment_term=in_advance (that's flat-only), so this uses
// in_arrears. featureID and featureKey must reference an existing metered
// feature; the rate card Key is set to featureKey (Key == FeatureKey is
// required by server validation).
func validUnitRateCard(featureID, featureKey string) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInArrears

	price := apiv3.BillingPrice{}
	if err := price.FromBillingPriceUnit(apiv3.BillingPriceUnit{
		Type:   apiv3.BillingPriceUnitTypeUnit,
		Amount: "0.10",
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            featureKey,
		Name:           "Test Unit Rate Card",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &apiv3.FeatureReferenceItem{Id: featureID},
	}
}

// validGraduatedRateCard returns a graduated tiered rate card with two tiers:
// 0–100 units at $0.10/unit and 100+ units at $0.05/unit. featureID and
// featureKey must reference an existing metered feature; the rate card Key is
// set to featureKey (Key == FeatureKey is required by server validation).
func validGraduatedRateCard(featureID, featureKey string) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInArrears

	price := apiv3.BillingPrice{}
	upTo := apiv3.Numeric("100")
	if err := price.FromBillingPriceGraduated(apiv3.BillingPriceGraduated{
		Type: apiv3.BillingPriceGraduatedTypeGraduated,
		Tiers: []apiv3.BillingPriceTier{
			{
				UpToAmount: &upTo,
				UnitPrice: &apiv3.BillingPriceUnit{
					Type:   apiv3.BillingPriceUnitTypeUnit,
					Amount: "0.10",
				},
			},
			{
				UnitPrice: &apiv3.BillingPriceUnit{
					Type:   apiv3.BillingPriceUnitTypeUnit,
					Amount: "0.05",
				},
			},
		},
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            featureKey,
		Name:           "Test Graduated Rate Card",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &apiv3.FeatureReferenceItem{Id: featureID},
	}
}

// validAddonRequest returns a baseline addon that publishes successfully.
func validAddonRequest(keyPrefix string) apiv3.CreateAddonRequest {
	return apiv3.CreateAddonRequest{
		Key:          uniqueKey(keyPrefix),
		Name:         "Test Addon " + keyPrefix,
		Currency:     "USD",
		InstanceType: apiv3.AddonInstanceTypeSingle,
		RateCards:    []apiv3.BillingRateCard{validFlatRateCard("addon_fee")},
	}
}

// validPlanAddonRequest returns a baseline attach request. The addon must be
// published and its currency/cadence compatible with the plan.
func validPlanAddonRequest(phaseKey, addonID string) apiv3.CreatePlanAddonRequest {
	return apiv3.CreatePlanAddonRequest{
		Name:          "Test Plan Addon",
		Addon:         apiv3.AddonReference{Id: addonID},
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
