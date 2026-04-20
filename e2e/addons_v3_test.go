package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

func openmeterAddress(t *testing.T) string {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	return strings.TrimRight(address, "/") + "/api/v3"
}

func doV3Addon(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, url, bodyReader)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, raw
}

func TestV3Addon(t *testing.T) {
	baseURL := openmeterAddress(t)

	addonV3URL := func(parts ...string) string {
		base := baseURL + "/openmeter/addons"
		if len(parts) > 0 {
			return base + "/" + strings.Join(parts, "/")
		}

		return base
	}

	addonKey := fmt.Sprintf("test_v3_addon_%d", time.Now().UnixMilli())

	billingCadence := apiv3.ISO8601Duration("P1M")
	paymentTerm := apiv3.BillingPricePaymentTermInAdvance

	price := apiv3.BillingPrice{}
	require.NoError(t, price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: "10",
	}))

	rateCards := []apiv3.BillingRateCard{
		{
			Key:            "addon_fee",
			Name:           "Addon Fee",
			Price:          price,
			BillingCadence: &billingCadence,
			PaymentTerm:    &paymentTerm,
		},
	}

	addonBody := apiv3.CreateAddonRequest{
		Key:          addonKey,
		Name:         "Test V3 Addon",
		Currency:     "USD",
		InstanceType: apiv3.AddonInstanceTypeSingle,
		RateCards:    rateCards,
	}

	var addonID string

	t.Run("Should create an addon in draft status", func(t *testing.T) {
		status, raw := doV3Addon(t, http.MethodPost, addonV3URL(), addonBody)
		require.Equal(t, http.StatusCreated, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.Equal(t, addonKey, addon.Key)
		assert.Equal(t, 1, addon.Version)
		assert.Equal(t, apiv3.AddonStatusDraft, addon.Status)
		assert.Nil(t, addon.EffectiveFrom)

		addonID = addon.Id
	})

	t.Run("Should get the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodGet, addonV3URL( addonID), nil)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.Equal(t, addonID, addon.Id)
		assert.Equal(t, apiv3.AddonStatusDraft, addon.Status)
	})

	t.Run("Should list addons and find the created addon", func(t *testing.T) {
		status, raw := doV3Addon(t, http.MethodGet, addonV3URL(), nil)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var page apiv3.AddonPagePaginatedResponse
		require.NoError(t, json.Unmarshal(raw, &page))

		found := false
		for _, a := range page.Data {
			if a.Id == addonID {
				found = true
				assert.NotEmpty(t, a.Currency)
				assert.NotEmpty(t, a.Status)
				assert.NotEmpty(t, a.RateCards)
				break
			}
		}
		assert.True(t, found, "created addon not found in list")
	})

	t.Run("Should update the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		updateBody := apiv3.UpsertAddonRequest{
			Name:         "Test V3 Addon Updated",
			InstanceType: apiv3.AddonInstanceTypeSingle,
			RateCards:    rateCards,
		}

		status, raw := doV3Addon(t, http.MethodPut, addonV3URL( addonID), updateBody)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.Equal(t, "Test V3 Addon Updated", addon.Name)
		assert.Equal(t, apiv3.AddonStatusDraft, addon.Status)
	})

	t.Run("Should publish the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodPost, addonV3URL( addonID, "publish"), nil)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.Equal(t, apiv3.AddonStatusActive, addon.Status)
		assert.NotNil(t, addon.EffectiveFrom)
	})

	t.Run("Should not allow deleting an active addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodDelete, addonV3URL( addonID), nil)
		assert.Equal(t, http.StatusBadRequest, status, "body: %s", raw)
	})

	t.Run("Should archive the published addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodPost, addonV3URL( addonID, "archive"), nil)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.Equal(t, apiv3.AddonStatusArchived, addon.Status)
		assert.NotNil(t, addon.EffectiveTo)
	})

	t.Run("Should delete an archived addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodDelete, addonV3URL( addonID), nil)
		assert.Equal(t, http.StatusNoContent, status, "body: %s", raw)
	})

	t.Run("Should return deleted_at after deletion", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		status, raw := doV3Addon(t, http.MethodGet, addonV3URL( addonID), nil)
		require.Equal(t, http.StatusOK, status, "body: %s", raw)

		var addon apiv3.Addon
		require.NoError(t, json.Unmarshal(raw, &addon))

		assert.NotNil(t, addon.DeletedAt)
	})
}
