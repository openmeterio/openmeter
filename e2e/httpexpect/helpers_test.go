package httpexpect_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

const requestTimeout = 30 * time.Second

// openmeterAddress returns the base server URL from $OPENMETER_ADDRESS,
// skipping the test when unset.
func openmeterAddress(t *testing.T) string {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	return strings.TrimRight(address, "/")
}

// newExpect returns an *httpexpect.Expect rooted at $OPENMETER_ADDRESS with
// require-semantics (first failure stops the test).
func newExpect(t *testing.T) *httpexpect.Expect {
	t.Helper()

	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  openmeterAddress(t),
		Reporter: httpexpect.NewRequireReporter(t),
		Client:   &http.Client{Timeout: requestTimeout},
	})
}

// newV3Expect returns an *httpexpect.Expect rooted at
// $OPENMETER_ADDRESS/api/v3/openmeter — the common prefix for all v3 endpoints.
func newV3Expect(t *testing.T) *httpexpect.Expect {
	t.Helper()

	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  openmeterAddress(t) + "/api/v3/openmeter",
		Reporter: httpexpect.NewRequireReporter(t),
		Client:   &http.Client{Timeout: requestTimeout},
	})
}

// newExpectCollect returns an *httpexpect.Expect with assert-semantics,
// suitable for use inside assert.EventuallyWithT callbacks (non-fatal failures
// allow the poller to retry).
func newExpectCollect(ct *assert.CollectT, address string) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  strings.TrimRight(address, "/") + "/api/v3/openmeter",
		Reporter: httpexpect.NewAssertReporter(ct),
		Client:   &http.Client{Timeout: requestTimeout},
	})
}

// ---------------------------------------------------------------------------
// Problem type
// ---------------------------------------------------------------------------

// Problem is the decoded shape of an application/problem+json response.
// The server uses two shapes interchangeably:
//   - Schema validation (TypeSpec/JSON Schema) returns RFC 7807 with
//     top-level invalid_parameters[].
//   - Domain validation (publish-time, create-time) returns RFC 7807 with
//     extensions.validationErrors[].
type Problem struct {
	Type              string             `json:"type"`
	Title             string             `json:"title"`
	Status            int                `json:"status"`
	Detail            string             `json:"detail"`
	Instance          string             `json:"instance"`
	Extensions        map[string]any     `json:"extensions"`
	InvalidParameters []InvalidParameter `json:"invalid_parameters"`
}

// InvalidParameter is the schema-layer validation entry.
type InvalidParameter struct {
	Field  string `json:"field"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
	Source string `json:"source"`
}

// ValidationError is the domain-layer validation entry nested under
// extensions.validationErrors.
type ValidationError struct {
	Code     string `json:"code"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ValidationErrors extracts domain validation errors from
// extensions.validationErrors. Returns nil when absent.
func (p *Problem) ValidationErrors() []ValidationError {
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
	var out []ValidationError
	_ = json.Unmarshal(b, &out)
	return out
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

// assertValidationCode asserts that the problem carries a domain ValidationError
// with the given code in extensions.validationErrors.
func assertValidationCode(t *testing.T, prob *Problem, code string) {
	t.Helper()
	require.NotNil(t, prob, "expected problem response")
	errs := prob.ValidationErrors()
	require.NotEmpty(t, errs, "no validation errors in problem: %+v", prob)
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

// decodeProblem decodes an application/problem+json response body.
// httpexpect's .JSON() rejects non-"application/json" content types, so we
// read the raw body and unmarshal ourselves.
func decodeProblem(resp *httpexpect.Response) *Problem {
	var p Problem
	_ = json.Unmarshal([]byte(resp.Body().Raw()), &p)
	return &p
}

// ---------------------------------------------------------------------------
// Fixture builders (mirrors e2e/v3helpers_test.go)
// ---------------------------------------------------------------------------

// uniqueKey returns a collision-safe fixture key for shared-DB test runs.
func uniqueKey(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), rand.Intn(10_000))
}

func validPlanRequest(keyPrefix string) apiv3.CreatePlanRequest {
	return apiv3.CreatePlanRequest{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Plan " + keyPrefix,
		Currency:       "USD",
		BillingCadence: apiv3.ISO8601Duration("P1M"),
		Phases:         []apiv3.BillingPlanPhase{validPlanPhase("phase_1", true)},
	}
}

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

func validUnitRateCard(keyPrefix string) apiv3.BillingRateCard {
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
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Unit Rate Card " + keyPrefix,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}
}

func validUsageRateCard(keyPrefix, featureID string) apiv3.BillingRateCard {
	rc := validUnitRateCard(keyPrefix)
	rc.Feature = &apiv3.FeatureReferenceItem{Id: featureID}
	return rc
}

func validGraduatedRateCard(keyPrefix string) apiv3.BillingRateCard {
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
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Graduated Rate Card " + keyPrefix,
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
	}
}

func validAddonRequest(keyPrefix string) apiv3.CreateAddonRequest {
	return apiv3.CreateAddonRequest{
		Key:          uniqueKey(keyPrefix),
		Name:         "Test Addon " + keyPrefix,
		Currency:     "USD",
		InstanceType: apiv3.AddonInstanceTypeSingle,
		RateCards:    []apiv3.BillingRateCard{validFlatRateCard("addon_fee")},
	}
}

func validPlanAddonRequest(phaseKey, addonID string) apiv3.CreatePlanAddonRequest {
	return apiv3.CreatePlanAddonRequest{
		Name:          "Test Plan Addon",
		Addon:         apiv3.AddonReference{Id: addonID},
		FromPlanPhase: phaseKey,
	}
}
