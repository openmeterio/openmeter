package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

func TestCustomerList(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create multiple customers with varying properties to test different filter scenarios
	var customers []*api.Customer

	// Customer 1: Tech startup with subscription
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_tech_startup"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		customerResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("tech-startup-001"),
			Name:         "Tech Startup Inc",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			Description:  lo.ToPtr("A technology startup company"),
			PrimaryEmail: lo.ToPtr("contact@techstartup.com"),
			BillingAddress: &api.Address{
				City:       lo.ToPtr("San Francisco"),
				Country:    lo.ToPtr("US"),
				Line1:      lo.ToPtr("123 Market St"),
				State:      lo.ToPtr("CA"),
				PostalCode: lo.ToPtr("94102"),
			},
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_tech_startup"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "body: %s", customerResp.Body)
		customers = append(customers, customerResp.JSON201)
	}

	// Customer 2: E-commerce company
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_ecommerce"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		customerResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("ecommerce-corp-002"),
			Name:         "E-Commerce Corp",
			Currency:     lo.ToPtr(api.CurrencyCode("EUR")),
			Description:  lo.ToPtr("An e-commerce platform"),
			PrimaryEmail: lo.ToPtr("admin@ecommerce-corp.com"),
			BillingAddress: &api.Address{
				City:       lo.ToPtr("Berlin"),
				Country:    lo.ToPtr("DE"),
				Line1:      lo.ToPtr("456 Commerce Ave"),
				PostalCode: lo.ToPtr("10115"),
			},
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_ecommerce"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "body: %s", customerResp.Body)
		customers = append(customers, customerResp.JSON201)
	}

	// Customer 3: Enterprise client with multiple subjects
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_enterprise_main"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		resp, err = client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_enterprise_backup"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		customerResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("enterprise-client-003"),
			Name:         "Enterprise Solutions Ltd",
			Currency:     lo.ToPtr(api.CurrencyCode("GBP")),
			Description:  lo.ToPtr("Large enterprise client"),
			PrimaryEmail: lo.ToPtr("billing@enterprise.co.uk"),
			BillingAddress: &api.Address{
				City:       lo.ToPtr("London"),
				Country:    lo.ToPtr("GB"),
				Line1:      lo.ToPtr("789 Business Rd"),
				PostalCode: lo.ToPtr("SW1A 1AA"),
			},
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_enterprise_main", "subject_enterprise_backup"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "body: %s", customerResp.Body)
		customers = append(customers, customerResp.JSON201)
	}

	// Customer 4: Tech company (for name filtering)
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_tech_company"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		customerResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("tech-innovations-004"),
			Name:         "Tech Innovations LLC",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			Description:  lo.ToPtr("Technology innovations company"),
			PrimaryEmail: lo.ToPtr("info@techinnovations.com"),
			BillingAddress: &api.Address{
				City:       lo.ToPtr("Austin"),
				Country:    lo.ToPtr("US"),
				Line1:      lo.ToPtr("321 Innovation Blvd"),
				State:      lo.ToPtr("TX"),
				PostalCode: lo.ToPtr("78701"),
			},
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_tech_company"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "body: %s", customerResp.Body)
		customers = append(customers, customerResp.JSON201)
	}

	// Customer 5: Small business (for email domain filtering)
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_small_business"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		customerResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("small-biz-005"),
			Name:         "Small Business Co",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("owner@smallbiz.com"),
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_small_business"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, customerResp.StatusCode(), "body: %s", customerResp.Body)
		customers = append(customers, customerResp.JSON201)
	}

	t.Run("Should list all customers with default pagination", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Should have at least our 5 customers
		assert.GreaterOrEqual(t, len(resp.JSON200.Items), 5)
		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 5)
	})

	t.Run("Should filter customers by key - exact match", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key: lo.ToPtr("tech-startup-001"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.Equal(t, 1, resp.JSON200.TotalCount)
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, "tech-startup-001", *resp.JSON200.Items[0].Key)
		assert.Equal(t, "Tech Startup Inc", resp.JSON200.Items[0].Name)
	})

	t.Run("Should filter customers by key - partial match", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key: lo.ToPtr("tech"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Should match "tech-startup-001" and "tech-innovations-004"
		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 2)
		for _, customer := range resp.JSON200.Items {
			assert.Contains(t, *customer.Key, "tech")
		}
	})

	t.Run("Should filter customers by name - partial match", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Name: lo.ToPtr("Tech"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Should match "Tech Startup Inc" and "Tech Innovations LLC"
		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 2)
		for _, customer := range resp.JSON200.Items {
			assert.Contains(t, customer.Name, "Tech")
		}
	})

	t.Run("Should filter customers by name - case insensitive", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Name: lo.ToPtr("enterprise"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.Name == "Enterprise Solutions Ltd" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find Enterprise Solutions Ltd")
	})

	t.Run("Should filter customers by primary email - exact domain", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			PrimaryEmail: lo.ToPtr("techstartup.com"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.PrimaryEmail != nil && *customer.PrimaryEmail == "contact@techstartup.com" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find customer with techstartup.com email")
	})

	t.Run("Should filter customers by primary email - partial match", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			PrimaryEmail: lo.ToPtr("ecommerce"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.PrimaryEmail != nil && *customer.PrimaryEmail == "admin@ecommerce-corp.com" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find E-Commerce Corp")
	})

	t.Run("Should filter customers by subject - single subject", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Subject: lo.ToPtr("subject_tech_startup"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.Name == "Tech Startup Inc" {
				found = true
				assert.Contains(t, customer.UsageAttribution.SubjectKeys, "subject_tech_startup")
				break
			}
		}
		assert.True(t, found, "Should find Tech Startup Inc")
	})

	t.Run("Should filter customers by subject - partial match", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Subject: lo.ToPtr("enterprise"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Should match "subject_enterprise_main" and "subject_enterprise_backup"
		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.Name == "Enterprise Solutions Ltd" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find Enterprise Solutions Ltd")
	})

	t.Run("Should order customers by name ascending", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			OrderBy: lo.ToPtr(api.CustomerOrderByName),
			Order:   lo.ToPtr(api.SortOrderASC),
			Name:    lo.ToPtr("Tech"), // Filter to limit results
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		require.GreaterOrEqual(t, len(resp.JSON200.Items), 2)
		// Verify ordering
		for i := 0; i < len(resp.JSON200.Items)-1; i++ {
			assert.LessOrEqual(t, resp.JSON200.Items[i].Name, resp.JSON200.Items[i+1].Name,
				"Customers should be ordered by name ascending")
		}
	})

	t.Run("Should order customers by name descending", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			OrderBy: lo.ToPtr(api.CustomerOrderByName),
			Order:   lo.ToPtr(api.SortOrderDESC),
			Name:    lo.ToPtr("Tech"), // Filter to limit results
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		require.GreaterOrEqual(t, len(resp.JSON200.Items), 2)
		// Verify ordering
		for i := 0; i < len(resp.JSON200.Items)-1; i++ {
			assert.GreaterOrEqual(t, resp.JSON200.Items[i].Name, resp.JSON200.Items[i+1].Name,
				"Customers should be ordered by name descending")
		}
	})

	t.Run("Should order customers by createdAt", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			OrderBy: lo.ToPtr(api.CustomerOrderByCreatedAt),
			Order:   lo.ToPtr(api.SortOrderASC),
			Key:     lo.ToPtr("tech"), // Filter to our test customers
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Verify ordering
		for i := 1; i < len(resp.JSON200.Items); i++ {
			prev := resp.JSON200.Items[i-1].CreatedAt
			curr := resp.JSON200.Items[i].CreatedAt
			assert.False(t, curr.Before(prev), "createdAt should be in ascending order")
		}
	})

	t.Run("Should respect page size parameter", func(t *testing.T) {
		pageSize := 2
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			PageSize: lo.ToPtr(pageSize),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.LessOrEqual(t, len(resp.JSON200.Items), pageSize)
		assert.Equal(t, pageSize, resp.JSON200.PageSize)
	})

	t.Run("Should paginate through customers", func(t *testing.T) {
		pageSize := 2
		page1Resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Page:     lo.ToPtr(1),
			PageSize: lo.ToPtr(pageSize),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, page1Resp.StatusCode(), "body: %s", page1Resp.Body)
		require.NotNil(t, page1Resp.JSON200)

		page2Resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Page:     lo.ToPtr(2),
			PageSize: lo.ToPtr(pageSize),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, page2Resp.StatusCode(), "body: %s", page2Resp.Body)
		require.NotNil(t, page2Resp.JSON200)

		// Pages should have same total count
		assert.Equal(t, page1Resp.JSON200.TotalCount, page2Resp.JSON200.TotalCount)

		// Items should not overlap across pages
		page1IDs := lo.Map(page1Resp.JSON200.Items, func(c api.Customer, _ int) string { return c.Id })
		page2IDs := lo.Map(page2Resp.JSON200.Items, func(c api.Customer, _ int) string { return c.Id })
		overlap := lo.Intersect(page1IDs, page2IDs)
		assert.Len(t, overlap, 0, "items should not repeat across pages")
	})

	t.Run("Should combine multiple filters", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Name:    lo.ToPtr("Tech"),
			Subject: lo.ToPtr("tech_startup"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		// Should only match "Tech Startup Inc" with subject_tech_startup
		found := false
		for _, customer := range resp.JSON200.Items {
			if customer.Name == "Tech Startup Inc" {
				found = true
				assert.Contains(t, customer.UsageAttribution.SubjectKeys, "subject_tech_startup")
			}
		}
		assert.True(t, found, "Should find Tech Startup Inc with combined filters")
	})

	t.Run("Should return empty list for non-existent filter", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key: lo.ToPtr("non-existent-customer-key-12345"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		assert.Equal(t, 0, resp.JSON200.TotalCount)
		assert.Len(t, resp.JSON200.Items, 0)
	})

	// Test deleted customers
	t.Run("Should filter deleted customers", func(t *testing.T) {
		// Create a customer to delete
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: "subject_to_delete"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		createResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:          lo.ToPtr("customer-to-delete"),
			Name:         "Customer To Delete",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("delete@test.com"),
			UsageAttribution: api.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_to_delete"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createResp.StatusCode(), "body: %s", createResp.Body)
		customerToDelete := createResp.JSON201

		// Delete the customer
		deleteResp, err := client.DeleteCustomerWithResponse(ctx, customerToDelete.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleteResp.StatusCode(), "body: %s", deleteResp.Body)

		// List without includeDeleted should not show the deleted customer
		listResp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key: lo.ToPtr("customer-to-delete"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "body: %s", listResp.Body)
		require.NotNil(t, listResp.JSON200)
		assert.Equal(t, 0, listResp.JSON200.TotalCount, "Deleted customer should not appear by default")

		// List with includeDeleted should show the deleted customer
		listWithDeletedResp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key:            lo.ToPtr("customer-to-delete"),
			IncludeDeleted: lo.ToPtr(true),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listWithDeletedResp.StatusCode(), "body: %s", listWithDeletedResp.Body)
		require.NotNil(t, listWithDeletedResp.JSON200)
		assert.Equal(t, 1, listWithDeletedResp.JSON200.TotalCount, "Deleted customer should appear with includeDeleted=true")
		require.Len(t, listWithDeletedResp.JSON200.Items, 1)
		assert.NotNil(t, listWithDeletedResp.JSON200.Items[0].DeletedAt, "Customer should have deletedAt timestamp")
	})

	// Test plan key filter (requires creating a plan and subscription)
	t.Run("Should filter customers by plan key", func(t *testing.T) {
		featureKey := "customer_list_test_feature"
		planKey := "customer_list_test_plan"
		rateCardKey := "test_rate_card_customer_list"

		// Create a simple feature for the plan
		featureResp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Key:  featureKey,
			Name: "Customer List Test Feature",
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, featureResp.StatusCode(), "body: %s", featureResp.Body)

		// Cleanup: Delete the feature after the test
		t.Cleanup(func() {
			_, _ = client.DeleteFeatureWithResponse(ctx, featureResp.JSON201.Id)
		})

		// Create a simple plan
		p1RC1 := api.RateCard{}
		err = p1RC1.FromRateCardFlatFee(api.RateCardFlatFee{
			Name: "Test Rate Card",
			Key:  rateCardKey,
			Price: &api.FlatPriceWithPaymentTerm{
				Amount:      "100",
				PaymentTerm: lo.ToPtr(api.PricePaymentTerm("in_advance")),
				Type:        api.FlatPriceWithPaymentTermType("flat"),
			},
			BillingCadence: lo.ToPtr("P1M"),
			Type:           api.RateCardFlatFeeType("flat"),
		})
		require.NoError(t, err)

		planResp, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            planKey,
			Name:           "Customer List Test Plan",
			Currency:       api.CurrencyCode("USD"),
			BillingCadence: "P1M",
			Phases: []api.PlanPhase{
				{
					Name:      "Test Phase",
					Key:       "test_phase",
					RateCards: []api.RateCard{p1RC1},
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, planResp.StatusCode(), "body: %s", planResp.Body)
		plan := planResp.JSON201

		// Cleanup: Archive the plan after the test
		t.Cleanup(func() {
			_, _ = client.ArchivePlanWithResponse(ctx, plan.Id)
		})

		// Publish the plan
		publishResp, err := client.PublishPlanWithResponse(ctx, plan.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, publishResp.StatusCode(), "body: %s", publishResp.Body)

		// Create a subscription for the first customer
		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		subCreate := api.SubscriptionCreate{}
		err = subCreate.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			Timing:     ct,
			CustomerId: &customers[0].Id,
			Plan: api.PlanReferenceInput{
				Key:     plan.Key,
				Version: lo.ToPtr(1),
			},
		})
		require.NoError(t, err)

		subResp, err := client.CreateSubscriptionWithResponse(ctx, subCreate)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, subResp.StatusCode(), "body: %s", subResp.Body)

		// Now filter customers by plan key
		listResp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			PlanKey: lo.ToPtr(planKey),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "body: %s", listResp.Body)
		require.NotNil(t, listResp.JSON200)

		// Should find at least the customer with the subscription
		assert.GreaterOrEqual(t, listResp.JSON200.TotalCount, 1)
		found := false
		for _, customer := range listResp.JSON200.Items {
			if customer.Id == customers[0].Id {
				found = true
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("Should find customer %s with plan subscription", customers[0].Name))
	})

	t.Run("Should expand subscriptions when requested", func(t *testing.T) {
		resp, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			Key:    lo.ToPtr("tech-startup-001"),
			Expand: &api.QueryCustomerListExpand{api.CustomerExpandSubscriptions},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.Body)
		require.NotNil(t, resp.JSON200)

		require.Len(t, resp.JSON200.Items, 1)
		customer := resp.JSON200.Items[0]

		// Customer should have subscriptions field populated
		assert.NotNil(t, customer.Subscriptions, "Subscriptions should be populated when expand is requested")
		assert.Greater(t, len(lo.FromPtr(customer.Subscriptions)), 0, "Customer should have at least one subscription expanded")
	})
}
